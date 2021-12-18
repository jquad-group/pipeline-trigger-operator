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

	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	meta "github.com/jquad-group/pipeline-trigger-operator/pkg/meta"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

const (
	imagePolicyField = ".spec.imagePolicy"
)

// PipelineTriggerReconciler reconciles a PipelineTrigger object
type PipelineTriggerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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

	log := logr.FromContext(ctx)

	var pipelineTrigger pipelinev1alpha1.PipelineTrigger
	if err := r.Get(ctx, req.NamespacedName, &pipelineTrigger); err != nil {
		log.Error(err, "unable to fetch PipelineTrigger")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//var imagePolicyVersion string
	if pipelineTrigger.Spec.ImagePolicy != "" {
		ImagePolicy := pipelineTrigger.Spec.ImagePolicy
		foundImagePolicy := &imagereflectorv1.ImagePolicy{}
		err := r.Get(ctx, types.NamespacedName{Name: ImagePolicy, Namespace: pipelineTrigger.Namespace}, foundImagePolicy)
		if err != nil {
			// If a imagePolicy name is provided, then it must exist
			// You will likely want to create an Event for the user to understand why their reconcile is failing.
			return ctrl.Result{}, err
		}

		// Hash the data in some way, or just use the version of the Object
		pipelineTrigger.Status.LatestImage = foundImagePolicy.Status.LatestImage
		log.Info("Found pipelinetrigger", "resource", pipelineTrigger.Status.LatestImage)

	}

	if pipelineTrigger.Spec.Pipeline != "" {
		Pipeline := pipelineTrigger.Spec.Pipeline
		foundTektonPipeline := &tektondevv1.Pipeline{}
		err := r.Get(ctx, types.NamespacedName{Name: Pipeline, Namespace: pipelineTrigger.Namespace}, foundTektonPipeline)
		if err != nil {
			// If a pipeline name is provided, then it must exist
			// You will likely want to create an Event for the user to understand why their reconcile is failing.
			return ctrl.Result{}, err
		}

		cfg := ctrl.GetConfigOrDie()

		tektonClient, err := clientsetversioned.NewForConfig(cfg)

		if err != nil {
			log.Info("Error building Serving clientset:", err)
		} else {
			log.Info("Successful initialized tekton client")
		}
		listOps_pipelines := v1.ListOptions{}

		pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
		pr := &tektondevv1.PipelineRun{
			TypeMeta:   pipelineRunTypeMeta,
			ObjectMeta: meta.ObjectMeta(meta.NamespacedName("", "ci-dryrun-from-push-pipeline-$(uid)")),
			Spec: tektondevv1.PipelineRunSpec{
				ServiceAccountName: "default",
				PipelineRef:        createPipelineRef("build-and-push-base-image"),
			},
		}

		opts := v1.CreateOptions{}

		tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).Create(ctx, pr, opts)

		tektonClient.TektonV1beta1().Pipelines(pipelineTrigger.GetNamespace()).List(ctx, listOps_pipelines)

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
		Watches(
			&source.Kind{Type: &imagereflectorv1.ImagePolicy{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForImagePolicy),
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

// getPIpelineNames returns the pipeline names of the array of pipelines passed in
func getPipelineNames(pipelines []tektondevv1.Pipeline) []string {
	var pipelineNames []string
	for _, pipeline := range pipelines {
		pipelineNames = append(pipelineNames, pipeline.Name)
	}
	return pipelineNames
}

func createPipelineRef(name string) *tektondevv1.PipelineRef {
	return &tektondevv1.PipelineRef{
		Name: name,
	}
}

/*
func createDevCDPipelineRun(saName string) tektondevv1.PipelineRun {
	return tektondevv1.PipelineRun{
		TypeMeta:   pipelineRunTypeMeta,
		ObjectMeta: meta.ObjectMeta(meta.NamespacedName("", "app-cd-pipeline-run-$(uid)")),
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: saName,
			PipelineRef:        createPipelineRef("app-cd-pipeline"),
			Resources:          createDevResource("$(params." + GitCommitID + ")"),
		},
	}
}
*/

/*
func createCIPipelineRun(saName string) tektondevv1.PipelineRun {
	return tektondevv1.PipelineRun{
		TypeMeta:   pipelineRunTypeMeta,
		ObjectMeta: meta.ObjectMeta(meta.NamespacedName("", "ci-dryrun-from-push-pipeline-$(uid)"), statusTrackerAnnotations("ci-dryrun-from-push-pipeline", "CI dry run on push event")),
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: saName,
			PipelineRef:        createPipelineRef("ci-dryrun-from-push-pipeline"),
			Resources:          createResources(),
		},
	}

}

func createResourceParams(name string, value string) tektondevv1.ResourceParam {
	return tektondevv1.ResourceParam{
		Name:  name,
		Value: value,
	}

}
func createPipelineRef(name string) *tektondevv1.PipelineRef {
	return &tektondevv1.PipelineRef{
		Name: name,
	}
}

func createPipelineBindingParam(name string, value string) tektondevv1.Param {
	return tektondevv1.Param{
		Name: name,
		Value: tektondevv1.ArrayOrString{
			StringVal: value,
			Type:      tektondevv1.ParamTypeString,
		},
	}
}
*/
