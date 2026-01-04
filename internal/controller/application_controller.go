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
	"encoding/json"
	"fmt"
	"text/template"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/james226/braid/api/v1"
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
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
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
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var template v1.ApplicationTemplate
	err = r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Spec.Template,
	}, &template)

	if err != nil {
		l.Error(err, "unable to fetch Application Template")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var deployments v1.ObjectTemplate
	err = r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      template.Spec.Objects[0].Template,
	}, &deployments)

	if err != nil {
		l.Error(err, "unable to fetch Deployment Template")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	variables := make(map[string]string)

	for k, v := range template.Spec.Objects[0].Variables {
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
	//object.SetGroupVersionKind(schema.GroupVersionKind{
	//	Kind:    "Deployment",
	//	Version: "v1",
	//	Group:   "apps",
	//})

	//var deployment appsv1.Deployment
	name := types.NamespacedName{Namespace: application.Namespace, Name: application.Name}
	err = r.Get(ctx, name, &object)

	if err != nil {
		if errors.IsNotFound(err) {

			object.SetName(application.Name)
			object.SetNamespace(application.Namespace)
			object.SetLabels(make(map[string]string))
			object.SetAnnotations(make(map[string]string))

			object.Object["spec"] = spec

			err = r.Create(ctx, &object)

			if err != nil {
				l.Error(err, "unable to create Deployment")
				// we'll ignore not-found errors, since they can't be fixed by an immediate
				// requeue (we'll need to wait for a new notification), and we can get them
				// on deleted requests.
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	foo1, err := json.Marshal(object.Object["spec"])

	test := MergeMaps(object.Object["spec"].(map[string]interface{}), spec.(map[string]interface{}))
	foo, err := json.Marshal(test)
	fmt.Println(foo, foo1)
	err = r.Update(ctx, &object)

	if err != nil {
		l.Error(err, "unable to update Deployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	return ctrl.Result{}, nil
}

func MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = MergeMaps(bv, v)
					continue
				}
			}
		}
		if s, ok := v.([]map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				out[k] = MergeSlices(bv.([]map[string]interface{}), s)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func MergeSlices(a, b []map[string]interface{}) []map[string]interface{} {
	var i int
	var result []map[string]interface{}

	for i = 0; i < len(a) && i < len(b); i++ {
		result = append(result, MergeMaps(a[i], b[i]))
	}

	return result
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

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Application{}).
		Named("application").
		Complete(r)
}
