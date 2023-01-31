/*
Copyright 2023.
*/

package controllers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/vbouchaud/wellerman/api/v1"
	"github.com/vbouchaud/wellerman/internal/gitlab"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Gitlab *gitlab.Client
}

const projectFinalizer = "app.heidrun.bouchaud.org/project-finalizer"

func projectPathDifference(a, b []appv1.ProjectPath) (diff []appv1.ProjectPath) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item.Path] = true
	}

	for _, item := range a {
		if _, ok := m[item.Path]; !ok {
			diff = append(diff, item)
		}
	}

	return
}

//+kubebuilder:rbac:groups=app.wellerman.bouchaud.org,resources=projects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.wellerman.bouchaud.org,resources=projects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.wellerman.bouchaud.org,resources=projects/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling Project.")

	// Fetch the Project instance
	project := &appv1.Project{}
	err := r.Get(ctx, req.NamespacedName, project)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Project resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Project.")
		return ctrl.Result{}, err
	}

	// Project deletion
	isProjectMarkedToBeDeleted := project.GetDeletionTimestamp() != nil
	if isProjectMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(project, projectFinalizer) {
			for _, projectPath := range project.Spec.Paths {
				if !projectPath.External {
					if err, _ /* changed */ := r.Gitlab.DeleteProject(projectPath); err != nil {
						logger.Error(err, "Error while removing gitlab project.", "project", project.Name, "project-path", projectPath.Name)
						return ctrl.Result{}, err
					}
				}
			}

			controllerutil.RemoveFinalizer(project, projectFinalizer)
			if err = r.Update(ctx, project); err != nil {
				logger.Error(err, "Failed to remove finalizer.", "project", project.Name)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Project Initialization
	if !controllerutil.ContainsFinalizer(project, projectFinalizer) {
		controllerutil.AddFinalizer(project, projectFinalizer)
		addCondition(logger, &project.Status.Conditions, conditionConfigured, metav1.ConditionFalse)
		if err = r.Update(ctx, project); err != nil {
			logger.Error(err, "Failed to initialize Project status.", "project", project.Name)
			return ctrl.Result{}, err
		}
	}

	// Project update
	changed := false

	var lastAppliedConfig appv1.Project
	// lastAppliedConfig should only be used to detect gitlab projects that the operator should delete. Might not be enough.
	if err = json.Unmarshal([]byte(project.GetObjectMeta().GetAnnotations()[v1.LastAppliedConfigAnnotation]), &lastAppliedConfig); err == nil {
		for _, projectPath := range projectPathDifference(project.Spec.Paths, lastAppliedConfig.Spec.Paths) {
			if !projectPath.External {
				err, pathChanged := r.Gitlab.DeleteProject(projectPath)
				if err != nil {
					logger.Error(err, "Error while removing gitlab project.", "project", project.Name, "project-path", projectPath.Name)
					return ctrl.Result{}, err
				}
				changed = pathChanged || changed
			}
		}
	}

	for _, projectPath := range project.Spec.Paths {
		if !projectPath.External {
			var pathChanged bool
			err, pathChanged = r.Gitlab.ReconcileProject(projectPath)
			if err != nil {
				logger.Error(err, "Failed to crupdate Project resource.", "project", project.Name, "project-path", projectPath.Name)
				return ctrl.Result{}, err
			}

			changed = pathChanged || changed
		}
	}

	if changed {
		addCondition(logger, &project.Status.Conditions, conditionConfigured, metav1.ConditionTrue)
		if err = r.Update(ctx, project); err != nil {
			logger.Error(err, "Failed to update Project status.", "ldap-group", project.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Project{}).
		Complete(r)
}
