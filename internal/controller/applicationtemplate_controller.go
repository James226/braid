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

    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    logf "sigs.k8s.io/controller-runtime/pkg/log"

    v1 "github.com/james226/braid/api/v1"
)

// ApplicationTemplateReconciler reconciles a ApplicationTemplate object
type ApplicationTemplateReconciler struct {
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
func (r *ApplicationTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    l := logf.FromContext(ctx)

    l.Info("Reconcile request", "namespace", req.Namespace, "name", req.Name)

    var application v1.ApplicationTemplate
    err := r.Get(ctx, req.NamespacedName, &application)

    if err != nil {
        l.Error(err, "unable to fetch ApplicationTemplate")
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    for _, application := range application.GetOwnerReferences() {
        l.Info("Owner Reference", "APIVersion", application.APIVersion, "Kind", application.Kind, "Name", application.Name)
    }

    return ctrl.Result{}, nil
}

func (r *ApplicationTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1.ApplicationTemplate{}).
        Named("applicationtemplate").
        Complete(r)
}
