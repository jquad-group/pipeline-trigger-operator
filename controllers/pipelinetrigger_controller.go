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

	core "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

const (
	sourceField                 = ".spec.source.name"
	pipelineField               = ".spec.pipeline.name"
	pipelineRunField            = ".name"
	myFinalizerName             = "pipeline.jquad.rocks/finalizer"
	pipelineRunLabelLatestEvent = "pipeline.jquad.rocks/latestEvent"
)

// PipelineTriggerReconciler reconciles a PipelineTrigger object
type PipelineTriggerReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

type Tekton struct {
	TektonClient *clientsetversioned.Clientset
}

// +kubebuilder:docs-gen:collapse=Reconciler Declaration

/*
There are two additional resources that the controller needs to have access to, other than PipelineTriggers.
- It needs to be able to fully manage Tekton PipelineRuns, as well as check their status.
- It also needs to be able to get, list and watch ImagePolicies and GitRepositories.
*/

//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pipelinetriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pipelinetriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=pipeline.jquad.rocks,resources=pipelinetriggers/finalizers,verbs=update
//+kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagepolicies,verbs=get;list;watch
//+kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagepolicies/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories,verbs=get;list;watch
//+kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=gitrepositories/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update;get;list;watch

/*
`Reconcile` will be in charge of reconciling the state of PipelineTriggers.
PipelineTriggers are used to manage PipelineRuns whose pods are started whenever the Source defined in the PipelineTrigger is updated.
When the Source detects new version, a new Tekton PipelineRun starts
*/
func (r *PipelineTriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	var pipelineTrigger pipelinev1alpha1.PipelineTrigger
	if err := r.Get(ctx, req.NamespacedName, &pipelineTrigger); err != nil {
		log.Error(err, "unable to fetch PipelineTrigger")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var err error
	var latestEvent string
	// Check if Source is from Kind ImagePolicy or GitRepository
	if pipelineTrigger.Spec.Source.Kind == "ImagePolicy" {
		foundSource := &imagereflectorv1.ImagePolicy{}
		err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		latestEvent = foundSource.Status.LatestImage
	} else if pipelineTrigger.Spec.Source.Kind == "GitRepository" {
		foundSource := &sourcev1.GitRepository{}
		err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		latestEvent = foundSource.Status.Artifact.Revision
	} else {
		return ctrl.Result{}, err
	}

	// Check if the Source resource exists
	if err != nil {
		// If a Source name is provided, then it must exist
		// Create an Event for the user to understand why their reconcile is failing.
		errorMsg := "Source " + pipelineTrigger.Spec.Source.Name + " in namespace " + pipelineTrigger.Namespace + "cannot be found."
		r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", errorMsg)
		return ctrl.Result{}, err
	}

	// check if the referenced tekton pipeline exists
	foundTektonPipeline := &tektondevv1.Pipeline{}
	err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Pipeline.Name, Namespace: pipelineTrigger.Namespace}, foundTektonPipeline)
	if err != nil {
		// If a pipeline name is provided, then it must exist
		// You will likely want to create an Event for the user to understand why their reconcile is failing.
		errorMsg := "Pipeline " + pipelineTrigger.Spec.Pipeline.Name + " in namespace " + pipelineTrigger.Namespace + " cannot be found."
		r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", errorMsg)
		pipelineTrigger.Status.PipelineStatus = "Error"
		r.Status().Update(ctx, &pipelineTrigger)
		return ctrl.Result{}, err
	}

	// check if there is a new event
	if latestEvent != pipelineTrigger.Status.LatestEvent {
		newVersionMsg := "Source " + pipelineTrigger.Spec.Source.Name + " in namespace " + pipelineTrigger.Namespace + " got new event " + latestEvent
		r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newVersionMsg)
		pipelineTrigger.Status.LatestPipelineRun = "Unknown"
		pipelineTrigger.Status.PipelineStatus = "Unknown"
		r.Status().Update(ctx, &pipelineTrigger)

		// check if a pipelinerun already exists, if not create a new one
		foundPipelineRun := &tektondevv1.PipelineRun{}
		err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Status.LatestPipelineRun, Namespace: pipelineTrigger.Namespace}, foundPipelineRun)
		if err != nil && errors.IsNotFound(err) {
			pipelineRunList := &tektondevv1.PipelineRunList{}
			pipelineRunListOpts := []client.ListOption{
				client.InNamespace(pipelineTrigger.Namespace),
				client.MatchingLabels{"pipeline.jquad.rocks/pipelinetrigger": pipelineTrigger.Name},
			}
			r.List(ctx, pipelineRunList, pipelineRunListOpts...)
			// if pipelineruns nr < MaxHistory, just create a new pipelinerun
			if len(pipelineRunList.Items) < int(pipelineTrigger.Spec.Pipeline.MaxHistory) {
				pr, pipelineRunError := pipelineTrigger.Spec.Pipeline.CreatePipelineRun(ctx, req, pipelineTrigger)
				pipelineTrigger.Status.LatestPipelineRun = pr.Name
				pipelineTrigger.Status.LatestEvent = latestEvent
				r.Status().Update(ctx, &pipelineTrigger)
				ctrl.SetControllerReference(&pipelineTrigger, pr, r.Scheme)
				if pipelineRunError != nil {
					log.Error(err, "Failed to create new PipelineRun", "PipelineTrigger.Namespace", pipelineTrigger.Namespace, "PipelineTrigger.Name", pipelineTrigger.Name)
					return ctrl.Result{}, err
				}
				newPipelineRunMsg := "PipelineRun " + pr.Name + " in namespace " + pr.Namespace + " created. "
				r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newPipelineRunMsg)
				return ctrl.Result{Requeue: true}, nil
			} else {
				// if pipelineruns > MaxHistory delete one pipelinerun, return and requeue
				deletePipelineRun := &pipelineRunList.Items[0]
				r.Delete(ctx, deletePipelineRun, client.GracePeriodSeconds(5))
				return ctrl.Result{Requeue: true}, nil
			}
		} else if err != nil {
			log.Error(err, "Failed to get PipelineRun")
			return ctrl.Result{}, err
		}

	}

	if (pipelineTrigger.Status.PipelineStatus != "Completed") && (pipelineTrigger.Status.PipelineStatus == "Succeeded") {
		pipelineTrigger.Status.CurrentPipelineRetry = 0
		r.Status().Update(ctx, &pipelineTrigger)
		return ctrl.Result{}, nil
	} else if (pipelineTrigger.Status.PipelineStatus == "Completed") && (pipelineTrigger.Status.PipelineStatus != "Succeeded") {
		pipelineTrigger.Status.CurrentPipelineRetry = 0
		r.Status().Update(ctx, &pipelineTrigger)
		return ctrl.Result{}, nil
	} else if pipelineTrigger.Status.PipelineStatus == "Failed" {
		if pipelineTrigger.Status.CurrentPipelineRetry < pipelineTrigger.Spec.Pipeline.Retries {
			pipelineTrigger.Status.CurrentPipelineRetry++
			pipelineTrigger.Status.PipelineStatus = "Unknown"
			pr, pipelineRunError := pipelineTrigger.Spec.Pipeline.CreatePipelineRun(ctx, req, pipelineTrigger)
			pipelineTrigger.Status.LatestPipelineRun = pr.Name
			r.Status().Update(ctx, &pipelineTrigger)
			ctrl.SetControllerReference(&pipelineTrigger, pr, r.Scheme)
			if pipelineRunError != nil {
				log.Error(err, "Failed to create new PipelineRun", "PipelineTrigger.Namespace", pipelineTrigger.Namespace, "PipelineTrigger.Name", pipelineTrigger.Name)
				return ctrl.Result{}, err
			}
			newPipelineRunMsg := "PipelineRun " + pr.Name + " in namespace " + pr.Namespace + " created. "
			r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newPipelineRunMsg)
			return ctrl.Result{Requeue: true}, nil
		} else {
			return ctrl.Result{}, nil
		}
	} else {
		foundPipelineRun := &tektondevv1.PipelineRun{}
		err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Status.LatestPipelineRun, Namespace: pipelineTrigger.Namespace}, foundPipelineRun)
		r.reconcilePipelineRun(ctx, foundPipelineRun, pipelineTrigger)
	}

	return ctrl.Result{}, nil

}

