/*
Copyright 2023.
*/

package main

import (
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	ldapClient "github.com/vbouchaud/wellerman/internal/ldap"

	appv1 "github.com/vbouchaud/wellerman/api/v1"
	"github.com/vbouchaud/wellerman/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(appv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// ldap related flags:
	var (
		ldapURL,
		bindDN,
		bindPassword,
		groupSearchBase,
		groupSearchScope,
		groupSearchFilter,
		groupNameProperty string
		groupSearchAttributes []string
	)
	flag.StringVar(&ldapURL, "ldap-url", "", "The address (host and scheme) of the LDAP service.")
	flag.StringVar(&bindDN, "bind-dn", "", "The service account DN to do the ldap search.")
	flag.StringVar(&bindPassword, "bind-password", "", "The service account password to authenticate against the LDAP service.")
	flag.StringVar(&groupSearchBase, "group-search-base", "", "The DN where the ldap search will take place.")
	flag.StringVar(&groupSearchScope, "group-search-scope", ldapClient.ScopeSingleLevel, fmt.Sprintf("The scope of the search. Can take to values base object: '%s', single level: '%s' or whole subtree: '%s'.", ldapClient.ScopeBaseObject, ldapClient.ScopeSingleLevel, ldapClient.ScopeWholeSubtree))
	flag.StringVar(&groupSearchFilter, "group-search-filter", "(&(objectClass=groupOfUniqueNames)(cn=%s))", "The filter to select groups.")
	flag.StringVar(&groupNameProperty, "group-name-property", "cn", "The attribute that contains group names.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ldap := ldapClient.NewInstance(
		ldapURL,
		bindDN,
		bindPassword,
		groupSearchBase,
		groupSearchScope,
		groupSearchFilter,
		groupNameProperty,
		groupSearchAttributes,
	)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "a201cea7.wellerman.bouchaud.org",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.TeamReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Ldap:   ldap,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Team")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
