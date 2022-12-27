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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	pipelineTriggerLabelKey            string = "pipeline.jquad.rocks"
	pipelineTriggerLabelValue          string = "pipelinetrigger"
	pipelineParamDynamicVariableMarker string = "$"
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

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
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

// GetLastCondition retruns the last condition based on the condition timestamp. if no condition is present it return false.
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

func (pipelineTrigger *PipelineTrigger) createParams(details string) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.PipelineRunSpec.Params); paramNr++ {
		pipelineParams = append(pipelineParams, createParam(pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr], details))
	}
	return pipelineParams
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResourceForBranch(currentBranch Branch, labels map[string]string) *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pipelineTrigger.Spec.PipelineRunSpec.Params = pipelineTrigger.createParams(currentBranch.Details)
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: currentBranch.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       labels,
		},
		Spec: pipelineTrigger.Spec.PipelineRunSpec,
	}

	return pr
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResource() *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	var pipelineRunLabels map[string]string
	var pipelineRunName string

	if pipelineTrigger.Spec.Source.Kind == "GitRepository" {
		pipelineTrigger.Spec.PipelineRunSpec.Params = pipelineTrigger.createParams(pipelineTrigger.Status.GitRepository.Details)
		pipelineRunLabels = pipelineTrigger.Status.GitRepository.GenerateGitRepositoryLabelsAsHash()
		pipelineRunName = pipelineTrigger.Status.GitRepository.Rewrite() + "-"
	}

	if pipelineTrigger.Spec.Source.Kind == "ImagePolicy" {
		pipelineTrigger.Spec.PipelineRunSpec.Params = pipelineTrigger.createParams(pipelineTrigger.Status.ImagePolicy.Details)
		pipelineRunLabels = pipelineTrigger.Status.ImagePolicy.GenerateImagePolicyLabelsAsHash()
		pipelineRunName = pipelineTrigger.Status.ImagePolicy.Rewrite() + "-"
	}

	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: pipelineRunName,
			Namespace:    pipelineTrigger.Namespace,
			Labels:       pipelineRunLabels,
		},
		Spec: pipelineTrigger.Spec.PipelineRunSpec,
	}
	return pr
}

func (pipelineTrigger *PipelineTrigger) StartPipelineRun(pr *tektondevv1.PipelineRun, ctx context.Context, req ctrl.Request, tektonClient client.Client) (string, *tektondevv1.PipelineRun) {
	log := log.FromContext(ctx)

	err := tektonClient.Create(ctx, pr)
	if err != nil {
		fmt.Println(err)
		log.Info("Cannot create tekton pipelinerun")
	}

	return pr.Name, pr
}

func createParam(inputParam tektondevv1.Param, details string) tektondevv1.Param {
	if !strings.HasPrefix(inputParam.Value.StringVal, pipelineParamDynamicVariableMarker) {
		return inputParam
	} else {
		res, _ := json.EvalExpr(details, inputParam.Value.StringVal)
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
