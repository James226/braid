/*
Copyright 2025.

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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	braidv1 "github.com/james226/braid/api/v1"
)

var _ = Describe("Application Controller", func() {
	Context("When reconciling an application with no objects", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		application := &braidv1.Application{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Application")
			err := k8sClient.Get(ctx, typeNamespacedName, application)
			if err != nil && errors.IsNotFound(err) {
				appTemplate := &braidv1.ApplicationTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: braidv1.ApplicationTemplateSpec{
						Objects: []braidv1.ApplicationObject{},
					},
				}
				Expect(k8sClient.Create(ctx, appTemplate)).To(Succeed())

				resource := &braidv1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: braidv1.ApplicationSpec{
						Template: resourceName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &braidv1.Application{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Application")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			template := &braidv1.ApplicationTemplate{}
			err = k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Application")
			Expect(k8sClient.Delete(ctx, template)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When reconciling an application with objects", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		application := &braidv1.Application{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Application")
			err := k8sClient.Get(ctx, typeNamespacedName, application)
			if err != nil && errors.IsNotFound(err) {
				object := &braidv1.ObjectVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: braidv1.ObjectVersionSpec{
						ApiVersion: "v1",
						Kind:       "Pod",
						Spec: `
containers:
- name: nginx
  image: "nginx"
  env:
  - name: foo
	value: bar`,
					},
				}
				Expect(k8sClient.Create(ctx, object)).To(Succeed())

				appTemplate := &braidv1.ApplicationTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: braidv1.ApplicationTemplateSpec{
						Objects: []braidv1.ApplicationObject{
							{Template: resourceName, Variables: map[string]string{}},
						},
					},
				}
				Expect(k8sClient.Create(ctx, appTemplate)).To(Succeed())

				resource := &braidv1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: braidv1.ApplicationSpec{
						Template: resourceName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &braidv1.Application{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Application")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			template := &braidv1.ApplicationTemplate{}
			err = k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ApplicationTemplate")
			Expect(k8sClient.Delete(ctx, template)).To(Succeed())

			object := &braidv1.ObjectVersion{}
			err = k8sClient.Get(ctx, typeNamespacedName, object)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance ObjectVersion")
			Expect(k8sClient.Delete(ctx, object)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ApplicationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("The pod is created")

			template := &v1.Pod{}
			err = k8sClient.Get(ctx, typeNamespacedName, template)
			Expect(err).NotTo(HaveOccurred())

			Expect(template.Spec.Containers[0].Image).To(Equal("nginx"))
			Expect(template.Spec.Containers[0].Env).To(Equal([]v1.EnvVar{{Name: "foo", Value: "bar"}}))
		})
	})
})
