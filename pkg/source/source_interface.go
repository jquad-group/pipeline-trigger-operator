package v1alpha1

import (
	"context"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SourceSubscriber interface {
	Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error
	List(ctx context.Context, client dynamic.Interface, namespace string, group string, version string) ([]unstructured.Unstructured, error)
	Get(ctx context.Context, client dynamic.Interface, name string, namespace string, group string, version string) (*unstructured.Unstructured, error)
	Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) error
	GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) (bool, error)
	CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*unstructured.Unstructured
	IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool
	ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request, r client.Client, message error) (reconcile.Result, error)
	SetCurrentPipelineRunStatus(pipelineRunList unstructured.UnstructuredList, obj *pipelinev1alpha1.PipelineTrigger)
	CalculateCurrentState(secondClusterEnabled bool, secondClusterAddr string, secondClusterBearerToken string, ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList unstructured.UnstructuredList, group string, version string) (bool, bool, error)
	HasIntersection(map1 map[string]string, map2 map[string]string) bool
	GetLastConditions(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) ([]string, []metav1.Condition)
	SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *unstructured.Unstructured, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger)
}
