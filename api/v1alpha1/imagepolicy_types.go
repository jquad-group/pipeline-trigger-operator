package v1alpha1

import (
	"encoding/json"
	"sort"
	"strings"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	repositoryNamePosition int    = 1
	imageNamePosition      int    = 0
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

	Details string `json:"details,omitempty"`

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

func (imagePolicy *ImagePolicy) GetImagePolicy(fluxImagePolicy imagereflectorv1.ImagePolicy) {
	imagePolicy.RepositoryName = getRepositoryName(fluxImagePolicy)
	imagePolicy.ImageName = getImageName(fluxImagePolicy)
	imagePolicy.ImageVersion = getImageVersion(fluxImagePolicy)
}

func getRepositoryName(fluxImagePolicy imagereflectorv1.ImagePolicy) string {
	pathSubdirSize := len(strings.Split(fluxImagePolicy.Status.LatestImage, imageNameDelimeter))
	repositoryName := strings.Split(fluxImagePolicy.Status.LatestImage, imageNameDelimeter)[pathSubdirSize-2]
	return repositoryName
}

func getImageName(fluxImagePolicy imagereflectorv1.ImagePolicy) string {
	pathSubdirSize := len(strings.Split(fluxImagePolicy.Status.LatestImage, imageNameDelimeter))
	imageNameWithVersion := strings.Split(fluxImagePolicy.Status.LatestImage, imageNameDelimeter)[pathSubdirSize-1]
	imageName := strings.Split(imageNameWithVersion, imageVersionDelimeter)[imageNamePosition]
	return imageName
}

func getImageVersion(fluxImagePolicy imagereflectorv1.ImagePolicy) string {
	imageVersion := strings.Split(fluxImagePolicy.Status.LatestImage, imageVersionDelimeter)[imageVersionPosition]
	return imageVersion
}

func (imagePolicy *ImagePolicy) AddOrReplaceCondition(c metav1.Condition) {
	found := false
	for i, condition := range imagePolicy.Conditions {
		if c.Type == condition.Type {
			imagePolicy.Conditions[i] = c
			found = true
		}
	}
	if !found {
		imagePolicy.Conditions = append(imagePolicy.Conditions, c)
	}
}

func (imagePolicy *ImagePolicy) GetCondition(conditionType string) (metav1.Condition, bool) {
	for _, condition := range imagePolicy.Conditions {
		if condition.Type == conditionType {
			return condition, true
		}
	}
	return metav1.Condition{}, false
}

// GetLastCondition retruns the last condition based on the condition timestamp. if no condition is present it return false.
func (imagePolicy *ImagePolicy) GetLastCondition() metav1.Condition {
	if len(imagePolicy.Conditions) == 0 {
		return metav1.Condition{}
	}
	//we need to make a copy of the slice
	copiedConditions := []metav1.Condition{}
	for _, condition := range imagePolicy.Conditions {
		ccondition := condition.DeepCopy()
		copiedConditions = append(copiedConditions, *ccondition)
	}
	sort.Slice(copiedConditions, func(i, j int) bool {
		return copiedConditions[i].LastTransitionTime.Before(&copiedConditions[j].LastTransitionTime)
	})
	return copiedConditions[len(copiedConditions)-1]
}

func (imagePolicy *ImagePolicy) Rewrite() string {
	// Replaces branch names from feature/newlogin to feature-newlogin
	return strings.ReplaceAll(imagePolicy.ImageName, ":", "-")
}

func (imagePolicy *ImagePolicy) GenerateDetails() {
	tempImagePolicy := &ImagePolicy{
		ImageName:      imagePolicy.ImageName,
		RepositoryName: imagePolicy.RepositoryName,
		ImageVersion:   imagePolicy.ImageVersion,
	}
	data, _ := json.Marshal(tempImagePolicy)
	imagePolicy.Details = string(data)
}
