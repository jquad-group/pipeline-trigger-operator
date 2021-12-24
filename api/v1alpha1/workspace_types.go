package v1alpha1

import (
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Workspace struct {
	// +required
	Name string `json:"name,omitempty"`

	// +required
	Size string `json:"size,omitempty"`

	// +required
	AccessMode string `json:"accessMode,omitempty"`
}

// Workspace adds a WorkspaceBinding to the PipelineRun spec.
func (workspace Workspace) CreateWorkspaceBinding() tektondevv1.WorkspaceBinding {
	return tektondevv1.WorkspaceBinding{
		Name: workspace.Name,
		VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.PersistentVolumeAccessMode(workspace.AccessMode)},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"storage": resource.MustParse(workspace.Size),
					},
				},
			},
		},
	}

}
