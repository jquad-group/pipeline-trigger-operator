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
	"fmt"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("PipelineTrigger controller", func() {

	const (
		gitRepositoryName   = "git-repo-1"
		pipelineTriggerName = "pipeline-trigger-1"
		taskName            = "build"
		pipelineName        = "build-and-push"
		namespace           = "default"

		timeout  = time.Second * 50000
		duration = time.Second * 10000
		interval = time.Millisecond * 250
	)

	Context("PipelineTrigger fails to create a PipelineRue to missing GitRepository", func() {
		It("Should not be able to create a PipelineRun", func() {
			By("Creating a Task custom resource")
			ctx := context.Background()
			taskMock := tektondevv1.Task{
				ObjectMeta: v1.ObjectMeta{
					Name:      taskName,
					Namespace: namespace,
				},
				Spec: tektondevv1.TaskSpec{
					Params: []tektondevv1.ParamSpec{
						{
							Name: taskName,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, &taskMock)).Should(Succeed())

			taskLookupKey := types.NamespacedName{Name: taskName, Namespace: namespace}
			createdTask := &tektondevv1.Task{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, taskLookupKey, createdTask)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a Pipeline, referencing a single Task")

			pipelineMock := tektondevv1.Pipeline{
				ObjectMeta: v1.ObjectMeta{
					Name:      pipelineName,
					Namespace: namespace,
				},
				Spec: tektondevv1.PipelineSpec{
					Tasks: []tektondevv1.PipelineTask{
						{
							Name: taskName,
							TaskRef: &tektondevv1.TaskRef{
								Name: taskName,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &pipelineMock)).Should(Succeed())

			pipelineLookupKey := types.NamespacedName{Name: pipelineName, Namespace: namespace}
			createdPipeline := &tektondevv1.Pipeline{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineLookupKey, createdPipeline)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a PipelineTrigger, referencing an existing Pipeline and not existing GitRepository")

			pipelineTrigger := pipelinev1alpha1.PipelineTrigger{
				ObjectMeta: v1.ObjectMeta{
					Name:      pipelineTriggerName,
					Namespace: namespace,
				},
				Spec: pipelinev1alpha1.PipelineTriggerSpec{
					Source: pipelinev1alpha1.Source{
						Kind: "GitRepository",
						Name: gitRepositoryName,
					},
					PipelineRunSpec: tektondevv1.PipelineRunSpec{
						PipelineRef: &tektondevv1.PipelineRef{
							Name: pipelineName,
						},
						Params: []tektondevv1.Param{
							{
								Name: "test",
								Value: tektondevv1.ArrayOrString{
									Type:      tektondevv1.ParamTypeString,
									StringVal: "test",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &pipelineTrigger)).Should(Succeed())

			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName, Namespace: namespace}
			createdPipelineTrigger := &pipelinev1alpha1.PipelineTrigger{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("The Status of the PipelineTrigger should be not found")

			Eventually(func() string {

				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)

				if err != nil {
					return ""
				}

				if len(createdPipelineTrigger.Status.GitRepository.Conditions) == 0 {
					return ""
				}
				fmt.Println(createdPipelineTrigger.Status.GitRepository.Conditions[0].Message)
				return createdPipelineTrigger.Status.GitRepository.Conditions[0].Message
			}, timeout, interval).Should(ContainSubstring("not found"))

			By("Delete the PipelineTrigger, Task, Pipeline, GitRepository")
			Expect(k8sClient.Delete(ctx, createdPipelineTrigger)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, createdPipeline)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, createdTask)).Should(Succeed())

		})
	})

	Context("PipelineTrigger creates a PipelineRun on the test cluster", func() {

		It("Should be able to create a PipelineRun custom resources", func() {

			By("Creating a GitRepository")
			ctx := context.Background()
			gitRepository := sourcev1.GitRepository{
				ObjectMeta: v1.ObjectMeta{
					Name:      gitRepositoryName,
					Namespace: namespace,
				},
				Spec: sourcev1.GitRepositorySpec{
					URL:      "http://github.com/org/repo.git",
					Interval: v1.Duration{},
				},
			}
			Expect(k8sClient.Create(ctx, &gitRepository)).Should(Succeed())

			gitRepositoryLookupKey := types.NamespacedName{Name: gitRepositoryName, Namespace: namespace}
			createdGitRepo := &sourcev1.GitRepository{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, gitRepositoryLookupKey, createdGitRepo)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the GitRepository status")

			gitRepoStatus := sourcev1.GitRepositoryStatus{
				Artifact: &sourcev1.Artifact{
					Checksum:       "cb0053b034ac7e74e2278b94b69db15871e9b3b40124adde8c585c1bdda48b25",
					Path:           "gitrepository/flux-system/flux-system/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69.tar.gz",
					URL:            "http://source-controller.flux-system.svc.cluster.local./gitrepository/flux-system/flux-system/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69.tar.gz",
					Revision:       "main/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69",
					LastUpdateTime: v1.Now(),
				},
				Conditions: []v1.Condition{
					{
						Type:               "Ready",
						Status:             v1.ConditionTrue,
						Reason:             v1.StatusSuccess,
						Message:            "Success",
						ObservedGeneration: 12,
						LastTransitionTime: v1.Now(),
					},
				},
			}

			k8sClient.Get(ctx, gitRepositoryLookupKey, createdGitRepo)
			createdGitRepo.Status = gitRepoStatus
			Expect(k8sClient.Status().Update(ctx, createdGitRepo)).Should(Succeed())

			By("Checking if the GitRepository artifact revision was updated")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, gitRepositoryLookupKey, createdGitRepo)
				if err != nil {
					return "", err
				}
				return createdGitRepo.Status.Artifact.Revision, nil
			}, duration, interval).Should(Equal("main/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69"))

			By("Creating a Tekton Task")

			taskMock := tektondevv1.Task{
				ObjectMeta: v1.ObjectMeta{
					Name:      taskName,
					Namespace: namespace,
				},
				Spec: tektondevv1.TaskSpec{
					Params: []tektondevv1.ParamSpec{
						{
							Name: taskName,
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, &taskMock)).Should(Succeed())

			taskLookupKey := types.NamespacedName{Name: taskName, Namespace: namespace}
			createdTask := &tektondevv1.Task{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, taskLookupKey, createdTask)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a Tekton Pipeline, referencing a single Task")

			pipelineMock := tektondevv1.Pipeline{
				ObjectMeta: v1.ObjectMeta{
					Name:      pipelineName,
					Namespace: namespace,
				},
				Spec: tektondevv1.PipelineSpec{
					Tasks: []tektondevv1.PipelineTask{
						{
							Name: taskName,
							TaskRef: &tektondevv1.TaskRef{
								Name: taskName,
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &pipelineMock)).Should(Succeed())

			pipelineLookupKey := types.NamespacedName{Name: pipelineName, Namespace: namespace}
			createdPipeline := &tektondevv1.Pipeline{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineLookupKey, createdPipeline)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a PipelineTrigger, referencing an existing Pipeline and GitRepository")

			pipelineTrigger := pipelinev1alpha1.PipelineTrigger{
				ObjectMeta: v1.ObjectMeta{
					Name:      pipelineTriggerName,
					Namespace: namespace,
				},
				Spec: pipelinev1alpha1.PipelineTriggerSpec{
					Source: pipelinev1alpha1.Source{
						Kind: "GitRepository",
						Name: gitRepositoryName,
					},
					PipelineRunSpec: tektondevv1.PipelineRunSpec{
						PipelineRef: &tektondevv1.PipelineRef{
							Name: pipelineName,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &pipelineTrigger)).Should(Succeed())

			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName, Namespace: namespace}
			createdPipelineTrigger := &pipelinev1alpha1.PipelineTrigger{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking if the PipelineTrigger was eventually created")
			pipelineTriggerList := &pipelinev1alpha1.PipelineTriggerList{}
			Eventually(func() ([]pipelinev1alpha1.PipelineTrigger, error) {
				err := k8sClient.List(
					context.Background(),
					pipelineTriggerList,
					client.InNamespace("default"),
				)
				return pipelineTriggerList.Items, err
			}, timeout, interval).ShouldNot(BeEmpty())

			By("Checking if the PipelineTrigger controller has started a new tekton pipeline")

			Eventually(func() (int, error) {
				err := k8sClient.Get(context.Background(), pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.GitRepository.Conditions), err
			}, timeout, interval).Should(Equal(1))

		})

	})

})
