package v1alpha1

import (
	//pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	"context"
	"errors"
	"strings"

	encodingJson "encoding/json"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/json"
	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PullrequestSubscriber struct {
	DynamicClient   dynamic.DynamicClient
	SubscriberError error
}

type PullRequestResult struct {
	BranchName           string
	BranchCommit         string
	Running              bool
	SecondClusterRunning bool
	Error                error
}

func NewPullrequestSubscriber(dynClient dynamic.DynamicClient) *PullrequestSubscriber {
	return &PullrequestSubscriber{DynamicClient: dynClient, SubscriberError: nil}
}

func (pullrequestSubscriber *PullrequestSubscriber) List(ctx context.Context, client dynamic.DynamicClient, namespace string, group string, version string) ([]unstructured.Unstructured, error) {
	var pullRequestResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "pullrequests"}
	list, err := client.Resource(pullRequestResource).Namespace(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (pullrequestSubscriber *PullrequestSubscriber) Get(ctx context.Context, client dynamic.DynamicClient, name string, namespace string, group string, version string) (*unstructured.Unstructured, error) {
	var pullRequestResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "pullrequests"}
	obj, err := client.Resource(pullRequestResource).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (pullrequestSubscriber PullrequestSubscriber) Subscribes(pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	return nil
}

func (pullrequestSubscriber PullrequestSubscriber) IsValid(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request) error {
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

func (pullrequestSubscriber PullrequestSubscriber) Exists(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) error {
	foundSource := &pullrequestv1alpha1.PullRequest{}
	err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (pullrequestSubscriber PullrequestSubscriber) CalculateCurrentState(secondClusterEnabled bool, secondClusterAddr string, secondClusterBearerToken string, ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, pipelineRunList unstructured.UnstructuredList, group string, version string) (bool, bool, error) {
	var res []bool
	var resExternal []bool
	// get the current branches from the pull request resource
	foundSource := &pullrequestv1alpha1.PullRequest{}
	if err := client.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource); err != nil {
		return false, false, err
	}
	var branches pipelinev1alpha1.Branches
	branches.GetPrBranches(foundSource.Status.SourceBranches)

	// check if there is a corresponding pipelinerun for every checkout branch
	if len(branches.Branches) > 0 {
		for key := range branches.Branches {
			tempBranch := branches.Branches[key]
			tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
			tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["pipelineRef"].(map[string]interface{})["name"].(string)
			for i := range pipelineRunList.Items {
				if pullrequestSubscriber.HasIntersection(tempBranchLabels, pipelineRunList.Items[i].GetLabels()) {
					res = append(res, true)
				}
			}
		}
	}

	if len(res) < len(branches.Branches) {
		if secondClusterEnabled {
			pRListExternal := &unstructured.UnstructuredList{}

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

			pipelineRunListExternal, err := myClientSet.RESTClient().Get().AbsPath("/apis/tekton.dev/v1").Namespace(pipelineTrigger.Namespace).Resource("pipelineruns").DoRaw(context.TODO())
			if err != nil {
				return false, false, err
			}
			if err := encodingJson.Unmarshal(pipelineRunListExternal, &pRListExternal); err != nil {
				return false, false, err
			}
			// check if there is a corresponding pipelinerun for every checkout branch
			if len(branches.Branches) > 0 {
				for key := range branches.Branches {
					tempBranch := branches.Branches[key]
					tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
					tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["pipelineRef"].(map[string]interface{})["name"].(string)
					for i := range pRListExternal.Items {
						if pullrequestSubscriber.HasIntersection(tempBranchLabels, pRListExternal.Items[i].GetLabels()) {
							resExternal = append(resExternal, true)
						}
					}
				}
			}
			if len(resExternal) >= len(branches.Branches) {
				return false, true, nil
			}
		}
	}

	return false, false, nil

}

func (pullrequestSubscriber PullrequestSubscriber) GetLatestEvent(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, client client.Client, req ctrl.Request, group string, version string) (bool, error) {
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

func (pullrequestSubscriber PullrequestSubscriber) SetCurrentPipelineRunName(ctx context.Context, client client.Client, pipelineRun *unstructured.Unstructured, pipelineRunName string, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
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

func (pullrequestSubscriber PullrequestSubscriber) CreatePipelineRunResource(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, r *runtime.Scheme) []*unstructured.Unstructured {
	var prs []*unstructured.Unstructured
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
	// Extract "params" field as a generic JSON object
	paramsJSON, _ := encodingJson.Marshal(pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["params"])

	// Create a slice of Param structs
	var params []Param

	// Unmarshal the JSON into the Param struct
	if err := encodingJson.Unmarshal(paramsJSON, &params); err != nil {
		// Handle the error
	}
	for _, param := range params {
		if strings.Contains(param.Value, "$") {
			_, err := json.Exists(currentBranch.Details, param.Value)
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

func (PullrequestSubscriber *PullrequestSubscriber) isStateIdentical(currentPipelineRun unstructured.Unstructured, currentBranch pipelinev1alpha1.Branch) bool {
	status, _ := currentPipelineRun.Object["status"].(map[string]interface{})
	conditions, _ := status["conditions"].([]interface{})
	for _, condition := range conditions {
		if conditionMap, ok := condition.(map[string]interface{}); ok {
			c := v1.Condition{
				Type:   conditionMap["type"].(string),
				Status: v1.ConditionStatus(conditionMap["status"].(string)),
				Reason: conditionMap["reason"].(string),
			}
			if (v1.ConditionStatus(c.Status) == v1.ConditionTrue) && (currentBranch.GetLastCondition().Status == v1.ConditionTrue) {
				return true
			}
			if (v1.ConditionStatus(c.Status) == v1.ConditionFalse) && (currentBranch.GetLastCondition().Status == v1.ConditionFalse) {
				return true
			}
		}
	}
	return false
}

func (pullrequestSubscriber *PullrequestSubscriber) SetCurrentPipelineRunStatus(pipelineRunList unstructured.UnstructuredList, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	if len(pipelineTrigger.Status.Branches.Branches) > 0 {
		for key := range pipelineTrigger.Status.Branches.Branches {
			tempBranch := pipelineTrigger.Status.Branches.Branches[key]
			//tempBranchLabels := tempBranch.GenerateBranchLabelsAsHash()
			// This label is automatically added by the Tekton Controller
			//tempBranchLabels["tekton.dev/pipeline"] = pipelineTrigger.Spec.PipelineRunSpec.PipelineRef.Name
			for _, pipelineRun := range pipelineRunList.Items {
				if pipelineRun.GetName() == tempBranch.LatestPipelineRun {
					if !pullrequestSubscriber.isStateIdentical(pipelineRun, tempBranch) {
						currBranch := tempBranch
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
				}
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
