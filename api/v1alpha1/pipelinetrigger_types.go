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
	encodedJson "encoding/json"
	"sort"
	"strings"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	PipelineRun unstructured.Unstructured `json:"pipelineRun"`
}

type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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

func (pipelineTrigger *PipelineTrigger) createParams(details string) []Param {

	// Extract "params" field as a generic JSON object
	paramsJSON, _ := encodedJson.Marshal(pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["params"])

	// Create a slice of Param structs
	var params []Param

	// Unmarshal the JSON into the Param struct
	if err := encodedJson.Unmarshal(paramsJSON, &params); err != nil {
		// Handle the error
	}

	var pipelineParams []Param

	for _, param := range params {
		pipelineParams = append(pipelineParams, createParam(param, details))
	}

	return pipelineParams
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResourceForBranch(currentBranch Branch, labels map[string]string) *unstructured.Unstructured {
	// Convert paramList to an unstructured array
	var unstructuredParams []interface{}
	paramList := pipelineTrigger.createParams(currentBranch.Details)
	for _, param := range paramList {
		unstructuredParam := map[string]interface{}{
			"name":  param.Name,
			"value": param.Value,
		}
		unstructuredParams = append(unstructuredParams, unstructuredParam)
	}
	//pipelineTrigger.Spec.PipelineRun.Object["params"] = unstructuredParams
	spec, specFound := pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})
	if !specFound {
		// Handle the case where "name" is not found or not of the expected type
	}
	spec["params"] = unstructuredParams

	pipelineTrigger.Spec.PipelineRun.Object["metadata"].(map[string]interface{})["labels"] = labels

	pipelineRunName := currentBranch.Rewrite() + "-"
	// First, assert the types step by step
	metadata, metadataFound := pipelineTrigger.Spec.PipelineRun.Object["metadata"].(map[string]interface{})
	if !metadataFound {
		// Handle the case where "metadata" is not found or not of the expected type
	}

	metadataName, nameFound := metadata["name"].(string)
	metadataGenerateName, generateNameFound := metadata["generateName"].(string)
	if !nameFound && !generateNameFound {
		// Handle the case where "name" is not found or not of the expected type
	}

	// Now, you can assign pipelineRunName to metadataGenerateName
	if len(metadataName) == 0 && len(metadataGenerateName) == 0 {
		metadataGenerateName = pipelineRunName
		// Update the "metadata" map within the object
		metadata["generateName"] = metadataName
		// Finally, update the object within pipelineTrigger
		pipelineTrigger.Spec.PipelineRun.Object["metadata"] = metadata
	}

	return &pipelineTrigger.Spec.PipelineRun
}

