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

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
)

var _ = Describe("PipelineTrigger controller", func() {

	const (
		resourceName = "foo-1"

		namespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When setting up the test environment", func() {

		It("Should fail to create a PipelineRun custom resources", func() {
			By("Creating a GitRepository custom resource")

			ctx := context.Background()
			gitRepository := sourcev1.GitRepository{
				ObjectMeta: v1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: sourcev1.GitRepositorySpec{
					URL:      "http://github.com/org/repo.git",
					Interval: v1.Duration{},
				},
				//Status: gitRepoStatus,
			}
			Expect(k8sClient.Create(ctx, &gitRepository)).Should(Succeed())

			By("Creating a PipelineTrigger custom resource, referencing not existing Pipeline custom resource")
			param1 := pipelinev1alpha1.InputParam{
				Name:  "test",
				Value: "test",
			}
			var params []pipelinev1alpha1.InputParam
			params = append(params, param1)

			source := pipelinev1alpha1.Source{
				Kind: "GitRepository",
				Name: resourceName,
			}

			pipelineTrigger1 := pipelinev1alpha1.PipelineTrigger{
				ObjectMeta: v1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				Spec: pipelinev1alpha1.PipelineTriggerSpec{
					Source: source,
					Pipeline: pipelinev1alpha1.Pipeline{
						Name:              resourceName,
						SericeAccountName: "test",
						InputParams:       params,
						Workspace: pipelinev1alpha1.Workspace{
							Name:       "test",
							Size:       "5Gi",
							AccessMode: "test",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, &pipelineTrigger1)).Should(Succeed())

			By("Updating the GitRepository status")

			gitRepoCondition := v1.Condition{
				Type:               "Ready",
				Status:             v1.ConditionTrue,
				Reason:             v1.StatusSuccess,
				Message:            "Success",
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
			k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, myGitRepo)
			myGitRepo.Status = gitRepoStatus
			Expect(k8sClient.Status().Update(ctx, myGitRepo)).Should(Succeed())

			By("By checking the status of the PipelineTrigger CR")
			myPipelineTrigger1 := &pipelinev1alpha1.PipelineTrigger{}

			Eventually(func() (string, error) {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, myPipelineTrigger1)
				if err != nil {
					return "", err
				}

				if len(myPipelineTrigger1.Status.GitRepository.Conditions) == 0 {
					return "", nil
				}

				return myPipelineTrigger1.Status.GitRepository.Conditions[0].Reason, nil
			}, timeout, interval).Should(ContainSubstring("Failed"), "Should have %s in the status", "Failed")

		})
	})

})
