/*
Copyright 2021 KML Ares-Operator Authors.

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

package operator

import (
	"fmt"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"ares-operator/cache"
	"ares-operator/cmd/server"
	"ares-operator/conf"
	"ares-operator/controllers"
	"ares-operator/reconciler"
	"ares-operator/reconciler/node"
	//+kubebuilder:scaffold:imports
)

func NewOperatorCommand() *cobra.Command {
	opts := NewOption()
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "The Ares Operator",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Option = server.CommonOptions
			return run(opts, server.OperatorConfig)
		},
	}
	opts.Bind(cmd.Flags())
	return cmd
}

func run(opts *OperatorOption, config *conf.Configuration) error {
	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info(" === [initializing...] === ")
	setupLog.Info(fmt.Sprintf("options: %s", opts.String()))

	// create manager
	restConfig, managerOpts := server.BuildManagerOptions(opts.Option, config)
	managerOpts.LeaderElection = opts.EnableLeaderElection
	mgr, err := ctrl.NewManager(restConfig, managerOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	// setup reconciler
	var c cache.Interface = nil
	if len(config.Cache.URI) > 0 {
		cacheOpts := &cache.CacheOptions{
			URI: config.Cache.URI,
		}
		c, err = cache.New(cacheOpts)
		if err != nil {
			setupLog.Error(err, "failed to create cache")
			return err
		}
	}
	r := &controllers.AresJobReconciler{
		Reconciler: &reconciler.Reconciler{
			Client:          mgr.GetClient(),
			Log:             ctrl.Log.WithName("controllers").WithName("AresJob"),
			Scheme:          mgr.GetScheme(),
			Cache:           c,
			Config:          config,
			NodeManager:     node.NewManager(),
			PodGroupManager: reconciler.NewPodGroupManager(mgr.GetClient()),
		},
	}
	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AresJob")
		return err
	}
	/* Disable Webhook
	if err = (&aresv1.AresJob{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "AresJob")
		os.Exit(1)
	}
	*/
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	r.Reconciler.Close()
	setupLog.Info(" === [terminated] === \n\n\n")
	return nil
}
