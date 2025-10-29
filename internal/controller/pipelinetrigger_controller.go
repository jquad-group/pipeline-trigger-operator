/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"errors"
	"strings"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1"

	"github.com/go-logr/logr"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	pipelinev1alpha1predicate "github.com/jquad-group/pipeline-trigger-operator/pkg/predicate"

	metricsApi "github.com/jquad-group/pipeline-trigger-operator/pkg/metrics"
	sourceApi "github.com/jquad-group/pipeline-trigger-operator/pkg/source"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	sourceField             = ".spec.source.name"
	pipelineRunField        = ".metadata.pipelinerun.name"
	FIELD_MANAGER           = "pipelinetrigger-controller"
	PULLREQUEST_KIND_NAME   = "PullRequest"
	IMAGEPOLICY_KIND_NAME   = "ImagePolicy"
	GITREPOSITORY_KIND_NAME = "GitRepository"
)

// PipelineTriggerReconciler reconciles a PipelineTrigger object
type PipelineTriggerReconciler struct {
	client.Client
	DynamicClient            dynamic.DynamicClient
	Log                      logr.Logger
	Scheme                   *runtime.Scheme
	recorder                 record.EventRecorder
	MetricsRecorder          *metricsApi.Recorder
	SecondClusterEnabled     bool
	SecondClusterAddr        string
	SecondClusterBearerToken string
}

type Param struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// +kubebuilder:docs-gen:collapse=Reconciler Declaration

/*
There are two additional resources that the controller needs to have access to, other than PipelineTriggers.
- It needs to be able to create Tekton PipelineRuns, as well as check their status.
- It also needs to be able to get, list and watch ImagePolicies, GitRepositories and PullRequests.
*/

//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pipelinetriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pipelinetriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pipelinetriggers/finalizers,verbs=update
//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pullrequests,verbs=get;list;watch
//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pullrequests/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagepolicies,verbs=get;list;watch
//+kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagepolicies/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=credentials.jquad.rocks,resources=managedcredentials,verbs=get;list;watch
//+kubebuilder:rbac:groups=credentials.jquad.rocks,resources=managedcredentials/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update;get;list;watch

