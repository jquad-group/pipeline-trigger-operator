package v1alpha1

type Source struct {
	// Kind of the source refernce.
	// +kubebuilder:validation:Enum=ImagePolicy;GitRepository
	// +required
	Kind string `json:"kind,omitempty"`

	// Name of the source reference.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +required
	Name string `json:"name"`
}
