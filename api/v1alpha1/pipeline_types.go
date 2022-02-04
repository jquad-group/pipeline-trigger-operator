package v1alpha1

import (
	"context"
	"strings"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/meta"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Pipeline struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	SericeAccountName string `json:"serviceAccountName"`

	// +kubebuilder:validation:Required
	InputParams []InputParam `json:"inputParams"`

	// +kubebuilder:validation:Required
	Workspace Workspace `json:"workspace"`

	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxFailedRetries int64 `json:"maxFailedRetries"`

	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Required
	MaxHistory int64 `json:"maxHistory"`
}

func (pipeline Pipeline) CreatePipelineRef() *tektondevv1.PipelineRef {
	return &tektondevv1.PipelineRef{
		Name: pipeline.Name,
	}
}

func (pipeline Pipeline) CreateParams(pipelineTrigger PipelineTrigger, latestEvent string) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipeline.InputParams); paramNr++ {
		pipelineParams = append(pipelineParams, pipeline.InputParams[paramNr].CreateParam(pipelineTrigger, latestEvent))
	}
	return pipelineParams
}

func (pipeline Pipeline) CreatePipelineRun(ctx context.Context, req ctrl.Request, pipelineTrigger PipelineTrigger, latestEvent string) (*tektondevv1.PipelineRun, error) {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			GenerateName: pipelineTrigger.Name + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       setLabel(pipelineTrigger.Name + "-" + strings.ReplaceAll(latestEvent, "/", "-")),
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipeline.SericeAccountName,
			PipelineRef:        pipeline.CreatePipelineRef(),
			Params:             pipeline.CreateParams(pipelineTrigger, latestEvent),
			Workspaces: []tektondevv1.WorkspaceBinding{
				pipeline.Workspace.CreateWorkspaceBinding(),
			},
		},
	}

	opts := v1.CreateOptions{}

	return tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).Create(ctx, pr, opts)

}

func (pipeline Pipeline) GetSuccessfulPipelineRuns(ctx context.Context, req ctrl.Request, pipelineTrigger PipelineTrigger) (*tektondevv1.PipelineRunList, error) {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	// the status is empty during the first reconciliation
	if len(pipelineTrigger.Status.Conditions) > 0 {
		opts := v1.ListOptions{LabelSelector: "pipeline.jquad.rocks/pipelinetrigger=" + pipelineTrigger.Name + "-" + strings.ReplaceAll(pipelineTrigger.Status.LatestEvent, "/", "-")}
		pipelineRunList, err := tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).List(ctx, opts)

		var successfulRuns []tektondevv1.PipelineRun
		for i := range pipelineRunList.Items {
			item := pipelineRunList.Items[i]

			if len(item.Status.Conditions) > 0 {
				if string(item.Status.Conditions[0].Reason) == "Succeeded" {
					successfulRuns = append(successfulRuns, item)
				}
			}
		}

		var successfulRunsList tektondevv1.PipelineRunList
		successfulRunsList.Items = successfulRuns
		return &successfulRunsList, err
	} else {
		var successfulRunsList tektondevv1.PipelineRunList
		return &successfulRunsList, err
	}
}

func (pipeline Pipeline) GetFailedPipelineRuns(ctx context.Context, req ctrl.Request, pipelineTrigger PipelineTrigger) (*tektondevv1.PipelineRunList, error) {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	// the status is empty during the first reconciliation
	if len(pipelineTrigger.Status.Conditions) > 0 {
		opts := v1.ListOptions{LabelSelector: "pipeline.jquad.rocks/pipelinetrigger=" + pipelineTrigger.Name + "-" + strings.ReplaceAll(pipelineTrigger.Status.LatestEvent, "/", "-")}

		pipelineRunList, err := tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).List(ctx, opts)

		var failedRuns []tektondevv1.PipelineRun
		for i := range pipelineRunList.Items {
			item := pipelineRunList.Items[i]

			if len(item.Status.Conditions) > 0 {
				if string(item.Status.Conditions[0].Reason) == "Failed" {
					failedRuns = append(failedRuns, item)
				}
			}
		}

		var failedRunsList tektondevv1.PipelineRunList
		failedRunsList.Items = failedRuns
		return &failedRunsList, err
	} else {
		var failedRunsList tektondevv1.PipelineRunList
		return &failedRunsList, err
	}

}

func setLabel(name string) map[string]string {
	return map[string]string{"pipeline.jquad.rocks/pipelinetrigger": name}
}