/*
`Reconcile` will be in charge of reconciling the state of PipelineTriggers.
PipelineTriggers are used to manage PipelineRuns, which are created whenever the Source defined in the PipelineTrigger is updated.
When the Source detects new version, a new Tekton PipelineRun starts
*/
func (r *PipelineTriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	var pipelineTrigger pipelinev1alpha1.PipelineTrigger
	if err := r.Get(ctx, req.NamespacedName, &pipelineTrigger); err != nil {
		// return and dont requeue
		return ctrl.Result{}, nil
	}

	patch := &unstructured.Unstructured{}
	patch.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   pipelinev1alpha1.GroupVersion.Group,
		Version: pipelinev1alpha1.GroupVersion.Version,
		Kind:    "PipelineTrigger",
	})
	patch.SetNamespace(pipelineTrigger.GetNamespace())
	patch.SetName(pipelineTrigger.GetName())
	patchOptions := &client.PatchOptions{
		FieldManager: FIELD_MANAGER,
		Force:        pointer.Bool(true),
	}

	subPatchOptions := &client.SubResourcePatchOptions{
		PatchOptions: *patchOptions,
	}

	// create the source subscriber
	sourceSubscriber := createSourceSubscriber(&pipelineTrigger, r.DynamicClient)
	// check if the API Version of the Source and PipelineRun are valid
	if err := sourceSubscriber.IsValid(ctx, pipelineTrigger, r.Client, req); err != nil {
		// requeue with error (maybe the resource was deployt afterwards?)
		return sourceSubscriber.ManageError(ctx, &pipelineTrigger, req, r.Client, err)
	}

	// Namespace cannot be defined in the metadata.namespace, due to multitenancy problems
	// need to implement impersonation first
	if r.isNamespaceDefinedInPipelineRun(pipelineTrigger) {
		namespaceDefinedError := errors.New("spec.pipelineRun.metadata.namespace not supported")
		return sourceSubscriber.ManageError(ctx, &pipelineTrigger, req, r.Client, namespaceDefinedError)
	}

	// set the group and version
	sourceApiVersion := pipelineTrigger.Spec.Source.APIVersion
	sourceApiVersionSplitted := strings.Split(sourceApiVersion, "/")
	sourceGroup := sourceApiVersionSplitted[0]
	sourceVersion := sourceApiVersionSplitted[1]

	tektonApiVersion := pipelineTrigger.Spec.PipelineRun.Object["apiVersion"].(string)
	apiVersionSplitted := strings.Split(tektonApiVersion, "/")
	tektonGroup := apiVersionSplitted[0]
	tektonVersion := apiVersionSplitted[1]

	// check if the referenced source exists
	if err := sourceSubscriber.Exists(ctx, pipelineTrigger, r.Client, req, sourceGroup, sourceVersion); err != nil {
		// requeue with error (maybe the resource was deployt afterwards?)
		return sourceSubscriber.ManageError(ctx, &pipelineTrigger, req, r.Client, err)
	}

	// check if the referenced tekton pipeline exists
	if err := r.existsPipelineResource(ctx, pipelineTrigger, r.DynamicClient, tektonGroup, tektonVersion); err != nil {
		// requeue with error (maybe the resource was deployt afterwards?)
		return sourceSubscriber.ManageError(ctx, &pipelineTrigger, req, r.Client, err)
	}

	//var pipelineRunListTest tektondevv1.PipelineRunList
	//errList := r.List(ctx, &pipelineRunListTest, client.InNamespace(req.Namespace), client.MatchingFields{pipelineRunField: req.Name})

	pipelineRunList, errList := ListTektonResources(ctx, r.DynamicClient, pipelineTrigger.Namespace, tektonGroup, tektonVersion, "pipelineruns")
	if errList != nil {
		log.Error(errList, "Failed to list PipelineRuns in ", pipelineTrigger.Namespace, " with label ", pipelineTrigger.Name)
	}

	running, runninOnSecondCluster, err := sourceSubscriber.CalculateCurrentState(r.SecondClusterEnabled, r.SecondClusterAddr, r.SecondClusterBearerToken, ctx, &pipelineTrigger, r.Client, pipelineRunList, sourceGroup, sourceVersion)
	if err != nil {
		log.Error(err, "Second cluster is enabled, but the controller cannot connect to it ", pipelineTrigger.Namespace)
	}

	if runninOnSecondCluster {
		newVersionMsg := "Pipeline is running on the second cluster in namespace " + pipelineTrigger.Namespace
		r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newVersionMsg)
	}

	if !running && !runninOnSecondCluster {

		// Get the Latest Source Event
		_, err := sourceSubscriber.GetLatestEvent(ctx, &pipelineTrigger, r.Client, req, sourceGroup, sourceVersion)
		if err != nil {
			return sourceSubscriber.ManageError(ctx, &pipelineTrigger, req, r.Client, err)
		}
		// Create the PipelineRun resources
		prs := sourceSubscriber.CreatePipelineRunResource(&pipelineTrigger, r.Scheme)

		// Apply ManagedCredential settings if referenced
		managedCred, err := pipelineTrigger.GetManagedCredential(ctx, r.Client)
		if err != nil {
			credRefName := "unknown"
			if pipelineTrigger.Spec.CredentialsRef != nil {
				credRefName = pipelineTrigger.Spec.CredentialsRef.Name
			}
			log.Error(err, "Failed to get ManagedCredential", "namespace", pipelineTrigger.Namespace, "name", credRefName)
			r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Warning", "Failed to get ManagedCredential: "+err.Error())
		} else if managedCred != nil {
			// Apply ManagedCredential settings to each PipelineRun
			for i := range prs {
				pipelineTrigger.ApplyManagedCredentialToPipelineRun(prs[i], managedCred)
			}
			log.Info("Applied ManagedCredential settings", "namespace", pipelineTrigger.Namespace, "name", pipelineTrigger.Spec.CredentialsRef.Name)
		}

		// Start the PipelineRun resources
		for pipelineRunCnt := 0; pipelineRunCnt < len(prs); pipelineRunCnt++ {
			instanceName, _, errRun := pipelineTrigger.StartPipelineRun(prs[pipelineRunCnt], ctx, req, r.Client)
			if errRun != nil {
				r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", errRun.Error())
			}
			sourceSubscriber.SetCurrentPipelineRunName(ctx, r.Client, prs[pipelineRunCnt], instanceName, &pipelineTrigger)
			newVersionMsg := "Started the pipeline " + instanceName + " in namespace " + pipelineTrigger.Namespace
			r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newVersionMsg)
		}

		patch.UnstructuredContent()["status"] = pipelineTrigger.Status
		errStatus := r.Status().Patch(ctx, patch, client.Apply, subPatchOptions)
		if errStatus != nil {
			r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Warning", errStatus.Error())
		}

	}

	sourceSubscriber.SetCurrentPipelineRunStatus(pipelineRunList, &pipelineTrigger)

	patch.UnstructuredContent()["status"] = pipelineTrigger.Status
	patch.SetManagedFields(nil)
	errStatus := r.Status().Patch(ctx, patch, client.Apply, subPatchOptions)
	if errStatus != nil {
		r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Warning", errStatus.Error())
	}

	if r.MetricsRecorder != nil {
		r.MetricsRecorder.RecordCondition(pipelineTrigger, sourceSubscriber)
	}

	return ctrl.Result{}, nil

}

