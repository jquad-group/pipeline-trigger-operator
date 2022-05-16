package v1alpha1

import (
	"context"
	"strings"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
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

type ImagepolicySubscriber struct {
}

func NewImagepolicySubscriber() *ImagepolicySubscriber {
	return &ImagepolicySubscriber{}
}

func (imagepolicySubscriber ImagepolicySubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	return nil
}

func (imagepolicySubscriber ImagepolicySubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error {
	foundSource := &imagereflectorv1.ImagePolicy{}
	err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (imagepolicySubscriber ImagepolicySubscriber) CalculateCurrentState(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList tektondevv1.PipelineRunList) bool {
	// get the current image policy from the Flux ImagePolicy resource
	foundSource := &imagereflectorv1.ImagePolicy{}
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return false
	}
	var imagePolicy pipelinev1alpha1.ImagePolicy
	imagePolicy.GetImagePolicy(*foundSource)
	imagePolicy.GenerateDetails()
	imagePolicyLabels := imagePolicy.GenerateImagePolicyLabelsAsHash()
	imagePolicyLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.Pipeline.Name
	for i := range pipelineRunList.Items {
		if imagepolicySubscriber.HasIntersection(imagePolicyLabels, pipelineRunList.Items[i].GetLabels()) {
			return true
		}
	}

	return false
}

func (imagepolicySubscriber ImagepolicySubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) (bool, error) {
	foundSource := &imagereflectorv1.ImagePolicy{}
	gotNewEvent := false
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return gotNewEvent, err
	}
	var imagePolicy pipelinev1alpha1.ImagePolicy
	imagePolicy.GetImagePolicy(*foundSource)
	imagePolicy.GenerateDetails()
	if !pipelineTrigger.Status.ImagePolicy.Equals(imagePolicy) {
		pipelineTrigger.Status.ImagePolicy = imagePolicy
		gotNewEvent = true
	} else {
		gotNewEvent = false
	}
	return gotNewEvent, nil
}

func (imagepolicySubscriber ImagepolicySubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*tektondevv1.PipelineRun {
	var prs []*tektondevv1.PipelineRun
	if len(pipelineTrigger.Status.ImagePolicy.Conditions) == 0 {
		paramsCorrectness, err := evaluatePipelineParamsForImage(pipelineTrigger)
		if paramsCorrectness {
			pr := pipelineTrigger.Spec.Pipeline.CreatePipelineRunResourceForImage(*pipelineTrigger)
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
			pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(condition)
		} else {
			condition := v1.Condition{
				Type:               apis.ReconcileError,
				LastTransitionTime: v1.Now(),
				ObservedGeneration: pipelineTrigger.GetGeneration(),
				Reason:             apis.ReconcileErrorReason,
				Status:             v1.ConditionFalse,
				Message:            err.Error(),
			}
			pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(condition)
		}
	}
	return prs
}

func evaluatePipelineParamsForImage(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) (bool, error) {
	for paramNr := 0; paramNr < len(pipelineTrigger.Spec.Pipeline.InputParams); paramNr++ {
		if strings.Contains(pipelineTrigger.Spec.Pipeline.InputParams[paramNr].Value, "$") {
			_, err := json.Exists(pipelineTrigger.Status.ImagePolicy.Details, pipelineTrigger.Spec.Pipeline.InputParams[paramNr].Value)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (imagepolicySubscriber ImagepolicySubscriber) SetCurrentPipelineRunStatus(pipelineRunList tektondevv1.PipelineRunList, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	for i := range pipelineRunList.Items {
		item := pipelineRunList.Items[i]
		pipelineTrigger.Status.ImagePolicy.LatestPipelineRun = item.Name
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
				pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(condition)
			} else if v1.ConditionStatus(c.Status) == v1.ConditionFalse {
				condition := v1.Condition{
					Type:               apis.ReconcileSuccess,
					LastTransitionTime: v1.Now(),
					ObservedGeneration: pipelineTrigger.GetGeneration(),
					Reason:             c.Reason,
					Status:             v1.ConditionFalse,
					Message:            c.Message,
				}
				pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(condition)
			} else {
				condition := v1.Condition{
					Type:               apis.ReconcileInProgress,
					LastTransitionTime: v1.Now(),
					ObservedGeneration: pipelineTrigger.GetGeneration(),
					Reason:             apis.ReconcileInProgress,
					Status:             v1.ConditionUnknown,
					Message:            "Progressing",
				}
				pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(condition)
			}
		}
	}
}

func (imagepolicySubscriber ImagepolicySubscriber) IsFinished(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) bool {
	result := true
	_, foundError := pipelineTrigger.Status.ImagePolicy.GetCondition(apis.ReconcileError)
	_, foundSuccess := pipelineTrigger.Status.ImagePolicy.GetCondition(apis.ReconcileSuccess)
	if (!foundError) && (!foundSuccess) {
		result = false
	} else {
		result = result && true
	}
	return result
}

func (imagepolicySubscriber *ImagepolicySubscriber) ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request, r client.Client, message error) (reconcile.Result, error) {
	patch := client.MergeFrom(obj.DeepCopy())

	condition := v1.Condition{
		Type:               apis.ReconcileError,
		LastTransitionTime: v1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileErrorReason,
		Status:             v1.ConditionFalse,
		Message:            message.Error(),
	}

	obj.Status.ImagePolicy.AddOrReplaceCondition(condition)

	err := r.Status().Patch(context, obj, patch)
	return reconcile.Result{}, err
}

func (imagepolicySubscriber *ImagepolicySubscriber) HasIntersection(map1 map[string]string, map2 map[string]string) bool {
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
