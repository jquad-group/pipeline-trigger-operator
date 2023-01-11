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

	imagereflectorv1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	opstatus "github.com/jquad-group/pipeline-trigger-operator/pkg/status"
	pullrequestv1alpha1 "github.com/jquad-group/pullrequest-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tektondevv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	clock "k8s.io/utils/clock/testing"
	"knative.dev/pkg/apis"
)

var taskMock tektondevv1.Task
var pipelineMock tektondevv1.Pipeline
var gitRepository sourcev1.GitRepository
var imagePolicy imagereflectorv1.ImagePolicy
var pullRequest pullrequestv1alpha1.PullRequest
var createdPipelineTrigger *pipelinev1alpha1.PipelineTrigger
var createdImagePolicy *imagereflectorv1.ImagePolicy
var createdPipelineRun *tektondevv1.PipelineRun
var createdGitRepo *sourcev1.GitRepository
var createdTask *tektondevv1.Task
var createdPipeline *tektondevv1.Pipeline
var createdPullRequest *pullrequestv1alpha1.PullRequest
var pipelineTriggerList *pipelinev1alpha1.PipelineTriggerList
var pipelineTrigger0 pipelinev1alpha1.PipelineTrigger
var pipelineTrigger1 pipelinev1alpha1.PipelineTrigger
var pipelineTrigger2 pipelinev1alpha1.PipelineTrigger
var pipelineTrigger3 pipelinev1alpha1.PipelineTrigger

