package metrics

import (
	"context"
	"testing"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNewRecorder(t *testing.T) {
	recorder := NewRecorder()

	assert.NotNil(t, recorder)
	assert.NotNil(t, recorder.conditionGauge)
}

func TestRecorder_Collectors(t *testing.T) {
	recorder := NewRecorder()

	collectors := recorder.Collectors()

	assert.NotNil(t, collectors)
	assert.Len(t, collectors, 1)
	assert.IsType(t, &prometheus.GaugeVec{}, collectors[0])
}

func TestRecorder_RecordCondition(t *testing.T) {
	recorder := NewRecorder()

	ref := pipelinev1alpha1.PipelineTrigger{
		TypeMeta: metav1.TypeMeta{
			Kind: "PipelineTrigger",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-trigger",
			Namespace: "namespace",
		},

		Spec: pipelinev1alpha1.PipelineTriggerSpec{
			Source: pipelinev1alpha1.Source{
				Kind: "TestSourceKind",
			},
		},
	}
	sourceSubscriber := &MockSourceSubscriber{
		names: []string{"Name1", "Name2"},
		conditions: []metav1.Condition{
			{Type: "ConditionType1", Status: metav1.ConditionTrue},
			{Type: "ConditionType2", Status: metav1.ConditionFalse},
		},
	}

	recorder.RecordCondition(ref, sourceSubscriber)

	// Perform assertions for your metrics, you can use your expected values here.
}

type MockSourceSubscriber struct {
	names      []string
	conditions []metav1.Condition
}

// CalculateCurrentState implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) CalculateCurrentState(secondClusterEnabled bool, secondClusterAddr string, secondClusterBearerToken string, ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList unstructured.UnstructuredList, group string, version string) (bool, bool, error) {
	panic("unimplemented")
}

// CreatePipelineRunResource implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*unstructured.Unstructured {
	panic("unimplemented")
}

// Exists implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req reconcile.Request, group string, version string) error {
	panic("unimplemented")
}

// Get implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) Get(ctx context.Context, client dynamic.DynamicClient, name string, namespace string, group string, version string) (*unstructured.Unstructured, error) {
	panic("unimplemented")
}

// GetLatestEvent implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req reconcile.Request, group string, version string) (bool, error) {
	panic("unimplemented")
}

// HasIntersection implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) HasIntersection(map1 map[string]string, map2 map[string]string) bool {
	panic("unimplemented")
}

// IsFinished implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool {
	panic("unimplemented")
}

// IsValid implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) IsValid(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req reconcile.Request) error {
	panic("unimplemented")
}

// List implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) List(ctx context.Context, client dynamic.DynamicClient, namespace string, group string, version string) ([]unstructured.Unstructured, error) {
	panic("unimplemented")
}

// ManageError implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req reconcile.Request, r client.Client, message error) (reconcile.Result, error) {
	panic("unimplemented")
}

// SetCurrentPipelineRunName implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *unstructured.Unstructured, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	panic("unimplemented")
}

// SetCurrentPipelineRunStatus implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) SetCurrentPipelineRunStatus(pipelineRunList unstructured.UnstructuredList, obj *pipelinev1alpha1.PipelineTrigger) {
	panic("unimplemented")
}

// Subscribes implements v1alpha1.SourceSubscriber.
func (*MockSourceSubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	panic("unimplemented")
}

func (m *MockSourceSubscriber) GetLastConditions(ref *pipelinev1alpha1.PipelineTrigger) ([]string, []metav1.Condition) {
	return m.names, m.conditions
}
