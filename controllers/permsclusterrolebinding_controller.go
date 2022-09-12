/*
Copyright 2022.

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
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	permsv1beta1 "github.com/infra-mgmt-io/perms/api/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PermsClusterRoleBindingReconciler reconciles a Perms object
type PermsClusterRoleBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//var logger logr.Logger

//+kubebuilder:rbac:groups=perms.infra-mgmt.io,resources=permsclusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=perms.infra-mgmt.io,resources=permsclusterrolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=perms.infra-mgmt.io,resources=permsclusterrolebindings/finalizers,verbs=update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete;bind
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete;bind

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *PermsClusterRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger = log.FromContext(ctx)

	// "Verify if a CRD of Permissions exists"
	permsclusterrolebinding := &permsv1beta1.PermsClusterRoleBinding{}
	err := r.Get(ctx, req.NamespacedName, permsclusterrolebinding)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource PermsClusterRoleBinding not found.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get PermsClusterRoleBinding")
		return ctrl.Result{}, err
	}

	// Check if the binding already exists, if not create a new one
	bindings := &rbacv1.ClusterRoleBinding{}
	if clusterBindingExistsErr := r.Get(ctx, types.NamespacedName{Name: permsclusterrolebinding.Name}, bindings); clusterBindingExistsErr != nil {
		logger.Info("Creating a new ClusterRolebinding", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name ", permsclusterrolebinding.Name)
		// Define a new ClusterRoleBinding
		rb := r.clusterRolebindingForPerms(permsclusterrolebinding, ctx)
		setProgressingStatus(ctx, &permsclusterrolebinding.Status.Conditions)
		if err = r.Create(ctx, rb); err != nil {
			logger.Error(err, "Failed to create new ClusterRoleBinding. Check if role exists.", "ClusterRolebinding.Namespace", rb.Namespace, "ClusterRolebinding.Name", rb.Name)
			setHoustonWeHaveAProblemStatus(ctx, &permsclusterrolebinding.Status.Conditions)
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}

	} else {
		// Check, if updates on immutable parts of rolebinding are configured
		// if so - leave the reconcile loop
		if bindings.RoleRef.Name != permsclusterrolebinding.Spec.Role {
			logger.Error(err, "Update immutable configuration (spec.Role)", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
			setHoustonWeHaveAProblemStatus(ctx, &permsclusterrolebinding.Status.Conditions)
			if updateErr := r.Status().Update(ctx, permsclusterrolebinding); updateErr != nil {
				logger.Error(updateErr, "Update rolebinding status failed")
			}
			return ctrl.Result{Requeue: false}, err
		}
		// Update ClusterRolebinding
		subs := subsForPermsClusterRoleBindings(permsclusterrolebinding)

		if !reflect.DeepEqual(bindings.Subjects, subs) {
			logger.Info("Updating ClusterRolebinding", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
			setProgressingStatus(ctx, &permsclusterrolebinding.Status.Conditions)
			bindings.Subjects = subs
			if err := r.Update(ctx, bindings); err != nil {
				logger.Error(err, "Failed to update ClusterRolebinding", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
				setHoustonWeHaveAProblemStatus(ctx, &permsclusterrolebinding.Status.Conditions)
				return ctrl.Result{}, err
			}
		}
	}

	// update the Resource Status
	r.updateCountsPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
	setEverythingIsFineStatus(ctx, &permsclusterrolebinding.Status.Conditions)
	if updateErr := r.Status().Update(ctx, permsclusterrolebinding); updateErr != nil {
		logger.Error(updateErr, "Update rolebinding status failed")
	}
	// return nil to stop reconcile loop
	return ctrl.Result{}, nil
}

// ClusterrolebindingForPerms returns a ClusterRolebinding object
func (r *PermsClusterRoleBindingReconciler) clusterRolebindingForPerms(p *permsv1beta1.PermsClusterRoleBinding, ctx context.Context) *rbacv1.ClusterRoleBinding {
	//logger := log.FromContext(ctx)

	// define labels
	labels := labelsForPermsClusterRoleBindings(p.Name)
	subs := subsForPermsClusterRoleBindings(p)

	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   p.Name,
			Labels: labels,
			Annotations: map[string]string{
				"infra-mgmt.io/perms": "operator-created",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     p.Spec.Role,
		},
	}
	rb.Subjects = subs

	// Set ClusterRolebindingForPermissions instance as the owner and controller
	ctrl.SetControllerReference(p, rb, r.Scheme)
	return rb
}

// Function returns the labels for selecting the resources
func labelsForPermsClusterRoleBindings(name string) map[string]string {
	return map[string]string{"crd": "PermsClusterRoleBinding", "permsclusterrolebinding_cr": name}
}

// Function returns the subjects for ClusterRolebinding
func subsForPermsClusterRoleBindings(p *permsv1beta1.PermsClusterRoleBinding) []rbacv1.Subject {
	subs := make([]rbacv1.Subject, len(p.Spec.Groups))
	for i, group := range p.Spec.Groups {
		subs[i] = rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Group",
			Name:     group,
		}
	}

	subUsers := make([]rbacv1.Subject, len(p.Spec.Users))
	for i, user := range p.Spec.Users {
		subUsers[i] = rbacv1.Subject{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "User",
			Name:     user,
		}
		subs = append(subs, subUsers[i])
	}

	subServiceaccounts := make([]rbacv1.Subject, len(p.Spec.Serviceaccounts))
	for i, serviceaccounts := range p.Spec.Serviceaccounts {
		subServiceaccounts[i] = rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      serviceaccounts.Name,
			Namespace: serviceaccounts.Namespace,
		}
		subs = append(subs, subServiceaccounts[i])
	}
	return subs
}

func (r *PermsClusterRoleBindingReconciler) updateCountsPermsClusterRoleBinding(ctx context.Context, p *permsv1beta1.PermsClusterRoleBinding, req ctrl.Request) {
	if p.Status.Count.Users != strconv.Itoa(len(p.Spec.Users)) ||
		p.Status.Count.Groups != strconv.Itoa(len(p.Spec.Groups)) ||
		p.Status.Count.Serviceaccounts != strconv.Itoa(len(p.Spec.Serviceaccounts)) {

		p.Status.Count.Users = strconv.Itoa(len(p.Spec.Users))
		p.Status.Count.Groups = strconv.Itoa(len(p.Spec.Groups))
		p.Status.Count.Serviceaccounts = strconv.Itoa(len(p.Spec.Serviceaccounts))

	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PermsClusterRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&permsv1beta1.PermsClusterRoleBinding{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}
