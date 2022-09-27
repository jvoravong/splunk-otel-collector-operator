// Copyright Splunk Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"fmt"
	"regexp"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/getter"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var defaultslog = logf.Log.WithName("defaults-resource")

func newEnvVar(name, value string) v1.EnvVar {
	return v1.EnvVar{
		Name:  name,
		Value: value,
	}
}

func newEnvVarWithFieldRef(name, path string) v1.EnvVar {
	return v1.EnvVar{
		Name: name,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  path,
			},
		},
	}
}

func getChartManifestString(r *Agent, k8sTypes string) runtime.Object {
	// This can be useful for local debugging because you can update the chart locally.
	// chartPath := "helm-charts/splunk-otel-collector"
	//
	// chart, err := loader.Load(chartPath)
	// if err != nil {
	// 	panic(err)
	// }

	httpgetter, err := getter.NewHTTPGetter()
	if err != nil {
		panic(err)
	}

	// This requires us to have this binary downloadable in order for the operator to deploy the collector.
	// TODO: Store a local version or versions of the chart in case of internet connection issues
	u := "https://github.com/signalfx/splunk-otel-collector-chart/releases/download/splunk-otel-collector-0.59.0/splunk-otel-collector-0.59.0.tgz"
	data, err := httpgetter.Get(u)
	if err != nil {
		panic(err)
	}

	chart, err := loader.LoadArchive(data)
	if err != nil {
		panic(err)
	}

	// The operator can set the values for the chart rendered k8s resources to use as base for operator created k8s resources.
	// TODO: Add these to the Agent spec and set the spec values here
	opVals := chartutil.Values{
		"clusterName":      r.Spec.ClusterName,
		"fullnameOverride": r.Name,
		// For now, only the otel logEngine is supported.
		// Using the fluentd logEngine would require several changes beforehand to the operator.
		"logsEngine": "otel",

		"splunkObservability": map[string]interface{}{
			"metricsEnabled": true,
			"tracesEnabled":  true,
			"logsEnabled":    true,
			"accessToken":    "${SPLUNK_OBSERVABILITY_ACCESS_TOKEN}",
			"realm":          "${SPLUNK_REALM}",
		},

		"agent": map[string]interface{}{
			"enabled": r.Spec.Agent.Enabled,
		},
		"clusterReceiver": map[string]interface{}{
			"enabled": r.Spec.ClusterReceiver.Enabled,
		},
		"gateway": map[string]interface{}{
			"enabled": r.Spec.Gateway.Enabled,
		},

		"secret": map[string]interface{}{
			"create": false,
			"name":   "splunk-access-token",
		},
	}
	ro := chartutil.ReleaseOptions{
		Name: r.ObjectMeta.Name,
	}
	v, err := chartutil.ToRenderValues(chart, opVals.AsMap(), ro, nil)
	if err != nil {
		panic(err)
	}
	sepYamlfiles, err := engine.Render(chart, v)
	if err != nil {
		panic(err)
	}
	acceptedK8sTypes := regexp.MustCompile(k8sTypes)

	for fileName, fileContent := range sepYamlfiles {
		if !acceptedK8sTypes.MatchString(fileName) {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, err := decode([]byte(fileContent), nil, nil)

		if err != nil {
			defaultslog.Info(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
			continue
		}

		return obj
	}
	return nil
}

const (
	defaultAgentCPU    = "200m"
	defaultAgentMemory = "500Mi"
)

func getDefaultAgentDaemonSet(r *Agent) *appsv1.DaemonSet {
	d := getChartManifestString(r, "(daemonset.yaml)").(*appsv1.DaemonSet)

	original := d.Spec.Template.Spec.Containers[0].VolumeMounts
	var filtered []v1.VolumeMount
	// We remove the otel-configmap volume mount because the operator will set it later.
	// See:
	//  https://github.com/signalfx/splunk-otel-collector-chart/blob/24a71781579dd80f4682189c25146d6ba0337c0e/rendered/manifests/agent-only/daemonset.yaml#L149
	//  https://github.com/signalfx/splunk-otel-collector-operator/blob/4bb7baf363a536dc124a6790495e952d4f3fad00/internal/collector/container.go#L54
	for i := 0; i < len(original); i++ {
		if original[i].Name != "otel-configmap" {
			filtered = append(filtered, original[i])
		}
	}
	d.Spec.Template.Spec.Containers[0].VolumeMounts = filtered

	return d
}

func getDefaultAgentConfigMapAsString(r *Agent) string {
	return getDefaultConfigMapAsString(r, "(configmap-agent.yaml)")
}

func getDefaultGatewayDeployment(r *Agent) *appsv1.Deployment {
	d := getChartManifestString(r, "(deployment-gateway.yaml)").(*appsv1.Deployment)

	original := d.Spec.Template.Spec.Containers[0].VolumeMounts
	var filtered []v1.VolumeMount
	// We remove the otel-configmap volume mount because the operator will set it later.
	// See:
	//  https://github.com/signalfx/splunk-otel-collector-chart/blob/24a71781579dd80f4682189c25146d6ba0337c0e/rendered/manifests/gateway-only/deployment-gateway.yaml#L113
	//  https://github.com/signalfx/splunk-otel-collector-operator/blob/4bb7baf363a536dc124a6790495e952d4f3fad00/internal/collector/container.go#L54
	for i := 0; i < len(original); i++ {
		if original[i].Name != "collector-configmap" {
			filtered = append(filtered, original[i])
		}
	}
	d.Spec.Template.Spec.Containers[0].VolumeMounts = filtered

	return d
}

func getDefaultGatewayConfigMapAsString(r *Agent) string {
	return getDefaultConfigMapAsString(r, "(configmap-gateway.yaml)")
}

func getDefaultConfigMapAsString(r *Agent, fileName string) string {
	return getChartManifestString(r, fileName).(*v1.ConfigMap).Data["relay"]
}

func getDefaultService(r *Agent) *v1.Service {
	return getChartManifestString(r, "(service.yaml)").(*v1.Service)
}

const (
	defaultClusterReceiverCPU    = "200m"
	defaultClusterReceiverMemory = "500Mi"
	defaultClusterReceiverConfig = `
extensions:
  health_check:
    endpoint: '0.0.0.0:13133'
  memory_ballast:
    size_mib: ${SPLUNK_BALLAST_SIZE_MIB}
receivers:
  k8s_cluster:
    auth_type: serviceAccount
    metadata_exporters:
      - signalfx
  prometheus/self:
    config:
      scrape_configs:
        - job_name: otel-k8s-cluster-receiver
          scrape_interval: 10s
          static_configs:
            - targets:
                - '${MY_POD_IP}:8888'
exporters:
  signalfx:
    access_token: '${SPLUNK_OBSERVABILITY_ACCESS_TOKEN}'
    api_url: 'https://api.${SPLUNK_REALM}.signalfx.com'
    ingest_url: 'https://ingest.${SPLUNK_REALM}.signalfx.com'
    timeout: 10s
  logging: null
  logging/debug:
    loglevel: debug
processors:
  batch: null
  memory_limiter:
    check_interval: 2s
    limit_mib: '${SPLUNK_MEMORY_LIMIT_MIB}'
  resource:
    attributes:
      - action: insert
        key: metric_source
        value: kubernetes
      - action: insert
        key: receiver
        value: k8scluster
      - action: upsert
        key: k8s.cluster.name
        value: '${MY_CLUSTER_NAME}'
      - action: upsert
        key: deployment.environment
        value: '${MY_CLUSTER_NAME}'
  resource/self:
    attributes:
      - action: insert
        key: k8s.node.name
        value: '${MY_NODE_NAME}'
      - action: insert
        key: k8s.pod.name
        value: '${MY_POD_NAME}'
      - action: insert
        key: k8s.pod.uid
        value: '${MY_POD_UID}'
      - action: insert
        key: k8s.namespace.name
        value: '${MY_NAMESPACE}'
  resourcedetection:
    override: false
    timeout: 10s
    detectors:
      - system
      - env
service:
  extensions:
    - health_check
    - memory_ballast
  pipelines:
    metrics:
      receivers:
        - k8s_cluster
      processors:
        - batch
        - resource
        - resourcedetection
      exporters:
        - signalfx
    metrics/self:
      receivers:
        - prometheus/self
      processors:
        - batch
        - resource
        - resource/self
        - resourcedetection
      exporters:
        - signalfx
`

	defaultClusterReceiverConfigOpenshift = `
extensions:
  health_check:
    endpoint: '0.0.0.0:13133'
  memory_ballast:
    size_mib: ${SPLUNK_BALLAST_SIZE_MIB}
receivers:
  k8s_cluster:
    distribution: openshift
    auth_type: serviceAccount
    metadata_exporters:
      - signalfx
  prometheus/self:
    config:
      scrape_configs:
        - job_name: otel-k8s-cluster-receiver
          scrape_interval: 10s
          static_configs:
            - targets:
                - '${MY_POD_IP}:8888'
exporters:
  signalfx:
    access_token: '${SPLUNK_OBSERVABILITY_ACCESS_TOKEN}'
    api_url: 'https://api.${SPLUNK_REALM}.signalfx.com'
    ingest_url: 'https://ingest.${SPLUNK_REALM}.signalfx.com'
    timeout: 10s
  logging: null
  logging/debug:
    loglevel: debug
processors:
  batch: null
  memory_limiter:
    check_interval: 2s
    limit_mib: '${SPLUNK_MEMORY_LIMIT_MIB}'
  resource:
    attributes:
      - action: insert
        key: metric_source
        value: kubernetes
      - action: insert
        key: receiver
        value: k8scluster
      - action: upsert
        key: k8s.cluster.name
        value: '${MY_CLUSTER_NAME}'
      - action: upsert
        key: deployment.environment
        value: '${MY_CLUSTER_NAME}'
  resource/self:
    attributes:
      - action: insert
        key: k8s.node.name
        value: '${MY_NODE_NAME}'
      - action: insert
        key: k8s.pod.name
        value: '${MY_POD_NAME}'
      - action: insert
        key: k8s.pod.uid
        value: '${MY_POD_UID}'
      - action: insert
        key: k8s.namespace.name
        value: '${MY_NAMESPACE}'
  resourcedetection:
    override: false
    timeout: 10s
    detectors:
      - system
      - env
service:
  extensions:
    - health_check
    - memory_ballast
  pipelines:
    metrics:
      receivers:
        - k8s_cluster
      processors:
        - batch
        - resource
        - resourcedetection
      exporters:
        - signalfx
    metrics/self:
      receivers:
        - prometheus/self
      processors:
        - batch
        - resource
        - resource/self
        - resourcedetection
      exporters:
        - signalfx
`

	defaultGatewayCPU    = "4"
	defaultGatewayMemory = "8Gi"

	// the javaagent version is managed by the update-javaagent-version.sh script.
	defaultJavaAgentVersion = "v1.14.1"
	defaultJavaAgentImage   = "quay.io/signalfx/splunk-otel-instrumentation-java:" + defaultJavaAgentVersion
)
