package v1alpha1

import (
	"context"
	"testing"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestImagepolicySubscriber_IsValid(t *testing.T) {
	var pipelineRunMockValid unstructured.Unstructured
	pipelineRunMockValid.SetAPIVersion("tekton.dev/v1beta1")
	pipelineRunMockValid.SetKind("PipelineRun")
	pipelineRunMockValid.SetNamespace("default")
	pipelineRunMockValid.Object["spec"] = map[string]interface{}{
		"pipelineRef": map[string]interface{}{
			"name": "build-and-push",
		},
		"params": []interface{}{
			map[string]interface{}{
				"name":  "param1",
				"value": "value1",
			},
			map[string]interface{}{
				"name":  "param2",
				"value": "value2",
			},
		},
	}

	var pipelineRunMockInValid unstructured.Unstructured
	pipelineRunMockInValid.SetAPIVersion("tekton.dev")
	pipelineRunMockInValid.SetKind("PipelineRun")
	pipelineRunMockInValid.SetNamespace("default")
	pipelineRunMockInValid.Object["spec"] = map[string]interface{}{
		"pipelineRef": map[string]interface{}{
			"name": "build-and-push",
		},
		"params": []interface{}{
			map[string]interface{}{
				"name":  "param1",
				"value": "value1",
			},
			map[string]interface{}{
				"name":  "param2",
				"value": "value2",
			},
		},
	}

	validPipelineTrigger := pipelinev1alpha1.PipelineTrigger{
		Spec: pipelinev1alpha1.PipelineTriggerSpec{
			Source: pipelinev1alpha1.Source{
				APIVersion: "v1alpha1/source",
			},
			PipelineRun: pipelineRunMockValid,
		},
	}

	invalidPipelineTrigger := pipelinev1alpha1.PipelineTrigger{
		Spec: pipelinev1alpha1.PipelineTriggerSpec{
			Source: pipelinev1alpha1.Source{
				APIVersion: "invalidversion",
			},
			PipelineRun: pipelineRunMockInValid,
		},
	}

	// Create an instance of your ImagepolicySubscriber
	imagepolicySubscriber := ImagepolicySubscriber{}

	// Test case 1: Valid input
	err := imagepolicySubscriber.IsValid(context.TODO(), validPipelineTrigger, nil, ctrl.Request{})
	assert.NoError(t, err)

	// Test case 2: Invalid source API version
	err = imagepolicySubscriber.IsValid(context.TODO(), invalidPipelineTrigger, nil, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not split the api version of the source as expected")

	// Test case 3: Invalid PipelineRun API version
	invalidPipelineTrigger.Spec.Source.APIVersion = "source.dev/valid"
	invalidPipelineTrigger.Spec.PipelineRun.Object["apiVersion"] = "invalidversion"
	err = imagepolicySubscriber.IsValid(context.TODO(), invalidPipelineTrigger, nil, ctrl.Request{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not split the api version of the pipelinerun as expected")
}
