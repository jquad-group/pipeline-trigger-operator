package v1alpha1

/*
import (
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

*/

const (
	pipelineTriggerLabelKey   string = "pipeline.jquad.rocks"
	pipelineTriggerLabelValue string = "pipelinetrigger"
)

type Event struct {
	// +kubebuilder:validation:Optional
	ImagePolicy ImagePolicy `json:"imagePolicy,omitempty"`

	// +kubebuilder:validation:Optional
	GitRepository GitRepository `json:"gitRepository,omitempty"`

	// +kubebuilder:validation:Optional
	Branches Branches `json:"branches,omitempty"`
}

func (event *Event) ToString() []string {
	result := []string{}
	if len(event.Branches.Branches) > 0 {
		for _, value := range event.Branches.Branches {
			result = append(result, value.Name+"/"+value.Commit)
		}
	}
	if event.ImagePolicy.RepositoryName != "" {
		result = append(result, event.ImagePolicy.ImageName+"/"+event.ImagePolicy.ImageVersion)
	}
	if event.GitRepository.RepositoryName != "" {
		result = append(result, event.GitRepository.BranchName+"/"+event.GitRepository.CommitId)
	}
	return result
}

func (event *Event) GetSize() int {
	if len(event.Branches.Branches) > 0 {
		return len(event.Branches.Branches)
	}
	if event.ImagePolicy.RepositoryName != "" {
		return 1
	}
	if event.GitRepository.RepositoryName != "" {
		return 1
	}
	return 0
}

func (event *Event) getLabelsAsHash(name string) []map[string]string {
	var result []map[string]string
	labels := make(map[string]string)

	labels[pipelineTriggerLabelKey+"/"+pipelineTriggerLabelValue] = name
	labels[pipelineTriggerLabelKey+"/"+"gitrepository"] = event.GitRepository.RepositoryName
	labels[pipelineTriggerLabelKey+"/"+"gitbranch"] = event.GitRepository.BranchName
	labels[pipelineTriggerLabelKey+"/"+"gitcommit"] = event.GitRepository.CommitId
	labels[pipelineTriggerLabelKey+"/"+"imagerepository"] = event.ImagePolicy.RepositoryName
	labels[pipelineTriggerLabelKey+"/"+"imagename"] = event.ImagePolicy.ImageName
	labels[pipelineTriggerLabelKey+"/"+"imageversion"] = event.ImagePolicy.ImageVersion
	prCounter := 0
	for _, value := range event.Branches.Branches {
		labels[pipelineTriggerLabelKey+"/"+"prbranchname"] = value.Name
		labels[pipelineTriggerLabelKey+"/"+"prbranchcommit"] = value.Commit
		result[prCounter] = labels
		prCounter++
	}

	return result
}

/*
func (event *Event) getLabelsAsHash(name string) []map[string]string {
	var result []map[string]string
	labels := make(map[string]string)

	labels[pipelineTriggerLabelKey+"/"+pipelineTriggerLabelValue] = name
	labels[pipelineTriggerLabelKey+"/"+"gitrepository"] = event.GitRepository.RepositoryName
	labels[pipelineTriggerLabelKey+"/"+"gitbranch"] = event.GitRepository.BranchName
	labels[pipelineTriggerLabelKey+"/"+"gitcommit"] = event.GitRepository.CommitId
	labels[pipelineTriggerLabelKey+"/"+"imagerepository"] = event.ImagePolicy.RepositoryName
	labels[pipelineTriggerLabelKey+"/"+"imagename"] = event.ImagePolicy.ImageName
	labels[pipelineTriggerLabelKey+"/"+"imageversion"] = event.ImagePolicy.ImageVersion
	prCounter := 0
	for _, value := range event.Branches.Branches {
		labels[pipelineTriggerLabelKey+"/"+"prbranchname"] = value.Name
		labels[pipelineTriggerLabelKey+"/"+"prbranchcommit"] = value.Commit
		result[prCounter] = labels
		prCounter++
	}

	return result
}

func (event *Event) GetLabelsAsString(name string) []string {
	var result []string
	labels := ""

	if len(event.Branches.Branches) > 0 {
		prCounter := 0
		for _, value := range event.Branches.Branches {
			labels = labels + "pipeline.jquad.rocks" + "/" + "prbranchname=" + value.Name + "," +
				"pipeline.jquad.rocks" + "/" + "prbranchcommit=" + value.Commit + ","
			labels = labels +
				pipelineTriggerLabelKey + "/" + pipelineTriggerLabelValue + "=" + name + "," +
				pipelineTriggerLabelKey + "/" + "gitrepository" + "=" + event.GitRepository.RepositoryName + "," +
				pipelineTriggerLabelKey + "/" + "gitbranch" + "=" + event.GitRepository.BranchName + "," +
				pipelineTriggerLabelKey + "/" + "gitcommit" + "=" + event.GitRepository.CommitId + "," +
				pipelineTriggerLabelKey + "/" + "imagerepository" + "=" + event.ImagePolicy.RepositoryName + "," +
				pipelineTriggerLabelKey + "/" + "imagename" + "=" + event.ImagePolicy.ImageName + "," +
				pipelineTriggerLabelKey + "/" + "imageversion" + "=" + event.ImagePolicy.ImageVersion + ","

			result[prCounter] = labels
			prCounter++
		}
		return result
	} else {

		labels = labels +
			pipelineTriggerLabelKey + "/" + pipelineTriggerLabelValue + "=" + name + "," +
			pipelineTriggerLabelKey + "/" + "gitrepository" + "=" + event.GitRepository.RepositoryName + "," +
			pipelineTriggerLabelKey + "/" + "gitbranch" + "=" + event.GitRepository.BranchName + "," +
			pipelineTriggerLabelKey + "/" + "gitcommit" + "=" + event.GitRepository.CommitId + "," +
			pipelineTriggerLabelKey + "/" + "imagerepository" + "=" + event.ImagePolicy.RepositoryName + "," +
			pipelineTriggerLabelKey + "/" + "imagename" + "=" + event.ImagePolicy.ImageName + "," +
			pipelineTriggerLabelKey + "/" + "imageversion" + "=" + event.ImagePolicy.ImageVersion + ","
		result[0] = labels
		return result
	}

}
*/

