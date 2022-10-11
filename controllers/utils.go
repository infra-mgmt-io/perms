package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// helper to set the "Available" status
func setEverythingIsFineStatus(ctx context.Context, conditions *[]metav1.Condition) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "Available",
		Message: "Permissions Operator is available",
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Progressing",
		Status:  metav1.ConditionFalse,
		Reason:  "Progressing",
		Message: "No Permissions Operator tasks are progressing",
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Degraded",
		Status:  metav1.ConditionFalse,
		Reason:  "Degraded",
		Message: "No Permissions Operator task are degraded",
	})
}

// helper to set the "Degraded" status
func setHoustonWeHaveAProblemStatus(ctx context.Context, conditions *[]metav1.Condition) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionFalse,
		Reason:  "Unavailable",
		Message: "Permissions Operator is unavailable",
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Progressing",
		Status:  metav1.ConditionFalse,
		Reason:  "Progressing",
		Message: "No Permissions Operator tasks are progressing",
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Degraded",
		Status:  metav1.ConditionTrue,
		Reason:  "Degraded",
		Message: "Permissions Operator task are degraded",
	})
}

// helper to set the "Progressing" status
func setProgressingStatus(ctx context.Context, conditions *[]metav1.Condition) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Available",
		Status:  metav1.ConditionTrue,
		Reason:  "Available",
		Message: "Permissions Operator is available",
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Progressing",
		Status:  metav1.ConditionTrue,
		Reason:  "Progressing",
		Message: "Permissions Operator tasks are progressing",
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Degraded",
		Status:  metav1.ConditionFalse,
		Reason:  "Degraded",
		Message: "No Permissions Operator task are degraded",
	})
}
