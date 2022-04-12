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
	"time"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	"github.com/go-logr/logr"
	apis "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	core "k8s.io/api/core/v1"
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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"

	sourceApi "github.com/jquad-group/pipeline-trigger-operator/pkg/source"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	sourceField                 = ".spec.source.name"
	pipelineField               = ".spec.pipeline.name"
	pipelineRunField            = ".name"
	myFinalizerName             = "pipeline.jquad.rocks/finalizer"
	pipelineRunLabelLatestEvent = "pipeline.jquad.rocks/latestEvent"
	PULLREQUEST_KIND_NAME       = "PullRequest"
	IMAGEPOLICY_KIND_NAME       = "ImagePolicy"
	GITREPOSITORY_KIND_NAME     = "GitRepository"
)

// PipelineTriggerReconciler reconciles a PipelineTrigger object
type PipelineTriggerReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// +kubebuilder:docs-gen:collapse=Reconciler Declaration

/*
There are two additional resources that the controller needs to have access to, other than PipelineTriggers.
- It needs to be able to create Tekton PipelineRuns, as well as check their status.
- It also needs to be able to get, list and watch ImagePolicies and GitRepositories.
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
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;delete;patch;update
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines,verbs=get;list;watch;create;delete;patch;update
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

	// check if the referenced source exists
	sourceSubscriber := createSourceSubscriber(&pipelineTrigger)
	if err := sourceSubscriber.Exists(ctx, pipelineTrigger, r.Client, req); err != nil {
		// requeue with error (maybe the resource was deployt afterwards?)
		return r.ManageError(ctx, &pipelineTrigger, req, err)
	}

	// check if the referenced tekton pipeline exists
	if err := r.existsPipelineResource(ctx, pipelineTrigger); err != nil {
		// requeue with error (maybe the resource was deployt afterwards?)
		return r.ManageError(ctx, &pipelineTrigger, req, err)
	}

	// Get the Latest Source Event
	gotNewEvent, err := sourceSubscriber.GetLatestEvent(ctx, &pipelineTrigger, r.Client, req)
	if err != nil {
		return r.ManageError(ctx, &pipelineTrigger, req, err)
	}

	if gotNewEvent {
		// Update the status conditions
		r.Status().Update(ctx, &pipelineTrigger)
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	prs := sourceSubscriber.CreatePipelineRunResource(&pipelineTrigger)
	for pipelineRunCnt := 0; pipelineRunCnt < len(prs); pipelineRunCnt++ {
		instanceName := pipelineTrigger.StartPipelineRun(prs[pipelineRunCnt], ctx, req, r.Scheme)
		newVersionMsg := "Started the pipeline " + instanceName + " in namespace " + pipelineTrigger.Namespace
		r.recorder.Event(&pipelineTrigger, core.EventTypeNormal, "Info", newVersionMsg)
	}

	_, errList := sourceSubscriber.GetPipelineRunsByLabel(ctx, req, &pipelineTrigger)
	if errList != nil {
		log.Error(errList, "Failed to list PipelineRuns in ", pipelineTrigger.Namespace, " with label ", pipelineTrigger.Name)
	}
	if !sourceSubscriber.IsFinished(&pipelineTrigger) {
		r.Status().Update(ctx, &pipelineTrigger)
		return ctrl.Result{RequeueAfter: time.Second * 50}, nil
	}

	r.Status().Update(ctx, &pipelineTrigger)

	log.Info("test")

	return ctrl.Result{}, nil

}

func createSourceSubscriber(pipelineTrigger *pipelinev1alpha1.PipelineTrigger) sourceApi.SourceSubscriber {
	switch pipelineTrigger.Spec.Source.Kind {
	case PULLREQUEST_KIND_NAME:
		return sourceApi.NewPullrequestSubscriber()
	}
	return nil
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
		var result []string
		for _, branch := range pipelineTrigger.Status.Branches.Branches {
			result = append(result, branch.LatestPipelineRun)
		}
		if len(result) == 0 {
			return nil
		}
		return result
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
			&source.Kind{Type: &pullrequestv1alpha1.PullRequest{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSource),
			//builder.WithPredicates(SourceRevisionChangePredicate{}),
		).
		Watches(
			&source.Kind{Type: &tektondevv1.PipelineRun{}},
			/*
				&handler.EnqueueRequestForOwner{
					IsController: true,
					OwnerType:    &pipelinev1alpha1.PipelineTrigger{},
				},
			*/
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

func (r *PipelineTriggerReconciler) deleteRandomPipelineRun(ctx context.Context, pipelineRunList *tektondevv1.PipelineRunList) error {
	log := log.FromContext(ctx)
	deletePipelineRun := &pipelineRunList.Items[0]
	err := r.Delete(ctx, deletePipelineRun, client.GracePeriodSeconds(5))
	if err != nil {
		log.Error(err, "unable to delete object ", "object", &pipelineRunList.Items[0])
		return err
	}
	return nil
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

func (r *PipelineTriggerReconciler) ManageUnknownWithRequeue(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
		log.Error(err, "unable to get obj")
		return reconcile.Result{}, err
	}
	condition := metav1.Condition{
		Type:               apis.ReconcileUnknown,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileUnknown,
		Status:             metav1.ConditionUnknown,
	}
	obj.AddOrReplaceCondition(condition)
	err := r.Status().Update(context, obj)
	if err != nil {
		log.Error(err, "unable to update status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: time.Second * 50}, nil
}

func (r *PipelineTriggerReconciler) ManagePipelineRunCreatedStatus(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)

	if err := r.Get(context, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
		log.Error(err, "unable to get obj")
		return reconcile.Result{}, err
	}

	//ctrl.SetControllerReference(obj, pipelineRun, r.Scheme)

	condition := metav1.Condition{
		Type:               apis.ReconcileUnknown,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileUnknown,
		Status:             metav1.ConditionUnknown,
	}
	obj.AddOrReplaceCondition(condition)
	err := r.Status().Update(context, obj)
	if err != nil {
		log.Error(err, "unable to update status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: time.Second * 50}, nil
}

func (r *PipelineTriggerReconciler) ManageError(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request, message error) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
		log.Error(err, "unable to get obj")
		return reconcile.Result{}, err
	}

	condition := metav1.Condition{
		Type:               apis.ReconcileError,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileErrorReason,
		Status:             metav1.ConditionFalse,
		Message:            message.Error(),
	}
	obj.AddOrReplaceCondition(condition)
	err := r.Status().Update(context, obj)
	if err != nil {
		log.Error(err, "unable to update status")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

func (r *PipelineTriggerReconciler) ManagePipelineRunSucceededStatus(context context.Context, obj *pipelinev1alpha1.PipelineTrigger, req ctrl.Request) (reconcile.Result, error) {
	log := log.FromContext(context)
	if err := r.Get(context, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj); err != nil {
		log.Error(err, "unable to get obj")
		return reconcile.Result{}, err
	}

	condition := metav1.Condition{
		Type:               apis.ReconcileSuccess,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: obj.GetGeneration(),
		Reason:             apis.ReconcileSuccessReason,
		Status:             metav1.ConditionTrue,
	}

	obj.AddOrReplaceCondition(condition)
	err := r.Status().Update(context, obj)
	if err != nil {
		log.Error(err, "unable to update status")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}
