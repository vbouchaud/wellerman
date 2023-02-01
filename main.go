/*
Copyright 2023.
*/

package main

import (
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

	"github.com/urfave/cli/v2"

	gitlabClient "github.com/vbouchaud/wellerman/internal/gitlab"
	ldapClient "github.com/vbouchaud/wellerman/internal/ldap"
	"github.com/vbouchaud/wellerman/internal/version"

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

func serve() *cli.Command {
	return &cli.Command{
		Name:     "serve",
		Aliases:  []string{"s"},
		Usage:    "start the operator",
		HideHelp: false,
		Flags: []cli.Flag{
			// Operator default flags
			&cli.StringFlag{
				Name:     "metrics-bind-address",
				Category: "operator related options:",
				Usage:    "The `ADDRESS` the metric endpoint binds to.",
				Value:    ":8080",
			},
			&cli.StringFlag{
				Name:     "health-probe-bind-address",
				Category: "operator related options:",
				Usage:    "The `ADDRESS` the probe endpoint binds to.",
				Value:    ":8081",
			},
			&cli.BoolFlag{
				Name:     "leader-elect",
				Category: "operator related options:",
				Usage:    "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.",
				Value:    false,
			},

			// ldap related flags
			&cli.StringFlag{
				Name:     "ldap-url",
				Category: "ldap related options:",
				Usage:    "The `ADDRESS` the metric endpoint binds to.",
				Value:    "ldap://localhost",
				EnvVars:  []string{"LDAP_ADDR"},
			},
			&cli.StringFlag{
				Name:     "bind-dn",
				Category: "ldap related options:",
				Usage:    "The service account `DN` to do the ldap search.",
				EnvVars:  []string{"LDAP_BINDDN"},
			},
			&cli.StringFlag{
				Name:     "bind-credentials",
				Category: "ldap related options:",
				EnvVars:  []string{"LDAP_BINDCREDENTIALS"},
				FilePath: "/etc/secrets/ldap/password",
				Usage:    "The service account `PASSWORD` to authenticate against the LDAP service., can be located in '/etc/secrets/ldap/password'.",
			},
			&cli.StringFlag{
				Name:     "group-search-base",
				Category: "ldap related options:",
				EnvVars:  []string{"LDAP_GROUP_SEARCHBASE"},
				Usage:    "The `DN` where the ldap search will take place.",
			},
			&cli.StringFlag{
				Name:     "group-search-scope",
				Category: "ldap related options:",
				EnvVars:  []string{"LDAP_GROUP_SEARCHSCOPE"},
				Usage:    fmt.Sprintf("The `SCOPE` of the search. Can take to values base object: '%s', single level: '%s' or whole subtree: '%s'.", ldapClient.ScopeBaseObject, ldapClient.ScopeSingleLevel, ldapClient.ScopeWholeSubtree),
				Value:    ldapClient.ScopeSingleLevel,
			},
			&cli.StringFlag{
				Name:     "group-search-filter",
				Category: "ldap related options:",
				EnvVars:  []string{"LDAP_GROUP_SEARCHFILTER"},
				Usage:    "The `FILTER` to select groups.",
				Value:    "(&(objectClass=groupOfUniqueNames)(cn=%s))",
			},
			&cli.StringFlag{
				Name:     "group-name-property",
				Category: "ldap related options:",
				EnvVars:  []string{"LDAP_GROUP_NAMEPROPERTY"},
				Usage:    "The `PROPERTY` that contains group names.",
				Value:    "cn",
			},

			// gitlab related flags
			&cli.StringFlag{
				Name:     "gitlab-url",
				Category: "gitlab related options:",
				EnvVars:  []string{"GITLAB_URL"},
				Usage:    "The `URL` of the gitlab API.",
			},
			&cli.StringFlag{
				Name:     "gitlab-token",
				Category: "gitlab related options:",
				EnvVars:  []string{"GITLAB_TOKEN"},
				Usage:    "The `TOKEN` to authenticate with.",
			},
		},
		Action: func(c *cli.Context) error {
			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme:                 scheme,
				MetricsBindAddress:     c.String("metrics-bin-address"),
				Port:                   9443,
				HealthProbeBindAddress: c.String("health-probe-bind-address"),
				LeaderElection:         c.Bool("leader-elect"),
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

			ldap := ldapClient.NewInstance(
				c.String("ldap-url"),
				c.String("bind-dn"),
				c.String("bind-credentials"),
				c.String("group-search-base"),
				c.String("group-search-scope"),
				c.String("group-search-filter"),
				c.String("group-name-property"),
				[]string{},
			)
			if err = (&controllers.TeamReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
				Ldap:   ldap,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "Team")
				os.Exit(1)
			}

			gitlab, err := gitlabClient.NewInstance(
				c.String("gitlab-url"),
				c.String("gitlab-token"),
			)
			if err != nil {
				setupLog.Error(err, "unable to create gitlab client")
				os.Exit(1)
			}
			if err = (&controllers.ProjectReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
				Gitlab: gitlab,
			}).SetupWithManager(mgr); err != nil {
				setupLog.Error(err, "unable to create controller", "controller", "Project")
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

			return nil
		},
	}
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(version.Version())
	}

	app := cli.NewApp()
	app.Name = "wellerman"
	app.Usage = "An operator to initialize resources for a custom definition of Project, including vsc, environment entitlements through ldap, vault, etc."
	app.Version = version.VERSION
	app.Compiled = version.Compiled()
	app.Authors = []*cli.Author{
		{
			Name:  "Vianney Bouchaud",
			Email: "vianney@bouchaud.org",
		},
	}

	app.UseShortOptionHandling = true
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "development-mode",
			Value:   false,
			EnvVars: []string{"DEVELOPMENT"},
		},
	}

	app.Before = func(c *cli.Context) error {
		// TODO: probably add some options to configure log level and some other options as operator-sdk do with https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/log/zap#Options.BindFlags
		ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{
			Development: c.Bool("development-mode"),
		})))

		return nil
	}

	app.Commands = []*cli.Command{
		serve(),
	}

	if err := app.Run(os.Args); err != nil {
		setupLog.Error(err, "problem running operator")
		os.Exit(1)
	}
}
