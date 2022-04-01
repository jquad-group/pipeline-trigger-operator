package v1alpha1

type Source struct {
	// Kind of the source refernce.
	// +kubebuilder:validation:Enum=ImagePolicy;GitRepository;PullRequest
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name of the source reference.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}
