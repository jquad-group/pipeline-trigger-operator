package v1alpha1

import (
	"context"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type SourceSubscriber interface {
	Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error
	Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error
	GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (bool, error)
	CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*tektondevv1.PipelineRun
	IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool
	ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request, r client.Client, message error) (reconcile.Result, error)
	SetCurrentPipelineRunStatus(pipelineRunList tektondevv1.PipelineRunList, obj *pipelinev1alpha1.PipelineTrigger)
	CalculateCurrentState(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList tektondevv1.PipelineRunList) bool
	HasIntersection(map1 map[string]string, map2 map[string]string) bool
	GetLastConditions(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) ([]string, []metav1.Condition)
	SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *tektondevv1.PipelineRun, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger)
}
