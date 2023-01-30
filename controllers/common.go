package controllers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	conditionInitialized = "Initialized"
	conditionConfigured  = "Configured"
	conditionReconciled  = "Reconciled"
)

func addCondition(l logr.Logger, c *[]metav1.Condition, t string, s metav1.ConditionStatus) {
	l.Info("Setting condition", "status", t, "condition", s)

	meta.SetStatusCondition(c, metav1.Condition{
		Type:   t,
		Status: s,
	})
}
