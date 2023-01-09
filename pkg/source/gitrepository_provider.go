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

func (gitrepositorySubscriber GitrepositorySubscriber) CalculateCurrentState(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList tektondevv1.PipelineRunList) bool {
	foundSource := &sourcev1.GitRepository{}
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return false
	}
	var gitRepository pipelinev1alpha1.GitRepository
	gitRepository.GetGitRepository(*foundSource)
	gitRepository.GenerateDetails()
	gitRepositoryLabels := gitRepository.GenerateGitRepositoryLabelsAsHash()
	gitRepositoryLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRunSpec.PipelineRef.Name
	for i := range pipelineRunList.Items {
		if gitrepositorySubscriber.HasIntersection(gitRepositoryLabels, pipelineRunList.Items[i].GetLabels()) {
			return true
		}
	}
	return false
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
			pr := pipelineTrigger.CreatePipelineRunResource()
			ctrl.SetControllerReference(pipelineTrigger, pr, r)
			prs = append(prs, pr)
			condition := v1.Condition{
				Type:               apis.ReconcileUnknown,
				LastTransitionTime: v1.Now(),
				ObservedGeneration: pipelineTrigger.GetGeneration(),
				Reason:             apis.ReconcileUnknown,
				Status:             v1.ConditionUnknown,
				Message:            "Unknown",
			}
			pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
		} else {
			condition := v1.Condition{
				Type:               apis.ReconcileError,
				LastTransitionTime: v1.Now(),
				ObservedGeneration: pipelineTrigger.GetGeneration(),
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
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.PipelineRunSpec.Params); paramNr++ {
		if strings.Contains(pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr].Value.StringVal, "$") {
			_, err := json.Exists(pipelineTrigger.Status.GitRepository.Details, pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr].Value.StringVal)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) SetCurrentPipelineRunStatus(pipelineRunList tektondevv1.PipelineRunList, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	for i := range pipelineRunList.Items {
		item := pipelineRunList.Items[i]
		pipelineTrigger.Status.GitRepository.LatestPipelineRun = item.Name
		for _, c := range pipelineRunList.Items[i].Status.Conditions {
			if v1.ConditionStatus(c.Status) == v1.ConditionTrue {
				condition := v1.Condition{
					Type:               apis.ReconcileSuccess,
					LastTransitionTime: v1.Now(),
					ObservedGeneration: pipelineTrigger.GetGeneration(),
					Reason:             apis.ReconcileSuccessReason,
					Status:             v1.ConditionTrue,
					Message:            "Reconciliation is successful.",
				}
				pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
			} else if v1.ConditionStatus(c.Status) == v1.ConditionFalse {
				condition := v1.Condition{
					Type:               apis.ReconcileSuccess,
					LastTransitionTime: v1.Now(),
					ObservedGeneration: pipelineTrigger.GetGeneration(),
					Reason:             c.Reason,
					Status:             v1.ConditionFalse,
					Message:            c.Message,
				}
				pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(condition)
			} else {
				condition := v1.Condition{
					Type:               apis.ReconcileInProgress,
					LastTransitionTime: v1.Now(),
					ObservedGeneration: pipelineTrigger.GetGeneration(),
					Reason:             apis.ReconcileInProgress,
					Status:             v1.ConditionUnknown,
					Message:            "Progressing",
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

	patch := client.MergeFrom(obj.DeepCopy())

	condition := v1.Condition{
		Type:               apis.ReconcileError,
		LastTransitionTime: v1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileErrorReason,
		Status:             v1.ConditionFalse,
		Message:            message.Error(),
	}
	obj.Status.GitRepository.AddOrReplaceCondition(condition)

	err := r.Status().Patch(context, obj, patch)
	return reconcile.Result{Requeue: true}, err
}

func (gitrepositorySubscriber *GitrepositorySubscriber) HasIntersection(map1 map[string]string, map2 map[string]string) bool {
	if len(map1) > len(map2) {
		return false
	}
	for key, valueM1 := range map1 {
		valueM2, exists := map2[key]
		if !exists {
			return false
		}
		if exists && (valueM1 != valueM2) {
			return false
		}
	}
	return true
}

func (gitrepositorySubscriber GitrepositorySubscriber) GetLastConditions(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) ([]string, []v1.Condition) {
	var conditions []v1.Condition
	var names []string
	conditions = append(conditions, pipelineTrigger.Status.GitRepository.GetLastCondition())
	names = append(names, pipelineTrigger.Status.GitRepository.BranchName)
	return names, conditions
}
