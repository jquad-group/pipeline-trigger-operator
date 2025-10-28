package credentials

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SecretReference references a secret in a specific namespace
type SecretReference struct {
	// Name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the secret
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
}

// ServiceAccountTemplate defines customization options for the generated ServiceAccount
type ServiceAccountTemplate struct {
	// Annotations to add to the ServiceAccount
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to add to the ServiceAccount
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// ImagePullSecrets to add to the ServiceAccount
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

// SecretTemplate defines customization options for the generated Secret
type SecretTemplate struct {
	// Annotations to add to the Secret
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels to add to the Secret
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Type overrides the automatic Secret type detection
	// +optional
	Type *corev1.SecretType `json:"type,omitempty"`

	// DataKey overrides the default data key for the token
	// +optional
	DataKey string `json:"dataKey,omitempty"`
}

const (
	// ManagedCredentialConditionReady indicates the ManagedCredential is ready
	ManagedCredentialConditionReady = "Ready"
	// ManagedCredentialConditionSyncFailed indicates sync failed
	ManagedCredentialConditionSyncFailed = "SyncFailed"
	// ManagedCredentialConditionSecretNotFound indicates the source secret is not found
	ManagedCredentialConditionSecretNotFound = "SecretNotFound"
)

// ManagedCredentialStatus defines the observed state of ManagedCredential
type ManagedCredentialStatus struct {
	// Conditions represent the latest available observations of the ManagedCredential's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SecretRef references the Tekton-compatible Kubernetes Secret created by this operator
	// +optional
	SecretRef *corev1.ObjectReference `json:"secretRef,omitempty"`

	// ServiceAccountRef references the ServiceAccount created by this operator
	// +optional
	ServiceAccountRef *corev1.ObjectReference `json:"serviceAccountRef,omitempty"`

	// ObservedGeneration is the most recent generation observed for this ManagedCredential.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ManagedCredential is a simplified version of the ManagedCredential API
type ManagedCredential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedCredentialSpec   `json:"spec,omitempty"`
	Status ManagedCredentialStatus `json:"status,omitempty"`
}

// DeepCopyObject implements the client.Object interface
func (m *ManagedCredential) DeepCopyObject() runtime.Object {
	if m == nil {
		return nil
	}

	out := &ManagedCredential{}
	m.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out. in must be non-nil.
func (m *ManagedCredential) DeepCopyInto(out *ManagedCredential) {
	*out = *m
	out.TypeMeta = m.TypeMeta
	m.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	if m.Spec.TokenSecretRef.Name != "" {
		out.Spec.TokenSecretRef = m.Spec.TokenSecretRef
	}

	if m.Spec.Description != "" {
		out.Spec.Description = m.Spec.Description
	}

	if m.Spec.ServiceAccountTemplate != nil {
		out.Spec.ServiceAccountTemplate = &ServiceAccountTemplate{}
		if m.Spec.ServiceAccountTemplate.Annotations != nil {
			out.Spec.ServiceAccountTemplate.Annotations = make(map[string]string)
			for k, v := range m.Spec.ServiceAccountTemplate.Annotations {
				out.Spec.ServiceAccountTemplate.Annotations[k] = v
			}
		}
		if m.Spec.ServiceAccountTemplate.Labels != nil {
			out.Spec.ServiceAccountTemplate.Labels = make(map[string]string)
			for k, v := range m.Spec.ServiceAccountTemplate.Labels {
				out.Spec.ServiceAccountTemplate.Labels[k] = v
			}
		}
		out.Spec.ServiceAccountTemplate.ImagePullSecrets = append([]corev1.LocalObjectReference{}, m.Spec.ServiceAccountTemplate.ImagePullSecrets...)
	}

	if m.Spec.SecretTemplate != nil {
		out.Spec.SecretTemplate = &SecretTemplate{}
		if m.Spec.SecretTemplate.Annotations != nil {
			out.Spec.SecretTemplate.Annotations = make(map[string]string)
			for k, v := range m.Spec.SecretTemplate.Annotations {
				out.Spec.SecretTemplate.Annotations[k] = v
			}
		}
		if m.Spec.SecretTemplate.Labels != nil {
			out.Spec.SecretTemplate.Labels = make(map[string]string)
			for k, v := range m.Spec.SecretTemplate.Labels {
				out.Spec.SecretTemplate.Labels[k] = v
			}
		}
		if m.Spec.SecretTemplate.Type != nil {
			out.Spec.SecretTemplate.Type = m.Spec.SecretTemplate.Type
		}
		out.Spec.SecretTemplate.DataKey = m.Spec.SecretTemplate.DataKey
	}

	if m.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(m.Status.Conditions))
		for i := range m.Status.Conditions {
			m.Status.Conditions[i].DeepCopyInto(&out.Status.Conditions[i])
		}
	}

	if m.Status.SecretRef != nil {
		out.Status.SecretRef = m.Status.SecretRef.DeepCopy()
	}

	if m.Status.ServiceAccountRef != nil {
		out.Status.ServiceAccountRef = m.Status.ServiceAccountRef.DeepCopy()
	}

	out.Status.ObservedGeneration = m.Status.ObservedGeneration
}

