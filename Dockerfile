FROM busybox:ubuntu-14.04

COPY bin/nfsbroker /nfsbroker
COPY nb-config /app/nb-config
RUN chmod a+x /nfsbroker

CMD /nfsbroker --listenAddr="0.0.0.0:$PORT" --serviceName="$SERVICENAME" --serviceId="nfsbroker" --dbDriver="$DBDRIVERNAME" --cfServiceName="$DBSERVICENAME" --dbHostname="$DBHOST" --dbPort="$DBPORT" --dbName="$DBNAME" --dbCACert="$DBCACERT" --logLevel="$LOGLEVEL" --allowedOptions="$ALLOWED_OPTIONS"
