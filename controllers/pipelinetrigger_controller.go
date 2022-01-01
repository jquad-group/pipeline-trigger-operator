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
	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		//log.Error(err, "unable to fetch PipelineTrigger")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		// return and dont requeue
		return ctrl.Result{}, nil
	}

	// check if the referenced tekton pipeline exists
	if err := r.existsPipelineResource(ctx, pipelineTrigger); err != nil {
		return r.ManageErrorWithRequeue(ctx, &pipelineTrigger, err, req)
	}

	var latestEvent string
	// Check if a GitRepository or ImagePolicy Source exists
	if err := r.existsSourceResource(ctx, pipelineTrigger); err != nil {
		// requeue with error (maybe the resource was deployt afterwards?)
		r.ManageErrorWithRequeue(ctx, &pipelineTrigger, err, req)
	} else {
		// Get the Latest Source Event
		latestEvent = r.getSourceLatestEvent(ctx, pipelineTrigger)
		if latestEvent != pipelineTrigger.Status.LatestEvent {
			newVersionMsg := "Source " + pipelineTrigger.Spec.Source.Name + " in namespace " + pipelineTrigger.Namespace + " got new event " + latestEvent
			r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newVersionMsg)
			pipelineTrigger.Status.LatestPipelineRun = "Unknown"
			pipelineTrigger.Status.LatestEvent = latestEvent
			err := r.Status().Update(ctx, &pipelineTrigger)
			if err != nil {
				log.Error(err, "unable to update latest event status")
				r.ManageErrorWithRequeue(ctx, &pipelineTrigger, err, req)
			}
		}

	}

	if pipelineTrigger.Status.LatestPipelineRun == "Unknown" {
		pipelineRunList := r.getPipelineRuns(ctx, pipelineTrigger)
		if len(pipelineRunList.Items) < int(pipelineTrigger.Spec.Pipeline.MaxHistory) {
			pr, pipelineRunError := pipelineTrigger.Spec.Pipeline.CreatePipelineRun(ctx, req, pipelineTrigger)
			r.setPipelineRunCreatedStatus(ctx, &pipelineTrigger, latestEvent, pr)
			if pipelineRunError != nil {
				log.Error(pipelineRunError, "Failed to create new PipelineRun", "PipelineTrigger.Namespace", pipelineTrigger.Namespace, "PipelineTrigger.Name", pipelineTrigger.Name)
				return ctrl.Result{}, pipelineRunError
			}
			r.ManageUnknownWithRequeue(ctx, &pipelineTrigger, req)
		} else {
			// if pipelineruns > MaxHistory delete one pipelinerun, return and requeue
			r.deleteRandomPipelineRun(ctx, pipelineRunList)
			return ctrl.Result{RequeueAfter: 10}, nil
		}
	}

	// Sometimes the status needs a little bit more time to get updated, therefore we start another reconciliation
	if pipelineTrigger.Status.LatestPipelineRun == "Unknown" {
		return ctrl.Result{RequeueAfter: 10}, nil
	}

	// Finds the started PipelineRun
	foundPipelineRun := &tektondevv1.PipelineRun{}
	pipelineRunError := r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Status.LatestPipelineRun, Namespace: pipelineTrigger.Namespace}, foundPipelineRun)
	if pipelineRunError != nil {
		log.Error(pipelineRunError, "Cannot get PipelineRun resource")
	}
	// Checks wheter the PipelineRun was Successful or not
	if isPipelineRunSuccessful(foundPipelineRun) {
		pipelineTrigger.Status.CurrentPipelineRetry = 0
		r.Status().Update(ctx, &pipelineTrigger)
		return r.ManageSuccess(ctx, &pipelineTrigger, req)
	} else if isPipelineRunFailure(foundPipelineRun) {
		// checks if the PipelineRun has failed, resets the LatestPipelineRun status in order to start another reconciliation
		if pipelineTrigger.Status.CurrentPipelineRetry < pipelineTrigger.Spec.Pipeline.Retries {
			pipelineTrigger.Status.CurrentPipelineRetry++
			pipelineTrigger.Status.LatestPipelineRun = "Unknown"
			r.Status().Update(ctx, &pipelineTrigger)
			return ctrl.Result{RequeueAfter: 10}, nil
		} else {
			// if the max number of failures has reached, stop the reconciliation and set error status
			return r.ManageError(ctx, &pipelineTrigger, req)
		}
	} else {
		// Requeue if the PipelineRun is still running
		return ctrl.Result{RequeueAfter: 10}, nil
	}
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

