package v1alpha1

import (
	"context"
	"strings"

	encodingJson "encoding/json"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"
	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ImagepolicySubscriber struct {
	DynamicClient   dynamic.Interface
	SubscriberError error
}

func NewImagepolicySubscriber() *ImagepolicySubscriber {
	cfg, err := config.GetConfig()
	if err != nil {
		return &ImagepolicySubscriber{DynamicClient: nil, SubscriberError: err}
	}

	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return &ImagepolicySubscriber{DynamicClient: nil, SubscriberError: err}
	}

	return &ImagepolicySubscriber{DynamicClient: dynClient, SubscriberError: nil}
}

func (imagepolicySubscriber *ImagepolicySubscriber) List(ctx context.Context, client dynamic.Interface, namespace string, group string, version string) ([]unstructured.Unstructured, error) {
	var imagePolicyResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "imagepolicies"}
	list, err := client.Resource(imagePolicyResource).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (imagepolicySubscriber *ImagepolicySubscriber) Get(ctx context.Context, client dynamic.Interface, name string, namespace string, group string, version string) (*unstructured.Unstructured, error) {
	var imagePolicyResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "imagepolicies"}
	obj, err := client.Resource(imagePolicyResource).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (imagepolicySubscriber ImagepolicySubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	return nil
}

func (imagepolicySubscriber ImagepolicySubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) error {
	_, err := imagepolicySubscriber.Get(ctx, imagepolicySubscriber.DynamicClient, pipelineTrigger.Spec.Source.Name, pipelineTrigger.Namespace, group, version)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (imagepolicySubscriber ImagepolicySubscriber) CalculateCurrentState(secondClusterEnabled bool, secondClusterAddr string, secondClusterBearerToken string, ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList unstructured.UnstructuredList, group string, version string) (bool, bool, error) {
	// get the current image policy from the Flux ImagePolicy resource
	foundSource, err := imagepolicySubscriber.Get(ctx, imagepolicySubscriber.DynamicClient, pipelineTrigger.Spec.Source.Name, pipelineTrigger.Namespace, group, version)
	if err != nil {
		return false, false, err
	}
	var imagePolicy pipelinev1alpha1.ImagePolicy
	imagePolicy.GetImagePolicy(*foundSource)
	imagePolicy.GenerateDetails()
	imagePolicyLabels := imagePolicy.GenerateImagePolicyLabelsAsHash()
	imagePolicyLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["pipelineRef"].(map[string]interface{})["name"].(string)
	for i := range pipelineRunList.Items {
		if imagepolicySubscriber.HasIntersection(imagePolicyLabels, pipelineRunList.Items[i].GetLabels()) {
			return true, false, nil
		}
	}

	if secondClusterEnabled {
		myConfig := rest.Config{
			Host:            secondClusterAddr,
			APIPath:         "/",
			BearerToken:     secondClusterBearerToken,
			TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		}

		myClientSet, err := kubernetes.NewForConfig(&myConfig)
		if err != nil {
			return false, false, err
		}

		pRListExternal := &unstructured.UnstructuredList{}
		pipelineRunListExternal, err := myClientSet.RESTClient().Get().AbsPath("/apis/tekton.dev/v1").Namespace(pipelineTrigger.Namespace).Resource("pipelineruns").DoRaw(context.TODO())
		if err != nil {
			return false, false, err
		}
		if err := encodingJson.Unmarshal(pipelineRunListExternal, &pRListExternal); err != nil {
			return false, false, err
		}

		for i := range pRListExternal.Items {
			if imagepolicySubscriber.HasIntersection(imagePolicyLabels, pRListExternal.Items[i].GetLabels()) {
				return false, true, nil
			}
		}
	}
	return false, false, nil
}

func (imagepolicySubscriber ImagepolicySubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) (bool, error) {
	gotNewEvent := false
	foundSource, err := imagepolicySubscriber.Get(ctx, imagepolicySubscriber.DynamicClient, pipelineTrigger.Spec.Source.Name, pipelineTrigger.Namespace, group, version)
	if err != nil {
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

func (imagepolicySubscriber ImagepolicySubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*unstructured.Unstructured {
	var prs []*unstructured.Unstructured
	if len(pipelineTrigger.Status.ImagePolicy.Conditions) == 0 {
		paramsCorrectness, err := evaluatePipelineParamsForImage(pipelineTrigger)
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
	// Extract "params" field as a generic JSON object
	paramsJSON, _ := encodingJson.Marshal(pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["params"])

	// Create a slice of Param structs
	var params []Param

	// Unmarshal the JSON into the Param struct
	if err := encodingJson.Unmarshal(paramsJSON, &params); err != nil {
		c := v1.Condition{
			Type:               "False",
			Status:             v1.ConditionFalse,
			Reason:             "Unmarshal of params failed.",
			Message:            err.Error(),
			LastTransitionTime: v1.Now(),
		}
		pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(c)
		return false, err
	}

	for _, param := range params {
		if strings.Contains(param.Value, "$") {
			_, err := json.Exists(pipelineTrigger.Status.ImagePolicy.Details, param.Value)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (imagepolicySubscriber ImagepolicySubscriber) SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *unstructured.Unstructured, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	pipelineTrigger.Status.ImagePolicy.LatestPipelineRun = pipelineRunName
}

func (imagepolicySubscriber ImagepolicySubscriber) SetCurrentPipelineRunStatus(pipelineRunList unstructured.UnstructuredList, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	for _, pipelineRun := range pipelineRunList.Items {
		if pipelineRun.GetName() == pipelineTrigger.Status.ImagePolicy.LatestPipelineRun {
			status, statusOk := pipelineRun.Object["status"].(map[string]interface{})
			if !statusOk {
				c := v1.Condition{
					Type:               "False",
					Status:             v1.ConditionFalse,
					Reason:             "Unmarshal of status failed.",
					Message:            "Unmarshal of status failed.",
					LastTransitionTime: v1.Now(),
				}
				pipelineTrigger.Status.ImagePolicy.AddOrReplaceCondition(c)
			}
			currentConditions, conditionsOk := status["conditions"].([]interface{})
			if !conditionsOk {
				c := v1.Condition{
					Type:               "False",
					Status:             v1.ConditionFalse,
					Reason:             "Unmarshal of conditions failed.",
					Message:            "Unmarshal of conditions failed.",
					LastTransitionTime: v1.Now(),
				}
				pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(c)
			}
			for _, current := range currentConditions {
				if conditionMap, ok := current.(map[string]interface{}); ok {
					// Then, try to convert it to a v1.Condition
					c := v1.Condition{
						Type:               conditionMap["type"].(string),
						Status:             v1.ConditionStatus(conditionMap["status"].(string)),
						Reason:             conditionMap["reason"].(string),
						Message:            conditionMap["message"].(string),
						LastTransitionTime: v1.Now(),
					}
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
	return reconcile.Result{Requeue: true}, err
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

func (imagepolicySubscriber ImagepolicySubscriber) GetLastConditions(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) ([]string, []v1.Condition) {
	var conditions []v1.Condition
	var names []string
	conditions = append(conditions, pipelineTrigger.Status.ImagePolicy.GetLastCondition())
	names = append(names, pipelineTrigger.Status.ImagePolicy.ImageName)
	return names, conditions
}
