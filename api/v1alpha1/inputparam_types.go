package v1alpha1

import (
	"strings"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type InputParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (inputParam InputParam) CreateParamGitRepository(gitRepository GitRepository) tektondevv1.Param {

	if !strings.Contains(inputParam.Value, "$") {
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: inputParam.Value,
			},
		}
	} else {
		res, _ := json.EvalExpr(gitRepository.Details, inputParam.Value)
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: res,
			},
		}
	}
}

func (inputParam InputParam) CreateParamImage(imagePolicy ImagePolicy) tektondevv1.Param {

	if !strings.Contains(inputParam.Value, "$") {
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: inputParam.Value,
			},
		}
	} else {
		res, _ := json.EvalExpr(imagePolicy.Details, inputParam.Value)
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: res,
			},
		}
	}
}

func (inputParam InputParam) CreateParam(currentBranch Branch) tektondevv1.Param {
	if !strings.Contains(inputParam.Value, "$") {
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: inputParam.Value,
			},
		}
	} else {
		res, _ := json.EvalExpr(currentBranch.Details, inputParam.Value)
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: res,
			},
		}
	}
}
