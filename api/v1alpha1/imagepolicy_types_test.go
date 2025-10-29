package v1alpha1

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestImagePolicy_GenerateImagePolicyLabelsAsHash(t *testing.T) {
	imagePol := ImagePolicy{
		ImageName:    "my",
		ImageVersion: "v1.0",
	}

	labels := imagePol.GenerateImagePolicyLabelsAsHash()

	expected := map[string]string{
		pipelineTriggerLabelKey + "/" + "ip.image.name":    "my",
		pipelineTriggerLabelKey + "/" + "ip.image.version": "v1.0",
	}

	if !reflect.DeepEqual(labels, expected) {
		t.Errorf("Generated labels are not as expected")
	}
}

func TestImagePolicy_GenerateImagePolicyLabelsAsString(t *testing.T) {
	imagePol := ImagePolicy{
		ImageName:    "my",
		ImageVersion: "v1.0",
	}

	label := imagePol.GenerateImagePolicyLabelsAsString()

	expected := pipelineTriggerLabelKey + "/" + "ip.image.name=my," + pipelineTriggerLabelKey + "/" + "ip.image.version=v1.0"

	if label != expected {
		t.Errorf("Generated label is not as expected")
	}
}

func TestImagePolicy_Equals(t *testing.T) {
	imagePol1 := ImagePolicy{
		RepositoryName: "my-repo",
		ImageName:      "my",
		ImageVersion:   "v1.0",
	}

	imagePol2 := ImagePolicy{
		RepositoryName: "my-repo",
		ImageName:      "my",
		ImageVersion:   "v1.0",
	}

	imagePol3 := ImagePolicy{
		RepositoryName: "other-repo",
		ImageName:      "other-image",
		ImageVersion:   "v2.0",
	}

	if !imagePol1.Equals(imagePol2) {
		t.Errorf("ImagePolicies are equal, but Equals function didn't return true.")
	}

	if imagePol1.Equals(imagePol3) {
		t.Errorf("ImagePolicies are not equal, but Equals function didn't return false.")
	}
}

func TestGetRepositoryName(t *testing.T) {
	fluxImagePol := unstructured.Unstructured{}
	fluxImagePol.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"latestRef": map[string]interface{}{
				"name": "ghcr.io/repo/my",
				"tag":  "v1.0",
			},
		},
	}

	name := getRepositoryName(fluxImagePol)
	expected := "repo"
	if name != expected {
		t.Errorf("Got Repository Name: %s, Expected: %s", name, expected)
	}
}

func TestGetImageName(t *testing.T) {
	fluxImagePol := unstructured.Unstructured{}
	fluxImagePol.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"latestRef": map[string]interface{}{
				"name": "ghcr.io/repo/my",
				"tag":  "v1.0",
			},
		},
	}

	name := getImageName(fluxImagePol)
	expected := "my"
	if name != expected {
		t.Errorf("Got Image Name: %s, Expected: %s", name, expected)
	}
}

func TestGetImageVersion(t *testing.T) {
	fluxImagePol := unstructured.Unstructured{}
	fluxImagePol.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"latestRef": map[string]interface{}{
				"name": "ghcr.io/repo/my",
				"tag":  "v1.0",
			},
		},
	}

	version := getImageVersion(fluxImagePol)
	expected := "v1.0"
	if version != expected {
		t.Errorf("Got Image Version: %s, Expected: %s", version, expected)
	}
}

func TestImagePolicy_GetImagePolicy(t *testing.T) {
	fluxImagePol := unstructured.Unstructured{}
	fluxImagePol.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"latestRef": map[string]interface{}{
				"name": "ghcr.io/repo/my",
				"tag":  "v1.0",
			},
		},
	}

	imagePol := ImagePolicy{}
	imagePol.GetImagePolicy(fluxImagePol)

	expectedImagePol := ImagePolicy{
		RepositoryName: "repo",
		ImageName:      "my",
		ImageVersion:   "v1.0",
	}

	if !reflect.DeepEqual(imagePol, expectedImagePol) {
		t.Errorf("Image Policy is not as expected")
	}
}

func TestImagePolicy_GetCondition(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
		},
		{
			Type:               "InProgress",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
		},
	}

	imagePol := ImagePolicy{
		Conditions: conditions,
	}

	conditionType := "InProgress"
	condition, found := imagePol.GetCondition(conditionType)

	if !found || condition.Type != conditionType {
		t.Errorf("Condition not found as expected")
	}
}

func TestImagePolicy_GetLastCondition(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
		},
		{
			Type:               "InProgress",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
		},
	}

	imagePol := ImagePolicy{
		Conditions: conditions,
	}

	lastCondition := imagePol.GetLastCondition()

	if lastCondition.Type != "InProgress" {
		t.Errorf("Last condition is not as expected")
	}
}

func TestImagePolicy_Rewrite(t *testing.T) {
	imagePol := ImagePolicy{
		ImageName: "my-image:v1.0",
	}

	rewrittenImageName := imagePol.Rewrite()

	if rewrittenImageName != "my-image-v1.0" {
		t.Errorf("Rewritten image name is not as expected")
	}
}

func TestImagePolicy_GenerateDetails(t *testing.T) {
	imagePol := ImagePolicy{
		ImageName:      "my",
		RepositoryName: "repo",
		ImageVersion:   "v1.0",
	}

	imagePol.GenerateDetails()

	expectedDetails := `{"repositoryName":"repo","imageName":"my","imageVersion":"v1.0"}`
	if imagePol.Details != expectedDetails {
		t.Errorf("Generated details are not as expected")
	}
}
