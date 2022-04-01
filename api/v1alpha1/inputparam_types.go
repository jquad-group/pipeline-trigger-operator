package v1alpha1

import (
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type InputParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (inputParam InputParam) CreateParam() tektondevv1.Param {

	return tektondevv1.Param{
		Name: inputParam.Name,
		Value: tektondevv1.ArrayOrString{
			Type:      tektondevv1.ParamTypeString,
			StringVal: inputParam.Value,
		},
	}
}