func (r *PipelineTriggerReconciler) reconcilePipelineRun(ctx context.Context, pr *tektondevv1.PipelineRun, pt v1alpha1.PipelineTrigger) (result ctrl.Result, err error) {
	log := logr.FromContext(ctx)

	if len(pr.Status.Conditions) > 0 {
		if pt.Status.PipelineStatus != pr.Status.Conditions[0].Reason {
			pt.Status.PipelineStatus = pr.Status.Conditions[0].Reason
			msg := "PipelineRun " + pr.Name + " in namespace " + pr.Namespace + " is with status: " + pr.Status.Conditions[0].Reason
			r.recorder.Event(&pt, core.EventTypeNormal, "Info", msg)
			ptErrStatus := r.Status().Update(ctx, &pt)
			if ptErrStatus != nil {
				log.Error(ptErrStatus, "Failed to update PipelineTrigger status")
				return ctrl.Result{}, ptErrStatus
			}
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ctrl.Result{}, nil
}

/*
Finally, we add this reconciler to the manager, so that it gets started when the manager is started.
Since we create dependency Tekton PipelineRuns during the reconcile, we can specify that the controller `Owns` Tekton PipelineRun.
However the ImagePolicies and GitRepositories that we want to watch are not owned by the PipelineRun object.
Therefore we must specify a custom way of watching those objects.
This watch logic is complex, so we have split it into a separate method.
*/

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.recorder = mgr.GetEventRecorderFor("PipelineTrigger")

	/*
		The `source.name` field must be indexed by the manager, so that we will be able to lookup `PipelineTriggers` by a referenced `source.name` name.
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
		- If pipelineRunField _x_ is updated, which PipelineTrigger are affected?
	*/

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pipelinev1alpha1.PipelineTrigger{}, pipelineRunField, func(rawObj client.Object) []string {
		// Extract the PipelineRun name from the PipelineTrigger Spec, if one is provided
		pipelineTrigger := rawObj.(*pipelinev1alpha1.PipelineTrigger)
		if pipelineTrigger.Status.LatestPipelineRun == "" {
			return nil
		}
		return []string{pipelineTrigger.Status.LatestPipelineRun}
	}); err != nil {
		return err
	}

	/*
		The controller will first register the Type that it manages, as well as the types of subresources that it controls.
		Since we also want to watch ImagePolicies and GitRepositories that are not controlled or managed by the controller, we will need to use the `Watches()` functionality as well.
		The `Watches()` function is a controller-runtime API that takes:
		- A Kind (i.e. `ImagePolicy, GitRepository`)
		- A mapping function that converts a `ImagePolicy,GitRepository` object to a list of reconcile requests for `PipelineTriggers`.
		We have separated this out into a separate function.
		- A list of options for watching the `ImagePolicies,GitRepositories`
		  - In our case, we only want the watch to be triggered when the LatestImage or LatestRevisioin of the ImagePolicy or GitRepository is changed.
	*/

	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1alpha1.PipelineTrigger{}).
		Owns(&tektondevv1.PipelineRun{}).
		Watches(
			&source.Kind{Type: &imagereflectorv1.ImagePolicy{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSource),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&source.Kind{Type: &sourcev1.GitRepository{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSource),
			builder.WithPredicates(SourceRevisionChangePredicate{}),
		).
		Watches(
			&source.Kind{Type: &tektondevv1.PipelineRun{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPipelineRun),
			//builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)

}

/*
	Because we have already created an index on the `source.name` reference field, this mapping function is quite straight forward.
	We first need to list out all `PipelineTriggers` that use `source.name` given in the mapping function.
	This is done by merely submitting a List request using our indexed field as the field selector.
	When the list of `PipelineTriggers` that reference the `ImagePolicy` or `GitRepository` is found,
	we just need to loop through the list and create a reconcile request for each one.
	If an error occurs fetching the list, or no `PipelineTriggers` are found, then no reconcile requests will be returned.
*/
func (r *PipelineTriggerReconciler) findObjectsForSource(source client.Object) []reconcile.Request {
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

func (r *PipelineTriggerReconciler) findObjectsForPipeline(pipeline client.Object) []reconcile.Request {
	attachedPipelineTriggers := &pipelinev1alpha1.PipelineTriggerList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(pipelineField, pipeline.GetName()),
		Namespace:     pipeline.GetNamespace(),
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

func (r *PipelineTriggerReconciler) findObjectsForPipelineRun(pipelineRun client.Object) []reconcile.Request {
	attachedPipelineTriggers := &pipelinev1alpha1.PipelineTriggerList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(pipelineRunField, pipelineRun.GetName()),
		Namespace:     pipelineRun.GetNamespace(),
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

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
