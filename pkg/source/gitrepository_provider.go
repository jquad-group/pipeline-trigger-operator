package v1alpha1

import (
	"context"
	"strings"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"
	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type GitrepositorySubscriber struct {
}

func NewGitrepositorySubscriber() *GitrepositorySubscriber {
	return &GitrepositorySubscriber{}
}

func (gitrepositorySubscriber GitrepositorySubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	return nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error {
	foundSource := &sourcev1.GitRepository{}
	err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (gitrepositorySubscriber GitrepositorySubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (bool, error) {
	foundSource := &sourcev1.GitRepository{}
	gotNewEvent := false
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return gotNewEvent, err
	}
	var gitRepository pipelinev1alpha1.GitRepository
	gitRepository.GetGitRepository(*foundSource)
	gitRepository.GenerateDetails()
	if !pipelineTrigger.Status.GitRepository.Equals(gitRepository) {
		pipelineTrigger.Status.GitRepository = gitRepository
		gotNewEvent = true
	} else {
		gotNewEvent = false
	}
	return gotNewEvent, nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*tektondevv1.PipelineRun {
	var prs []*tektondevv1.PipelineRun
	if len(pipelineTrigger.Status.GitRepository.Conditions) == 0 {
		paramsCorrectness, err := evaluatePipelineParamsForGitRepository(pipelineTrigger)
		if paramsCorrectness {
			pr := pipelineTrigger.Spec.Pipeline.CreatePipelineRunResourceForGit(*pipelineTrigger)
			ctrl.SetControllerReference(pipelineTrigger, pr, r)
			prs = append(prs, pr)
			condition := v1.Condition{
				Type:               apis.ReconcileUnknown,
				LastTransitionTime: v1.Now(),
				Reason:             apis.ReconcileUnknown,
				Status:             v1.ConditionTrue,
				Message:            "Unknown",
			}
			pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
		} else {
			condition := v1.Condition{
				Type:               apis.ReconcileError,
				LastTransitionTime: v1.Now(),
				Reason:             apis.ReconcileErrorReason,
				Status:             v1.ConditionFalse,
				Message:            err.Error(),
			}
			pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
		}
	}
	return prs
}

func evaluatePipelineParamsForGitRepository(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) (bool, error) {
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.Pipeline.InputParams); paramNr++ {
		if strings.Contains(pipelineTrigger.Spec.Pipeline.InputParams[paramNr].Value, "$") {
			_, err := json.Exists(pipelineTrigger.Status.GitRepository.Details, pipelineTrigger.Spec.Pipeline.InputParams[paramNr].Value)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) UpdateStatus(ctx context.Context, req ctrl.Request, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, pipelineRunList *tektondevv1.PipelineRunList) {
	for i := range pipelineRunList.Items {
		item := pipelineRunList.Items[i]
		pipelineTrigger.Status.GitRepository.LatestPipelineRun = item.Name
		for _, c := range pipelineRunList.Items[i].Status.Conditions {
			if (tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonCompleted || tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonSuccessful) && v1.ConditionStatus(c.Status) == v1.ConditionTrue {
				condition := v1.Condition{
					Type:               apis.ReconcileSuccess,
					LastTransitionTime: v1.Now(),
					Reason:             apis.ReconcileSuccessReason,
					Status:             v1.ConditionTrue,
					Message:            "Reconciliation is successful.",
				}
				pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
			}
			if (tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonFailed && v1.ConditionStatus(c.Status) == v1.ConditionFalse) ||
				(tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonCancelled) ||
				(tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonTimedOut) {
				condition := v1.Condition{
					Type:               apis.ReconcileError,
					LastTransitionTime: v1.Now(),
					Reason:             apis.ReconcileErrorReason,
					Status:             v1.ConditionTrue,
					Message:            "Reconciliation is successful.",
				}
				pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
			}
		}
	}
}

func (gitrepositorySubscriber GitrepositorySubscriber) IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool {
	result := true
	_, foundError := pipelineTrigger.Status.GitRepository.GetCondition(apis.ReconcileError)
	_, foundSuccess := pipelineTrigger.Status.GitRepository.GetCondition(apis.ReconcileSuccess)
	if (!foundError) && (!foundSuccess) {
		result = false
	} else {
		result = result && true
	}
	return result
}

func (gitrepositorySubscriber *GitrepositorySubscriber) ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request, r client.Client, message error) (reconcile.Result, error) {

	if err := r.Get(context, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
		return reconcile.Result{}, err
	}

	condition := v1.Condition{
		Type:               apis.ReconcileError,
		LastTransitionTime: v1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileErrorReason,
		Status:             v1.ConditionFalse,
		Message:            message.Error(),
	}
	obj.Status.GitRepository.AddOrReplaceCondition(condition)
	err := r.Status().Update(context, obj)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
