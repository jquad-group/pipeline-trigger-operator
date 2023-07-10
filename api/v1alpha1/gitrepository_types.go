package v1alpha1

import (
	"encoding/json"
	"sort"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// +kubebuilder:validation:Required
	BranchName string `json:"branchName,omitempty"`

	// +kubebuilder:validation:Required
	CommitId string `json:"commitId,omitempty"`

	// +kubebuilder:validation:Required
	RepositoryName string `json:"repositoryName,omitempty"`

	// +kubebuilder:validation:Required
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

func getGitRepositoryName(fluxGitRepository sourcev1.GitRepository) string {
	repositoryName := strings.Split(fluxGitRepository.Status.Artifact.Path, repositoryNameDelimeter)[gitRepositoryNamePosition]
	return repositoryName
}

func getBranchName(fluxGitRepository sourcev1.GitRepository) string {
	branchName := strings.Split(fluxGitRepository.Status.Artifact.Revision, revisionDelimiter)[branchNamePosition]
	return branchName
}

func getCommitId(fluxGitRepository sourcev1.GitRepository) string {
	commitId := strings.Split(fluxGitRepository.Status.Artifact.Revision, commitIdDelimeter)[commitIdPosition]
	return commitId
}

func (gitRepository *GitRepository) GetGitRepository(fluxGitRepository sourcev1.GitRepository) {
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
