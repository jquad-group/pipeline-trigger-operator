package v1alpha1

import (
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
)

type Branches struct {
	// +kubebuilder:validation:Optional
	Branches map[string]Branch `json:"branch,omitempty"`
}

func (branches *Branches) GetPrBranches(prBranches pullrequestv1alpha1.Branches) {
	branches.Branches = make(map[string]Branch)
	for branchNr := 0; branchNr < len(prBranches.Branches); branchNr++ {
		var tempBranch Branch
		tempBranch.GetBranch(prBranches.Branches[branchNr])
		branches.Branches[tempBranch.Name] = tempBranch
	}
}

func (branches *Branches) GenerateLabelsAsString() []string {
	result := []string{}
	for _, value := range branches.Branches {
		label := value.GenerateBranchLabelsAsString()
		result = append(result, label)
	}
	return result
}

func (branches *Branches) Equals(newBranch Branch) bool {
	_, found := branches.Branches[newBranch.Name]
	return found
}

func (branches *Branches) GetSize() int {
	return len(branches.Branches)
}
