package v1alpha1

import (
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCreateParam(t *testing.T) {
	inputParam := Param{
		Name:  "example",
		Value: "$.id",
	}

	details := "{\"id\":1163006807}"

	result := createParam(inputParam, details)

	expected := Param{
		Name:  "example",
		Value: "1163006807",
	}

	if result != expected {
		t.Errorf("createParam failed. Expected: %v, Got: %v", expected, result)
	}
}

func TestReplaceCondition(t *testing.T) {
	// Create a sample PipelineTrigger
	pipelineTrigger := &PipelineTrigger{
		Status: PipelineTriggerStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "ConditionA",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
				},
				{
					Type:               "ConditionB",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
				},
			},
		},
	}

	// Create a new condition to replace an existing condition
	newCondition := metav1.Condition{
		Type:               "ConditionB",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
	}

	// Call the ReplaceCondition method
	pipelineTrigger.ReplaceCondition(newCondition)

	// Check if the condition was replaced
	if len(pipelineTrigger.Status.Conditions) != 2 {
		t.Errorf("ReplaceCondition() failed. Expected 2 conditions, got %d", len(pipelineTrigger.Status.Conditions))
	}

	// Check if the replaced condition matches the new condition
	if pipelineTrigger.Status.Conditions[0] != newCondition {
		t.Errorf("ReplaceCondition() failed. Condition not replaced as expected.")
	}
}

func TestGetLastCondition(t *testing.T) {
	// Create a sample PipelineTrigger with conditions
	pipelineTrigger := &PipelineTrigger{
		Status: PipelineTriggerStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "ConditionA",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
				},
				{
					Type:               "ConditionB",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
				},
				{
					Type:               "ConditionC",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
				},
			},
		},
	}

	// Call the function you want to test
	lastCondition := pipelineTrigger.GetLastCondition()

	// Define the expected result (the last condition based on time)
	expectedCondition := metav1.Condition{
		Type:               "ConditionC",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Date(2023, time.October, 26, 12, 0, 0, 0, time.UTC)),
	}

	// Check if the result matches the expected condition
	if lastCondition != expectedCondition {
		t.Errorf("GetLastCondition() failed. Expected: %v, Got: %v", expectedCondition, lastCondition)
	}
}

func TestGetLastConditionNoConditions(t *testing.T) {
	// Create a sample PipelineTrigger with no conditions
	pipelineTrigger := &PipelineTrigger{
		Status: PipelineTriggerStatus{
			Conditions: []metav1.Condition{},
		},
	}

	// Call the function you want to test
	lastCondition := pipelineTrigger.GetLastCondition()

	// Define the expected result (an empty condition)
	expectedCondition := metav1.Condition{}

	// Check if the result matches the expected condition
	if lastCondition != expectedCondition {
		t.Errorf("GetLastCondition() failed. Expected: %v, Got: %v", expectedCondition, lastCondition)
	}
}

func TestAddOrReplaceCondition(t *testing.T) {
	// Create a sample PipelineTrigger
	pipelineTrigger := &PipelineTrigger{
		Status: PipelineTriggerStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "ConditionA",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               "ConditionB",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a new condition to add or replace
	newCondition := metav1.Condition{
		Type:               "ConditionB",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}

	// Call the AddOrReplaceCondition method
	pipelineTrigger.AddOrReplaceCondition(newCondition)

	// Check if the condition was replaced
	if len(pipelineTrigger.Status.Conditions) != 2 {
		t.Errorf("AddOrReplaceCondition() failed. Expected 2 conditions, got %d", len(pipelineTrigger.Status.Conditions))
	}

	// Check if the replaced condition matches the new condition
	if pipelineTrigger.Status.Conditions[1] != newCondition {
		t.Errorf("AddOrReplaceCondition() failed. Condition not replaced as expected.")
	}
}

func TestAddOrReplaceConditionNewCondition(t *testing.T) {
	// Create a sample PipelineTrigger
	pipelineTrigger := &PipelineTrigger{
		Status: PipelineTriggerStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "ConditionA",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
				},
				{
					Type:               "ConditionB",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}

	// Create a new condition with a different type
	newCondition := metav1.Condition{
		Type:               "ConditionC",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}

	// Call the AddOrReplaceCondition method
	pipelineTrigger.AddOrReplaceCondition(newCondition)

	// Check if a new condition was added
	if len(pipelineTrigger.Status.Conditions) != 3 {
		t.Errorf("AddOrReplaceCondition() failed. Expected 3 conditions, got %d", len(pipelineTrigger.Status.Conditions))
	}

	// Check if the new condition was added as expected
	if pipelineTrigger.Status.Conditions[2] != newCondition {
		t.Errorf("AddOrReplaceCondition() failed. New condition not added as expected.")
	}
}

/////asdadasd

func TestCreateParams(t *testing.T) {
	// Create a sample PipelineTrigger
	pipelineTrigger := &PipelineTrigger{
		Spec: PipelineTriggerSpec{
			PipelineRun: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"params": []map[string]interface{}{
							{
								"name":  "param1",
								"value": "value1",
							},
							{
								"name":  "param2",
								"value": "value2",
							},
						},
					},
				},
			},
		},
	}

	// Create a sample 'details' string
	details := "{\"id\":1163006807}"

	// Call the createParams method
	pipelineParams := pipelineTrigger.createParams(details)

	// Define the expected Param structs
	expectedParams := []Param{
		{
			Name:  "param1",
			Value: "value1",
		},
		{
			Name:  "param2",
			Value: "value2",
		},
	}

	// Check if the result matches the expected Params
	if len(pipelineParams) != len(expectedParams) {
		t.Errorf("createParams() failed. Expected %d params, got %d", len(expectedParams), len(pipelineParams))
	}

	for i, expectedParam := range expectedParams {
		if pipelineParams[i] != expectedParam {
			t.Errorf("createParams() failed. Param %d does not match the expected value.", i)
		}
	}
}

