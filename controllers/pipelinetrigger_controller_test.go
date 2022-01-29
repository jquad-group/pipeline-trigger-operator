package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("PipelineTrigger controller", func() {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = imagereflectorv1.AddToScheme(scheme)
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		PipelineTriggerName      = "test-pipelinetrigger"
		PipelineTriggerNamespace = "default"
		PipelineRunName          = "test-pipelinerun"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating PipelineTrigger resource", func() {
		It("Should create PipelineRun", func() {
			//By("By creating a new PipelineTrigger")
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
				Status: imagereflectorv1.ImagePolicyStatus{
					LatestImage: "rannox/testimage:0.0.1",
				},
			}
			Expect(k8sClient.Create(ctx, imagePolicy)).Should(Succeed())
			pipeline := &tektondevv1.Pipeline{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "tekton.dev/v1beta1",
					Kind:       "Pipeline",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "build-and-push-base-image",
					Namespace: PipelineTriggerNamespace,
				},
				Spec: tektondevv1.PipelineSpec{
					Params: []tektondevv1.ParamSpec{
						{Name: "testparam", Type: tektondevv1.ParamTypeString},
						{Name: "testparam2", Type: tektondevv1.ParamTypeString},
					},
				},
			}
			Expect(k8sClient.Create(ctx, pipeline)).Should(Succeed())
			/*
				pipelineRun := &tektondevv1.PipelineRun{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "tekton.dev/v1beta1",
						Kind:       "PipelineRun",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pr",
						Namespace: PipelineTriggerNamespace,
					},
					Spec: tektondevv1.PipelineRunSpec{
						PipelineRef: &tektondevv1.PipelineRef{
							Name: "build-and-push-base-image",
						},
						Params: []tektondevv1.Param{
							{Name: "testparam", Value: tektondevv1.ArrayOrString{
								Type:      tektondevv1.ParamTypeString,
								StringVal: "param",
							},
							},
							{Name: "testparam2", Value: tektondevv1.ArrayOrString{
								Type:      tektondevv1.ParamTypeString,
								StringVal: "param2",
							},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, pipelineRun)).Should(Succeed())

				pipelineRunList := &tektondevv1.PipelineRunList{}
				pipelineRunListOpts := []client.ListOption{
					client.InNamespace(PipelineTriggerNamespace),
					//client.MatchingLabels{"pipeline.jquad.rocks/pipelinetrigger": pipelineTrigger.Name},
				}
				k8sClient.List(ctx, pipelineRunList, pipelineRunListOpts...)
				Expect(IsPipelineRunCreated(pipelineRunList)).Should(Equal(1))
			*/

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
						Name:              "build-and-push-base-image",
						Retries:           1,
						MaxHistory:        3,
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
			imagePolicy.Status.LatestImage = "rannox/testimage:0.0.1"
			Expect(k8sClient.Status().Update(ctx, imagePolicy)).Should(Succeed())

			//pipelineTrigger.Status.LatestEvent = "rannox/testimage:0.0.1"
			//pipelineTrigger.Status.LatestPipelineRun = "blabla"
			//Expect(k8sClient.Status().Update(ctx, pipelineTrigger)).Should(Succeed())
			fmt.Println("LatestEvent: " + pipelineTrigger.Status.LatestEvent)
			fmt.Println("LatestPipelineRun: " + pipelineTrigger.Status.LatestPipelineRun)

			imagePolicyList := &imagereflectorv1.ImagePolicyList{}
			k8sClient.List(ctx, imagePolicyList)
			for i := 0; i < len(imagePolicyList.Items); i++ {
				fmt.Println("item: " + imagePolicyList.Items[i].Status.LatestImage)
			}

			/*
				pipelineRunListOpts := []client.ListOption{
					client.InNamespace(PipelineTriggerNamespace),
					//client.MatchingLabels{"pipeline.jquad.rocks/pipelinetrigger": pipelineTrigger.Name},
				}

				k8sClient.List(ctx, pipelineRunList, pipelineRunListOpts...)
			*/

			/*
				Eventually(func() bool {
					err := k8sClient.List(ctx, pipelineRunList)
					if err != nil {
						return false
					}
					return true
				}, timeout, interval).Should(BeTrue())
			*/

			Eventually(func() bool {
				fetched := &tektondevv1.PipelineRun{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: pipelineTrigger.Status.LatestPipelineRun, Namespace: "default"}, fetched)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			//Eventually(HasPipelineRunCreated(), timeout, interval).Should(BeTrue())
			//k8sClient.List(ctx, pipelineRunList)
			//Expect(IsPipelineRunCreated(pipelineRunList)).Should(Equal(1))

		})
	})
})

func HasPipelineRunCreated() bool {
	pipelineRunList := &tektondevv1.PipelineRunList{}
	pipelineRunListOpts := []client.ListOption{
		client.InNamespace("default"),
		client.MatchingLabels{"pipeline.jquad.rocks/pipelinetrigger": "test-pipelinetrigger"},
	}

	//k8sClient.List(ctx, pipelineRunList)
	k8sClient.List(ctx, pipelineRunList, pipelineRunListOpts...)
	if len(pipelineRunList.Items) == 1 {
		return true
	} else {
		return false
	}
}

func IsPipelineRunCreated(pipelineRunList *tektondevv1.PipelineRunList) (int, error) {
	if len(pipelineRunList.Items) == 1 {
		return 1, nil
	} else {
		return -1, errors.New("PipelineRun not created")
	}
}
