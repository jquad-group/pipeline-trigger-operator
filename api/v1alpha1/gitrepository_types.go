package v1alpha1

import (
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gitRepositoryNamePosition int    = 2
	branchNamePosition        int    = 0
	commitIdPosition          int    = 1
	revisionDelimiter         string = "/"
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

func (currentGitRepository *GitRepository) isUpdated(newGitRepository GitRepository) bool {
	if currentGitRepository.BranchName == newGitRepository.BranchName && currentGitRepository.RepositoryName == newGitRepository.RepositoryName && currentGitRepository.CommitId != newGitRepository.CommitId {
		return true
	} else {
		return false
	}
}

func getGitRepositoryName(fluxGitRepository sourcev1.GitRepository) string {
	repositoryName := strings.Split(fluxGitRepository.Status.Artifact.Path, revisionDelimiter)[gitRepositoryNamePosition]
	return repositoryName
}

func getBranchName(fluxGitRepository sourcev1.GitRepository) string {
	branchName := strings.Split(fluxGitRepository.Status.Artifact.Revision, revisionDelimiter)[branchNamePosition]
	return branchName
}

func getCommitId(fluxGitRepository sourcev1.GitRepository) string {
	commitId := strings.Split(fluxGitRepository.Status.Artifact.Revision, revisionDelimiter)[commitIdPosition]
	return commitId
}

func (gitRepository *GitRepository) GetGitRepository(fluxGitRepository sourcev1.GitRepository) {
	gitRepository.RepositoryName = getGitRepositoryName(fluxGitRepository)
	gitRepository.BranchName = getBranchName(fluxGitRepository)
	gitRepository.CommitId = getCommitId(fluxGitRepository)
}
