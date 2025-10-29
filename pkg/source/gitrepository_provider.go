package v1alpha1

import (
	"context"
	"errors"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type GitrepositorySubscriber struct {
	DynamicClient   dynamic.DynamicClient
	SubscriberError error
}

func NewGitrepositorySubscriber(dynClient dynamic.DynamicClient) *GitrepositorySubscriber {
	return &GitrepositorySubscriber{DynamicClient: dynClient, SubscriberError: nil}
}

func (gitrepositorySubscriber *GitrepositorySubscriber) List(ctx context.Context, client dynamic.DynamicClient, namespace string, group string, version string) ([]unstructured.Unstructured, error) {
	var gitRepositoryResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "gitrepositories"}
	list, err := client.Resource(gitRepositoryResource).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (gitrepositorySubscriber *GitrepositorySubscriber) Get(ctx context.Context, client dynamic.DynamicClient, name string, namespace string, group string, version string) (*unstructured.Unstructured, error) {
	var gitRepositoryResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "gitrepositories"}
	obj, err := client.Resource(gitRepositoryResource).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	return nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) IsValid(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error {
	sourceApiVersion := pipelineTrigger.Spec.Source.APIVersion
	sourceApiVersionSplitted := strings.Split(sourceApiVersion, "/")
	if len(sourceApiVersionSplitted) != 2 {
		return errors.New("could not split the api version of the source as expected")
	}

	tektonApiVersion := pipelineTrigger.Spec.PipelineRun.Object["apiVersion"].(string)
	apiVersionSplitted := strings.Split(tektonApiVersion, "/")
	if len(apiVersionSplitted) != 2 {
		return errors.New("could not split the api version of the pipelinerun as expected")
	}

	return nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) error {
	_, err := gitrepositorySubscriber.Get(ctx, gitrepositorySubscriber.DynamicClient, pipelineTrigger.Spec.Source.Name, pipelineTrigger.Namespace, group, version)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (gitrepositorySubscriber GitrepositorySubscriber) CalculateCurrentState(secondClusterEnabled bool, secondClusterAddr string, secondClusterBearerToken string, ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList unstructured.UnstructuredList, group string, version string) (bool, bool, error) {
	foundSource, err := gitrepositorySubscriber.Get(ctx, gitrepositorySubscriber.DynamicClient, pipelineTrigger.Spec.Source.Name, pipelineTrigger.Namespace, group, version)
	if err != nil {
		return false, false, err
	}
	var gitRepository pipelinev1alpha1.GitRepository
	gitRepository.GetGitRepository(*foundSource)
	gitRepository.GenerateDetails()
	gitRepositoryLabels := gitRepository.GenerateGitRepositoryLabelsAsHash()

	// Safely extract pipeline name with nil checks
	if spec, ok := pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{}); ok && spec != nil {
		if pipelineRef, ok := spec["pipelineRef"].(map[string]interface{}); ok && pipelineRef != nil {
			if name, ok := pipelineRef["name"].(string); ok && name != "" {
				gitRepositoryLabels["tekton.dev/pipeline"] = name
			}
		}
	}

	for i := range pipelineRunList.Items {
		if gitrepositorySubscriber.HasIntersection(gitRepositoryLabels, pipelineRunList.Items[i].GetLabels()) {
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
			if gitrepositorySubscriber.HasIntersection(gitRepositoryLabels, pRListExternal.Items[i].GetLabels()) {
				return false, true, nil
			}
		}
	}
	return false, false, nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) (bool, error) {
	gotNewEvent := false
	foundSource, err := gitrepositorySubscriber.Get(ctx, gitrepositorySubscriber.DynamicClient, pipelineTrigger.Spec.Source.Name, pipelineTrigger.Namespace, group, version)
	if err != nil {
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

func (gitrepositorySubscriber GitrepositorySubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*unstructured.Unstructured {
	var prs []*unstructured.Unstructured
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
	// Extract "params" field as a generic JSON object
	paramsJSON, _ := encodingJson.Marshal(pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["params"])

	// Create a slice of Param structs
	var params []Param

	// Unmarshal the JSON into the Param struct
	if err := encodingJson.Unmarshal(paramsJSON, &params); err != nil {
		c := v1.Condition{
			Type:               "False",
			Status:             v1.ConditionFalse,
			Reason:             "ParamsUnmarshalFailed",
			Message:            err.Error(),
			LastTransitionTime: v1.Now(),
		}
		pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(c)
		return false, err
	}

	for _, param := range params {
		if strings.Contains(param.Value, "$") {
			_, err := json.Exists(pipelineTrigger.Status.GitRepository.Details, param.Value)
			if err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

func (gitrepositorySubscriber GitrepositorySubscriber) SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *unstructured.Unstructured, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	pipelineTrigger.Status.GitRepository.LatestPipelineRun = pipelineRunName
}

func (gitrepositorySubscriber GitrepositorySubscriber) SetCurrentPipelineRunStatus(pipelineRunList unstructured.UnstructuredList, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	for _, pipelineRun := range pipelineRunList.Items {
		if pipelineRun.GetName() == pipelineTrigger.Status.GitRepository.LatestPipelineRun {
			status, statusOk := pipelineRun.Object["status"].(map[string]interface{})
			if !statusOk {
				c := v1.Condition{
					Type:               "False",
					Status:             v1.ConditionFalse,
					Reason:             "ParamsUnmarshalFailed",
					Message:            "Unmarshal of status failed.",
					LastTransitionTime: v1.Now(),
				}
				pipelineTrigger.Status.GitRepository.AddOrReplaceCondition(c)
			}
			currentConditions, conditionsOk := status["conditions"].([]interface{})
			if !conditionsOk {
				c := v1.Condition{
					Type:               "False",
					Status:             v1.ConditionFalse,
					Reason:             "ParamsUnmarshalFailed",
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
