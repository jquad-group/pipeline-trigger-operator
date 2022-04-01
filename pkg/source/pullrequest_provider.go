package v1alpha1

import (
	//pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	"context"

	"errors"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"

	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	var err error
	if pipelineTrigger.Spec.Source.Kind == "PullRequest" {
		foundSource := &pullrequestv1alpha1.PullRequest{}
		err = client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		return err
	}
}

func (pullrequestSubscriber PullrequestSubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (pipelinev1alpha1.Event, error) {
	var emptyEvent pipelinev1alpha1.Event
	emptyEventError := errors.New("Undefined error")
	if pipelineTrigger.Spec.Source.Kind == "PullRequest" {
		foundSource := &pullrequestv1alpha1.PullRequest{}
		if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
			return emptyEvent, err
		}
		var branches pipelinev1alpha1.Branches
		branches.GetPrBranches(foundSource.Status.SourceBranches)

		newEvent := pipelinev1alpha1.Event{
			Branches: branches,
		}
		return newEvent, nil
		//pipelineTrigger.Status.LatestEvent.AddOrReplace(newEvent)
	}

	return emptyEvent, emptyEventError
}

func (pullrequestSubscriber PullrequestSubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, newEvent pipelinev1alpha1.Event) []*tektondevv1.PipelineRun {
	var prs []*tektondevv1.PipelineRun
	if newEvent.Branches.GetSize() > 0 {
		for key := range newEvent.Branches.Branches {
			tempBranch := newEvent.Branches.Branches[key]
			pr := pipelineTrigger.Spec.Pipeline.CreatePipelineRunResource(*pipelineTrigger, tempBranch.GenerateBranchLabelsAsHash())
			prs = append(prs, pr)
			condition := metav1.Condition{
				Type:               apis.ReconcileUnknown,
				LastTransitionTime: metav1.Now(),
				Reason:             apis.ReconcileUnknown,
				Status:             metav1.ConditionTrue,
				Message:            "Unknown",
			}
			tempBranch.AddOrReplaceCondition(condition)
			pipelineTrigger.Status.LatestEvent.Branches.Branches[key] = tempBranch
		}
	}
	return prs
}

func (pullrequestSubscriber PullrequestSubscriber) GetPipelineRunsByLabel(ctx context.Context, req ctrl.Request, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) (*tektondevv1.PipelineRunList, error) {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	// the status is empty during the first reconciliation
	if len(pipelineTrigger.Status.LatestEvent.Branches.Branches) > 0 {
		var pipelineRunsByLabel []tektondevv1.PipelineRun
		for key := range pipelineTrigger.Status.LatestEvent.Branches.Branches {
			tempBranch := pipelineTrigger.Status.LatestEvent.Branches.Branches[key]
			opts := v1.ListOptions{LabelSelector: tempBranch.GenerateBranchLabelsAsString()}
			pipelineRunList, err := tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).List(ctx, opts)
			if err != nil {
				log.Info("Cannot get pipelineruns by label.")
			}

			for i := range pipelineRunList.Items {
				item := pipelineRunList.Items[i]
				tempBranch.LatestPipelineRun = item.Name
				pipelineRunsByLabel = append(pipelineRunsByLabel, item)
				for _, c := range pipelineRunList.Items[i].Status.Conditions {
					if (tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonCompleted || tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonSuccessful) && metav1.ConditionStatus(c.Status) == metav1.ConditionTrue {
						condition := metav1.Condition{
							Type:               apis.ReconcileSuccess,
							LastTransitionTime: metav1.Now(),
							Reason:             apis.ReconcileSuccessReason,
							Status:             metav1.ConditionTrue,
							Message:            "Reconciliation is successful.",
						}
						tempBranch.AddOrReplaceCondition(condition)
						pipelineTrigger.Status.LatestEvent.Branches.Branches[key] = tempBranch
					}
					if tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonFailed && metav1.ConditionStatus(c.Status) == metav1.ConditionFalse {
						condition := metav1.Condition{
							Type:               apis.ReconcileError,
							LastTransitionTime: metav1.Now(),
							Reason:             apis.ReconcileErrorReason,
							Status:             metav1.ConditionTrue,
							Message:            "Reconciliation is successful.",
						}
						tempBranch.AddOrReplaceCondition(condition)
						pipelineTrigger.Status.LatestEvent.Branches.Branches[key] = tempBranch
					}
				}
			}
		}

		var pipelineRunsList tektondevv1.PipelineRunList
		pipelineRunsList.Items = pipelineRunsByLabel
		return &pipelineRunsList, err
	} else {
		var pipelineRunsList tektondevv1.PipelineRunList
		return &pipelineRunsList, err
	}
}

func (pullrequestSubscriber PullrequestSubscriber) IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool {
	result := true
	for key := range pipelineTrigger.Status.LatestEvent.Branches.Branches {
		tempBranch := pipelineTrigger.Status.LatestEvent.Branches.Branches[key]
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
