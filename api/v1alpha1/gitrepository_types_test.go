package v1alpha1

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGitRepository_GenerateGitRepositoryLabelsAsHash(t *testing.T) {
	gitRepo := GitRepository{
		BranchName: "main",
		CommitId:   "abc123",
	}

	labels := gitRepo.GenerateGitRepositoryLabelsAsHash()

	expected := map[string]string{
		pipelineTriggerLabelKey + "/" + "git.repository.branch.name":   "main",
		pipelineTriggerLabelKey + "/" + "git.repository.branch.commit": "abc123",
	}

	if !reflect.DeepEqual(labels, expected) {
		t.Errorf("Generated labels are not as expected")
	}
}

func TestGitRepository_GenerateGitRepositoryLabelsAsString(t *testing.T) {
	gitRepo := GitRepository{
		BranchName: "main",
		CommitId:   "abc123",
	}

	label := gitRepo.GenerateGitRepositoryLabelsAsString()

	expected := pipelineTriggerLabelKey + "/" + "git.repository.branch.name=main," + pipelineTriggerLabelKey + "/" + "git.repository.branch.commit=abc123"

	if label != expected {
		t.Errorf("Generated label is not as expected")
	}
}

func TestGitRepository_Equals(t *testing.T) {
	gitRepo1 := GitRepository{
		BranchName:     "main",
		CommitId:       "abc123",
		RepositoryName: "myrepo",
	}

	gitRepo2 := GitRepository{
		BranchName:     "main",
		CommitId:       "abc123",
		RepositoryName: "myrepo",
	}

	gitRepo3 := GitRepository{
		BranchName:     "feature",
		CommitId:       "def456",
		RepositoryName: "otherrepo",
	}

	if !gitRepo1.Equals(gitRepo2) {
		t.Errorf("GitRepositories are equal, but Equals function didn't return true.")
	}

	if gitRepo1.Equals(gitRepo3) {
		t.Errorf("GitRepositories are not equal, but Equals function didn't return false.")
	}
}

func TestGetGitRepositoryName(t *testing.T) {
	artifactPath := "gitrepository/001-flux-system/repo/c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2.tar.gz"
	fluxGitRepo := unstructured.Unstructured{}
	fluxGitRepo.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"artifact": map[string]interface{}{
				"path": artifactPath,
			},
		},
	}

	name := getGitRepositoryName(fluxGitRepo)
	expected := "repo"
	if name != expected {
		t.Errorf("Got Git Repository Name: %s, Expected: %s", name, expected)
	}
}

func TestGetBranchName(t *testing.T) {
	artifactRevision := "main@sha1:abc123"
	fluxGitRepo := unstructured.Unstructured{}
	fluxGitRepo.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"artifact": map[string]interface{}{
				"revision": artifactRevision,
			},
		},
	}

	name := getBranchName(fluxGitRepo)
	expected := "main"
	if name != expected {
		t.Errorf("Got Branch Name: %s, Expected: %s", name, expected)
	}
}

func TestGetCommitId(t *testing.T) {
	artifactRevision := "main@sha1:abc123"
	fluxGitRepo := unstructured.Unstructured{}
	fluxGitRepo.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"artifact": map[string]interface{}{
				"revision": artifactRevision,
			},
		},
	}

	commitID := getCommitId(fluxGitRepo)
	expected := "abc123"
	if commitID != expected {
		t.Errorf("Got Commit ID: %s, Expected: %s", commitID, expected)
	}
}

func TestGitRepository_GetGitRepository(t *testing.T) {
	artifactPath := "gitrepository/001-flux-system/myrepo/c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2.tar.gz"
	fluxGitRepo := unstructured.Unstructured{}
	fluxGitRepo.Object = map[string]interface{}{
		"status": map[string]interface{}{
			"artifact": map[string]interface{}{
				"path":     artifactPath,
				"revision": "main@sha1:c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2",
			},
		},
	}

	gitRepo := GitRepository{}
	gitRepo.GetGitRepository(fluxGitRepo)

	expectedGitRepo := GitRepository{
		BranchName:     "main",
		CommitId:       "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2",
		RepositoryName: "myrepo",
	}

	if !reflect.DeepEqual(gitRepo, expectedGitRepo) {
		t.Errorf("Git Repository is not as expected")
	}
}

func TestGitRepository_GetCondition(t *testing.T) {
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

	gitRepo := GitRepository{
		Conditions: conditions,
	}

	conditionType := "InProgress"
	condition, found := gitRepo.GetCondition(conditionType)

	if !found || condition.Type != conditionType {
		t.Errorf("Condition not found as expected")
	}
}

func TestGitRepository_GetLastCondition(t *testing.T) {
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

	gitRepo := GitRepository{
		Conditions: conditions,
	}

	lastCondition := gitRepo.GetLastCondition()

	if lastCondition.Type != "InProgress" {
		t.Errorf("Last condition is not as expected")
	}
}

func TestGitRepository_Rewrite(t *testing.T) {
	gitRepo := GitRepository{
		BranchName: "feature/newlogin",
	}

	rewrittenBranchName := gitRepo.Rewrite()

	if rewrittenBranchName != "feature-newlogin" {
		t.Errorf("Rewritten branch name is not as expected")
	}
}

func TestGitRepository_GenerateDetails(t *testing.T) {
	gitRepo := GitRepository{
		BranchName:     "main",
		CommitId:       "abc123",
		RepositoryName: "myrepo",
	}

	gitRepo.GenerateDetails()

	expectedDetails := `{"branchName":"main","commitId":"abc123","repositoryName":"myrepo"}`
	if gitRepo.Details != expectedDetails {
		t.Errorf("Generated details are not as expected")
	}
}
