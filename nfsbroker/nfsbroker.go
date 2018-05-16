package nfsbroker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"

	"crypto/md5"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/service-broker-store/brokerstore"
	"github.com/pivotal-cf/brokerapi"
	"github.com/thecodeteam/goisilon"
)

const (
	PermissionVolumeMount = brokerapi.RequiredPermission("volume_mount")
	DefaultContainerPath  = "/var/vcap/data"
)

const (
	Username string = "kerberosPrincipal"
	Secret   string = "kerberosKeytab"
)

const (
	_        = iota // ignore first value
	KB int64 = 1 << (10 * iota)
	MB
	GB
)

type staticState struct {
	ServiceName string `json:"ServiceName"`
	ServiceId   string `json:"ServiceId"`
}

type lock interface {
	Lock()
	Unlock()
}

type clientConfig struct {
}

type Broker struct {
	logger  lager.Logger
	dataDir string
	os      osshim.Os
	mutex   lock
	clock   clock.Clock
	static  staticState
	store   brokerstore.Store
	config  Config
	isilonClientConfig
}

type isilonClientConfig struct {
	insecure string
	endpoint string
	username string
	password string
	group    string
	volPath  string
}

func New(
	logger lager.Logger,
	serviceName, serviceId, dataDir string,
	os osshim.Os,
	clock clock.Clock,
	store brokerstore.Store,
	config *Config,
	isilConf map[string]string,
) *Broker {

	theBroker := Broker{
		logger:  logger,
		dataDir: dataDir,
		os:      os,
		mutex:   &sync.Mutex{},
		clock:   clock,
		store:   store,
		static: staticState{
			ServiceName: serviceName,
			ServiceId:   serviceId,
		},
		config: *config,
		isilonClientConfig: isilonClientConfig{
			isilConf["insecure"],
			isilConf["endpoint"],
			isilConf["username"],
			isilConf["password"],
			isilConf["group"],
			isilConf["volpath"],
		},
	}

	theBroker.store.Restore(logger)

	return &theBroker
}

func (b *Broker) Services(_ context.Context) []brokerapi.Service {
	logger := b.logger.Session("services")
	logger.Info("start")
	defer logger.Info("end")

	return []brokerapi.Service{{
		ID:            b.static.ServiceId,
		Name:          b.static.ServiceName,
		Description:   "DELL EMC Isilon",
		Bindable:      true,
		PlanUpdatable: false,
		Tags:          []string{"nfs", "isilon"},
		Requires:      []brokerapi.RequiredPermission{PermissionVolumeMount},

		Plans: []brokerapi.ServicePlan{
			{
				ID:          "5",
				Name:        "5GB",
				Description: "5GB Dell EMC Isilon NFS Share.",
			},
			{
				ID:          "10",
				Name:        "10GB",
				Description: "10GB Dell EMC Isilon NFS Share.",
			},
		},
	}}
}

func (b *Broker) Provision(context context.Context, instanceID string, details brokerapi.ProvisionDetails, asyncAllowed bool) (_ brokerapi.ProvisionedServiceSpec, e error) {
	logger := b.logger.Session("provision").WithData(lager.Data{"instanceID": instanceID, "details": details})
	logger.Info("start")
	defer logger.Info("end")

	if b.isilonClientConfig.insecure == "" {
		b.isilonClientConfig.insecure = "false"
	} // set default to false

	cliIsInsecure, _ := strconv.ParseBool(b.isilonClientConfig.insecure)
	client, e := goisilon.NewClientWithArgs(
		context,
		b.endpoint,
		cliIsInsecure,
		b.username,
		b.group,
		b.password,
		b.volPath)
	if e != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("failed to create isilon client %s with error %s", instanceID, e)
	}

	// Create Volume
	_, e = client.CreateVolume(context, instanceID)
	if e != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("failed to create isilon volume %s with error %s", instanceID, e)
	}

	// Create Export
	_, e = client.ExportVolume(context, instanceID)
	if e != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("failed to create isilon export %s with error %s", instanceID, e)
	}

	// Create Quota
	n, e := strconv.ParseInt(details.PlanID, 10, 64)
	if e != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("failed to convert plan size to bytes for plan - %s", details.PlanID)
	}
	size := n * GB
	if size == 0 {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("plan size must be greater than 0 bytes")
	}
	e = client.SetQuotaSize(context, instanceID, size)
	if e != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("failed to set isilon quota for %s with error %s", instanceID, e)
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	defer func() {
		out := b.store.Save(logger)
		if e == nil {
			e = out
		}
	}()

	volumePath := os.Getenv("GOISILON_VOLUMEPATH") + "/" + instanceID
	instanceDetails := brokerstore.ServiceInstance{
		details.ServiceID,
		details.PlanID,
		details.OrganizationGUID,
		details.SpaceGUID,
		volumePath}

	if b.instanceConflicts(instanceDetails, instanceID) {
		return brokerapi.ProvisionedServiceSpec{}, brokerapi.ErrInstanceAlreadyExists
	}

	e = b.store.CreateInstanceDetails(instanceID, instanceDetails)
	if e != nil {
		return brokerapi.ProvisionedServiceSpec{}, fmt.Errorf("failed to store instance details %s", instanceID)
	}

	logger.Info("service-instance-created", lager.Data{"instanceDetails": instanceDetails})

	return brokerapi.ProvisionedServiceSpec{IsAsync: false}, nil
}

