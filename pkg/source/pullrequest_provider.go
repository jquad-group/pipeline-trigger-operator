package v1alpha1

import (
	//pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	"context"
	"fmt"
	"strings"

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

func (pullrequestSubscriber PullrequestSubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (bool, error) {
	foundSource := &pullrequestv1alpha1.PullRequest{}
	/*
		condition := v1.Condition{
			Type:               apis.ReconcileUnknown,
			LastTransitionTime: v1.Now(),
			Reason:             apis.ReconcileUnknown,
			Status:             v1.ConditionTrue,
			Message:            "Unknown",
		}
	*/
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
			if !found {
				// add start condition to every new found branch
				//value.AddOrReplaceCondition(condition)
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
		// add start condition to every new found branch
		/*
			for key, value := range branches.Branches {
				value.AddOrReplaceCondition(condition)
				branches.Branches[key] = value
			}
		*/
		pipelineTrigger.Status.Branches = branches
		gotNewEvent = true
	}
	return gotNewEvent, nil
}

func (pullrequestSubscriber PullrequestSubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*tektondevv1.PipelineRun {
	var prs []*tektondevv1.PipelineRun
	for key := range pipelineTrigger.Status.Branches.Branches {
		tempBranch := pipelineTrigger.Status.Branches.Branches[key]
		// there is no pipeline for the current branch
		if len(tempBranch.Conditions) == 0 {
			paramsCorrectness, err := evaluatePipelineParams(pipelineTrigger, tempBranch)
			if paramsCorrectness {
				pr := pipelineTrigger.Spec.Pipeline.CreatePipelineRunResourceForBranch(*pipelineTrigger, tempBranch, tempBranch.GenerateBranchLabelsAsHash())
				ctrl.SetControllerReference(pipelineTrigger, pr, r)
				prs = append(prs, pr)
				condition := v1.Condition{
					Type:               apis.ReconcileUnknown,
					LastTransitionTime: v1.Now(),
					Reason:             apis.ReconcileUnknown,
					Status:             v1.ConditionTrue,
					Message:            "Unknown",
				}
				tempBranch.Details = ""
				tempBranch.AddOrReplaceCondition(condition)
				pipelineTrigger.Status.Branches.Branches[key] = tempBranch
			} else {
				condition := v1.Condition{
					Type:               apis.ReconcileError,
					LastTransitionTime: v1.Now(),
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
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.Pipeline.InputParams); paramNr++ {
		if strings.Contains(pipelineTrigger.Spec.Pipeline.InputParams[paramNr].Value, "$") {
			_, err := json.Exists(currentBranch.Details, pipelineTrigger.Spec.Pipeline.InputParams[paramNr].Value)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (pullrequestSubscriber PullrequestSubscriber) UpdateStatus(ctx context.Context, req ctrl.Request, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, pipelineRunList *tektondevv1.PipelineRunList) {
	// the status is empty during the first reconciliation
	if len(pipelineTrigger.Status.Branches.Branches) > 0 {
		for key := range pipelineTrigger.Status.Branches.Branches {
			tempBranch := pipelineTrigger.Status.Branches.Branches[key]
			for i := range pipelineRunList.Items {
				item := pipelineRunList.Items[i]
				tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
				tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.Pipeline.Name
				itemLabels := item.GetLabels()
				if fmt.Sprint(tempBranchLabels) == fmt.Sprint(itemLabels) {
					tempBranch.LatestPipelineRun = item.Name
					for _, c := range pipelineRunList.Items[i].Status.Conditions {
						if (tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonCompleted || tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonSuccessful) && v1.ConditionStatus(c.Status) == v1.ConditionTrue {
							condition := v1.Condition{
								Type:               apis.ReconcileSuccess,
								LastTransitionTime: v1.Now(),
								Reason:             apis.ReconcileSuccessReason,
								Status:             v1.ConditionTrue,
								Message:            "Reconciliation is successful.",
							}
							tempBranch.AddOrReplaceCondition(condition)
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
							tempBranch.AddOrReplaceCondition(condition)
						}
					}
					pipelineTrigger.Status.Branches.Branches[key] = tempBranch
				}
			}
		}
	}
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
