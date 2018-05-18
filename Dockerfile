FROM golang:1.9.2 as builder
WORKDIR /go/src/github.com/nimbus-cloud/isilon-nfs-broker
COPY . .
RUN go get github.com/tools/godep
RUN godep restore
RUN GOOS=linux GOARCH=amd64 go build -o bin/nfsbroker

FROM busybox:ubuntu-14.04
WORKDIR /root/
COPY --from=builder /go/src/github.com/nimbus-cloud/isilon-nfs-broker/bin/nfsbroker .
CMD ./nfsbroker --listenAddr="0.0.0.0:$PORT" --serviceName="$SERVICENAME" --serviceId="nfsbroker" --dbDriver="$DBDRIVERNAME" --cfServiceName="$DBSERVICENAME" --dbHostname="$DBHOST" --dbPort="$DBPORT" --dbName="$DBNAME" --dbCACert="$DBCACERT" --logLevel="$LOGLEVEL" --allowedOptions="$ALLOWED_OPTIONS"
