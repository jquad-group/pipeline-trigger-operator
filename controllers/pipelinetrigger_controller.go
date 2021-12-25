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
	//"errors"

	core "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

const (
	imagePolicyField = ".spec.imagePolicy"
	pipelineRunField = ".name"
	myFinalizerName  = "pipeline.jquad.rocks/finalizer"
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
- It also needs to be able to get, list and watch ImagePolicies.
*/

//+kubebuilder:rbac:groups=jquad.rocks.pipeline,resources=pipelinetriggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jquad.rocks.pipeline,resources=pipelinetriggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jquad.rocks.pipeline,resources=pipelinetriggers/finalizers,verbs=update
//+kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagepolicies,verbs=get;list;watch
//+kubebuilder:rbac:groups=image.toolkit.fluxcd.io,resources=imagepolicies/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update;get;list;watch

/*
`Reconcile` will be in charge of reconciling the state of PipelineTriggers.
PipelineTriggers are used to manage Tekton PipelineRuns whose pods are started whenever the imagePolicy defined in the PipelineRun is updated.
For that reason we need to add an annotation to the PodTemplate within the Tekton PipelineRun we create.
This annotation will keep track of the latest version of the base image used for the build process.
Therefore when the imagePolicy detects new base image version, the PodTemplate in the Tekton PipelineRun will change.
This will start new Tekton PipelineRun.
Skip down to the `SetupWithManager` function to see how we ensure that `Reconcile` is called when the referenced `ImagePolicies` are updated.
*/
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PipelineTriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	var pipelineTrigger pipelinev1alpha1.PipelineTrigger
	if err := r.Get(ctx, req.NamespacedName, &pipelineTrigger); err != nil {
		log.Error(err, "unable to fetch PipelineTrigger")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else {

		//var imagePolicyVersion string
		if pipelineTrigger.Spec.ImagePolicy != "" {
			ImagePolicy := pipelineTrigger.Spec.ImagePolicy
			foundImagePolicy := &imagereflectorv1.ImagePolicy{}
			err := r.Get(ctx, types.NamespacedName{Name: ImagePolicy, Namespace: pipelineTrigger.Namespace}, foundImagePolicy)
			if err != nil {
				// If a imagePolicy name is provided, then it must exist
				// Create an Event for the user to understand why their reconcile is failing.
				errorMsg := "Image Policy " + pipelineTrigger.Spec.ImagePolicy + " in namespace " + pipelineTrigger.Namespace + "cannot be found."
				r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", errorMsg)
				return ctrl.Result{}, err

			} else {
				var pipelineRun *tektondevv1.PipelineRun
				if pipelineTrigger.Status.LatestImage != foundImagePolicy.Status.LatestImage {
					pipelineTrigger.Status.LatestImage = foundImagePolicy.Status.LatestImage
					msg := "Image Policy " + pipelineTrigger.Spec.ImagePolicy + " in namespace " + pipelineTrigger.Namespace + " got new version " + foundImagePolicy.Status.LatestImage
					r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", msg)
					ptErr := r.Status().Update(ctx, &pipelineTrigger)
					if ptErr != nil {
						log.Error(ptErr, "Failed to update PipelineTrigger status")
						return ctrl.Result{}, ptErr
					}

					if pipelineTrigger.Spec.Pipeline.Name != "" {
						Pipeline := pipelineTrigger.Spec.Pipeline
						foundTektonPipeline := &tektondevv1.Pipeline{}
						err := r.Get(ctx, types.NamespacedName{Name: Pipeline.Name, Namespace: pipelineTrigger.Namespace}, foundTektonPipeline)
						if err != nil {
							// If a pipeline name is provided, then it must exist
							// You will likely want to create an Event for the user to understand why their reconcile is failing.
							errorMsg := "Pipeline " + pipelineTrigger.Spec.Pipeline.Name + " in namespace " + pipelineTrigger.Namespace + " cannot be found."
							r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", errorMsg)
							return ctrl.Result{}, err
						}

						pr, err := Pipeline.CreatePipelineRun(ctx, req, pipelineTrigger)
						pipelineRun = pr
						msg := "PipelineRun " + pr.Name + " in namespace " + pr.Namespace + " created. "
						r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", msg)

					}

				}

				foundTektonPipelineRun := &tektondevv1.PipelineRun{}
				prErr := r.Get(ctx, types.NamespacedName{Name: pipelineRun.Name, Namespace: pipelineRun.Namespace}, foundTektonPipelineRun)
				if prErr != nil {
					errorMsg := "PipelineRun " + pipelineRun.Name + " in namespace " + pipelineRun.Namespace + " cannot be found."
					r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", errorMsg)
					return ctrl.Result{}, err
				} else {
					if foundTektonPipelineRun.Status.Conditions[0].Reason == tektondevv1.PipelineRunReasonFailed.String() {
						pipelineTrigger.Status.PipelineStatus = "Failed"
						msg := "PipelineRun " + pipelineRun.Name + " in namespace " + pipelineRun.Namespace + " has failed. See the logs for more information."
						r.recorder.Event(&pipelineTrigger, core.EventTypeWarning, "Error", msg)
					} else {
						pipelineTrigger.Status.PipelineStatus = foundTektonPipelineRun.Status.Conditions[0].Reason
						msg := "PipelineRun " + pipelineRun.Name + " in namespace " + pipelineRun.Namespace + " is with status: " + foundTektonPipelineRun.Status.Conditions[0].Reason
						r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", msg)
					}

					ptErr := r.Status().Update(ctx, &pipelineTrigger)
					if ptErr != nil {
						log.Error(ptErr, "Failed to update PipelineTrigger status")
						return ctrl.Result{}, ptErr
					}
				}

				// examine DeletionTimestamp to determine if object is under deletion
				if pipelineTrigger.ObjectMeta.DeletionTimestamp.IsZero() {
					// The object is not being deleted, so if it does not have our finalizer,
					// then lets add the finalizer and update the object. This is equivalent
					// registering our finalizer.
					if !containsString(pipelineTrigger.GetFinalizers(), myFinalizerName) {
						controllerutil.AddFinalizer(&pipelineTrigger, myFinalizerName)
						if err := r.Update(ctx, &pipelineTrigger); err != nil {
							return ctrl.Result{}, err
						}
					}
				} else {
					// The object is being deleted
					if containsString(pipelineTrigger.GetFinalizers(), myFinalizerName) {
						// our finalizer is present, so lets handle any external dependency
						if err := r.Delete(ctx, pipelineRun); err != nil {
							// if fail to delete the external dependency here, return with error
							// so that it can be retried
							return ctrl.Result{}, err
						}

						// remove our finalizer from the list and update it.
						controllerutil.RemoveFinalizer(&pipelineTrigger, myFinalizerName)
						if err := r.Update(ctx, &pipelineTrigger); err != nil {
							return ctrl.Result{}, err
						}
					}

					// Stop reconciliation as the item is being deleted
					return ctrl.Result{}, nil
				}

			}

		}

	}

	return ctrl.Result{}, nil

}

/*
Finally, we add this reconciler to the manager, so that it gets started
when the manager is started.
Since we create dependency Tekton PipelineRuns during the reconcile, we can specify that the controller `Owns` Tekton PipelineRun.
However the ImagePolicies that we want to watch are not owned by the PipelineRun object.
Therefore we must specify a custom way of watching those objects.
This watch logic is complex, so we have split it into a separate method.
*/

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.recorder = mgr.GetEventRecorderFor("PipelineTrigger")

	/*
		The `imagePolicy` field must be indexed by the manager, so that we will be able to lookup `PipelineTriggers` by a referenced `ImagePolicy` name.
		This will allow for quickly answer the question:
		- If ImagePolicy _x_ is updated, which PipelineTrigger are affected?
	*/

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pipelinev1alpha1.PipelineTrigger{}, imagePolicyField, func(rawObj client.Object) []string {
		// Extract the ImagePolicy name from the PipelineTrigger Spec, if one is provided
		pipelineTrigger := rawObj.(*pipelinev1alpha1.PipelineTrigger)
		if pipelineTrigger.Spec.ImagePolicy == "" {
			return nil
		}
		return []string{pipelineTrigger.Spec.ImagePolicy}
	}); err != nil {
		return err
	}

	/*
		The `pipelineRunField` field must be indexed by the manager, so that we will be able to lookup `PipelineTriggers` by a referenced `PipelineRun` name.
		This will allow for quickly answer the question:
		- If PipelineRun _x_ is updated, which PipelineTrigger are affected?
	*/

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pipelinev1alpha1.PipelineTrigger{}, pipelineRunField, func(rawObj client.Object) []string {
		// Extract the PipelineRun name from the PipelineTrigger Spec, if one is provided
		pipelineTrigger := rawObj.(*pipelinev1alpha1.PipelineTrigger)
		if pipelineTrigger.Name == "" {
			return nil
		}
		return []string{pipelineTrigger.Name}
	}); err != nil {
		return err
	}

	/*
		The controller will first register the Type that it manages, as well as the types of subresources that it controls.
		Since we also want to watch ImagePolicies that are not controlled or managed by the controller, we will need to use the `Watches()` functionality as well.
		The `Watches()` function is a controller-runtime API that takes:
		- A Kind (i.e. `ImagePolicy`)
		- A mapping function that converts a `ImagePolicy` object to a list of reconcile requests for `PipelineTriggers`.
		We have separated this out into a separate function.
		- A list of options for watching the `ImagePolicies`
		  - In our case, we only want the watch to be triggered when the LatestImage of the ImagePolicy is changed.
	*/

	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1alpha1.PipelineTrigger{}).
		Owns(&tektondevv1.PipelineRun{}).
		Watches(
			&source.Kind{Type: &imagereflectorv1.ImagePolicy{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForImagePolicy),
			//builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(
			&source.Kind{Type: &tektondevv1.PipelineRun{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForPipelineRun),
			//builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)

}

/*
	Because we have already created an index on the `imagePolicy` reference field, this mapping function is quite straight forward.
	We first need to list out all `PipelineTriggers` that use `ImagePolicy` given in the mapping function.
	This is done by merely submitting a List request using our indexed field as the field selector.
	When the list of `PipelineTriggers` that reference the `ImagePolicy` is found,
	we just need to loop through the list and create a reconcile request for each one.
	If an error occurs fetching the list, or no `PipelineTriggers` are found, then no reconcile requests will be returned.
*/
func (r *PipelineTriggerReconciler) findObjectsForImagePolicy(imagePolicy client.Object) []reconcile.Request {
	attachedPipelineTriggers := &pipelinev1alpha1.PipelineTriggerList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(imagePolicyField, imagePolicy.GetName()),
		Namespace:     imagePolicy.GetNamespace(),
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
