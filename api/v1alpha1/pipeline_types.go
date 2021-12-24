package v1alpha1

import (
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type Pipeline struct {
	// +required
	Name string `json:"name,omitempty"`

	// +required
	InputParams []InputParam `json:"inputParams,omitempty"`

	// +required
	Workspace Workspace `json:"workspace,omitempty"`

	// +optional
	Status PipelineStatus `json:"status,omitempty"`
}

type PipelineStatus struct {
	// PipelineStatus gives the status of the
	// tekton pipeline currently running.
	PipelineStatus string `json:"PipelineStatus,omitempty"`
	// +optional
}

func (pipeline Pipeline) CreatePipelineRef() *tektondevv1.PipelineRef {
	return &tektondevv1.PipelineRef{
		Name: pipeline.Name,
	}
}

func (pipeline Pipeline) CreateParams() []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipeline.InputParams); paramNr++ {
		pipelineParams = append(pipelineParams, pipeline.InputParams[paramNr].CreateParam())
	}
	return pipelineParams
}
