package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateBranchLabelsAsHash(t *testing.T) {
	currentBranch := &Branch{
		Name:   "branchName",
		Commit: "commitHash",
	}

	expectedLabels := map[string]string{
		pipelineTriggerLabelKey + "/" + "pr.branch.name":   "branchName",
		pipelineTriggerLabelKey + "/" + "pr.branch.commit": "commitHash",
	}

	resultLabels := currentBranch.GenerateBranchLabelsAsHash()
	if !reflect.DeepEqual(resultLabels, expectedLabels) {
		t.Errorf("GenerateBranchLabelsAsHash() failed. Got %v, expected %v", resultLabels, expectedLabels)
	}
}

func TestGenerateBranchLabelsAsString(t *testing.T) {
	currentBranch := &Branch{
		Name:   "branchName",
		Commit: "commitHash",
	}

	expectedLabel := pipelineTriggerLabelKey + "/" + "pr.branch.name=branchName," + pipelineTriggerLabelKey + "/" + "pr.branch.commit=commitHash"

	resultLabel := currentBranch.GenerateBranchLabelsAsString()
	if resultLabel != expectedLabel {
		t.Errorf("GenerateBranchLabelsAsString() failed. Got %s, expected %s", resultLabel, expectedLabel)
	}
}

func TestEquals(t *testing.T) {
	currentBranch := &Branch{
		Name:   "branchName",
		Commit: "commitHash",
	}

	// Branch with the same name and commit
	sameBranch := Branch{
		Name:   "branchName",
		Commit: "commitHash",
	}

	// Branch with a different name
	differentNameBranch := Branch{
		Name:   "differentBranch",
		Commit: "commitHash",
	}

	// Branch with a different commit
	differentCommitBranch := Branch{
		Name:   "branchName",
		Commit: "differentCommitHash",
	}

	if !currentBranch.Equals(sameBranch) {
		t.Errorf("Equals() failed for the same branch.")
	}

	if currentBranch.Equals(differentNameBranch) {
		t.Errorf("Equals() failed for branches with different names.")
	}

	if currentBranch.Equals(differentCommitBranch) {
		t.Errorf("Equals() failed for branches with different commits.")
	}
}

func TestGetBranch(t *testing.T) {
	jqBranch := pullrequestv1alpha1.Branch{
		Name:    "jqBranchName",
		Commit:  "jqCommitHash",
		Details: "jqBranchDetails",
	}

	currentBranch := &Branch{}
	currentBranch.GetBranch(jqBranch)

	if currentBranch.Name != "jqBranchName" || currentBranch.Commit != "jqCommitHash" || currentBranch.Details != "jqBranchDetails" {
		t.Errorf("GetBranch() failed. Got %+v, expected %+v", currentBranch, jqBranch)
	}
}

func TestRewriteBranchName(t *testing.T) {
	originalBranchName := "feature/newlogin"
	expectedRewrittenName := "feature-newlogin"

	result := rewriteBranchName(originalBranchName)

	if result != expectedRewrittenName {
		t.Errorf("RewriteBranchName() failed. Got %s, expected %s", result, expectedRewrittenName)
	}
}

func TestRewrite(t *testing.T) {
	currentBranch := &Branch{
		Name: "feature/newlogin",
	}

	expectedRewrittenName := "feature-newlogin"

	result := currentBranch.Rewrite()

	if result != expectedRewrittenName {
		t.Errorf("Rewrite() failed. Got %s, expected %s", result, expectedRewrittenName)
	}
}

func TestAddOrReplaceConditionBranch(t *testing.T) {
	branch := &Branch{
		Conditions: []metav1.Condition{
			{
				Type:               "ConditionType1",
				LastTransitionTime: metav1.NewTime(time.Now()),
			},
		},
	}

	newCondition := metav1.Condition{
		Type:               "ConditionType1",
		Status:             "True",
		LastTransitionTime: metav1.NewTime(time.Now().Add(5 * time.Second)),
	}

	branch.AddOrReplaceCondition(newCondition)

	if len(branch.Conditions) != 1 {
		t.Errorf("AddOrReplaceCondition() added a duplicate condition. Expected 1 condition, got %d", len(branch.Conditions))
	}

	if !reflect.DeepEqual(branch.Conditions[0], newCondition) {
		t.Errorf("AddOrReplaceCondition() didn't replace the existing condition correctly.")
	}

	newCondition.Type = "ConditionType2"
	branch.AddOrReplaceCondition(newCondition)

	if len(branch.Conditions) != 2 {
		t.Errorf("AddOrReplaceCondition() didn't add a new condition. Expected 2 conditions, got %d", len(branch.Conditions))
	}

	if !reflect.DeepEqual(branch.Conditions[1], newCondition) {
		t.Errorf("AddOrReplaceCondition() didn't add a new condition correctly.")
	}
}

func TestGetCondition(t *testing.T) {
	condition1 := metav1.Condition{
		Type: "ConditionType1",
	}

	condition2 := metav1.Condition{
		Type: "ConditionType2",
	}

	branch := &Branch{
		Conditions: []metav1.Condition{
			condition1,
			condition2,
		},
	}

	foundCondition, found := branch.GetCondition("ConditionType1")
	if !found || !reflect.DeepEqual(foundCondition, condition1) {
		t.Errorf("GetCondition() for ConditionType1 failed. Got %+v, expected %+v", foundCondition, condition1)
	}

	_, found = branch.GetCondition("NonExistentCondition")
	if found {
		t.Errorf("GetCondition() for a non-existent condition should return false.")
	}
}

func TestGetLastConditionBranch(t *testing.T) {
	condition1 := metav1.Condition{
		Type:               "ConditionType1",
		LastTransitionTime: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
	}

	condition2 := metav1.Condition{
		Type:               "ConditionType2",
		LastTransitionTime: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
	}

	branch := &Branch{
		Conditions: []metav1.Condition{condition1, condition2},
	}

	lastCondition := branch.GetLastCondition()

	if !reflect.DeepEqual(lastCondition, condition1) {
		t.Errorf("GetLastCondition() did not return the latest condition. Got %+v, expected %+v", lastCondition, condition1)
	}
}