func (r *PipelineTriggerReconciler) getPipelineRuns(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger) *tektondevv1.PipelineRunList {
	pipelineRunList := &tektondevv1.PipelineRunList{}
	pipelineRunListOpts := []client.ListOption{
		client.InNamespace(pipelineTrigger.Namespace),
		client.MatchingLabels{"pipeline.jquad.rocks/pipelinetrigger": pipelineTrigger.Name},
	}
	r.List(ctx, pipelineRunList, pipelineRunListOpts...)
	return pipelineRunList
}

func (r *PipelineTriggerReconciler) getSourceLatestEvent(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger) string {
	if pipelineTrigger.Spec.Source.Kind == "ImagePolicy" {
		foundSource := &imagereflectorv1.ImagePolicy{}
		r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		return foundSource.Status.LatestImage
	} else if pipelineTrigger.Spec.Source.Kind == "GitRepository" {
		foundSource := &sourcev1.GitRepository{}
		r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		return foundSource.Status.Artifact.Revision
	} else {
		return ""
	}
}

func (r *PipelineTriggerReconciler) existsSourceResource(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	var err error
	// Check if Source is from Kind ImagePolicy or GitRepository
	if pipelineTrigger.Spec.Source.Kind == "ImagePolicy" {
		foundSource := &imagereflectorv1.ImagePolicy{}
		err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else if pipelineTrigger.Spec.Source.Kind == "GitRepository" {
		foundSource := &sourcev1.GitRepository{}
		err = r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, foundSource)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		return err
	}

}

func (r *PipelineTriggerReconciler) existsPipelineResource(ctx context.Context, pipelineTrigger pipelinev1alpha1.PipelineTrigger) error {
	// check if the referenced tekton pipeline exists
	foundTektonPipeline := &tektondevv1.Pipeline{}
	err := r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Pipeline.Name, Namespace: pipelineTrigger.Namespace}, foundTektonPipeline)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (r *PipelineTriggerReconciler) resetRetriesStatus(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger) {
	pipelineTrigger.Status.CurrentPipelineRetry = 0
	r.Status().Update(ctx, pipelineTrigger)
}

func (r *PipelineTriggerReconciler) setNewRetryStatus(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, pipelineRun *tektondevv1.PipelineRun) {
	pipelineTrigger.Status.CurrentPipelineRetry++
	pipelineTrigger.Status.PipelineStatus = "Unknown"
	pipelineTrigger.Status.LatestPipelineRun = pipelineRun.Name
	r.Status().Update(ctx, pipelineTrigger)
	ctrl.SetControllerReference(pipelineTrigger, pipelineRun, r.Scheme)
}

func (r *PipelineTriggerReconciler) setPipelineRunCreatedStatus(ctx context.Context, pipelineTrigger *pipelinev1alpha1.PipelineTrigger, latestEvent string, pipelineRun *tektondevv1.PipelineRun) {
	log := log.FromContext(ctx)
	r.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Spec.Source.Name, Namespace: pipelineTrigger.Namespace}, pipelineTrigger)
	pipelineTrigger.Status.LatestPipelineRun = pipelineRun.Name
	pipelineTrigger.Status.LatestEvent = latestEvent
	err := r.Status().Update(ctx, pipelineTrigger)
	if err != nil {
		log.Error(err, "unable to update status after the creation of the pipeline run")
	}
	ctrl.SetControllerReference(pipelineTrigger, pipelineRun, r.Scheme)
}

func (r *PipelineTriggerReconciler) deleteRandomPipelineRun(ctx context.Context, pipelineRunList *tektondevv1.PipelineRunList) error {
	log := log.FromContext(ctx)
	deletePipelineRun := &pipelineRunList.Items[0]
	err := r.Delete(ctx, deletePipelineRun, client.GracePeriodSeconds(5))
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "unable to delete object ", "object", &pipelineRunList.Items[0])
		return err
	}
	return nil
}

func (r *PipelineTriggerReconciler) ManageUnknownWithRequeue(context context.Context, obj client.Object, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to get obj (ManageUnknownWithRequeue)")
		return reconcile.Result{}, err
	}
	if conditionsAware, updateStatus := (obj).(apis.ConditionsAware); updateStatus {
		condition := metav1.Condition{
			Type:               apis.ReconcileUnknown,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Reason:             apis.ReconcileUnknown,
			Status:             metav1.ConditionUnknown,
		}
		conditionsAware.SetConditions(apis.ReplaceCondition(condition, conditionsAware.GetConditions()))
		err := r.Status().Update(context, obj)
		if err != nil {
			log.Error(err, "unable to update status (ManageUnknownWithRequeue)")
			return reconcile.Result{}, err
		}
	} else {
		log.V(1).Info("object is not ConditionsAware, not setting status")
	}
	return reconcile.Result{RequeueAfter: 10}, nil
}