func TestCreateParamsEmptyParams(t *testing.T) {
	// Create a sample PipelineTrigger with empty params
	pipelineTrigger := &PipelineTrigger{
		Spec: PipelineTriggerSpec{
			PipelineRun: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"params": []map[string]interface{}{},
					},
				},
			},
		},
	}

	// Create a sample 'details' string
	details := "{\"id\":1163006807}"

	// Call the createParams method
	pipelineParams := pipelineTrigger.createParams(details)

	// Check if the result is an empty slice
	if len(pipelineParams) != 0 {
		t.Errorf("createParams() failed. Expected 0 params for empty 'params' field, got %d", len(pipelineParams))
	}
}

func TestCreateParamsUnmarshalError(t *testing.T) {
	// Create a sample PipelineTrigger with invalid JSON in 'params' field
	pipelineTrigger := &PipelineTrigger{
		Spec: PipelineTriggerSpec{
			PipelineRun: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"params": "invalid-json",
					},
				},
			},
		},
	}

	// Create a sample 'details' string
	details := "{\"id\":1163006807}"

	// Call the createParams method
	pipelineParams := pipelineTrigger.createParams(details)

	// Check if the result is an empty slice due to unmarshal error
	if len(pipelineParams) != 0 {
		t.Errorf("createParams() failed. Expected 0 params for invalid 'params' field, got %d", len(pipelineParams))
	}
}

func TestTrimQuotes(t *testing.T) {
	// Test case 1: Param value with leading and trailing quotes
	input1 := `"quoted value"`
	expected1 := "quoted value"
	result1 := trimQuotes(input1)
	if result1 != expected1 {
		t.Errorf("TrimQuotes(%s) = %s, expected %s", input1, result1, expected1)
	}

	// Test case 2: Param value without quotes
	input2 := "unquoted value"
	expected2 := "unquoted value"
	result2 := trimQuotes(input2)
	if result2 != expected2 {
		t.Errorf("TrimQuotes(%s) = %s, expected %s", input2, result2, expected2)
	}

	/*
		// Test case 3: Param value with only a leading quote
		input3 := `"leading quote`
		expected3 := `"leading quote`
		result3 := trimQuotes(input3)
		if result3 != expected3 {
			t.Errorf("TrimQuotes(%s) = %s, expected %s", input3, result3, expected3)
		}

		// Test case 4: Param value with only a trailing quote
		input4 := `trailing quote"`
		expected4 := `trailing quote"`
		result4 := trimQuotes(input4)
		if result4 != expected4 {
			t.Errorf("TrimQuotes(%s) = %s, expected %s", input4, result4, expected4)
		}

		// Test case 5: Param value with multiple leading and trailing quotes
		input5 := `"""multi-quote"""`
		expected5 := `"multi-quote"`
		result5 := trimQuotes(input5)
		if result5 != expected5 {
			t.Errorf("TrimQuotes(%s) = %s, expected %s", input5, result5, expected5)
		}
	*/
}

func TestConvertStringMapToInterfaceMap(t *testing.T) {
	// Test case 1: Convert an empty string map
	input1 := map[string]string{}
	expected1 := map[string]interface{}{}
	result1 := ConvertStringMapToInterfaceMap(input1)
	if !mapsAreEqual(result1, expected1) {
		t.Errorf("ConvertStringMapToInterfaceMap(%v) = %v, expected %v", input1, result1, expected1)
	}

	// Test case 2: Convert a string map with values
	input2 := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	expected2 := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	result2 := ConvertStringMapToInterfaceMap(input2)
	if !mapsAreEqual(result2, expected2) {
		t.Errorf("ConvertStringMapToInterfaceMap(%v) = %v, expected %v", input2, result2, expected2)
	}
}

func mapsAreEqual(map1, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, value1 := range map1 {
		value2, exists := map2[key]
		if !exists || value1 != value2 {
			return false
		}
	}

	return true
}

