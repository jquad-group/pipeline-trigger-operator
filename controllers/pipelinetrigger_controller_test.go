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

		timeout  = time.Second * 50
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When setting up the test environment", func() {

		It("Should be able to create a PipelineRun custom resources", func() {
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
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Creating a Pipeline custom resource, referencing a single Task")

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
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Creating a PipelineTrigger custom resource, referencing an existing Pipeline custom resource")

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
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("By checking the status of the PipelineTrigger that should be not found")

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

			By("Delete the Pipeline Trigger CRD")
			Expect(k8sClient.Delete(ctx, createdPipelineTrigger))

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				if err != nil {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())

			/*
				By("Updating the GitRepository status")

				gitRepoCondition := v1.Condition{
					Type:               "Ready",
					Status:             v1.ConditionTrue,
					Reason:             v1.StatusSuccess,
					Message:            "Success",
					ObservedGeneration: 12,
					LastTransitionTime: v1.Now(),
				}
				var gitRepoConditions []v1.Condition
				gitRepoConditions = append(gitRepoConditions, gitRepoCondition)
				gitRepoStatus := sourcev1.GitRepositoryStatus{
					Artifact: &sourcev1.Artifact{
						Checksum:       "cb0053b034ac7e74e2278b94b69db15871e9b3b40124adde8c585c1bdda48b25",
						Path:           "gitrepository/flux-system/flux-system/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69.tar.gz",
						URL:            "http://source-controller.flux-system.svc.cluster.local./gitrepository/flux-system/flux-system/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69.tar.gz",
						Revision:       "main/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69",
						LastUpdateTime: v1.Now(),
					},
					Conditions: gitRepoConditions,
				}

				myGitRepo := &sourcev1.GitRepository{}
				k8sClient.Get(ctx, types.NamespacedName{Name: gitRepository1Name, Namespace: namespace}, myGitRepo)
				myGitRepo.Status = gitRepoStatus
				Expect(k8sClient.Status().Update(ctx, myGitRepo)).Should(Succeed())

				By("By checking the status of the PipelineTrigger CR")

				testGit := &sourcev1.GitRepository{}
				k8sClient.Get(ctx, types.NamespacedName{Name: gitRepository1Name, Namespace: namespace}, testGit)
				fmt.Println("GIT")
				fmt.Println(testGit.Name)
				fmt.Println(testGit.Namespace)
				fmt.Println(testGit.Status.Artifact.Revision)
				fmt.Println(testGit.Status.Artifact.Path)

				Eventually(func() (string, error) {
					myPipelineTrigger1 := &pipelinev1alpha1.PipelineTrigger{}
					fmt.Println("GETTING PT STATUS")

					err := k8sClient.Get(ctx, types.NamespacedName{Name: pipelineTrigger1Name, Namespace: namespace}, myPipelineTrigger1)
					fmt.Println(myPipelineTrigger1.Name)
					fmt.Println(myPipelineTrigger1.Namespace)
					fmt.Println(myPipelineTrigger1.Spec.Source.Name)
					fmt.Println(myPipelineTrigger1.Status.GitRepository.CommitId)
					if err != nil {
						fmt.Println(err)
						return "", err
					}

					if len(myPipelineTrigger1.Status.GitRepository.Conditions) == 0 {
						return "", nil
					}

					return myPipelineTrigger1.Status.GitRepository.Conditions[0].Reason, nil
				}, time.Hour, time.Second).Should(ContainSubstring("Failed"), "Should have %s in the status", "Failed")
			*/
		})

		It("Should start a new test", func() {

			By("Creating a GitRepository custom resource")
			//ctx := context.Background()
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
				if err != nil {
					return false
				}
				return true
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

			By("By checking the GitRepository artifact revision")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, gitRepositoryLookupKey, createdGitRepo)
				if err != nil {
					return "", err
				}
				return createdGitRepo.Status.Artifact.Revision, nil
			}, duration, interval).Should(Equal("main/dc0fd09d0915f47cbda5f235a8a9c30b2d8baa69"))

			By("Re-Creating a PipelineTrigger custom resource, referencing an existing Pipeline custom resource")

			pipelineTriggerNew := pipelinev1alpha1.PipelineTrigger{
				TypeMeta: v1.TypeMeta{
					Kind:       "PipelineTrigger",
					APIVersion: "pipeline.jquad.rocks/v1alpha1",
				},
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
			Expect(k8sClient.Create(ctx, &pipelineTriggerNew)).Should(Succeed())

			pipelineTriggerNewLookupKey := types.NamespacedName{Name: pipelineTriggerName, Namespace: namespace}
			createdPipelineTriggerNew := &pipelinev1alpha1.PipelineTrigger{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerNewLookupKey, createdPipelineTriggerNew)
				if err != nil {
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())

			By("Gettint the PR")
			createdPrList := &tektondevv1.PipelineRunList{}
			Eventually(func() bool {
				err := k8sClient.List(ctx, createdPrList, client.InNamespace(namespace))
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			fmt.Println("PR LIST")
			fmt.Println(len(createdPrList.Items))
			fmt.Println(createdPrList.Items)

			By("Checking the new PipelineTrigger custom resource status")

			Eventually(func() (int, bool) {

				err := k8sClient.Get(ctx, pipelineTriggerNewLookupKey, createdPipelineTriggerNew)
				if err != nil {
					return -1, false
				}
				if len(createdPipelineTriggerNew.Status.GitRepository.Conditions) == 1 {
					return 1, true
				}
				return 0, false
			}, timeout*5000, interval).Should(Equal(1))

		})

	})

})
