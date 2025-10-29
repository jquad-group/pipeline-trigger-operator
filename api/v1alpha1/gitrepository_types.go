package v1alpha1

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	gitRepositoryNamePosition int    = 2
	branchNamePosition        int    = 0
	commitIdPosition          int    = 1
	repositoryNameDelimeter   string = "/"
	revisionDelimiter         string = "@"
	commitIdDelimeter         string = ":"
)

type GitRepository struct {
	// +kubebuilder:validation:Optional
	BranchName string `json:"branchName,omitempty"`

	// +kubebuilder:validation:Optional
	CommitId string `json:"commitId,omitempty"`

	// +kubebuilder:validation:Optional
	RepositoryName string `json:"repositoryName,omitempty"`

	// +kubebuilder:validation:Optional
	LatestPipelineRun string `json:"latestPipelineRun,omitempty"`

	Details string `json:"details,omitempty"`

	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

func (currentGitRepository *GitRepository) GenerateGitRepositoryLabelsAsHash() map[string]string {
	labels := make(map[string]string)

	labels[pipelineTriggerLabelKey+"/"+"git.repository.branch.name"] = currentGitRepository.BranchName
	labels[pipelineTriggerLabelKey+"/"+"git.repository.branch.commit"] = currentGitRepository.CommitId

	return labels
}

func (currentGitRepository *GitRepository) GenerateGitRepositoryLabelsAsString() string {
	label :=
		pipelineTriggerLabelKey + "/" + "git.repository.branch.name=" + currentGitRepository.BranchName + "," +
			pipelineTriggerLabelKey + "/" + "git.repository.branch.commit=" + currentGitRepository.CommitId

	return label
}

func (currentGitRepository *GitRepository) Equals(newGitRepository GitRepository) bool {
	if currentGitRepository.BranchName == newGitRepository.BranchName && currentGitRepository.RepositoryName == newGitRepository.RepositoryName && currentGitRepository.CommitId == newGitRepository.CommitId {
		return true
	} else {
		return false
	}
}

func getGitRepositoryName(fluxGitRepository unstructured.Unstructured) string {
	repositoryPath := fluxGitRepository.Object["status"].(map[string]interface{})["artifact"].(map[string]interface{})["path"]
	repositoryPathStr := fmt.Sprintf("%v", repositoryPath)
	repositoryName := strings.Split(repositoryPathStr, repositoryNameDelimeter)[gitRepositoryNamePosition]
	return repositoryName
}

func getBranchName(fluxGitRepository unstructured.Unstructured) string {
	repositoryRevision := fluxGitRepository.Object["status"].(map[string]interface{})["artifact"].(map[string]interface{})["revision"]
	repositoryRevisionStr := fmt.Sprintf("%v", repositoryRevision)
	branchName := strings.Split(repositoryRevisionStr, revisionDelimiter)[branchNamePosition]
	return branchName
}

func getCommitId(fluxGitRepository unstructured.Unstructured) string {
	repositoryCommitId := fluxGitRepository.Object["status"].(map[string]interface{})["artifact"].(map[string]interface{})["revision"]
	repositoryCommitIdStr := fmt.Sprintf("%v", repositoryCommitId)
	commitId := strings.Split(repositoryCommitIdStr, commitIdDelimeter)[commitIdPosition]
	return commitId
}

func (gitRepository *GitRepository) GetGitRepository(fluxGitRepository unstructured.Unstructured) {
	gitRepository.RepositoryName = getGitRepositoryName(fluxGitRepository)
	gitRepository.BranchName = getBranchName(fluxGitRepository)
	gitRepository.CommitId = getCommitId(fluxGitRepository)
}

func (gitRepository *GitRepository) AddOrReplaceCondition(c metav1.Condition) {
	found := false
	for i, condition := range gitRepository.Conditions {
		if c.Type == condition.Type {
			gitRepository.Conditions[i] = c
			found = true
		}
	}
	if !found {
		gitRepository.Conditions = append(gitRepository.Conditions, c)
	}
}

func (gitRepository *GitRepository) GetCondition(conditionType string) (metav1.Condition, bool) {
	for _, condition := range gitRepository.Conditions {
		if condition.Type == conditionType {
			return condition, true
		}
	}
	return metav1.Condition{}, false
}

// GetLastCondition retruns the last condition based on the condition timestamp. if no condition is present it return false.
func (gitRepository *GitRepository) GetLastCondition() metav1.Condition {
	if len(gitRepository.Conditions) == 0 {
		return metav1.Condition{}
	}
	//we need to make a copy of the slice
	copiedConditions := []metav1.Condition{}
	for _, condition := range gitRepository.Conditions {
		ccondition := condition.DeepCopy()
		copiedConditions = append(copiedConditions, *ccondition)
	}
	sort.Slice(copiedConditions, func(i, j int) bool {
		return copiedConditions[i].LastTransitionTime.Before(&copiedConditions[j].LastTransitionTime)
	})
	return copiedConditions[len(copiedConditions)-1]
}

func (gitRepository *GitRepository) Rewrite() string {
	// Replaces branch names from feature/newlogin to feature-newlogin
	return strings.ReplaceAll(gitRepository.BranchName, "/", "-")
}

func (gitRepository *GitRepository) GenerateDetails() {
	tempGitRepository := &GitRepository{
		BranchName:     gitRepository.BranchName,
		CommitId:       gitRepository.CommitId,
		RepositoryName: gitRepository.RepositoryName,
	}
	data, _ := json.Marshal(tempGitRepository)
	gitRepository.Details = string(data)
}
