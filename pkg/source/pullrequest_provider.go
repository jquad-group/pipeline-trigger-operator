package v1alpha1

import (
	//pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"
	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PullrequestSubscriber struct {
}

func NewPullrequestSubscriber() *PullrequestSubscriber {
	return &PullrequestSubscriber{}
}

func (pullrequestSubscriber PullrequestSubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	return nil
}

func (pullrequestSubscriber PullrequestSubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error {
	foundSource := &pullrequestv1alpha1.PullRequest{}
	err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (pullrequestSubscriber PullrequestSubscriber) CalculateCurrentState(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList tektondevv1.PipelineRunList) bool {
	var res []bool
	// get the current branches from the pull request resource
	foundSource := &pullrequestv1alpha1.PullRequest{}
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return false
	}
	var branches pipelinev1alpha1.Branches
	branches.GetPrBranches(foundSource.Status.SourceBranches)

	// check if there is a corresponding pipelinerun for every checkout branch
	if len(branches.Branches) > 0 {
		for key := range branches.Branches {
			tempBranch := branches.Branches[key]
			tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
			tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRunSpec.PipelineRef.Name
			for i := range pipelineRunList.Items {
				if pullrequestSubscriber.HasIntersection(tempBranchLabels, pipelineRunList.Items[i].GetLabels()) {
					res = append(res, true)
				}
			}
		}
	}

	if len(res) < len(branches.Branches) {
		return false
	} else {
		return true
	}

}

func (pullrequestSubscriber PullrequestSubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (bool, error) {
	foundSource := &pullrequestv1alpha1.PullRequest{}
	gotNewEvent := false
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return gotNewEvent, err
	}
	var branches pipelinev1alpha1.Branches
	branches.GetPrBranches(foundSource.Status.SourceBranches)
	if len(pipelineTrigger.Status.Branches.Branches) > 0 {
		// add all new branches to the status of the pipelinetrigger
		for key, value := range branches.Branches {
			_, found := pipelineTrigger.Status.Branches.Branches[key]
			if found {
				newCommit := branches.Branches[key].Commit
				oldCommit := pipelineTrigger.Status.Branches.Branches[key].Commit
				// Pull Request was updated after it was created
				if newCommit != oldCommit {
					pipelineTrigger.Status.Branches.Branches[key] = value
					gotNewEvent = true
				}
			}
			if !found {
				// add the branch
				pipelineTrigger.Status.Branches.Branches[key] = value
				gotNewEvent = true
			}
		}
		// remove all branches from the pipelinetrigger that were not part of the request
		for key := range pipelineTrigger.Status.Branches.Branches {
			_, found := branches.Branches[key]
			if !found {
				// remove the branch
				delete(pipelineTrigger.Status.Branches.Branches, key)
			}
		}
	} else {
		pipelineTrigger.Status.Branches = branches
		gotNewEvent = true
	}
	return gotNewEvent, nil
}

func (pullrequestSubscriber PullrequestSubscriber) SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *tektondevv1.PipelineRun, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	if len(pipelineTrigger.Status.Branches.Branches) > 0 {
		for key := range pipelineTrigger.Status.Branches.Branches {
			tempBranch := pipelineTrigger.Status.Branches.Branches[key]
			tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
			// This label is automatically added by the Tekton Controller
			//tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRunSpec.PipelineRef.Name
			if pullrequestSubscriber.HasIntersection(tempBranchLabels, pipelineRun.GetLabels()) {
				tempBranch.LatestPipelineRun = pipelineRunName
				pipelineTrigger.Status.Branches.Branches[key] = tempBranch
			}
		}
	}
}

func (pullrequestSubscriber PullrequestSubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*tektondevv1.PipelineRun {
	var prs []*tektondevv1.PipelineRun
	for key := range pipelineTrigger.Status.Branches.Branches {
		tempBranch := pipelineTrigger.Status.Branches.Branches[key]
		// there is no pipeline for the current branch
		if len(tempBranch.Conditions) == 0 {
			paramsCorrectness, err := evaluatePipelineParams(pipelineTrigger, tempBranch)
			if paramsCorrectness {
				pr := pipelineTrigger.CreatePipelineRunResourceForBranch(tempBranch, tempBranch.GenerateBranchLabelsAsHash())
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
				tempBranch.Details = ""
				tempBranch.AddOrReplaceCondition(condition)
				pipelineTrigger.Status.Branches.Branches[key] = tempBranch
			} else {
				condition := v1.Condition{
					Type:               apis.ReconcileError,
					LastTransitionTime: v1.Now(),
					ObservedGeneration: pipelineTrigger.GetGeneration(),
					Reason:             apis.ReconcileErrorReason,
					Status:             v1.ConditionFalse,
					Message:            err.Error(),
				}
				tempBranch.Details = ""
				tempBranch.AddOrReplaceCondition(condition)
				pipelineTrigger.Status.Branches.Branches[key] = tempBranch
			}
		}
	}

	return prs
}

func evaluatePipelineParams(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, currentBranch pipelinev1alpha1.Branch) (bool, error) {
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.PipelineRunSpec.Params); paramNr++ {
		if strings.Contains(pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr].Value.StringVal, "$") {
			_, err := json.Exists(currentBranch.Details, pipelineTrigger.Spec.PipelineRunSpec.Params[paramNr].Value.StringVal)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (pullrequestSubscriber PullrequestSubscriber) IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool {
	result := true
	for key := range pipelineTrigger.Status.Branches.Branches {
		tempBranch := pipelineTrigger.Status.Branches.Branches[key]
		_, foundError := tempBranch.GetCondition(apis.ReconcileError)
		_, foundSuccess := tempBranch.GetCondition(apis.ReconcileSuccess)
		if (!foundError) && (!foundSuccess) {
			result = false
		} else {
			result = result && true
		}
	}
	return result
}

func (pullrequestSubscriber *PullrequestSubscriber) ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request, r client.Client, message error) (reconcile.Result, error) {
	patch := client.MergeFrom(obj.DeepCopy())

	condition := v1.Condition{
		Type:               apis.ReconcileError,
		LastTransitionTime: v1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileErrorReason,
		Status:             v1.ConditionFalse,
		Message:            message.Error(),
	}

	if len(obj.Status.Branches.Branches) <= 0 {
		branches := make(map[string]pipelinev1alpha1.Branch)
		tempBranch := pipelinev1alpha1.Branch{}
		tempBranch.AddOrReplaceCondition(condition)
		branches["null"] = tempBranch
		obj.Status.Branches.Branches = branches
	} else {
		for key := range obj.Status.Branches.Branches {
			tempBranch := obj.Status.Branches.Branches[key]
			tempBranch.AddOrReplaceCondition(condition)
			obj.Status.Branches.Branches[key] = tempBranch
		}
	}

	err := r.Status().Patch(context, obj, patch)
	return reconcile.Result{Requeue: true}, err
}

func (PullrequestSubscriber *PullrequestSubscriber) isStateIdentical(currentPipelineRun tektondevv1.PipelineRun, currentBranch pipelinev1alpha1.Branch) bool {
	conditions := currentPipelineRun.Status.Conditions
	for i := range conditions {
		if (v1.ConditionStatus(conditions[i].Status) == v1.ConditionTrue) && (currentBranch.GetLastCondition().Status == v1.ConditionTrue) {
			return true
		}
		if (v1.ConditionStatus(conditions[i].Status) == v1.ConditionFalse) && (currentBranch.GetLastCondition().Status == v1.ConditionFalse) {
			return true
		}
	}
	return false
}

func (pullrequestSubscriber *PullrequestSubscriber) SetCurrentPipelineRunStatus(pipelineRunList tektondevv1.PipelineRunList, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	if len(pipelineTrigger.Status.Branches.Branches) > 0 {
		for key := range pipelineTrigger.Status.Branches.Branches {
			tempBranch := pipelineTrigger.Status.Branches.Branches[key]
			//tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
			// This label is automatically added by the Tekton Controller
			//tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRunSpec.PipelineRef.Name
			for i := range pipelineRunList.Items {
				//if pullrequestSubscriber.HasIntersection(tempBranchLabels, pipelineRunList.Items[i].GetLabels()) {
				if pipelineRunList.Items[i].Name == tempBranch.LatestPipelineRun {
					if !pullrequestSubscriber.isStateIdentical(pipelineRunList.Items[i], tempBranch) {
						currBranch := tempBranch
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
								currBranch.AddOrReplaceCondition(condition)
								pipelineTrigger.Status.Branches.Branches[key] = currBranch
							} else if v1.ConditionStatus(c.Status) == v1.ConditionFalse {
								condition := v1.Condition{
									Type:               apis.ReconcileSuccess,
									LastTransitionTime: v1.Now(),
									ObservedGeneration: pipelineTrigger.GetGeneration(),
									Reason:             c.Reason,
									Status:             v1.ConditionFalse,
									Message:            c.Message,
								}
								currBranch.AddOrReplaceCondition(condition)
								pipelineTrigger.Status.Branches.Branches[key] = currBranch
							} else {
								condition := v1.Condition{
									Type:               apis.ReconcileInProgress,
									LastTransitionTime: v1.Now(),
									ObservedGeneration: pipelineTrigger.GetGeneration(),
									Reason:             apis.ReconcileInProgress,
									Status:             v1.ConditionUnknown,
									Message:            "Progressing",
								}
								currBranch.AddOrReplaceCondition(condition)
								pipelineTrigger.Status.Branches.Branches[key] = currBranch
							}
						}
					}
				}
				//}
			}
		}
	}

}

func (pullrequestSubscriber *PullrequestSubscriber) HasIntersection(map1 map[string]string, map2 map[string]string) bool {
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

func (pullrequestSubscriber PullrequestSubscriber) GetLastConditions(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) ([]string, []v1.Condition) {
	var conditions []v1.Condition
	var names []string
	for key := range pipelineTrigger.Status.Branches.Branches {
		tempBranch := pipelineTrigger.Status.Branches.Branches[key]
		conditions = append(conditions, tempBranch.GetLastCondition())
		names = append(names, tempBranch.Name)
	}
	return names, conditions

}
