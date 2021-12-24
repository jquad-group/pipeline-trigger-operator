/*
Copyright 2021.

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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PipelineTrigger is the Schema for the pipelinetriggers API
type PipelineTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PipelineTriggerSpec `json:"spec,omitempty"`

	// +kubebuilder:default={"observedGeneration":-1}
	Status PipelineTriggerStatus `json:"status,omitempty"`
}

// PipelineTriggerSpec defines the desired state of PipelineTrigger
type PipelineTriggerSpec struct {
	// AppVersion is the version of the application
	// that should be built
	AppVersion string `json:"appVersion,omitempty"`

	// ImagePolicy points at the object specifying the latest image
	// found
	// +required
	ImagePolicy string `json:"imagePolicy"`

	// Pipeline points at the object specifying the tekton pipeline
	// +required
	Pipeline Pipeline `json:"pipeline"`
}

// PipelineTriggerStatus defines the observed state of PipelineTrigger
type PipelineTriggerStatus struct {
	// LatestImage gives the first in the list of images scanned by
	// the image repository, when filtered and ordered according to
	// the policy.
	LatestImage string `json:"latestImage,omitempty"`
	// +optional

	// PipelineStatus gives the status of the
	// tekton pipeline currently running.
	PipelineStatus string `json:"pipelineStatus,omitempty"`
	// +optional

	// PipelineReason gives the reason of the
	// tekton pipeline currently running.
	PipelineReason string `json:"pipelineReason,omitempty"`
	// +optional
}

//+kubebuilder:object:root=true

// PipelineTriggerList contains a list of PipelineTrigger
type PipelineTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineTrigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineTrigger{}, &PipelineTriggerList{})
}
