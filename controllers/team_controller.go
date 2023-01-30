/*
Copyright 2023.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/vbouchaud/wellerman/api/v1"
	"github.com/vbouchaud/wellerman/internal/ldap"
)

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Ldap   *ldap.Client
}

const teamFinalizer = "app.heidrun.bouchaud.org/team-finalizer"

//+kubebuilder:rbac:groups=app.wellerman.bouchaud.org,resources=teams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.wellerman.bouchaud.org,resources=teams/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.wellerman.bouchaud.org,resources=teams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling Team.")

	// Fetch the Team instance
	team := &appv1.Team{}
	err := r.Get(ctx, req.NamespacedName, team)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Team resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Team.")
		return ctrl.Result{}, err
	}

	// Team deletion
	isTeamMarkedToBeDeleted := team.GetDeletionTimestamp() != nil
	if isTeamMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(team, teamFinalizer) {
			if err := r.Ldap.DeleteGroup(team.Name); err != nil {
				if ldap.IsNotFound(err) {
					logger.Info("Ldap group not found.", "ldap-group", team.Name)
				} else {
					logger.Error(err, "Error while removing group.", "ldap-group", team.Name)
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(team, teamFinalizer)
			if err = r.Update(ctx, team); err != nil {
				logger.Error(err, "Failed to remove finalizer.", "ldap-group", team.Name)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Team Initialization
	if !controllerutil.ContainsFinalizer(team, teamFinalizer) {
		controllerutil.AddFinalizer(team, teamFinalizer)
		addCondition(logger, &team.Status.Conditions, conditionConfigured, metav1.ConditionFalse)
		if err = r.Update(ctx, team); err != nil {
			logger.Error(err, "Failed to initialize Team status.", "ldap-group", team.Name)
			return ctrl.Result{}, err
		}
	}

	// Team update
	changed := false

	team.Status.DistinguishedName, err, changed = r.Ldap.ReconcileGroup(team)
	if err != nil {
		logger.Error(err, "Failed to crupdate Team resource.", "ldap-group", team.Name)
		return ctrl.Result{}, err
	}

	if changed {
		addCondition(logger, &team.Status.Conditions, conditionConfigured, metav1.ConditionTrue)
		if err = r.Update(ctx, team); err != nil {
			logger.Error(err, "Failed to update Team status.", "ldap-group", team.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Team{}).
		Complete(r)
}
