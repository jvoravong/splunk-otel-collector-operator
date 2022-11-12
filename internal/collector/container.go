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

package collector

import (
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/signalfx/splunk-otel-collector-operator/apis/otel/v1alpha1"
	"github.com/signalfx/splunk-otel-collector-operator/internal/naming"
)

// Container builds a container for the given collector.
func Container(logger logr.Logger, spec v1alpha1.CollectorSpec) corev1.Container {
	image := spec.Image
	if len(image) == 0 {
		image = defaultCollectorImage
	}

	argsMap := spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}

	if _, exists := argsMap["config"]; exists {
		logger.Info("the 'config' flag isn't allowed and is being ignored")
	}

	// this effectively overrides any 'config' entry that might exist in the CR
	argsMap["config"] = fmt.Sprintf("/conf/%s", configmapEntry)

	var args []string
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	configMapVolumeMountDefined := false
	for i := 0; i < len(spec.VolumeMounts); i++ {
		if spec.VolumeMounts[i].MountPath == "/conf" {
			configMapVolumeMountDefined = true
			break
		}
	}

	volumeMounts := spec.VolumeMounts
	if !configMapVolumeMountDefined {
		// We could filter out /conf volumeMounts if they were defined
		volumeMount := corev1.VolumeMount{
			Name:      naming.ConfigMapVolume(),
			MountPath: "/conf",
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	var envVars = spec.Env
	if spec.Env == nil {
		envVars = []corev1.EnvVar{}
	}

	// The CollectorSpec uses service ports so we have to convert them to container ports here
	// TODO: Fix this
	var containerPorts []corev1.ContainerPort
	if spec.Ports != nil {
		for _, servicePort := range spec.Ports {
			containerPorts = append(containerPorts, corev1.ContainerPort{
				Name:          servicePort.Name,
				ContainerPort: servicePort.Port,
				Protocol:      servicePort.Protocol,
			})
		}
	}

	return corev1.Container{
		Name:            naming.Container(),
		Image:           image,
		ImagePullPolicy: spec.ImagePullPolicy,
		VolumeMounts:    volumeMounts,
		Args:            args,
		Env:             envVars,
		Resources:       spec.Resources,
		SecurityContext: spec.SecurityContext,
		Ports:           containerPorts,
	}
}