func createSourceSubscriber(pipelineTrigger *pipelinev1alpha1.PipelineTrigger, dynClient dynamic.DynamicClient) sourceApi.SourceSubscriber {
	switch pipelineTrigger.Spec.Source.Kind {
	case PULLREQUEST_KIND_NAME:
		return sourceApi.NewPullrequestSubscriber(dynClient)
	case GITREPOSITORY_KIND_NAME:
		return sourceApi.NewGitrepositorySubscriber(dynClient)
	case IMAGEPOLICY_KIND_NAME:
		return sourceApi.NewImagepolicySubscriber(dynClient)
	}

	return nil
}

/*
Finally, we add this reconciler to the manager, so that it gets started when the manager is started.
Since we create dependency Tekton PipelineRuns during the reconcile, we can specify that the controller `Owns` Tekton PipelineRun.
However the ImagePolicies, GitRepositories and PullRequests that we want to watch are not owned by the PipelineTrigger object.
Therefore we must specify a custom way of watching those objects.
This watch logic is complex, so we have split it into a separate method.
*/

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("PipelineTrigger")

	/*
		The `sourceField` field must be indexed by the manager, so that we will be able to lookup `PipelineTriggers` by a referenced `sourceField` name.
		This will allow for quickly answer the question:
		- If source _x_ is updated, which PipelineTrigger are affected?
	*/

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pipelinev1alpha1.PipelineTrigger{}, sourceField, func(rawObj client.Object) []string {
		// Extract the Source name from the PipelineTrigger Spec, if one is provided
		pipelineTrigger := rawObj.(*pipelinev1alpha1.PipelineTrigger)
		if pipelineTrigger.Spec.Source.Name == "" {
			return nil
		}
		return []string{pipelineTrigger.Spec.Source.Name}
	}); err != nil {
		return err
	}

	/*
		The `pipelineRunField` field must be indexed by the manager, so that we will be able to lookup `PipelineTriggers` by a referenced `pipelineRunField` name.
		This will allow for quickly answer the question:
		- If PipelineRun _x_ is updated, which PipelineTrigger are affected?
	*/
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &tektondevv1.PipelineRun{}, pipelineRunField, func(rawObj client.Object) []string {
		// grab the job object, extract the owner...
		job := rawObj.(*tektondevv1.PipelineRun)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...make sure it's a PipelineTrigger...
		if owner.Kind != "PipelineTrigger" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	/*
		The controller will first register the Type that it manages, as well as the types of subresources that it controls.
		Since we also want to watch ImagePolicies, GitRepositories and PullRequests that are not controlled or managed by the controller, we will need to use the `Watches()` functionality as well.
		The `Watches()` function is a controller-runtime API that takes:
		- A Kind (i.e. `ImagePolicy, GitRepository, PullRequest`)
		- A mapping function that converts a `ImagePolicy,GitRepository,PullRequest` object to a list of reconcile requests for `PipelineTriggers`.
		We have separated this out into a separate function.
		- A list of options for watching the `ImagePolicies,GitRepositories,PullRequests,PipelineRuns`
		  - In our case, we only want the watch to be triggered when the LatestImage or LatestRevisioin of the ImagePolicy or GitRepository or Branches of the PullRequest is changed.
		  - Reconcile is triggered when the status of the PipelineRun, that is owned by the controller is changed
	*/

	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1alpha1.PipelineTrigger{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&imagereflectorv1.ImagePolicy{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSource),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&sourcev1.GitRepository{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSource),
			builder.WithPredicates(pipelinev1alpha1predicate.SourceRevisionChangePredicate{}),
		).
		Watches(
			&pullrequestv1alpha1.PullRequest{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSource),
			builder.WithPredicates(pipelinev1alpha1predicate.PullRequestStatusChangePredicate{}),
		).
		Watches(
			&tektondevv1.PipelineRun{},
			handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &pipelinev1alpha1.PipelineTrigger{}, handler.OnlyControllerOwner()),
			builder.WithPredicates(pipelinev1alpha1predicate.StatusChangePredicate{}),
		).
		Complete(r)

}

