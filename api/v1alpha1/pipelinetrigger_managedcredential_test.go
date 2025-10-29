package v1alpha1

import (
	"context"
	"testing"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/credentials"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPipelineTrigger_GetManagedCredential(t *testing.T) {

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = AddToScheme(scheme)
	_ = (&credentials.ManagedCredential{}).AddToScheme(scheme)

	ctx := context.Background()

	tests := []struct {
		name            string
		pipelineTrigger *PipelineTrigger
		managedCred     *credentials.ManagedCredential
		expectError     bool
		expectNil       bool
	}{
		{
			name: "No ManagedCredential reference",
			pipelineTrigger: &PipelineTrigger{
				Spec: PipelineTriggerSpec{
					CredentialsRef: nil,
				},
			},
			expectNil: true,
		},
		{
			name: "Valid ManagedCredential reference",
			pipelineTrigger: &PipelineTrigger{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pipeline",
					Namespace: "default",
				},
				Spec: PipelineTriggerSpec{
					CredentialsRef: &CredentialsRef{
						Name:      "test-cred",
						Namespace: "default",
					},
				},
			},
			managedCred: &credentials.ManagedCredential{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cred",
					Namespace: "default",
				},
				Spec: credentials.ManagedCredentialSpec{
					CredentialType: "git",
					Provider:       "github.com",
					TokenSecretRef: credentials.SecretReference{
						Name:      "secret",
						Namespace: "default",
					},
				},
				Status: credentials.ManagedCredentialStatus{
					SecretRef: &corev1.ObjectReference{
						Name:      "generated-secret",
						Namespace: "default",
					},
					ServiceAccountRef: &corev1.ObjectReference{
						Name:      "generated-sa",
						Namespace: "default",
					},
				},
			},
			expectNil: false,
		},
		{
			name: "ManagedCredential not found",
			pipelineTrigger: &PipelineTrigger{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pipeline",
					Namespace: "default",
				},
				Spec: PipelineTriggerSpec{
					CredentialsRef: &CredentialsRef{
						Name:      "nonexistent-cred",
						Namespace: "default",
					},
				},
			},
			expectError: true,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up fake client for each test
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Setup test data
			if tt.managedCred != nil {
				err := fakeClient.Create(ctx, tt.managedCred)
				if err != nil {
					t.Fatalf("Failed to create ManagedCredential: %v", err)
				}
			}

			// Test the method
			result, err := tt.pipelineTrigger.GetManagedCredential(ctx, fakeClient)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.expectNil && result != nil {
				t.Error("Expected nil result but got value")
			}
			if !tt.expectNil && result == nil {
				t.Error("Expected non-nil result but got nil")
			}
		})
	}
}

func TestPipelineTrigger_ApplyManagedCredentialToPipelineRun(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = AddToScheme(scheme)
	_ = (&credentials.ManagedCredential{}).AddToScheme(scheme)

	tests := []struct {
		name                   string
		pipelineTrigger        *PipelineTrigger
		managedCred            *credentials.ManagedCredential
		initialPipelineRun     map[string]interface{}
		expectedServiceAccount string
		expectedWorkspaceCount int
	}{
		{
			name:            "Apply nil ManagedCredential",
			pipelineTrigger: &PipelineTrigger{},
			managedCred:     nil,
			initialPipelineRun: map[string]interface{}{
				"apiVersion": "tekton.dev/v1",
				"kind":       "PipelineRun",
				"spec":       map[string]interface{}{},
			},
			expectedServiceAccount: "",
			expectedWorkspaceCount: 0,
		},
		{
			name:            "Apply ManagedCredential with service account and secret",
			pipelineTrigger: &PipelineTrigger{},
			managedCred: &credentials.ManagedCredential{
				Status: credentials.ManagedCredentialStatus{
					SecretRef: &corev1.ObjectReference{
						Name: "test-secret",
					},
					ServiceAccountRef: &corev1.ObjectReference{
						Name: "test-sa",
					},
				},
			},
			initialPipelineRun: map[string]interface{}{
				"apiVersion": "tekton.dev/v1",
				"kind":       "PipelineRun",
				"spec":       map[string]interface{}{},
			},
			expectedServiceAccount: "test-sa",
			expectedWorkspaceCount: 1,
		},
		{
			name:            "Apply ManagedCredential to existing PipelineRun with workspaces",
			pipelineTrigger: &PipelineTrigger{},
			managedCred: &credentials.ManagedCredential{
				Status: credentials.ManagedCredentialStatus{
					SecretRef: &corev1.ObjectReference{
						Name: "test-secret",
					},
					ServiceAccountRef: &corev1.ObjectReference{
						Name: "test-sa",
					},
				},
			},
			initialPipelineRun: map[string]interface{}{
				"apiVersion": "tekton.dev/v1",
				"kind":       "PipelineRun",
				"spec": map[string]interface{}{
					"workspaces": []interface{}{
						map[string]interface{}{
							"name":     "existing-workspace",
							"emptyDir": map[string]interface{}{},
						},
					},
				},
			},
			expectedServiceAccount: "test-sa",
			expectedWorkspaceCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create unstructured PipelineRun
			pipelineRun := &unstructured.Unstructured{}
			pipelineRun.SetUnstructuredContent(tt.initialPipelineRun)

			// Apply ManagedCredential
			tt.pipelineTrigger.ApplyManagedCredentialToPipelineRun(pipelineRun, tt.managedCred)

			// Check results
			spec, found := pipelineRun.Object["spec"].(map[string]interface{})
			if !found {
				t.Error("Expected spec to be present")
				return
			}

			// Check service account
			if tt.expectedServiceAccount != "" {
				if sa, ok := spec["serviceAccountName"].(string); !ok || sa != tt.expectedServiceAccount {
					t.Errorf("Expected serviceAccountName to be %s, got %v", tt.expectedServiceAccount, spec["serviceAccountName"])
				}
			} else if _, ok := spec["serviceAccountName"]; ok {
				t.Error("Expected no serviceAccountName, but found one")
			}

			// Check workspaces
			if workspaces, ok := spec["workspaces"].([]interface{}); ok {
				if len(workspaces) != tt.expectedWorkspaceCount {
					t.Errorf("Expected %d workspaces, got %d", tt.expectedWorkspaceCount, len(workspaces))
				}
			} else if tt.expectedWorkspaceCount > 0 {
				t.Error("Expected workspaces to be present, but found none")
			}
		})
	}
}
