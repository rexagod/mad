/*
Copyright 2023 The Kubernetes mad Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:rbac:groups=mad.instrumentation.k8s-sigs.io,resources=metricsanomalydetectorresources;metricsanomalydetectorresources/status,verbs=*

// MetricsAnomalyDetectorResource is a specification for a MetricsAnomalyDetectorResource resource.
// +kubebuilder:subresource:status
type MetricsAnomalyDetectorResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MetricsAnomalyDetectorResourceSpec `json:"spec"`

	// +kubebuilder:validation:Optional
	// +optional
	Status MetricsAnomalyDetectorResourceStatus `json:"status,omitempty"`
}

// MetricsAnomalyDetectorResourceSpec is the spec for a MetricsAnomalyDetectorResource resource.
type MetricsAnomalyDetectorResourceSpec struct {

	// BufferSize is the size of the circular buffer at any given time.
	// If the specified value is less than the current, last excessive entries will be dropped.
	// +kube:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=255
	// +kubebuilder:default=10
	BufferSize int `json:"bufferSize"`

	// HealthcheckEndpoints is the list of endpoints to query.
	// +kube:validation:Required
	HealthcheckEndpoints []string `json:"healthcheckEndpoints"`

	// QueryInterval is the interval at which the endpoints are queried, in seconds.
	// +kube:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +kubebuilder:default=60
	QueryInterval int `json:"queryInterval"`
}

// HealthcheckRecord is a record of a healthcheck event.
type HealthcheckRecord struct {

	// Timestamp is the time when the event was received.
	// NOTE: The difference between timestamps may not be same as the query interval due to network latency, throttling, etc.
	// +kubebuilder:validation:Optional
	// +optional
	Timestamp *metav1.Time `json:"timestamp"`

	// Healthy is the health status of the component.
	// +kubebuilder:validation:Optional
	// +optional
	Healthy *bool `json:"healthy"`
}

// MetricsAnomalyDetectorResourceStatus is the status for a MetricsAnomalyDetectorResource resource.
type MetricsAnomalyDetectorResourceStatus struct {

	// CurrentBufferSize is the current size of the buffer.
	// +kubebuilder:validation:Optional
	// +optional
	CurrentBufferSize int `json:"currentBufferSize"`

	// LastBufferModificationTime is the time when the buffer was last modified.
	// +kubebuilder:validation:Optional
	// +optional
	LastBufferModificationTime metav1.Time `json:"lastBufferModificationTime"`

	// LastBuffer is the last buffer of events.
	// +kubebuilder:validation:Optional
	// +optional
	LastBuffer []HealthcheckRecord `json:"lastBuffer"`

	// HealthcheckEndpointsHealthy is the list of healthy endpoints.
	// +kubebuilder:validation:Optional
	// +optional
	HealthcheckEndpointsHealthy map[string]bool `json:"healthcheckEndpointsHealthy"`

	// LastHealthcheckQueryTime is the time when the healthcheck endpoints were last queried.
	// +kubebuilder:validation:Optional
	// +optional
	LastHealthcheckQueryTime metav1.Time `json:"lastHealthcheckQueryTime"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// MetricsAnomalyDetectorResourceList is a list of MetricsAnomalyDetectorResource resources.
type MetricsAnomalyDetectorResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MetricsAnomalyDetectorResource `json:"items"`
}