func (r *PipelineTriggerReconciler) ManageSuccess(context context.Context, obj client.Object, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to get obj (ManageSuccess)")
		return reconcile.Result{}, err
	}
	if conditionsAware, updateStatus := (obj).(apis.ConditionsAware); updateStatus {
		condition := metav1.Condition{
			Type:               apis.ReconcileSuccess,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Reason:             apis.ReconcileSuccessReason,
			Status:             metav1.ConditionTrue,
		}
		conditionsAware.SetConditions(apis.ReplaceCondition(condition, conditionsAware.GetConditions()))
		err := r.Status().Update(context, obj)
		if err != nil {
			log.Error(err, "unable to update status (ManageSuccess)")
			return reconcile.Result{}, err
		}
	} else {
		log.V(1).Info("object is not ConditionsAware, not setting status")
	}
	return reconcile.Result{}, nil
}

func (r *PipelineTriggerReconciler) ManageError(context context.Context, obj client.Object, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to get obj (ManageError)")
		return reconcile.Result{}, err
	}
	if conditionsAware, updateStatus := (obj).(apis.ConditionsAware); updateStatus {
		condition := metav1.Condition{
			Type:               apis.ReconcileError,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Reason:             apis.ReconcileErrorReason,
			Status:             metav1.ConditionFalse,
		}
		conditionsAware.SetConditions(apis.ReplaceCondition(condition, conditionsAware.GetConditions()))
		err := r.Status().Update(context, obj)
		if err != nil {
			log.Error(err, "unable to update status (ManageError)")
			return reconcile.Result{}, err
		}
	} else {
		log.V(1).Info("object is not ConditionsAware, not setting status")
	}
	return reconcile.Result{}, nil
}

func (r *PipelineTriggerReconciler) ManageSuccessWithRequeue(context context.Context, obj client.Object, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to get obj (ManageSuccessWithRequeue)")
		return reconcile.Result{}, err
	}
	if conditionsAware, updateStatus := (obj).(apis.ConditionsAware); updateStatus {
		condition := metav1.Condition{
			Type:               apis.ReconcileSuccess,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Reason:             apis.ReconcileSuccessReason,
			Status:             metav1.ConditionTrue,
		}
		conditionsAware.SetConditions(apis.ReplaceCondition(condition, conditionsAware.GetConditions()))
		err := r.Status().Update(context, obj)
		if err != nil {
			log.Error(err, "unable to update status (ManageSuccessWithRequeue)")
			return reconcile.Result{RequeueAfter: 10}, err
		}
	} else {
		log.V(1).Info("object is not ConditionsAware, not setting status")
	}
	return reconcile.Result{RequeueAfter: 10}, nil
}

//ManageErrorWithRequeue will take care of the following:
// 1. generate a warning event attached to the passed CR
// 2. set the status of the passed CR to a error condition if the object implements the apis.ConditionsStatusAware interface
// 3. return a reconcile status with with the passed requeueAfter and error
func (r *PipelineTriggerReconciler) ManageErrorWithRequeue(context context.Context, obj client.Object, issue error, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to get obj (ManageErrorWithRequeue)")
		return reconcile.Result{}, err
	}
	r.recorder.Event(obj, "Warning", "ProcessingError", issue.Error())
	if conditionsAware, updateStatus := (obj).(apis.ConditionsAware); updateStatus {
		condition := metav1.Condition{
			Type:               apis.ReconcileError,
			LastTransitionTime: metav1.Now(),
			ObservedGeneration: obj.GetGeneration(),
			Message:            issue.Error(),
			Reason:             apis.ReconcileErrorReason,
			Status:             metav1.ConditionTrue,
		}
		conditionsAware.SetConditions(apis.ReplaceCondition(condition, conditionsAware.GetConditions()))
		err := r.Status().Update(context, obj)
		if err != nil {
			log.Error(err, "unable to update status (ManageErrorWithRequeue)")
			return reconcile.Result{Requeue: true}, err
		}
	} else {
		log.V(1).Info("object is not ConditionsAware, not setting status")
	}
	return reconcile.Result{RequeueAfter: 10}, issue
}

func isPipelineRunSuccessful(pipelineRun *tektondevv1.PipelineRun) bool {
	for _, c := range pipelineRun.Status.Conditions {
		if (tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonCompleted || tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonSuccessful) && metav1.ConditionStatus(c.Status) == metav1.ConditionTrue {
			return true
		}
	}

	return false
}

func isPipelineRunFailure(pipelineRun *tektondevv1.PipelineRun) bool {
	for _, c := range pipelineRun.Status.Conditions {
		if tektondevv1.PipelineRunReason(c.GetReason()) == tektondevv1.PipelineRunReasonFailed && metav1.ConditionStatus(c.Status) == metav1.ConditionFalse {
			return true
		}
	}

	return false
}