func TestCreatePipelineRunResource(t *testing.T) {
	// PipelineRun
	var pipelineRun1 unstructured.Unstructured
	pipelineRun1.SetAPIVersion("tekton.dev/v1beta1")
	pipelineRun1.SetKind("PipelineRun")
	//pipelineRun1.SetName("does-not-exist")
	//pipelineRun1.SetNamespace("default")
	pipelineRun1.Object["spec"] = map[string]interface{}{
		"pipelineRef": map[string]interface{}{
			"name": "does-not-exist",
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
	// Test case 1: GitRepository source
	pipelineTriggerGit := &PipelineTrigger{
		Spec: PipelineTriggerSpec{
			Source: Source{
				APIVersion: "source.toolkit.fluxcd.io",
				Kind:       "GitRepository",
				Name:       "repo",
			},
			PipelineRun: pipelineRun1,
		},
		Status: PipelineTriggerStatus{
			GitRepository: GitRepository{
				BranchName: "git-details",
				CommitId:   "129381292513502359235da123da",
				Details:    "{\"id\":1163006807}",
			},
		},
	}

	expectedPipelineRunGit := pipelineTriggerGit.Spec.PipelineRun.DeepCopy()
	expectedPipelineRunGit.Object["metadata"] = map[string]interface{}{
		"generateName": "git-details-",
		"labels":       ConvertStringMapToInterfaceMap(pipelineTriggerGit.Status.GitRepository.GenerateGitRepositoryLabelsAsHash()),
	}
	paramListGit := pipelineTriggerGit.createParams(pipelineTriggerGit.Status.GitRepository.Details)
	unstructuredParamsGit := make([]interface{}, len(paramListGit))
	for i, param := range paramListGit {
		unstructuredParamsGit[i] = map[string]interface{}{
			"name":  param.Name,
			"value": param.Value,
		}
	}
	expectedPipelineRunGit.Object["spec"].(map[string]interface{})["params"] = unstructuredParamsGit

	resultPipelineRunGit := pipelineTriggerGit.CreatePipelineRunResource()
	if !reflect.DeepEqual(resultPipelineRunGit, expectedPipelineRunGit) {
		t.Errorf("CreatePipelineRunResource() for GitRepository source failed. Got %v, expected %v", resultPipelineRunGit, expectedPipelineRunGit)
	}

	// Test case 2: ImagePolicy source
	var pipelineRun2 unstructured.Unstructured
	pipelineRun2.SetAPIVersion("tekton.dev/v1beta1")
	pipelineRun2.SetKind("PipelineRun")
	//pipelineRun1.SetName("does-not-exist")
	//pipelineRun1.SetNamespace("default")
	pipelineRun2.Object["spec"] = map[string]interface{}{
		"pipelineRef": map[string]interface{}{
			"name": "does-not-exist",
		},
		"params": []interface{}{
			map[string]interface{}{
				"name":  "param1",
				"value": "$.id",
			},
			map[string]interface{}{
				"name":  "param2",
				"value": "value2",
			},
		},
	}
	pipelineTriggerImage := &PipelineTrigger{
		Spec: PipelineTriggerSpec{
			Source: Source{
				APIVersion: "image.toolkit.fluxcd.io/v1",
				Kind:       "ImagePolicy",
				Name:       "test-image-policy",
			},
			PipelineRun: pipelineRun2,
		},
		Status: PipelineTriggerStatus{
			ImagePolicy: ImagePolicy{
				RepositoryName: "gcr.io",
				ImageName:      "repo",
				ImageVersion:   "v0.0.1",
				Details:        "{\"id\":1163006807}",
			},
		},
	}

	expectedPipelineRunImage := pipelineTriggerImage.Spec.PipelineRun.DeepCopy()
	expectedPipelineRunImage.Object["metadata"] = map[string]interface{}{
		"generateName": "repo-",
		"labels":       ConvertStringMapToInterfaceMap(pipelineTriggerImage.Status.ImagePolicy.GenerateImagePolicyLabelsAsHash()),
	}
	paramListImage := pipelineTriggerImage.createParams(pipelineTriggerImage.Status.ImagePolicy.Details)
	unstructuredParamsImage := make([]interface{}, len(paramListImage))
	for i, param := range paramListImage {
		unstructuredParamsImage[i] = map[string]interface{}{
			"name":  param.Name,
			"value": param.Value,
		}
	}
	expectedPipelineRunImage.Object["spec"].(map[string]interface{})["params"] = unstructuredParamsImage

	resultPipelineRunImage := pipelineTriggerImage.CreatePipelineRunResource()
	if !reflect.DeepEqual(resultPipelineRunImage, expectedPipelineRunImage) {
		t.Errorf("CreatePipelineRunResource() for ImagePolicy source failed. Got %v, expected %v", resultPipelineRunImage, expectedPipelineRunImage)
	}
}
