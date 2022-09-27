# Collector Agent and Gateway walk through
In this example we will deploy the collector in agent mode and gateway mode only.
See
[splunk-otel-collector gateway architecture](https://github.com/signalfx/splunk-otel-collector/blob/main/docs/architecture.md#advanced)
for more details about.
### 1. Complete the [Getting Started](https://github.com/signalfx/splunk-otel-collector-operator#getting-started) steps

### 2. Deploy the spring-petclinic (cloud version) project
Original Instructions: [spring-petclinic-cloud Setting Things Up In Kubernetes](https://github.com/spring-petclinic/spring-petclinic-cloud#setting-things-up-in-kubernetes)


```console
$ kubectl apply -f - <<EOF
apiVersion: otel.splunk.com/v1alpha1
kind: Agent
metadata:
  name: splunk-otel
  namespace: splunk-otel-operator-system
spec:
  clusterName: <MY_CLUSTER_NAME>
  realm: <SPLUNK_REALM>
  agent:
    enabled: true
  clusterReceiver:
    enabled: false
  gateway:
    enabled: true
    resources:
      limits:
        cpu: 200m
        memory: 200Mi
      requests:
        memory: 100Mi
        cpu: 100m
    replicas: 1
EOF
```
