package v1alpha1

import (
	"strings"

	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type InputParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (inputParam InputParam) CreateParam(pipelineTrigger PipelineTrigger, latestEvent string) tektondevv1.Param {

	if inputParam.Value == "$(branch)" {
		inputParam.Value = getBranchName(latestEvent)
	}

	if inputParam.Value == "$(commit)" {
		inputParam.Value = getCommitId(latestEvent)
	}

	return tektondevv1.Param{
		Name: inputParam.Name,
		Value: tektondevv1.ArrayOrString{
			Type:      tektondevv1.ParamTypeString,
			StringVal: inputParam.Value,
		},
	}
}

func getBranchName(value string) string {
	branchName := strings.Split(value, "/")
	return branchName[0]
}

func getCommitId(value string) string {
	commitId := strings.Split(value, "/")
	return commitId[1]
}
