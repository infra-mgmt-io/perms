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

	// "Verify if a CRD of Permissions exists"
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
	err = r.Get(ctx, types.NamespacedName{Name: permsrolebinding.Name, Namespace: permsrolebinding.Namespace}, bindings)
	if err != nil && errors.IsNotFound(err) {
		// Define a new RoleBinding
		logger.Info("Creating a new Rolebinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name ", permsrolebinding.Name)
		rb := r.rolebindingForPerms(permsrolebinding, ctx)
		err = r.Create(ctx, rb)
		if err != nil {
			logger.Error(err, "Failed to create new RoleBinding. Check if role exists.", "Rolebinding.Namespace", rb.Namespace, "Rolebinding.Name", rb.Name)
			// Update state and configure progressing
			permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
			meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionTrue,
				Reason:  "Available",
				Message: "Permissions Operator is available",
			})
			meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Progressing",
				Status:  metav1.ConditionTrue,
				Reason:  "Progressing",
				Message: "Permissions Operator tasks are progressing - create PermsRoleBinding",
			})
			meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionTrue,
				Reason:  "Degraded",
				Message: "Permissions Operator task are degraded - create PermsRoleBinding",
			})
			r.updateCountsPermsRoleBinding(ctx, permsrolebinding, req)
			//r.updateStatus(ctx, permsrolebinding, req)
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		// Deployment created successfully - update status and requeue
		// Update state and configure progressing
		permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionFalse,
			Reason:  "Progressing",
			Message: "No Permissions Operator tasks are progressing",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionFalse,
			Reason:  "Degraded",
			Message: "No Permissions Operator task are degraded",
		})
		r.updateCountsPermsRoleBinding(ctx, permsrolebinding, req)
		//r.updateStatus(ctx, permsrolebinding, req)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Rolebinding")
		permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionTrue,
			Reason:  "Degraded",
			Message: "Permissions Operator (create) task are degraded",
		})
		r.updateStatus(ctx, permsrolebinding, req)
		return ctrl.Result{}, err
	}

	// Check, if updates on immutable parts of rolebinding are configured
	err = r.Get(ctx, req.NamespacedName, permsrolebinding)
	if err != nil {
		logger.Error(err, "Failed to update RoleBinding - update cache failed", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
	}
	if bindings.RoleRef.Kind != permsrolebinding.Spec.Kind || bindings.RoleRef.Name != permsrolebinding.Spec.Role {
		logger.Error(err, "Update immutable configuration (spec.kind || spec.Role)", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
		permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionTrue,
			Reason:  "Degraded",
			Message: "Permissions Operator task degraded, immutable Spec.Kind || Spec.Role changed",
		})
		permsrolebinding = r.updateStatus(ctx, permsrolebinding, req)
	} else if !(errors.IsNotFound(err)) {
		permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Degraded",
			Status:  metav1.ConditionFalse,
			Reason:  "Degraded",
			Message: "No Permissions Operator task are degraded",
		})
		permsrolebinding = r.updateStatus(ctx, permsrolebinding, req)
	}

	// Update rolebinding
	subs := subsForPermsRoleBindings(permsrolebinding)
	err = r.Get(ctx, req.NamespacedName, permsrolebinding)
	if err != nil {
		logger.Error(err, "Failed to update RoleBinding - update cache failed", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
	}
	if !reflect.DeepEqual(bindings.Subjects, subs) {
		logger.Info("Updating rolebinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
		//logger.Info("Debug", "bindings.Subjects", bindings.Subjects, "Rolebinding.Name", Rolebinding.Name)
		//logger.Info("Debug", "subs", subs, "Rolebinding.Name ", Rolebinding.Name)
		// update status
		permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionTrue,
			Reason:  "Progressing",
			Message: "Permissions Operator (update) tasks are progressing",
		})
		bindings.Subjects = subs

		permsrolebinding = r.updateStatus(ctx, permsrolebinding, req)
		err = r.Update(ctx, bindings)
		if err != nil {
			logger.Error(err, "Failed to update RoleBinding", "Rolebinding.Namespace", permsrolebinding.Namespace, "Rolebinding.Name", permsrolebinding.Name)
			permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
			meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Available",
				Status:  metav1.ConditionTrue,
				Reason:  "Available",
				Message: "Permissions Operator is available",
			})
			meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Progressing",
				Status:  metav1.ConditionFalse,
				Reason:  "Progressing",
				Message: "No Permissions Operator tasks are progressing",
			})
			meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
				Type:    "Degraded",
				Status:  metav1.ConditionTrue,
				Reason:  "Degraded",
				Message: "Permissions Operator (update) task are degraded",
			})
			r.updateStatus(ctx, permsrolebinding, req)
			return ctrl.Result{}, err
		}

		r.updateCountsPermsRoleBinding(ctx, permsrolebinding, req)
		//return ctrl.Result{Requeue: true}, nil
	} else if !(errors.IsNotFound(err)) {
		permsrolebinding = r.refreshPermsRoleBinding(ctx, permsrolebinding, req)
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "Available",
			Message: "Permissions Operator is available",
		})
		meta.SetStatusCondition(&permsrolebinding.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionFalse,
			Reason:  "Progressing",
			Message: "No Permissions Operator tasks are progressing",
		})
		r.updateStatus(ctx, permsrolebinding, req)
	}

	return ctrl.Result{}, nil
}

// rolebindingForPerms returns a Rolebinding object
func (r *PermsRoleBindingReconciler) rolebindingForPerms(p *permsv1beta1.PermsRoleBinding, ctx context.Context) *rbacv1.RoleBinding {
	//logger := log.FromContext(ctx)

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
	ctrl.SetControllerReference(p, rb, r.Scheme)
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

// Update the status
func (r *PermsRoleBindingReconciler) updateStatus(ctx context.Context, p *permsv1beta1.PermsRoleBinding, req ctrl.Request) *permsv1beta1.PermsRoleBinding {
	err := r.Status().Update(ctx, p)
	if err != nil {
		logger.Error(err, "Unable to update Status")
	}
	time.Sleep(5 * time.Second)
	p = r.refreshPermsRoleBinding(ctx, p, req)
	return p
}

// Update the Perms custom ressource status
func (r *PermsRoleBindingReconciler) refreshPermsRoleBinding(ctx context.Context, p *permsv1beta1.PermsRoleBinding, req ctrl.Request) *permsv1beta1.PermsRoleBinding {
	permsrolebinding := &permsv1beta1.PermsRoleBinding{}
	err := r.Get(ctx, req.NamespacedName, permsrolebinding)
	if err != nil {
		logger.Error(err, "Unable to update Cache")
	}
	return permsrolebinding
}

func (r *PermsRoleBindingReconciler) updateCountsPermsRoleBinding(ctx context.Context, p *permsv1beta1.PermsRoleBinding, req ctrl.Request) {
	p = r.refreshPermsRoleBinding(ctx, p, req)
	if p.Status.Count.Users != strconv.Itoa(len(p.Spec.Users)) ||
		p.Status.Count.Groups != strconv.Itoa(len(p.Spec.Groups)) ||
		p.Status.Count.Serviceaccounts != strconv.Itoa(len(p.Spec.Serviceaccounts)) {
		p.Status.Count.Users = strconv.Itoa(len(p.Spec.Users))
		p.Status.Count.Groups = strconv.Itoa(len(p.Spec.Groups))
		p.Status.Count.Serviceaccounts = strconv.Itoa(len(p.Spec.Serviceaccounts))
		r.updateStatus(ctx, p, req)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PermsRoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&permsv1beta1.PermsRoleBinding{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}