// DeepCopy creates a deep copy of ManagedCredential
func (m *ManagedCredential) DeepCopy() *ManagedCredential {
	if m == nil {
		return nil
	}
	out := new(ManagedCredential)
	m.DeepCopyInto(out)
	return out
}

// ManagedCredentialList contains a list of ManagedCredentials
type ManagedCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedCredential `json:"items"`
}

// DeepCopyObject implements the client.Object interface for ManagedCredentialList
func (m *ManagedCredentialList) DeepCopyObject() runtime.Object {
	if m == nil {
		return nil
	}

	out := &ManagedCredentialList{}
	m.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out. in must be non-nil.
func (m *ManagedCredentialList) DeepCopyInto(out *ManagedCredentialList) {
	*out = *m
	out.TypeMeta = m.TypeMeta
	m.ListMeta.DeepCopyInto(&out.ListMeta)

	if m.Items != nil {
		out.Items = make([]ManagedCredential, len(m.Items))
		for i := range m.Items {
			m.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy creates a deep copy of ManagedCredentialList
func (m *ManagedCredentialList) DeepCopy() *ManagedCredentialList {
	if m == nil {
		return nil
	}
	out := new(ManagedCredentialList)
	m.DeepCopyInto(out)
	return out
}

// AddToScheme adds the ManagedCredential type to the given scheme
func (ManagedCredential) AddToScheme(scheme *runtime.Scheme) error {
	gv := schema.GroupVersion{Group: "credentials.jquad.rocks", Version: "v1alpha1"}
	scheme.AddKnownTypes(gv, &ManagedCredential{}, &ManagedCredentialList{})
	return nil
}

// ManagedCredentialSpec defines the desired state of ManagedCredential
type ManagedCredentialSpec struct {
	// CredentialType defines the type of credential for proper Tekton annotation handling
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=git;docker-registry;cloud-provider;generic
	CredentialType string `json:"credentialType"`

	// Provider specifies the provider (e.g., github.com, docker.io) for Tekton annotations
	// +kubebuilder:validation:Required
	Provider string `json:"provider"`

	// Description provides human-readable explanation of the credential's purpose
	// +optional
	Description string `json:"description,omitempty"`

	// TokenSecretRef references the Kubernetes Secret containing the raw sensitive token
	// +kubebuilder:validation:Required
	TokenSecretRef SecretReference `json:"tokenSecretRef"`

	// ServiceAccountTemplate allows customization of the generated ServiceAccount
	// +optional
	ServiceAccountTemplate *ServiceAccountTemplate `json:"serviceAccountTemplate,omitempty"`

	// SecretTemplate allows customization of the generated Secret
	// +optional
	SecretTemplate *SecretTemplate `json:"secretTemplate,omitempty"`
}

// IsReady returns true if the ManagedCredential is ready
func (m *ManagedCredential) IsReady() bool {
	condition := m.GetCondition(ManagedCredentialConditionReady)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// GetCondition returns the condition with the given type, or nil if not found
func (m *ManagedCredential) GetCondition(conditionType string) *metav1.Condition {
	for i := range m.Status.Conditions {
		if m.Status.Conditions[i].Type == conditionType {
			return &m.Status.Conditions[i]
		}
	}
	return nil
}

// SetCondition sets the condition with the given type and status
func (m *ManagedCredential) SetCondition(conditionType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	for i := range m.Status.Conditions {
		if m.Status.Conditions[i].Type == conditionType {
			if m.Status.Conditions[i].Status != status {
				m.Status.Conditions[i].LastTransitionTime = metav1.Now()
			}
			m.Status.Conditions[i].Status = status
			m.Status.Conditions[i].Reason = reason
			m.Status.Conditions[i].Message = message
			return
		}
	}

	m.Status.Conditions = append(m.Status.Conditions, condition)
}
