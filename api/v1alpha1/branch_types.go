package v1alpha1

import (
	"sort"
	"strings"

	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Branch struct {
	Name              string `json:"name"`
	Commit            string `json:"commit,omitempty"`
	LatestPipelineRun string `json:"latestPipelineRun,omitempty"`
	Details           string `json:"details,omitempty"`
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

func (currentBranch *Branch) GenerateBranchLabelsAsHash() map[string]string {
	labels := make(map[string]string)

	labels[pipelineTriggerLabelKey+"/"+"pr.branch.name"] = rewriteBranchName(currentBranch.Name)
	labels[pipelineTriggerLabelKey+"/"+"pr.branch.commit"] = currentBranch.Commit

	return labels
}

func (currentBranch *Branch) GenerateBranchLabelsAsString() string {
	label :=
		pipelineTriggerLabelKey + "/" + "pr.branch.name=" + rewriteBranchName(currentBranch.Name) + "," +
			pipelineTriggerLabelKey + "/" + "pr.branch.commit=" + currentBranch.Commit

	return label
}

func (currentBranch *Branch) Equals(newBranch Branch) bool {
	if currentBranch.Name == newBranch.Name && currentBranch.Commit == newBranch.Commit {
		return true
	} else {
		return false
	}
}

func (currentBranch *Branch) GetBranch(jqBranch pullrequestv1alpha1.Branch) {
	currentBranch.Name = getPrBranchName(jqBranch)
	currentBranch.Commit = getPrCommit(jqBranch)
	currentBranch.Details = getPrDetails(jqBranch)
}

func getPrBranchName(jqBranch pullrequestv1alpha1.Branch) string {
	branchName := jqBranch.Name
	return branchName
}

func getPrCommit(jqBranch pullrequestv1alpha1.Branch) string {
	commitId := jqBranch.Commit
	return commitId
}

func getPrDetails(jqBranch pullrequestv1alpha1.Branch) string {
	details := jqBranch.Details
	return details
}

/*
Invalid value: "bugfix/mega-bug": a valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?')
*/
func rewriteBranchName(branchName string) string {
	// Replaces branch names from feature/newlogin to feature-newlogin
	return strings.ReplaceAll(branchName, "/", "-")
}

func (currentBranch *Branch) Rewrite() string {
	// Replaces branch names from feature/newlogin to feature-newlogin
	return strings.ReplaceAll(currentBranch.Name, "/", "-")
}

func (branch *Branch) AddOrReplaceCondition(c metav1.Condition) {
	found := false
	for i, condition := range branch.Conditions {
		if c.Type == condition.Type {
			branch.Conditions[i] = c
			found = true
		}
	}
	if !found {
		branch.Conditions = append(branch.Conditions, c)
	}
}

func (branch *Branch) GetCondition(conditionType string) (metav1.Condition, bool) {
	for _, condition := range branch.Conditions {
		if condition.Type == conditionType {
			return condition, true
		}
	}
	return metav1.Condition{}, false
}

// GetLastCondition retruns the last condition based on the condition timestamp. if no condition is present it return false.
func (branch *Branch) GetLastCondition() metav1.Condition {
	if len(branch.Conditions) == 0 {
		return metav1.Condition{}
	}
	//we need to make a copy of the slice
	copiedConditions := []metav1.Condition{}
	for _, condition := range branch.Conditions {
		ccondition := condition.DeepCopy()
		copiedConditions = append(copiedConditions, *ccondition)
	}
	sort.Slice(copiedConditions, func(i, j int) bool {
		return copiedConditions[i].LastTransitionTime.Before(&copiedConditions[j].LastTransitionTime)
	})
	return copiedConditions[len(copiedConditions)-1]
}
