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
	"bytes"
	"context"
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/james226/braid/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=braid.james-parker.dev,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=braid.james-parker.dev,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=braid.james-parker.dev,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	l.Info("Reconcile request", "namespace", req.Namespace, "name", req.Name)

	var application v1.Application
	err := r.Get(ctx, req.NamespacedName, &application)

	if err != nil {
		l.Error(err, "unable to fetch Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var tmpl v1.ApplicationTemplate

	err = r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Spec.Template,
	}, &tmpl)

	if err != nil {
		l.Error(err, "unable to fetch Application Template")
		return ctrl.Result{}, err
	}

	if application.GetOwnerReferences() == nil {
		err = ctrl.SetControllerReference(&tmpl, &application, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Update(ctx, &application)
	}

	for _, o := range tmpl.Spec.Objects {

		var deployments v1.ObjectVersion
		err = r.Get(ctx, types.NamespacedName{
			Namespace: application.Namespace,
			Name:      o.Template,
		}, &deployments)

		if err != nil {
			l.Error(err, "unable to fetch Deployment Version")
			return ctrl.Result{}, err
		}

		variables := make(map[string]string)

		for k, v := range o.Variables {
			variables[k] = v
		}

		for k, v := range application.Spec.Variables {
			variables[k] = v
		}

		spec, err := replaceVariables(deployments.Spec.Spec, variables)
		if err != nil {
			l.Error(err, "unable to create Deployment spec")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		object := unstructured.Unstructured{}
		groupVersion, err := schema.ParseGroupVersion(deployments.Spec.ApiVersion)
		if err != nil {
			l.Error(err, "unable to parse GroupVersion")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		object.SetGroupVersionKind(groupVersion.WithKind(deployments.Spec.Kind))

		name := types.NamespacedName{Namespace: application.Namespace, Name: application.Name}
		err = r.Get(ctx, name, &object)

		if err != nil {
			if errors.IsNotFound(err) {

				object.SetName(application.Name)
				object.SetNamespace(application.Namespace)
				object.SetLabels(make(map[string]string))
				object.SetAnnotations(make(map[string]string))
				object.SetOwnerReferences([]metav1.OwnerReference{{
					APIVersion: "braid.james-parker.dev/v1",
					Kind:       "Application",
					Name:       application.Name,
					UID:        application.UID,
					Controller: ptr.To(true),
				}})

				object.Object["spec"] = spec

				err = r.Apply(ctx, client.ApplyConfigurationFromUnstructured(&object), &client.ApplyOptions{FieldManager: "braid"})

				if err != nil {
					l.Error(err, "unable to create Deployment")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		object = unstructured.Unstructured{}
		object.SetGroupVersionKind(groupVersion.WithKind(deployments.Spec.Kind))
		object.SetName(application.Name)
		object.SetNamespace(application.Namespace)
		object.SetLabels(make(map[string]string))
		object.SetAnnotations(make(map[string]string))

		object.Object["spec"] = spec

		err = r.Apply(ctx, client.ApplyConfigurationFromUnstructured(&object), &client.ApplyOptions{FieldManager: "braid"})

		if err != nil {
			l.Error(err, "unable to update Deployment")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func replaceVariables(spec string, variables map[string]string) (interface{}, error) {
	tmpl, err := template.New("object").Option("missingkey=zero").Parse(spec)
	if err != nil {
		return struct{}{}, err
	}
	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, variables)

	if err != nil {
		return struct{}{}, err
	}
	var result interface{}
	err = yaml.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return struct{}{}, err
	}

	return result, nil
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Application{}).
		Named("application").
		Complete(r)
}
