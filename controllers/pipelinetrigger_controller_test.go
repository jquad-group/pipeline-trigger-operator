package controllers

import (
	"context"
	"time"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("PipelineTrigger controller", func() {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = imagereflectorv1.AddToScheme(scheme)
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		PipelineTriggerName      = "test-cronjob"
		PipelineTriggerNamespace = "default"
		PipelineRunName          = "test-job"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating PipelineTrigger resource", func() {
		It("Should create PipelineRun", func() {
			By("By creating a new PipelineTrigger")
			ctx := context.Background()
			imagePolicy := &imagereflectorv1.ImagePolicy{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "image.toolkit.fluxcd.io/v1beta1",
					Kind:       "ImagePolicy",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app1",
					Namespace: PipelineTriggerNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, imagePolicy)).Should(Succeed())
			pipelineTrigger := &pipelinev1alpha1.PipelineTrigger{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "pipeline.jquad.rocks/v1alpha1",
					Kind:       "PipelineTrigger",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      PipelineTriggerName,
					Namespace: PipelineTriggerNamespace,
				},
				Spec: pipelinev1alpha1.PipelineTriggerSpec{
					Source: pipelinev1alpha1.Source{
						Kind: "ImagePolicy",
						Name: "app1",
					},
					Pipeline: pipelinev1alpha1.Pipeline{
						Name:              "build-and-push",
						Retries:           1,
						MaxHistory:        1,
						SericeAccountName: "default",
						Workspace: pipelinev1alpha1.Workspace{
							Name:       "workspace",
							Size:       "1Gi",
							AccessMode: "ReadWriteOnce",
						},
						InputParams: []pipelinev1alpha1.InputParam{
							{Name: "testparam", Value: "testvalue"},
							{Name: "testparam2", Value: "testvalue2"},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pipelineTrigger)).Should(Succeed())
		})
	})
})
