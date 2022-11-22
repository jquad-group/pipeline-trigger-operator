/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/meta"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	pipelineTriggerLabelKey   string = "pipeline.jquad.rocks"
	pipelineTriggerLabelValue string = "pipelinetrigger"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PipelineTrigger is the Schema for the pipelinetriggers API
type PipelineTrigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PipelineTriggerSpec `json:"spec,omitempty"`

	Status PipelineTriggerStatus `json:"status,omitempty"`
}

// PipelineTriggerSpec defines the desired state of PipelineTrigger
type PipelineTriggerSpec struct {
	// Source points at the object specifying the Image Policy, Git Repository or Pull Request found
	// +kubebuilder:validation:Required
	Source Source `json:"source"`

	// +kubebuilder:validation:Required
	PipelineRunSpec tektondevv1.PipelineRunSpec `json:"pipelineRunSpec"`
}

// PipelineTriggerStatus defines the observed state of PipelineTrigger
type PipelineTriggerStatus struct {
	// +kubebuilder:validation:Optional
	ImagePolicy ImagePolicy `json:"imagePolicy,omitempty"`

	// +kubebuilder:validation:Optional
	GitRepository GitRepository `json:"gitRepository,omitempty"`

	// +kubebuilder:validation:Optional
	Branches Branches `json:"branches,omitempty"`

	// https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

func (m *PipelineTrigger) GetConditions() []metav1.Condition {
	return m.Status.Conditions
}

func (m *PipelineTrigger) SetConditions(conditions []metav1.Condition) {
	m.Status.Conditions = conditions
}

//GetLastCondition retruns the last condition based on the condition timestamp. if no condition is present it return false.
func (m *PipelineTrigger) GetLastCondition() metav1.Condition {
	if len(m.Status.Conditions) == 0 {
		return metav1.Condition{}
	}
	//we need to make a copy of the slice
	copiedConditions := []metav1.Condition{}
	for _, condition := range m.Status.Conditions {
		ccondition := condition.DeepCopy()
		copiedConditions = append(copiedConditions, *ccondition)
	}
	sort.Slice(copiedConditions, func(i, j int) bool {
		return copiedConditions[i].LastTransitionTime.Before(&copiedConditions[j].LastTransitionTime)
	})
	return copiedConditions[len(copiedConditions)-1]
}

func (m *PipelineTrigger) GetCondition(conditionType string) (metav1.Condition, bool) {
	for _, condition := range m.Status.Conditions {
		if condition.Type == conditionType {
			return condition, true
		}
	}
	return metav1.Condition{}, false
}

func (m *PipelineTrigger) ReplaceCondition(c metav1.Condition) {
	if len(m.Status.Conditions) == 0 {
		m.Status.Conditions = append(m.Status.Conditions, c)
	} else {
		m.Status.Conditions[0] = c
	}
}

func (m *PipelineTrigger) AddOrReplaceCondition(c metav1.Condition) {
	found := false
	for i, condition := range m.Status.Conditions {
		if c.Type == condition.Type {
			m.Status.Conditions[i] = c
			found = true
		}
	}
	if !found {
		m.Status.Conditions = append(m.Status.Conditions, c)
	}
}

//+kubebuilder:object:root=true

// PipelineTriggerList contains a list of PipelineTrigger
type PipelineTriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PipelineTrigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PipelineTrigger{}, &PipelineTriggerList{})
}

func (pipelineTrigger *PipelineTrigger) createPipelineRef() *tektondevv1.PipelineRef {
	return &tektondevv1.PipelineRef{
		Name: pipelineTrigger.Spec.PipelineRunSpec.PipelineRef.Name,
	}
}

func (pipelineTrigger *PipelineTrigger) createParams(currentBranch Branch) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.PipelineRunSpec.Params); paramNr++ {
		pipelineParams = append(pipelineParams, CreateParam(pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr], currentBranch))
	}
	return pipelineParams
}

func (pipelineTrigger *PipelineTrigger) createParamsGitRepository(gitRepository GitRepository) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.PipelineRunSpec.Params); paramNr++ {
		pipelineParams = append(pipelineParams, CreateParamGitRepository(pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr], gitRepository))
	}
	return pipelineParams
}