func (b *Broker) Deprovision(context context.Context, instanceID string, details brokerapi.DeprovisionDetails, asyncAllowed bool) (_ brokerapi.DeprovisionServiceSpec, e error) {
	logger := b.logger.Session("deprovision")
	logger.Info("start")
	defer logger.Info("end")

	if b.insecure == "" {
		b.insecure = "false"
	} // set default to false

	cliIsInsecure, _ := strconv.ParseBool(b.insecure)
	client, e := goisilon.NewClientWithArgs(
		context,
		b.endpoint,
		cliIsInsecure,
		b.username,
		b.group,
		b.password,
		b.volPath)
	if e != nil {
		return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("failed to delete isilon client %s with error %s", instanceID, e)
	}

	// Delete Export
	e = client.UnexportVolume(context, instanceID)
	if e != nil {
		return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("failed to delete isilon export %s with error %s", instanceID, e)
	}

	// Delete Quota
	e = client.ClearQuota(context, instanceID)
	if e != nil {
		return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("failed to unset isilon quota for %s with error %s", instanceID, e)
	}

	// Delete Volume
	e = client.DeleteVolume(context, instanceID)
	if e != nil {
		return brokerapi.DeprovisionServiceSpec{}, fmt.Errorf("failed to delete isilon volume %s with error %s", instanceID, e)
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()
	defer func() {
		out := b.store.Save(logger)
		if e == nil {
			e = out
		}
	}()

	_, err := b.store.RetrieveInstanceDetails(instanceID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, brokerapi.ErrInstanceDoesNotExist
	}

	err = b.store.DeleteInstanceDetails(instanceID)
	if err != nil {
		return brokerapi.DeprovisionServiceSpec{}, err
	}

	return brokerapi.DeprovisionServiceSpec{IsAsync: false, OperationData: "deprovision"}, nil
}

func (b *Broker) Bind(context context.Context, instanceID string, bindingID string, bindDetails brokerapi.BindDetails) (_ brokerapi.Binding, e error) {
	logger := b.logger.Session("bind")
	logger.Info("start", lager.Data{"bindingID": bindingID, "details": bindDetails})
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()
	defer func() {
		out := b.store.Save(logger)
		if e == nil {
			e = out
		}
	}()

	logger.Info("starting-nfsbroker-bind")
	instanceDetails, err := b.store.RetrieveInstanceDetails(instanceID)
	if err != nil {
		return brokerapi.Binding{}, brokerapi.ErrInstanceDoesNotExist
	}

	if bindDetails.AppGUID == "" {
		return brokerapi.Binding{}, brokerapi.ErrAppGuidNotProvided
	}

	var opts map[string]interface{}
	if err := json.Unmarshal(bindDetails.RawParameters, &opts); err != nil {
		return brokerapi.Binding{}, err
	}
	mode, err := evaluateMode(opts)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	if b.bindingConflicts(bindingID, bindDetails) {
		return brokerapi.Binding{}, brokerapi.ErrBindingAlreadyExists
	}

	logger.Info("retrieved-instance-details", lager.Data{"instanceDetails": instanceDetails})

	err = b.store.CreateBindingDetails(bindingID, bindDetails)
	if err != nil {
		return brokerapi.Binding{}, err
	}

	source := fmt.Sprintf("nfs://%s", instanceDetails.ServiceFingerPrint)

	// TODO--brokerConfig is not re-entrant because it stores state in SetEntries--we should modify it to
	// TODO--be stateless.  Until we do that, we will just make a local copy, but we should really
	// TODO--refactor this to something more efficient.
	tempConfig := b.config.Copy()
	if err := tempConfig.SetEntries(logger, source, opts, []string{
		"share", "mount", "kerberosPrincipal", "kerberosKeytab", "readonly",
	}); err != nil {
		logger.Info("parameters-error-assign-entries", lager.Data{
			"given_source":  source,
			"given_options": opts,
			"mount":         tempConfig.mount,
			"sloppy_mount":  tempConfig.sloppyMount,
		})
		return brokerapi.Binding{}, err
	}

	mountConfig := tempConfig.MountConfig()
	mountConfig["source"] = tempConfig.Share(source)
	if mode == "r" {
		mountConfig["readonly"] = true
		mode = "rw"
	}

	logger.Info("volume-service-binding", lager.Data{"Driver": "nfsv3driver", "mountConfig": mountConfig, "source": source})

	s, err := b.hash(mountConfig)
	if err != nil {
		logger.Error("error-calculating-volume-id", err, lager.Data{"config": mountConfig, "bindingID": bindingID, "instanceID": instanceID})
		return brokerapi.Binding{}, err
	}
	volumeId := fmt.Sprintf("%s-%s", instanceID, s)

	ret := brokerapi.Binding{
		Credentials: struct{}{}, // if nil, cloud controller chokes on response
		VolumeMounts: []brokerapi.VolumeMount{{
			ContainerDir: evaluateContainerPath(opts, instanceID),
			Mode:         mode,
			Driver:       "nfsv3driver",
			DeviceType:   "shared",
			Device: brokerapi.SharedDevice{
				VolumeId:    volumeId,
				MountConfig: mountConfig,
			},
		}},
	}
	return ret, nil
}

func (b *Broker) hash(mountConfig map[string]interface{}) (string, error) {
	var (
		bytes []byte
		err   error
	)
	if bytes, err = json.Marshal(mountConfig); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(bytes)), nil
}

