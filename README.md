# nfsbroker
A Cloud Foundry service broker for Dell EMC Isilon nfsv3 shares.

For details on how to use this broker, please refer to [the nfs-volume-release README](https://github.com/cloudfoundry/nfs-volume-release)

Build: 
```
 GOOS=linux GOARCH=amd64 go build -o bin/nfsbroker
```