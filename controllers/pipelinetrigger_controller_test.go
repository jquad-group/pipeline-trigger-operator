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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pipelinev1alpha1 "github.com/jquad-group/pipeline-trigger-operator/api/v1alpha1"
)

var _ = Describe("PipelineTriggerController", func() {
	var (
		name      string
		namespace string
		request   reconcile.Request
	)
	BeforeEach(func() {
		name = "test-resource"
		namespace = "test-namespace"
		request = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("PipelineTrigger CRD", func() {
		var (
			instance *pipelinev1alpha1.PipelineTrigger
		)
		Context("with one base S3V1 resource", func() {
			BeforeEach(func() {
				// Create a new resource using k8sClient.Create()
				// I'm just going to assume you've done this in
				// a method called createInstanceInCluster()
				instance = createInstanceInCluster(name, namespace, instance)
			})
			AfterEach(func() {
				// Remember to clean up the cluster after each test
				deleteInstanceInCluster(instance)
			})
			It("should have created the correct pipeline trigger", func() {
				// Some method where you've implemented Gomega Expect()'s
				//assertResourceUpdated(instance)
				Expect(instance).To(Equal(name))
			})
			// This polls the interior function for a result
			// and checks it against a condition until either
			// it is true, or the timeout has been reached
			Eventually(func() error {
				defer GinkgoRecover()
				err := k8sClient.Get(context.TODO(), request.NamespacedName, instance)
				return err
			}, "10s", "1s").ShouldNot(HaveOccurred())
		})
	})
})

func deleteInstanceInCluster(instance *pipelinev1alpha1.PipelineTrigger) {
	k8sClient.Delete(ctx, instance)
}

func createInstanceInCluster(name string, namespace string, instance *pipelinev1alpha1.PipelineTrigger) *pipelinev1alpha1.PipelineTrigger {
	pipelineTrigger := &pipelinev1alpha1.PipelineTrigger{
		TypeMeta: v1.TypeMeta{
			APIVersion: "pipeline.jquad.rocks/v1alpha1",
			Kind:       "PipelineTrigger",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: pipelinev1alpha1.PipelineTriggerSpec{
			Source: pipelinev1alpha1.Source{
				Kind: "ImagePolicy",
				Name: "app1",
			},
			Pipeline: pipelinev1alpha1.Pipeline{
				Name:              "build-and-push-base-image",
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
	k8sClient.Create(ctx, pipelineTrigger)
	return pipelineTrigger
}
