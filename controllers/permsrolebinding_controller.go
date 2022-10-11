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

// PermsRoleBindingReconciler reconciles a Perms object
type PermsRoleBindingReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var logger logr.Logger

//+kubebuilder:rbac:groups=perms.infra-mgmt.io,resources=permsrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=perms.infra-mgmt.io,resources=permsrolebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=perms.infra-mgmt.io,resources=permsrolebindings/finalizers,verbs=update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete;bind

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *PermsRoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger = log.FromContext(ctx)

	// Verify if a CRD of Permissions exists
	permsrolebinding := &permsv1beta1.PermsRoleBinding{}
	err := r.Get(ctx, req.NamespacedName, permsrolebinding)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Resource PermsRoleBinding not found.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get PermsRoleBinding")
		return ctrl.Result{}, err
	}

	// Check if the binding already exists, if not create a new one
	bindings := &rbacv1.RoleBinding{}
	if bindingExistsErr := r.Get(ctx, types.NamespacedName{Name: permsrolebinding.Name, Namespace: permsrolebinding.Namespace}, bindings); bindingExistsErr != nil {
		logger.Info("Creating a new Rolebinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name ", permsrolebinding.Name)
		// Define a new RoleBinding
		rb := r.rolebindingForPerms(permsrolebinding, ctx)
		setProgressingStatus(ctx, &permsrolebinding.Status.Conditions)
		if err = r.Create(ctx, rb); err != nil {
			logger.Error(err, "Failed to create RoleBinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
			setHoustonWeHaveAProblemStatus(ctx, &permsrolebinding.Status.Conditions)
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
	} else {
		// Check, if updates on immutable parts of rolebinding are configured
		// if so - leave the reconcile loop
		if bindings.RoleRef.Kind != permsrolebinding.Spec.Kind || bindings.RoleRef.Name != permsrolebinding.Spec.Role {
			logger.Error(err, "Update immutable configuration (spec.kind || spec.Role)", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
			setHoustonWeHaveAProblemStatus(ctx, &permsrolebinding.Status.Conditions)
			if updateErr := r.Status().Update(ctx, permsrolebinding); updateErr != nil {
				logger.Error(updateErr, "Update rolebinding status failed")
			}
			return ctrl.Result{Requeue: false}, err
		}
		// Update rolebinding if possible
		subs := subsForPermsRoleBindings(permsrolebinding)
		if !reflect.DeepEqual(bindings.Subjects, subs) {
			logger.Info("Updating rolebinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
			setProgressingStatus(ctx, &permsrolebinding.Status.Conditions)
			bindings.Subjects = subs
			if err := r.Update(ctx, bindings); err != nil {
				logger.Error(err, "Failed to update RoleBinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
				setHoustonWeHaveAProblemStatus(ctx, &permsrolebinding.Status.Conditions)
				return ctrl.Result{}, err
			}
		}
	}

	// update the Resource Status
	r.updateCountsPermsRoleBinding(ctx, permsrolebinding, req)
	setEverythingIsFineStatus(ctx, &permsrolebinding.Status.Conditions)
	if updateErr := r.Status().Update(ctx, permsrolebinding); updateErr != nil {
		logger.Error(updateErr, "Update rolebinding status failed")
	}
	// return nil to stop reconcile loop
	return ctrl.Result{}, nil
}

// rolebindingForPerms returns a Rolebinding object
func (r *PermsRoleBindingReconciler) rolebindingForPerms(p *permsv1beta1.PermsRoleBinding, ctx context.Context) *rbacv1.RoleBinding {
	// define labels
	labels := labelsForPermsRoleBindings(p.Name)
	subs := subsForPermsRoleBindings(p)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"infra-mgmt.io/perms": "operator-created",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     p.Spec.Kind,
			Name:     p.Spec.Role,
		},
	}
	rb.Subjects = subs

	// Set rolebindingForPermissions instance as the owner and controller
	if err := ctrl.SetControllerReference(p, rb, r.Scheme); err != nil {
		logger.Error(err, "Failed to set as owner")
	}
	return rb
}

// Function returns the labels for selecting the resources
func labelsForPermsRoleBindings(name string) map[string]string {
	return map[string]string{"crd": "PermsRoleBinding", "permsrolebinding_cr": name}
}

// Function returns the subjects for rolebinding
func subsForPermsRoleBindings(p *permsv1beta1.PermsRoleBinding) []rbacv1.Subject {
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

// compare "Status" with "Spec" and update the Status if needed
func (r *PermsRoleBindingReconciler) updateCountsPermsRoleBinding(ctx context.Context, p *permsv1beta1.PermsRoleBinding, req ctrl.Request) {
	if p.Status.Count.Users != strconv.Itoa(len(p.Spec.Users)) ||
		p.Status.Count.Groups != strconv.Itoa(len(p.Spec.Groups)) ||
		p.Status.Count.Serviceaccounts != strconv.Itoa(len(p.Spec.Serviceaccounts)) {

		p.Status.Count.Users = strconv.Itoa(len(p.Spec.Users))
		p.Status.Count.Groups = strconv.Itoa(len(p.Spec.Groups))
		p.Status.Count.Serviceaccounts = strconv.Itoa(len(p.Spec.Serviceaccounts))

	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PermsRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&permsv1beta1.PermsRoleBinding{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