func (pipelineTrigger *PipelineTrigger) CreatePipelineRunResource() *unstructured.Unstructured {

	var pipelineRunLabels map[string]string
	var pipelineRunName string

	if pipelineTrigger.Spec.Source.Kind == "GitRepository" {
		// Convert paramList to an unstructured array
		var unstructuredParams []interface{}
		paramList := pipelineTrigger.createParams(pipelineTrigger.Status.GitRepository.Details)
		for _, param := range paramList {
			unstructuredParam := map[string]interface{}{
				"name":  param.Name,
				"value": param.Value,
			}
			unstructuredParams = append(unstructuredParams, unstructuredParam)
		}
		//pipelineTrigger.Spec.PipelineRun.Object["params"] = unstructuredParams
		spec, specFound := pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})
		if !specFound {
			// Handle the case where "name" is not found or not of the expected type
		}
		spec["params"] = unstructuredParams

		pipelineRunLabels = pipelineTrigger.Status.GitRepository.GenerateGitRepositoryLabelsAsHash()
		pipelineTrigger.Spec.PipelineRun.Object["metadata"].(map[string]interface{})["labels"] = pipelineRunLabels

		pipelineRunName = pipelineTrigger.Status.GitRepository.Rewrite() + "-"
		// First, assert the types step by step
		metadata, metadataFound := pipelineTrigger.Spec.PipelineRun.Object["metadata"].(map[string]interface{})
		if !metadataFound {
			// Handle the case where "metadata" is not found or not of the expected type
		}

		metadataName, nameFound := metadata["name"].(string)
		metadataGenerateName, generateNameFound := metadata["generateName"].(string)
		if !nameFound && !generateNameFound {
			// Handle the case where "name" is not found or not of the expected type
		}

		// Now, you can assign pipelineRunName to metadataGenerateName
		if len(metadataName) == 0 && len(metadataGenerateName) == 0 {
			metadataGenerateName = pipelineRunName
			// Update the "metadata" map within the object
			metadata["generateName"] = metadataName
			// Finally, update the object within pipelineTrigger
			pipelineTrigger.Spec.PipelineRun.Object["metadata"] = metadata
		}

	}

	if pipelineTrigger.Spec.Source.Kind == "ImagePolicy" {
		// Convert paramList to an unstructured array
		var unstructuredParams []interface{}
		paramList := pipelineTrigger.createParams(pipelineTrigger.Status.ImagePolicy.Details)
		for _, param := range paramList {
			unstructuredParam := map[string]interface{}{
				"name":  param.Name,
				"value": param.Value,
			}
			unstructuredParams = append(unstructuredParams, unstructuredParam)
		}

		spec, specFound := pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})
		if !specFound {
			// Handle the case where "name" is not found or not of the expected type
		}
		spec["params"] = unstructuredParams
		pipelineRunLabels = pipelineTrigger.Status.ImagePolicy.GenerateImagePolicyLabelsAsHash()
		pipelineTrigger.Spec.PipelineRun.Object["metadata"].(map[string]interface{})["labels"] = pipelineRunLabels

		pipelineRunName = pipelineTrigger.Status.ImagePolicy.Rewrite() + "-"
		// First, assert the types step by step
		metadata, metadataFound := pipelineTrigger.Spec.PipelineRun.Object["metadata"].(map[string]interface{})
		if !metadataFound {
			// Handle the case where "metadata" is not found or not of the expected type
		}

		metadataName, nameFound := metadata["name"].(string)
		metadataGenerateName, generateNameFound := metadata["generateName"].(string)
		if !nameFound && !generateNameFound {
			// Handle the case where "name" is not found or not of the expected type
		}

		// Now, you can assign pipelineRunName to metadataGenerateName
		if len(metadataName) == 0 && len(metadataGenerateName) == 0 {
			metadataGenerateName = pipelineRunName
			// Update the "metadata" map within the object
			metadata["generateName"] = metadataName
			// Finally, update the object within pipelineTrigger
			pipelineTrigger.Spec.PipelineRun.Object["metadata"] = metadata
		}

	}
	/*
		pr := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "tekton.dev/v1",
				"kind":       "PipelineRun",
				"metadata": map[string]interface{}{
					"generateName": pipelineRunName,
					"namespace":    pipelineTrigger.Namespace,
					"labels":       pipelineRunLabels,
				},
				"spec": pipelineTrigger.Spec.PipelineRun,
			},
		}
	*/

	return &pipelineTrigger.Spec.PipelineRun
}

func (pipelineTrigger *PipelineTrigger) StartPipelineRun(pr *unstructured.Unstructured, ctx context.Context, req ctrl.Request, tektonClient client.Client) (string, *unstructured.Unstructured, error) {
	log := log.FromContext(ctx)

	err := tektonClient.Create(ctx, pr)
	if err != nil {
		log.Info("Cannot create tekton pipelinerun")
		return "", nil, err
	}

	return pr.GetName(), pr, nil
}

func createParam(inputParam Param, details string) Param {

	if !strings.HasPrefix(inputParam.Value, pipelineParamDynamicVariableMarker) {
		return inputParam
	} else {
		res, _ := json.EvalExpr(details, inputParam.Value)
		return Param{
			Name:  inputParam.Name,
			Value: trimQuotes(res),
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
