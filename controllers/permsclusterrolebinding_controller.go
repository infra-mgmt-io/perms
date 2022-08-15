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
	"k8s.io/apimachinery/pkg/api/meta"
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
	err = r.Get(ctx, types.NamespacedName{Name: permsclusterrolebinding.Name}, bindings)
	if err != nil && errors.IsNotFound(err) {
		// Define a new ClusterRoleBinding
		logger.Info("Creating a new ClusterRolebinding", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name ", permsclusterrolebinding.Name)
		rb := r.clusterRolebindingForPerms(permsclusterrolebinding, ctx)
		err = r.Create(ctx, rb)
		if err != nil {
			logger.Error(err, "Failed to create new ClusterRoleBinding. Check if role exists.", "ClusterRolebinding.Namespace", rb.Namespace, "ClusterRolebinding.Name", rb.Name)
			// Update state and configure progressing
			permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
			meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionTrue,
				Reason:  "Available",
				Message: "Permissions Operator is available",
			})
			meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Progressing",
				Status:  metav1.ConditionTrue,
				Reason:  "Progressing",
				Message: "Permissions Operator tasks are progressing - create PermsClusterRoleBinding",
			})
			meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionTrue,
				Reason:  "Degraded",
				Message: "Permissions Operator task are degraded - create PermsClusterRoleBinding",
			})
			r.updateCountsPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		// Deployment created successfully - update status and requeue
		// Update state and configure progressing
		permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionFalse,
			Reason:  "Progressing",
			Message: "No Permissions Operator tasks are progressing",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionFalse,
			Reason:  "Degraded",
			Message: "No Permissions Operator task are degraded",
		})
		r.updateCountsPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get ClusterRolebinding")
		permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionTrue,
			Reason:  "Degraded",
			Message: "Permissions Operator (create) task are degraded",
		})
		r.updatePermsClusterRoleBindingStatus(ctx, permsclusterrolebinding, req)
		return ctrl.Result{}, err
	}

	// Check, if updates on immutable parts of Clusterrolebinding are configured
	err = r.Get(ctx, req.NamespacedName, permsclusterrolebinding)
	if err != nil {
		logger.Error(err, "Failed to update ClusterRoleBinding - update cache failed", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
	}
	if bindings.RoleRef.Name != permsclusterrolebinding.Spec.Role {
		logger.Error(err, "Update immutable configuration (spec.Role)", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
		permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionTrue,
			Reason:  "Degraded",
			Message: "Permissions Operator task degraded, immutable Spec.Role changed",
		})
		permsclusterrolebinding = r.updatePermsClusterRoleBindingStatus(ctx, permsclusterrolebinding, req)
	} else if !(errors.IsNotFound(err)) {
		permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionFalse,
			Reason:  "Degraded",
			Message: "No Permissions Operator task are degraded",
		})
		permsclusterrolebinding = r.updatePermsClusterRoleBindingStatus(ctx, permsclusterrolebinding, req)
	}

	// Update ClusterRolebinding
	subs := subsForPermsClusterRoleBindings(permsclusterrolebinding)
	err = r.Get(ctx, req.NamespacedName, permsclusterrolebinding)
	if err != nil {
		logger.Error(err, "Failed to update ClusterRolebinding - update cache failed", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
	}
	if !reflect.DeepEqual(bindings.Subjects, subs) {
		logger.Info("Updating ClusterRolebinding", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
		//logger.Info("Debug", "bindings.Subjects", bindings.Subjects, "ClusterRolebinding.Name", ClusterRolebinding.Name)
		//logger.Info("Debug", "subs", subs, "ClusterRolebinding.Name ", ClusterRolebinding.Name)
		// update status
		permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionTrue,
			Reason:  "Progressing",
			Message: "Permissions Operator (update) tasks are progressing",
		})
		bindings.Subjects = subs

		permsclusterrolebinding = r.updatePermsClusterRoleBindingStatus(ctx, permsclusterrolebinding, req)
		err = r.Update(ctx, bindings)
		if err != nil {
			logger.Error(err, "Failed to update ClusterRolebinding", "ClusterRolebinding.Namespace", permsclusterrolebinding.Namespace, "ClusterRolebinding.Name", permsclusterrolebinding.Name)
			permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
			meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionTrue,
				Reason:  "Available",
				Message: "Permissions Operator is available",
			})
			meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Progressing",
				Status:  metav1.ConditionFalse,
				Reason:  "Progressing",
				Message: "No Permissions Operator tasks are progressing",
			})
			meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionTrue,
				Reason:  "Degraded",
				Message: "Permissions Operator (update) task are degraded",
			})
			r.updatePermsClusterRoleBindingStatus(ctx, permsclusterrolebinding, req)
			return ctrl.Result{}, err
		}

		r.updateCountsPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		//return ctrl.Result{Requeue: true}, nil
	} else if !(errors.IsNotFound(err)) {
		permsclusterrolebinding = r.refreshPermsClusterRoleBinding(ctx, permsclusterrolebinding, req)
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsclusterrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionFalse,
			Reason:  "Progressing",
			Message: "No Permissions Operator tasks are progressing",
		})
		r.updatePermsClusterRoleBindingStatus(ctx, permsclusterrolebinding, req)
	}

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

// Update the status
func (r *PermsClusterRoleBindingReconciler) updatePermsClusterRoleBindingStatus(ctx context.Context, p *permsv1beta1.PermsClusterRoleBinding, req ctrl.Request) *permsv1beta1.PermsClusterRoleBinding {
	err := r.Status().Update(ctx, p)
	if err != nil {
		logger.Error(err, "Unable to update Status")
	}
	time.Sleep(5 * time.Second)
	p = r.refreshPermsClusterRoleBinding(ctx, p, req)
	return p
}

// Update the Perms custom ressource status
func (r *PermsClusterRoleBindingReconciler) refreshPermsClusterRoleBinding(ctx context.Context, p *permsv1beta1.PermsClusterRoleBinding, req ctrl.Request) *permsv1beta1.PermsClusterRoleBinding {
	permsclusterrolebinding := &permsv1beta1.PermsClusterRoleBinding{}
	err := r.Get(ctx, req.NamespacedName, permsclusterrolebinding)
	if err != nil {
		logger.Error(err, "Unable to update Cache")
	}
	return permsclusterrolebinding
}

func (r *PermsClusterRoleBindingReconciler) updateCountsPermsClusterRoleBinding(ctx context.Context, p *permsv1beta1.PermsClusterRoleBinding, req ctrl.Request) {
	p = r.refreshPermsClusterRoleBinding(ctx, p, req)
	if p.Status.Count.Users != strconv.Itoa(len(p.Spec.Users)) ||
		p.Status.Count.Groups != strconv.Itoa(len(p.Spec.Groups)) ||
		p.Status.Count.Serviceaccounts != strconv.Itoa(len(p.Spec.Serviceaccounts)) {
		p.Status.Count.Users = strconv.Itoa(len(p.Spec.Users))
		p.Status.Count.Groups = strconv.Itoa(len(p.Spec.Groups))
		p.Status.Count.Serviceaccounts = strconv.Itoa(len(p.Spec.Serviceaccounts))
		r.updatePermsClusterRoleBindingStatus(ctx, p, req)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PermsClusterRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&permsv1beta1.PermsClusterRoleBinding{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}