func (event *Event) GetUpdates(newEvent Event) Event {
	var myEvent Event
	// event is from type branch
	if len(newEvent.Branches.Branches) > 0 {
		myEvent.Branches.Branches = make(map[string]Branch)
		if event.Branches.Branches == nil {
			event.Branches.Branches = make(map[string]Branch)
		}
		for key, value := range newEvent.Branches.Branches {
			tempBranch := event.Branches.Branches[key]
			if !tempBranch.Equals(value) {
				myEvent.Branches.Branches[key] = value
			}
		}
	}
	// event is from type imagepolicy
	if newEvent.ImagePolicy.RepositoryName != "" {
		if !event.ImagePolicy.Equals(newEvent.ImagePolicy) {
			myEvent.ImagePolicy = newEvent.ImagePolicy
		}
	}
	// event is from type gitrepository
	if newEvent.GitRepository.RepositoryName != "" {
		if !event.GitRepository.Equals(newEvent.GitRepository) {
			myEvent.GitRepository = newEvent.GitRepository
		}
	}
	return myEvent
}

func (event *Event) AddOrReplace(newEvent Event) {
	// event is from type branch
	if len(newEvent.Branches.Branches) > 0 {
		if event.Branches.Branches == nil {
			event.Branches.Branches = make(map[string]Branch)
		}
		//for key, value := range newEvent.Branches.Branches {
		event.Branches.Branches = newEvent.Branches.Branches
		//}
	}
	// event is from type imagepolicy
	if newEvent.ImagePolicy.RepositoryName != "" {
		if !event.ImagePolicy.Equals(newEvent.ImagePolicy) {
			if event.ImagePolicy.isUpdated(newEvent.ImagePolicy) {
				event.ImagePolicy.ImageVersion = newEvent.ImagePolicy.ImageVersion
			} else {
				event.ImagePolicy = newEvent.ImagePolicy
			}
		}
	}
	// event is from type gitrepository
	if newEvent.GitRepository.RepositoryName != "" {
		if !event.GitRepository.Equals(newEvent.GitRepository) {
			if event.GitRepository.isUpdated(newEvent.GitRepository) {
				event.GitRepository.CommitId = newEvent.GitRepository.CommitId
			} else {
				event.GitRepository = newEvent.GitRepository
			}
		}
	}

}

func (event *Event) Equals(newEvent Event) bool {
	isEqual := true
	// event from type branch
	if len(newEvent.Branches.Branches) > 0 {
		for key, value := range event.Branches.Branches {
			newEventBranch := newEvent.Branches.Branches[key]
			if newEventBranch.Equals(value) {
				isEqual = false
				return isEqual
			}
		}
	}
	// event is from type imagepolicy
	if newEvent.ImagePolicy.RepositoryName != "" {
		if !event.ImagePolicy.Equals(newEvent.ImagePolicy) {
			isEqual = false
			return isEqual
		}
	}
	// event is from type gitrepository
	if newEvent.GitRepository.RepositoryName != "" {
		if !event.GitRepository.Equals(newEvent.GitRepository) {
			isEqual = false
			return isEqual
		}
	}

	return isEqual
}
