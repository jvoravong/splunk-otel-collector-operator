// Copyright The OpenTelemetry Authors
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

// Package naming is for determining the names for components (containers, services, ...).
package naming

import (
	"fmt"

	"github.com/signalfx/splunk-otel-collector-operator/apis/otel/v1alpha1"
)

// ConfigMap builds the name for the config map used in the SplunkOtelAgent containers.
func ConfigMap(spec v1alpha1.Agent, kind string) string {
	return fmt.Sprintf("%s-%s", spec.Name, kind)
}

// ConfigMapVolume returns the name to use for the config map's volume in the pod.
func ConfigMapVolume() string {
	return "otel-configmap"
}

// Container returns the name to use for the container in the pod.
func Container() string {
	return "otc-container"
}

// Gateway builds the gateway name based on the instance.
func Gateway(otelcol v1alpha1.Agent) string {
	return fmt.Sprintf("%s-gateway", otelcol.Name)
}

// Agent builds the agent name based on the instance.
func Agent(otelcol v1alpha1.Agent) string {
	return fmt.Sprintf("%s-agent", otelcol.Name)
}

// ClusterReceiver builds the agent name based on the instance.
func ClusterReceiver(otelcol v1alpha1.Agent) string {
	return fmt.Sprintf("%s-cluster-receiver", otelcol.Name)
}

// HeadlessService builds the name for the headless service based on the instance.
func HeadlessService(otelcol v1alpha1.Agent) string {
	return fmt.Sprintf("%s-headless", Service(otelcol))
}

// MonitoringService builds the name for the monitoring service based on the instance.
func MonitoringService(otelcol v1alpha1.Agent) string {
	return fmt.Sprintf("%s-monitoring", Service(otelcol))
}

// Service builds the service name based on the instance.
func Service(otelcol v1alpha1.Agent) string {
	// We use this specific name value here for the service so the operator and chart agents talk to the proper gateway endpoint.
	//  https://github.com/signalfx/splunk-otel-collector-operator/blob/4bb7baf363a536dc124a6790495e952d4f3fad00/internal/collector/reconcile/service.go#L113
	//  https://github.com/signalfx/splunk-otel-collector-chart/blob/24a71781579dd80f4682189c25146d6ba0337c0e/helm-charts/splunk-otel-collector/templates/service.yaml#L6
	return fmt.Sprintf("%s", otelcol.Name)
}

// ServiceAccount builds the service account name based on the instance.
func ServiceAccount(otelcol v1alpha1.Agent) string {
	// TODO(splunk): create separate accounts for agent, clusterreceiver
	// and gateway.
	// return fmt.Sprintf("%s-account", otelcol.Name)
	return "splunk-otel-operator-account"
}

// Namespace builds the namespace name based on the instance.
func Namespace(otelcol v1alpha1.Agent) string {
	return "splunk-otel-operator-system"
}