var _ = Describe("PipelineTrigger controller", FlakeAttempts(5), func() {

	const (
		gitRepositoryName    = "git-repo-1"
		imagePolicyName      = "image-policy-1"
		pullRequestName      = "pr-1"
		pipelineTriggerName0 = "pipeline-trigger-0"
		pipelineTriggerName1 = "pipeline-trigger-1"
		pipelineTriggerName2 = "pipeline-trigger-2"
		pipelineTriggerName3 = "pipeline-trigger-3"
		taskName             = "build"
		pipelineName         = "build-and-push"
		namespace            = "default"

		timeout  = time.Second * 50
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	BeforeEach(func() {
		ctx = context.Background()
		createdPipelineTrigger = &pipelinev1alpha1.PipelineTrigger{}
		createdImagePolicy = &imagereflectorv1.ImagePolicy{}
		createdPipelineRun = &tektondevv1.PipelineRun{}
		createdGitRepo = &sourcev1.GitRepository{}
		createdTask = &tektondevv1.Task{}
		createdPipeline = &tektondevv1.Pipeline{}
		createdPullRequest = &pullrequestv1alpha1.PullRequest{}
		pipelineTriggerList = &pipelinev1alpha1.PipelineTriggerList{}
		taskMock = tektondevv1.Task{
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

		pipelineMock = tektondevv1.Pipeline{
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

		gitRepository = sourcev1.GitRepository{
			ObjectMeta: v1.ObjectMeta{
				Name:      gitRepositoryName,
				Namespace: namespace,
			},
			Spec: sourcev1.GitRepositorySpec{
				URL:      "http://github.com/org/repo.git",
				Interval: v1.Duration{},
			},
		}

		imagePolicy = imagereflectorv1.ImagePolicy{
			ObjectMeta: v1.ObjectMeta{
				Name:      imagePolicyName,
				Namespace: namespace,
			},
			Spec: imagereflectorv1.ImagePolicySpec{
				ImageRepositoryRef: meta.NamespacedObjectReference{},
				Policy:             imagereflectorv1.ImagePolicyChoice{},
			},
		}

		pullRequest = pullrequestv1alpha1.PullRequest{
			ObjectMeta: v1.ObjectMeta{
				Name:      pullRequestName,
				Namespace: namespace,
			},
			Spec: pullrequestv1alpha1.PullRequestSpec{
				GitProvider: pullrequestv1alpha1.GitProvider{
					Provider:           "Github",
					InsecureSkipVerify: true,
					Github: pullrequestv1alpha1.Github{
						Url:        "https://github.com/example-org/microservice",
						Owner:      "example-org",
						Repository: "microservice",
					},
				},
				TargetBranch: pullrequestv1alpha1.Branch{
					Name: "main",
				},
				Interval: v1.Duration{},
			},
		}

		pipelineTrigger0 = pipelinev1alpha1.PipelineTrigger{
			TypeMeta: v1.TypeMeta{
				Kind:       "PipelineTrigger",
				APIVersion: "pipeline.jquad.rocks/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      pipelineTriggerName0,
				Namespace: namespace,
			},
			Spec: pipelinev1alpha1.PipelineTriggerSpec{
				Source: pipelinev1alpha1.Source{
					Kind: "GitRepository",
					Name: gitRepositoryName,
				},
				PipelineRunSpec: tektondevv1.PipelineRunSpec{
					PipelineRef: &tektondevv1.PipelineRef{
						Name: "does-not-exist",
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

		pipelineTrigger1 = pipelinev1alpha1.PipelineTrigger{
			TypeMeta: v1.TypeMeta{
				Kind:       "PipelineTrigger",
				APIVersion: "pipeline.jquad.rocks/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      pipelineTriggerName1,
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

		pipelineTrigger2 = pipelinev1alpha1.PipelineTrigger{
			TypeMeta: v1.TypeMeta{
				Kind:       "PipelineTrigger",
				APIVersion: "pipeline.jquad.rocks/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      pipelineTriggerName2,
				Namespace: namespace,
			},
			Spec: pipelinev1alpha1.PipelineTriggerSpec{
				Source: pipelinev1alpha1.Source{
					Kind: "ImagePolicy",
					Name: imagePolicyName,
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

		pipelineTrigger3 = pipelinev1alpha1.PipelineTrigger{
			TypeMeta: v1.TypeMeta{
				Kind:       "PipelineTrigger",
				APIVersion: "pipeline.jquad.rocks/v1alpha1",
			},
			ObjectMeta: v1.ObjectMeta{
				Name:      pipelineTriggerName3,
				Namespace: namespace,
			},
			Spec: pipelinev1alpha1.PipelineTriggerSpec{
				Source: pipelinev1alpha1.Source{
					Kind: "PullRequest",
					Name: pullRequestName,
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

	})

	Context("PipelineTrigger fails to create a PipelineRun due to missing GitRepository", func() {

		It("Should not be able to create a PipelineRun", func() {

			By("Creating a Task")
			Expect(k8sClient.Create(ctx, &taskMock)).Should(Succeed())
			taskLookupKey := types.NamespacedName{Name: taskName, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, taskLookupKey, createdTask)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a Pipeline, referencing a single Task")
			Expect(k8sClient.Create(ctx, &pipelineMock)).Should(Succeed())
			pipelineLookupKey := types.NamespacedName{Name: pipelineName, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineLookupKey, createdPipeline)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a PipelineTrigger, referencing an existing Pipeline and not existing GitRepository")
			Expect(k8sClient.Create(ctx, &pipelineTrigger1)).Should(Succeed())
			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName1, Namespace: namespace}
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
				return createdPipelineTrigger.Status.GitRepository.Conditions[0].Message
			}, timeout, interval).Should(ContainSubstring("not found"))
		})
	})

	Context("PipelineTrigger creates a PipelineRun on the test cluster for GitRepository", func() {

		It("Should be able to create a PipelineRun custom resources", func() {

			By("Creating a GitRepository")
			Expect(k8sClient.Create(ctx, &gitRepository)).Should(Succeed())
			gitRepositoryLookupKey := types.NamespacedName{Name: gitRepositoryName, Namespace: namespace}

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

			By("Creating a PipelineTrigger, referencing an existing Pipeline and GitRepository")
			Expect(k8sClient.Create(ctx, &pipelineTrigger1)).Should(Succeed())
			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName1, Namespace: namespace}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking if the PipelineTrigger was eventually created")
			Eventually(func() ([]pipelinev1alpha1.PipelineTrigger, error) {
				err := k8sClient.List(
					ctx,
					pipelineTriggerList,
					client.InNamespace("default"),
				)
				return pipelineTriggerList.Items, err
			}, timeout, interval).ShouldNot(BeEmpty())

			By("Checking if the PipelineTrigger controller has started a new tekton pipeline")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.GitRepository.Conditions), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller has started a single pipeline")
			pipelineRuns := &tektondevv1.PipelineRunList{}
			Eventually(func() (int, error) {
				err := k8sClient.List(ctx, pipelineRuns)
				return len(pipelineRuns.Items), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller is managing the PipelineRun")
			pipelineRunList := &tektondevv1.PipelineRunList{}
			Eventually(func() string {
				k8sClient.List(ctx, pipelineRunList)
				pipelineRun := pipelineRunList.Items[0]
				return pipelineRun.GetOwnerReferences()[0].Name
			}, timeout, interval).Should(Equal(pipelineTriggerName1))

			By("Updating the PipelineRun status to reason started (status: Unknown)")
			pipelineRunLookupKey := types.NamespacedName{Name: pipelineRunList.Items[0].Name, Namespace: namespace}

			var now = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
			var testClock = clock.NewFakePassiveClock(now)
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.InitializeConditions(testClock)
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())
			// https://tekton.dev/docs/pipelines/pipelineruns/#pipelinerun-status
			By("Checking if the PipelineRun status reason was updated to reason: Started (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Started"))

			By("Updating the PipelineRun status to reason: Running (status: Unknown)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionUnknown,
				Reason:  "Running",
				Message: "Tasks Running: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Running (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Running"))

			By("Updating the PipelineRun status to reason: Succeeded (status: True)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionTrue,
				Reason:  "Succeeded",
				Message: "Tasks Completed: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Succeeded (status: True)")
			Eventually(func() (corev1.ConditionStatus, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Status, err
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			By("Checking if the PipelineTrigger LatestPipelineRun is correctly set")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return createdPipelineTrigger.Status.GitRepository.LatestPipelineRun, err
			}, timeout, interval).Should(ContainSubstring("main"))

			By("Checking if the PipelineTrigger status is updated to succeeded when the corresponding PipelineRun is completed")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				cond, _ := createdPipelineTrigger.Status.GitRepository.GetCondition(opstatus.ReconcileSuccess)
				return cond.Type, err
			}, timeout, interval).Should(ContainSubstring(opstatus.ReconcileSuccess))

			By("Checking if the PipelineTrigger status conditions = PipelineRuns status changes")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.GitRepository.Conditions), err
			}, timeout, interval).Should(Equal(3))

		})
	})

	Context("PipelineTriggers status conditions are set according to the PipelineRun status condition", func() {

		It("Should be able to create a PipelineRun custom resources", func() {

			By("Creating a GitRepository")
			Expect(k8sClient.Create(ctx, &gitRepository)).Should(Succeed())
			gitRepositoryLookupKey := types.NamespacedName{Name: gitRepositoryName, Namespace: namespace}

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

			By("Creating a PipelineTrigger, referencing an existing Pipeline and GitRepository")
			Expect(k8sClient.Create(ctx, &pipelineTrigger1)).Should(Succeed())
			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName1, Namespace: namespace}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking if the PipelineTrigger was eventually created")
			Eventually(func() ([]pipelinev1alpha1.PipelineTrigger, error) {
				err := k8sClient.List(
					ctx,
					pipelineTriggerList,
					client.InNamespace("default"),
				)
				return pipelineTriggerList.Items, err
			}, timeout, interval).ShouldNot(BeEmpty())

			By("Checking if the PipelineTrigger controller has started a new tekton pipeline")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.GitRepository.Conditions), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller has started a single pipeline")
			pipelineRuns := &tektondevv1.PipelineRunList{}
			Eventually(func() (int, error) {
				err := k8sClient.List(ctx, pipelineRuns)
				return len(pipelineRuns.Items), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller is managing the PipelineRun")
			pipelineRunList := &tektondevv1.PipelineRunList{}
			Eventually(func() string {
				k8sClient.List(ctx, pipelineRunList)
				pipelineRun := pipelineRunList.Items[0]
				return pipelineRun.GetOwnerReferences()[0].Name
			}, timeout, interval).Should(Equal(pipelineTriggerName1))

			// PipelineRun      -     PipelineTrigger
			// Status: Unknown        Unknown
			// Reason: Started		  Unknown
			By("Checking if the PipelineTrigger status is Unknown")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return string(createdPipelineTrigger.Status.GitRepository.GetLastCondition().Status), err
			}, timeout, interval).Should(Equal("Unknown"))

			By("Checking if the PipelineTrigger reason is Unknown")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return string(createdPipelineTrigger.Status.GitRepository.GetLastCondition().Status), err
			}, timeout, interval).Should(Equal("Unknown"))

			// PipelineRun      -     PipelineTrigger
			// Status: Unknown        Unknown
			// Reason: Running		  InProgress
			By("Updating the PipelineRun status to reason: Running (status: Unknown)")
			pipelineRunLookupKey := types.NamespacedName{Name: pipelineRunList.Items[0].Name, Namespace: namespace}
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionUnknown,
				Reason:  "Running",
				Message: "Tasks Running: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Running (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Running"))

			By("Checking if the PipelineTrigger status is Unknown")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return string(createdPipelineTrigger.Status.GitRepository.GetLastCondition().Status), err
			}, timeout, interval).Should(Equal("Unknown"))

			By("Checking if the PipelineTrigger reason is InProgress")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return string(createdPipelineTrigger.Status.GitRepository.GetLastCondition().Reason), err
			}, timeout, interval).Should(Equal("InProgress"))

			// PipelineRun      -     PipelineTrigger
			// Status: False          False
			// Reason: Cancelled	  Cancelled
			By("Updating the PipelineRun status to reason: Cancelled (status: False)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "Cancelled",
				Message: "PipelineRun xxx was cancelled",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Cancelled (status: False)")
			Eventually(func() (corev1.ConditionStatus, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Status, err
			}, timeout, interval).Should(Equal(corev1.ConditionFalse))

			// PipelineRun is Cancelled
			By("Checking if the PipelineTrigger status is False")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return string(createdPipelineTrigger.Status.GitRepository.GetLastCondition().Status), err
			}, timeout, interval).Should(Equal("False"))

			By("Checking if the PipelineTrigger reason is Cancelled")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return string(createdPipelineTrigger.Status.GitRepository.GetLastCondition().Reason), err
			}, timeout, interval).Should(Equal("Cancelled"))

		})
	})

	Context("PipelineTrigger fails to create a PipelineRun due to missing Pipeline", func() {

		It("Should not be able to create a PipelineRun", func() {

			By("Creating a GitRepository")
			Expect(k8sClient.Create(ctx, &gitRepository)).Should(Succeed())
			gitRepositoryLookupKey := types.NamespacedName{Name: gitRepositoryName, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, gitRepositoryLookupKey, createdGitRepo)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating a PipelineTrigger, referencing a non-existing Pipeline and existing GitRepository")
			Expect(k8sClient.Create(ctx, &pipelineTrigger0)).Should(Succeed())
			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName0, Namespace: namespace}
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
				return createdPipelineTrigger.Status.GitRepository.Conditions[0].Message
			}, timeout, interval).Should(ContainSubstring("not found"))
		})
	})

	Context("PipelineTrigger fails to create a PipelineRun to missing ImagePolicy", func() {

		It("Should not be able to create a PipelineRun", func() {

			By("Creating a PipelineTrigger, referencing an existing Pipeline and not existing GitRepository")
			Expect(k8sClient.Create(ctx, &pipelineTrigger2)).Should(Succeed())
			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName2, Namespace: namespace}
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
				if len(createdPipelineTrigger.Status.ImagePolicy.Conditions) == 0 {
					return ""
				}
				return createdPipelineTrigger.Status.ImagePolicy.Conditions[0].Message
			}, timeout, interval).Should(ContainSubstring("not found"))
		})
	})

	Context("PipelineTrigger creates a PipelineRun on the test cluster for ImagePolicy", func() {

		It("Should be able to create a PipelineRun custom resources", func() {

			By("Creating a ImagePolicy")
			Expect(k8sClient.Create(ctx, &imagePolicy)).Should(Succeed())
			imagePolicyLookupKey := types.NamespacedName{Name: imagePolicyName, Namespace: namespace}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, imagePolicyLookupKey, createdImagePolicy)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the ImagePolicy status")
			imagePolicyStatus := imagereflectorv1.ImagePolicyStatus{
				LatestImage: "ghcr.io/test/test:v0.0.1",
				Conditions: []v1.Condition{
					{
						Type:               "Ready",
						Status:             v1.ConditionTrue,
						Reason:             "ReconciliationSucceeded",
						Message:            "Latest image tag for 'ghcr.io/test/test' resolved to: v0.0.1",
						LastTransitionTime: v1.Now(),
					},
				},
			}

			k8sClient.Get(ctx, imagePolicyLookupKey, createdImagePolicy)
			createdImagePolicy.Status = imagePolicyStatus
			Expect(k8sClient.Status().Update(ctx, createdImagePolicy)).Should(Succeed())

			By("Checking if the ImagePolciy artifact revision was updated")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, imagePolicyLookupKey, createdImagePolicy)
				if err != nil {
					return "", err
				}
				return createdImagePolicy.Status.LatestImage, nil
			}, duration, interval).Should(Equal("ghcr.io/test/test:v0.0.1"))

			By("Creating a PipelineTrigger, referencing an existing Pipeline and ImagePolicy")

			Expect(k8sClient.Create(ctx, &pipelineTrigger2)).Should(Succeed())

			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName2, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking if the PipelineTrigger was eventually created")
			Eventually(func() ([]pipelinev1alpha1.PipelineTrigger, error) {
				err := k8sClient.List(
					ctx,
					pipelineTriggerList,
					client.InNamespace("default"),
				)
				return pipelineTriggerList.Items, err
			}, timeout, interval).ShouldNot(BeEmpty())

			By("Checking if the PipelineTrigger controller has started a new tekton pipeline")

			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.ImagePolicy.Conditions), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller is managing the PipelineRun")

			pipelineRunList := &tektondevv1.PipelineRunList{}
			Eventually(func() string {
				k8sClient.List(ctx, pipelineRunList)
				pipelineRun := pipelineRunList.Items[0]
				return pipelineRun.GetOwnerReferences()[0].Name
			}, timeout, interval).Should(Equal(pipelineTriggerName2))

			By("Updating the PipelineRun status to reason started (status: Unknown)")
			pipelineRunLookupKey := types.NamespacedName{Name: pipelineRunList.Items[0].Name, Namespace: namespace}

			var now = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
			var testClock = clock.NewFakePassiveClock(now)
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.InitializeConditions(testClock)
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())
			// https://tekton.dev/docs/pipelines/pipelineruns/#pipelinerun-status
			By("Checking if the PipelineRun status reason was updated to reason: Started (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Started"))

			By("Updating the PipelineRun status to reason: Running (status: Unknown)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionUnknown,
				Reason:  "Running",
				Message: "Tasks Running: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Running (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Running"))

			By("Updating the PipelineRun status to reason: Succeeded (status: True)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionTrue,
				Reason:  "Succeeded",
				Message: "Tasks Completed: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Succeeded (status: True)")
			Eventually(func() (corev1.ConditionStatus, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Status, err
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			By("Checking if the PipelineTrigger LatestPipelineRun is correctly set")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return createdPipelineTrigger.Status.ImagePolicy.LatestPipelineRun, err
			}, timeout, interval).Should(ContainSubstring("test"))

			By("Checking if the PipelineTrigger status is updated to succeeded when the corresponding PipelineRun is completed")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				cond, _ := createdPipelineTrigger.Status.ImagePolicy.GetCondition(opstatus.ReconcileSuccess)
				return cond.Type, err
			}, timeout, interval).Should(ContainSubstring(opstatus.ReconcileSuccess))

			By("Checking if the PipelineTrigger status conditions = PipelineRuns status changes")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.ImagePolicy.Conditions), err
			}, timeout*10, interval).Should(Equal(3))

		})

	})

	Context("PipelineTrigger fails to create a PipelineRun to missing PullRequest", func() {
		It("Should not be able to create a PipelineRun", func() {

			By("Creating a PipelineTrigger, referencing an existing Pipeline and not existing PullRequest")

			Expect(k8sClient.Create(ctx, &pipelineTrigger3)).Should(Succeed())

			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName3, Namespace: namespace}

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

				if len(createdPipelineTrigger.Status.Branches.Branches["null"].Conditions) == 0 {
					return ""
				}
				return createdPipelineTrigger.Status.Branches.Branches["null"].Conditions[0].Message
			}, timeout, interval).Should(ContainSubstring("not found"))

		})
	})

	Context("PipelineTrigger creates a PipelineRun on the test cluster for PullRequest", func() {

		It("Should be able to create a PipelineRun custom resources", func() {

			By("Creating a PullRequest")
			Expect(k8sClient.Create(ctx, &pullRequest)).Should(Succeed())

			pullRequestLookupKey := types.NamespacedName{Name: pullRequestName, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pullRequestLookupKey, createdPullRequest)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Updating the PullRequest status")

			pullRequestStatus := pullrequestv1alpha1.PullRequestStatus{
				SourceBranches: pullrequestv1alpha1.Branches{
					Branches: []pullrequestv1alpha1.Branch{
						{
							Name:    "feature-branch-test",
							Commit:  "8932484a2017a3784608c2db429553a94f1e2f4b",
							Details: "{\"id\":1163006807}",
						},
					},
				},
				Conditions: []v1.Condition{
					{
						Type:               "Success",
						Status:             v1.ConditionTrue,
						Reason:             "Succeded",
						Message:            "Success",
						LastTransitionTime: v1.Now(),
					},
				},
			}

			k8sClient.Get(ctx, pullRequestLookupKey, createdPullRequest)
			createdPullRequest.Status = pullRequestStatus
			Expect(k8sClient.Status().Update(ctx, createdPullRequest)).Should(Succeed())

			By("Checking if the PullRequest artifact revision was updated")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pullRequestLookupKey, createdPullRequest)
				if err != nil {
					return "", err
				}
				return createdPullRequest.Status.SourceBranches.Branches[0].Name, nil
			}, duration, interval).Should(Equal("feature-branch-test"))

			By("Creating a PipelineTrigger, referencing an existing Pipeline and PullRequest")

			Expect(k8sClient.Create(ctx, &pipelineTrigger3)).Should(Succeed())

			pipelineTriggerLookupKey := types.NamespacedName{Name: pipelineTriggerName3, Namespace: namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Checking if the PipelineTrigger was eventually created")
			Eventually(func() (int, error) {
				err := k8sClient.List(
					ctx,
					pipelineTriggerList,
					client.InNamespace("default"),
				)
				return len(pipelineTriggerList.Items), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller has started a new tekton pipeline")

			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.Branches.Branches["feature-branch-test"].Conditions), err
			}, timeout, interval).Should(Equal(1))

			By("Checking if the PipelineTrigger controller is managing the PipelineRun")

			pipelineRunList := &tektondevv1.PipelineRunList{}
			Eventually(func() string {
				k8sClient.List(ctx, pipelineRunList)
				pipelineRun := pipelineRunList.Items[0]
				return pipelineRun.GetOwnerReferences()[0].Name
			}, timeout, interval).Should(Equal(pipelineTriggerName3))

			By("Adding the tekton.dev label to the PipelineRun")
			pipelineRunLookupKey := types.NamespacedName{Name: pipelineRunList.Items[0].Name, Namespace: namespace}

			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.SetLabels(map[string]string{
				"pipeline.jquad.rocks/pr.branch.commit": "8932484a2017a3784608c2db429553a94f1e2f4b",
				"pipeline.jquad.rocks/pr.branch.name":   "feature-branch-test",
				"tekton.dev/pipeline":                   pipelineName})
			Expect(k8sClient.Update(ctx, createdPipelineRun)).Should(Succeed())
			By("Updating the PipelineRun status to reason started (status: Unknown)")
			var now = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
			var testClock = clock.NewFakePassiveClock(now)
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.InitializeConditions(testClock)
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())
			// https://tekton.dev/docs/pipelines/pipelineruns/#pipelinerun-status
			By("Checking if the PipelineRun status reason was updated to reason: Started (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Started"))

			By("Updating the PipelineRun status to reason: Running (status: Unknown)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionUnknown,
				Reason:  "Running",
				Message: "Tasks Running: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Running (status: Unknown)")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return string(createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Reason), err
			}, timeout, interval).Should(Equal("Running"))

			By("Updating the PipelineRun status to reason: Succeeded (status: True)")
			k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
			createdPipelineRun.Status.SetCondition(&apis.Condition{
				Type:    apis.ConditionSucceeded,
				Status:  corev1.ConditionTrue,
				Reason:  "Succeeded",
				Message: "Tasks Completed: 1 (Failed: 0, Cancelled 0), Skipped: 0",
			})
			createdPipelineRun.Status.ObservedGeneration = 2
			Expect(k8sClient.Status().Update(ctx, createdPipelineRun)).Should(Succeed())

			By("Checking if the PipelineRun status was updated to reason: Succeeded (status: True)")
			Eventually(func() (corev1.ConditionStatus, error) {
				err := k8sClient.Get(ctx, pipelineRunLookupKey, createdPipelineRun)
				return createdPipelineRun.Status.GetCondition(apis.ConditionSucceeded).Status, err
			}, timeout, interval).Should(Equal(corev1.ConditionTrue))

			By("Checking if the PipelineTrigger LatestPipelineRun is correctly set")
			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return createdPipelineTrigger.Status.Branches.Branches["feature-branch-test"].LatestPipelineRun, err
			}, timeout, interval).Should(ContainSubstring("feature-branch-test"))

			By("Checking if the PipelineTrigger status conditions = PipelineRuns status changes")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, pipelineTriggerLookupKey, createdPipelineTrigger)
				return len(createdPipelineTrigger.Status.Branches.Branches["feature-branch-test"].Conditions), err
			}, timeout, interval).Should(Equal(3))

		})

	})

	AfterEach(func() {
		createdImagePolicyList := imagereflectorv1.ImagePolicyList{}
		createdPipelineRunList := tektondevv1.PipelineRunList{}
		createdGitRepositoryList := sourcev1.GitRepositoryList{}
		createdPullRequestList := pullrequestv1alpha1.PullRequestList{}
		err := k8sClient.Delete(ctx, createdPipelineTrigger)
		k8sClient.Delete(ctx, createdImagePolicy)
		k8sClient.Delete(ctx, createdPipelineRun)
		k8sClient.Delete(ctx, createdGitRepo)
		k8sClient.Delete(ctx, createdPullRequest)
		if err != nil {
			fmt.Println(err, "failed to delete object", "name")
		}
		Eventually(func() bool {
			errPipelineTrigger := k8sClient.List(ctx, pipelineTriggerList)
			if errPipelineTrigger != nil {
				return false
			}
			errImagePolicy := k8sClient.List(ctx, &createdImagePolicyList)
			if errImagePolicy != nil {
				return false
			}
			errGitRepository := k8sClient.List(ctx, &createdGitRepositoryList)
			if errGitRepository != nil {
				return false
			}
			errPipelineRun := k8sClient.List(ctx, &createdPipelineRunList)
			if errPipelineRun != nil {
				return false
			}
			errPullRequest := k8sClient.List(ctx, &createdPullRequestList)
			if errPullRequest != nil {
				return false
			}
			if (len(pipelineTriggerList.Items) > 0) ||
				(len(createdImagePolicyList.Items) > 0) ||
				(len(createdPipelineRunList.Items) > 0) ||
				(len(createdGitRepositoryList.Items) > 0) ||
				(len(createdPullRequestList.Items) > 0) {
				return false
			}
			return true
		}, timeout*10, interval).Should(BeTrue())

	})

})
