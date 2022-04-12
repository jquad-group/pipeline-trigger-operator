package v1alpha1

import (
	"context"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SourceSubscriber interface {
	Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error
	Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error
	GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (bool, error)
	CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) []*tektondevv1.PipelineRun
	GetPipelineRunsByLabel(ctx context.Context, req ctrl.Request, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) (*tektondevv1.PipelineRunList, error)
	IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool
}