func (b *Broker) Unbind(context context.Context, instanceID string, bindingID string, details brokerapi.UnbindDetails) (e error) {
	logger := b.logger.Session("unbind")
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()
	defer func() {
		out := b.store.Save(logger)
		if e == nil {
			e = out
		}
	}()

	if _, err := b.store.RetrieveInstanceDetails(instanceID); err != nil {
		return brokerapi.ErrInstanceDoesNotExist
	}

	if _, err := b.store.RetrieveBindingDetails(bindingID); err != nil {
		return brokerapi.ErrBindingDoesNotExist
	}

	if err := b.store.DeleteBindingDetails(bindingID); err != nil {
		return err
	}
	return nil
}

func (b *Broker) Update(context context.Context, instanceID string, details brokerapi.UpdateDetails, asyncAllowed bool) (brokerapi.UpdateServiceSpec, error) {
	panic("not implemented")
}

func (b *Broker) LastOperation(_ context.Context, instanceID string, operationData string) (brokerapi.LastOperation, error) {
	logger := b.logger.Session("last-operation").WithData(lager.Data{"instanceID": instanceID})
	logger.Info("start")
	defer logger.Info("end")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	switch operationData {
	default:
		return brokerapi.LastOperation{}, errors.New("unrecognized operationData")
	}
}

func (b *Broker) instanceConflicts(details brokerstore.ServiceInstance, instanceID string) bool {
	return b.store.IsInstanceConflict(instanceID, brokerstore.ServiceInstance(details))
}

func (b *Broker) bindingConflicts(bindingID string, details brokerapi.BindDetails) bool {
	return b.store.IsBindingConflict(bindingID, details)
}

func evaluateContainerPath(parameters map[string]interface{}, volId string) string {
	if containerPath, ok := parameters["mount"]; ok && containerPath != "" {
		return containerPath.(string)
	}

	return path.Join(DefaultContainerPath, volId)
}

func evaluateMode(parameters map[string]interface{}) (string, error) {
	if ro, ok := parameters["readonly"]; ok {
		switch ro := ro.(type) {
		case bool:
			return readOnlyToMode(ro), nil
		default:
			return "", brokerapi.ErrRawParamsInvalid
		}
	}
	return "rw", nil
}

func readOnlyToMode(ro bool) string {
	if ro {
		return "r"
	}
	return "rw"
}
