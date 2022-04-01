package v1alpha1

import (
	"strings"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	repositoryNamePosition int    = 1
	imageNamePosition      int    = 2
	imageVersionPosition   int    = 1
	imageNameDelimeter     string = "/"
	imageVersionDelimeter  string = ":"
)

type ImagePolicy struct {

	// +kubebuilder:validation:Required
	RepositoryName string `json:"repositoryName,omitempty"`

	// +kubebuilder:validation:Required
	ImageName string `json:"imageName,omitempty"`

	// +kubebuilder:validation:Required
	ImageVersion string `json:"imageVersion,omitempty"`

	// +kubebuilder:validation:Required
	LatestPipelineRun string `json:"latestPipelineRun,omitempty"`

	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

func (currentImagePolicy *ImagePolicy) GenerateImagePolicyLabelsAsHash() map[string]string {
	labels := make(map[string]string)

	labels[pipelineTriggerLabelKey+"/"+"ip.image.name"] = currentImagePolicy.ImageName
	labels[pipelineTriggerLabelKey+"/"+"ip.image.version"] = currentImagePolicy.ImageVersion

	return labels
}

func (currentImagePolicy *ImagePolicy) GenerateImagePolicyLabelsAsString() string {
	label :=
		pipelineTriggerLabelKey + "/" + "ip.image.name=" + currentImagePolicy.ImageName + "," +
			pipelineTriggerLabelKey + "/" + "ip.image.version=" + currentImagePolicy.ImageVersion

	return label
}

func (currentImagePolicy *ImagePolicy) Equals(newImagePolicy ImagePolicy) bool {
	if currentImagePolicy.RepositoryName == newImagePolicy.RepositoryName && currentImagePolicy.ImageName == newImagePolicy.ImageName && currentImagePolicy.ImageVersion == newImagePolicy.ImageVersion {
		return true
	} else {
		return false
	}
}

func (currentImagePolicy *ImagePolicy) isUpdated(newImagePolicy ImagePolicy) bool {
	if currentImagePolicy.RepositoryName == newImagePolicy.RepositoryName && currentImagePolicy.ImageName == newImagePolicy.ImageName && currentImagePolicy.ImageVersion != newImagePolicy.ImageVersion {
		return true
	} else {
		return false
	}
}

func (imagePolicy *ImagePolicy) GetImagePolicy(fluxImagePolicy imagereflectorv1.ImagePolicy) {
	imagePolicy.RepositoryName = getRepositoryName(fluxImagePolicy)
	imagePolicy.ImageName = getImageName(fluxImagePolicy)
	imagePolicy.ImageVersion = getImageVersion(fluxImagePolicy)
}

func getRepositoryName(fluxImagePolicy imagereflectorv1.ImagePolicy) string {
	repositoryName := strings.Split(fluxImagePolicy.Status.LatestImage, imageNameDelimeter)[repositoryNamePosition]
	return repositoryName
}

func getImageName(fluxImagePolicy imagereflectorv1.ImagePolicy) string {
	imageName := strings.Split(fluxImagePolicy.Status.LatestImage, imageNameDelimeter)[imageNamePosition]
	return imageName
}

func getImageVersion(fluxImagePolicy imagereflectorv1.ImagePolicy) string {
	imageName := strings.Split(fluxImagePolicy.Status.LatestImage, imageVersionDelimeter)[imageVersionPosition]
	return imageName
}
