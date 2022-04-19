package v1alpha1

import (
	"context"

	"fmt"

	"github.com/jquad-group/pipeline-trigger-operator/pkg/meta"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientsetversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	// +kubebuilder:validation:Optional
	SecurityContext SecurityContext `json:"securityContext"`
}

func (pipeline *Pipeline) createPipelineRef() *tektondevv1.PipelineRef {
	return &tektondevv1.PipelineRef{
		Name: pipeline.Name,
	}
}

func (pipeline *Pipeline) createParams(currentBranch Branch) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipeline.InputParams); paramNr++ {
		pipelineParams = append(pipelineParams, pipeline.InputParams[paramNr].CreateParam(currentBranch))
	}
	return pipelineParams
}

func (pipeline *Pipeline) createParamsGitRepository(gitRepository GitRepository) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipeline.InputParams); paramNr++ {
		pipelineParams = append(pipelineParams, pipeline.InputParams[paramNr].CreateParamGitRepository(gitRepository))
	}
	return pipelineParams
}

func (pipeline *Pipeline) createParamsImagePolicy(imagePolicy ImagePolicy) []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipeline.InputParams); paramNr++ {
		pipelineParams = append(pipelineParams, pipeline.InputParams[paramNr].CreateParamImage(imagePolicy))
	}
	return pipelineParams
}

func (pipeline *Pipeline) CreatePipelineRunResourceForBranch(pipelineTrigger PipelineTrigger, currentBranch Branch, labels map[string]string) *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			GenerateName: currentBranch.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       labels,
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipelineTrigger.Spec.Pipeline.SericeAccountName,
			PipelineRef:        pipeline.createPipelineRef(),
			Params:             pipeline.createParams(currentBranch),
			Workspaces: []tektondevv1.WorkspaceBinding{
				pipelineTrigger.Spec.Pipeline.Workspace.CreateWorkspaceBinding(),
			},
			PodTemplate: &pod.Template{
				SecurityContext: pipelineTrigger.Spec.Pipeline.SecurityContext.CreatePodSecurityContext(),
			},
		},
	}
	return pr
}

func (pipeline *Pipeline) CreatePipelineRunResourceForGit(pipelineTrigger PipelineTrigger) *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			GenerateName: pipelineTrigger.Status.GitRepository.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       pipelineTrigger.Status.GitRepository.GenerateGitRepositoryLabelsAsHash(),
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipelineTrigger.Spec.Pipeline.SericeAccountName,
			PipelineRef:        pipeline.createPipelineRef(),
			Params:             pipeline.createParamsGitRepository(pipelineTrigger.Status.GitRepository),
			Workspaces: []tektondevv1.WorkspaceBinding{
				pipelineTrigger.Spec.Pipeline.Workspace.CreateWorkspaceBinding(),
			},
			PodTemplate: &pod.Template{
				SecurityContext: pipelineTrigger.Spec.Pipeline.SecurityContext.CreatePodSecurityContext(),
			},
		},
	}
	return pr
}

func (pipeline *Pipeline) CreatePipelineRunResourceForImage(pipelineTrigger PipelineTrigger) *tektondevv1.PipelineRun {
	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		ObjectMeta: v1.ObjectMeta{
			GenerateName: pipelineTrigger.Status.ImagePolicy.Rewrite() + "-",
			Namespace:    pipelineTrigger.Namespace,
			Labels:       pipelineTrigger.Status.ImagePolicy.GenerateImagePolicyLabelsAsHash(),
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipelineTrigger.Spec.Pipeline.SericeAccountName,
			PipelineRef:        pipeline.createPipelineRef(),
			Params:             pipeline.createParamsImagePolicy(pipelineTrigger.Status.ImagePolicy),
			Workspaces: []tektondevv1.WorkspaceBinding{
				pipelineTrigger.Spec.Pipeline.Workspace.CreateWorkspaceBinding(),
			},
			PodTemplate: &pod.Template{
				SecurityContext: pipelineTrigger.Spec.Pipeline.SecurityContext.CreatePodSecurityContext(),
			},
		},
	}
	return pr
}

func (pipelineTrigger *PipelineTrigger) StartPipelineRun(pr *tektondevv1.PipelineRun, ctx context.Context, req ctrl.Request, r *runtime.Scheme) string {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	opts := v1.CreateOptions{}
	prInstance, err := tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).Create(ctx, pr, opts)
	if err != nil {
		fmt.Println(err)
		log.Info("Cannot create tekton pipelinerun")
	}

	ctrl.SetControllerReference(pipelineTrigger, prInstance, r)

	return prInstance.Name
}