func (pipelineTrigger *PipelineTrigger) createParamsImagePolicy(imagePolicy ImagePolicy) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.PipelineRunSpec.Params); paramNr++ {
		pipelineParams = append(pipelineParams, CreateParamImage(pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr], imagePolicy))
	}
	return pipelineParams
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResourceForBranch(currentBranch Branch, labels map[string]string) *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: currentBranch.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       labels,
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipelineTrigger.Spec.PipelineRunSpec.ServiceAccountName,
			PipelineRef:        pipelineTrigger.createPipelineRef(),
			Params:             pipelineTrigger.createParams(currentBranch),
			Workspaces:         pipelineTrigger.Spec.PipelineRunSpec.Workspaces,
			PodTemplate:        pipelineTrigger.Spec.PipelineRunSpec.PodTemplate,
		},
	}

	return pr
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResourceForGit() *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pipelineTrigger.Status.GitRepository.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       pipelineTrigger.Status.GitRepository.GenerateGitRepositoryLabelsAsHash(),
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipelineTrigger.Spec.PipelineRunSpec.ServiceAccountName,
			PipelineRef:        pipelineTrigger.createPipelineRef(),
			Params:             pipelineTrigger.createParamsGitRepository(pipelineTrigger.Status.GitRepository),
			Workspaces:         pipelineTrigger.Spec.PipelineRunSpec.Workspaces,
			PodTemplate:        pipelineTrigger.Spec.PipelineRunSpec.PodTemplate,
		},
	}
	return pr
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResourceForImage() *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pipelineTrigger.Status.ImagePolicy.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       pipelineTrigger.Status.ImagePolicy.GenerateImagePolicyLabelsAsHash(),
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipelineTrigger.Spec.PipelineRunSpec.ServiceAccountName,
			PipelineRef:        pipelineTrigger.createPipelineRef(),
			Params:             pipelineTrigger.createParamsImagePolicy(pipelineTrigger.Status.ImagePolicy),
			Workspaces:         pipelineTrigger.Spec.PipelineRunSpec.Workspaces,
			PodTemplate:        pipelineTrigger.Spec.PipelineRunSpec.PodTemplate,
		},
	}
	return pr
}

func (pipelineTrigger *PipelineTrigger) StartPipelineRun(pr *tektondevv1.PipelineRun, ctx context.Context, req ctrl.Request) (string, *tektondevv1.PipelineRun) {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	opts := metav1.CreateOptions{}
	prInstance, err := tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).Create(ctx, pr, opts)
	if err != nil {
		fmt.Println(err)
		log.Info("Cannot create tekton pipelinerun")
	}

	return prInstance.Name, prInstance
}

func CreateParamGitRepository(inputParam tektondevv1.Param, gitRepository GitRepository) tektondevv1.Param {

	if !strings.HasPrefix(inputParam.Value.StringVal, "$") {
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: inputParam.Value.StringVal,
			},
		}
	} else {
		res, _ := json.EvalExpr(gitRepository.Details, inputParam.Value.StringVal)
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: trimQuotes(res),
			},
		}
	}
}

func CreateParamImage(inputParam tektondevv1.Param, imagePolicy ImagePolicy) tektondevv1.Param {

	if !strings.HasPrefix(inputParam.Value.StringVal, "$") {
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: inputParam.Value.StringVal,
			},
		}
	} else {
		res, _ := json.EvalExpr(imagePolicy.Details, inputParam.Value.StringVal)
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: trimQuotes(res),
			},
		}
	}
}

func CreateParam(inputParam tektondevv1.Param, currentBranch Branch) tektondevv1.Param {
	if !strings.HasPrefix(inputParam.Value.StringVal, "$") {
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: inputParam.Value.StringVal,
			},
		}
	} else {
		res, _ := json.EvalExpr(currentBranch.Details, inputParam.Value.StringVal)
		return tektondevv1.Param{
			Name: inputParam.Name,
			Value: tektondevv1.ArrayOrString{
				Type:      tektondevv1.ParamTypeString,
				StringVal: trimQuotes(res),
			},
		}
	}
}

func trimQuotes(paramValue string) string {
	trimedParam := paramValue
	if trimedParam[0] == '"' {
		trimedParam = trimedParam[1:]
	}
	if i := len(trimedParam) - 1; trimedParam[i] == '"' {
		trimedParam = trimedParam[:i]
	}
	return trimedParam
}
