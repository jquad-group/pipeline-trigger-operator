package v1alpha1

import (
	"reflect"
	"sort"
	"testing"

	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
)

func TestGetPrBranches(t *testing.T) {
	// Create a sample pullrequestv1alpha1.Branches
	prBranches := pullrequestv1alpha1.Branches{
		Branches: []pullrequestv1alpha1.Branch{
			{
				Name:    "branch1",
				Commit:  "commit1",
				Details: "details1",
			},
			{
				Name:    "branch2",
				Commit:  "commit2",
				Details: "details2",
			},
		},
	}

	branches := &Branches{}
	branches.GetPrBranches(prBranches)

	if len(branches.Branches) != 2 {
		t.Errorf("GetPrBranches() failed. Expected 2 branches, got %d", len(branches.Branches))
	}

	// Check if the branches were correctly converted
	expectedBranch1 := Branch{
		Name:    "branch1",
		Commit:  "commit1",
		Details: "details1",
	}
	expectedBranch2 := Branch{
		Name:    "branch2",
		Commit:  "commit2",
		Details: "details2",
	}

	if !reflect.DeepEqual(branches.Branches["branch1"], expectedBranch1) {
		t.Errorf("GetPrBranches() failed for branch1. Got %+v, expected %+v", branches.Branches["branch1"], expectedBranch1)
	}

	if !reflect.DeepEqual(branches.Branches["branch2"], expectedBranch2) {
		t.Errorf("GetPrBranches() failed for branch2. Got %+v, expected %+v", branches.Branches["branch2"], expectedBranch2)
	}
}

func TestGenerateLabelsAsString(t *testing.T) {
	// Create sample Branches
	branches := &Branches{
		Branches: map[string]Branch{
			"branch1": {
				Name:    "branch1",
				Commit:  "commit1",
				Details: "{\"id\":1163006807}",
			},
			"branch2": {
				Name:    "branch2",
				Commit:  "commit2",
				Details: "{\"id\":1163006809}",
			},
		},
	}

	expectedLabels := []string{
		pipelineTriggerLabelKey + "/" + "pr.branch.name=branch1," + pipelineTriggerLabelKey + "/" + "pr.branch.commit=commit1",
		pipelineTriggerLabelKey + "/" + "pr.branch.name=branch2," + pipelineTriggerLabelKey + "/" + "pr.branch.commit=commit2",
	}

	resultLabels := branches.GenerateLabelsAsString()
	// Sort both the expected and actual slices
	sort.Strings(expectedLabels)
	sort.Strings(resultLabels)

	if !reflect.DeepEqual(resultLabels, expectedLabels) {
		t.Errorf("GenerateLabelsAsString() failed. Got %v, expected %v", resultLabels, expectedLabels)
	}
}

func TestEqualsBranches(t *testing.T) {
	// Create sample Branches
	branches := &Branches{
		Branches: map[string]Branch{
			"branch1": {
				Name:   "branch1",
				Commit: "commit1",
			},
			"branch2": {
				Name:   "branch2",
				Commit: "commit2",
			},
		},
	}

	// Branch that exists in the branches
	existingBranch := Branch{
		Name:   "branch1",
		Commit: "commit1",
	}

	// Branch that does not exist in the branches
	nonExistentBranch := Branch{
		Name:   "branch3",
		Commit: "commit3",
	}

	if !branches.Equals(existingBranch) {
		t.Errorf("Equals() failed for an existing branch.")
	}

	if branches.Equals(nonExistentBranch) {
		t.Errorf("Equals() failed for a non-existent branch.")
	}
}

func TestGetSize(t *testing.T) {
	// Create sample Branches
	branches := &Branches{
		Branches: map[string]Branch{
			"branch1": {
				Name:   "branch1",
				Commit: "commit1",
			},
			"branch2": {
				Name:   "branch2",
				Commit: "commit2",
			},
		},
	}

	size := branches.GetSize()
	expectedSize := 2

	if size != expectedSize {
		t.Errorf("GetSize() failed. Got %d, expected %d", size, expectedSize)
	}
}
