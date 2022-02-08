package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type SecurityContext struct {

	// +required
	RunAsUser int64 `json:"runAsUser,omitempty"`
	// +required
	RunAsGroup int64 `json:"runAsGroup,omitempty"`
	// +required
	FsGroup int64 `json:"fsGroup,omitempty"`
}

func (securityContext SecurityContext) CreatePodSecurityContext() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsUser:  &securityContext.RunAsUser,
		RunAsGroup: &securityContext.RunAsGroup,
		FSGroup:    &securityContext.FsGroup,
	}
}
