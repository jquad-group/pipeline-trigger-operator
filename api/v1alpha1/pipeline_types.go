package v1alpha1

import (
	"context"
	"math/rand"
	"time"

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
	Retries int64 `json:"retries"`

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

func (pipeline Pipeline) CreateParams() []tektondevv1.Param {

	var pipelineParams []tektondevv1.Param
	for paramNr := 0; paramNr < len(pipeline.InputParams); paramNr++ {
		pipelineParams = append(pipelineParams, pipeline.InputParams[paramNr].CreateParam())
	}
	return pipelineParams
}

func (pipeline Pipeline) CreatePipelineRun(ctx context.Context, req ctrl.Request, pipelineTrigger PipelineTrigger) (*tektondevv1.PipelineRun, error) {
	log := log.FromContext(ctx)

	cfg := ctrl.GetConfigOrDie()

	tektonClient, err := clientsetversioned.NewForConfig(cfg)

	if err != nil {
		log.Info("Cannot create tekton client.")
	}

	pipelineRunTypeMeta := meta.TypeMeta("PipelineRun", "tekton.dev/v1beta1")
	pipelineRunName := pipelineTrigger.Name + "-" + generateRandomString(4, "abcdefghijklmnopqrstuvwxyz")
	pr := &tektondevv1.PipelineRun{
		TypeMeta: pipelineRunTypeMeta,
		//		ObjectMeta: meta.ObjectMeta(meta.NamespacedName(pipelineTrigger.Namespace, pipelineRunName)),
		ObjectMeta: v1.ObjectMeta{
			Name:      pipelineRunName,
			Namespace: pipelineTrigger.Namespace,
			Labels:    setLabel(pipelineTrigger.Name),
		},
		Spec: tektondevv1.PipelineRunSpec{
			ServiceAccountName: pipeline.SericeAccountName,
			PipelineRef:        pipeline.CreatePipelineRef(),
			Params:             pipeline.CreateParams(),
			Workspaces: []tektondevv1.WorkspaceBinding{
				pipeline.Workspace.CreateWorkspaceBinding(),
			},
		},
	}

	opts := v1.CreateOptions{}

	return tektonClient.TektonV1beta1().PipelineRuns(pipelineTrigger.Namespace).Create(ctx, pr, opts)

}

func generateRandomString(length int, charset string) string {
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func setLabel(name string) map[string]string {
	return map[string]string{"pipeline.jquad.rocks/pipelinetrigger": name}
}