/*
Because we have already created an index on the `source.name` reference field, this mapping function is quite straight forward.
We first need to list out all `PipelineTriggers` that use `source.name` given in the mapping function.
This is done by merely submitting a List request using our indexed field as the field selector.
When the list of `PipelineTriggers` that reference the `ImagePolicy`,`GitRepository` or `PullRequest` is found,
we just need to loop through the list and create a reconcile request for each one.
If an error occurs fetching the list, or no `PipelineTriggers` are found, then no reconcile requests will be returned.
*/
func (r *PipelineTriggerReconciler) findObjectsForSource(ctx context.Context, source client.Object) []reconcile.Request {
	attachedPipelineTriggers := &pipelinev1alpha1.PipelineTriggerList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(sourceField, source.GetName()),
		Namespace:     source.GetNamespace(),
	}
	err := r.List(context.TODO(), attachedPipelineTriggers, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedPipelineTriggers.Items))
	for i, item := range attachedPipelineTriggers.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func ListTektonResources(ctx context.Context, client dynamic.DynamicClient, namespace string, tektonGroup string, tektonVersion string, tektonResource string) (unstructured.UnstructuredList, error) {
	var tektonApiResource = schema.GroupVersionResource{Group: tektonGroup, Version: tektonVersion, Resource: tektonResource}

	list, err := client.Resource(tektonApiResource).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return unstructured.UnstructuredList{}, err
	}

	return *list, nil
}

func (r *PipelineTriggerReconciler) existsPipelineResource(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger, client dynamic.DynamicClient, group string, version string) error {
	// check if the referenced tekton pipeline exists
	var pipelineRunResource = schema.GroupVersionResource{Group: group, Version: version, Resource: "pipelines"}
	pipelineRef := pipelineTrigger.Spec.PipelineRun.Object["spec"].(map[string]interface{})["pipelineRef"].(map[string]interface{})["name"].(string)
	_, err := client.Resource(pipelineRunResource).Namespace(pipelineTrigger.Namespace).Get(ctx, pipelineRef, metav1.GetOptions{})
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (r *PipelineTriggerReconciler) isNamespaceDefinedInPipelineRun(pipelineTrigger pipelinev1alpha1.PipelineTrigger) bool {
	if (pipelineTrigger.Spec.PipelineRun.GetNamespace() != pipelineTrigger.GetNamespace()) && len(pipelineTrigger.Spec.PipelineRun.GetNamespace()) > 0 {
		return true
	}
	return false
}
