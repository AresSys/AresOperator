package observer

import (
	"context"
	"fmt"
	"net/http"

	"ares-operator/cmd/server"
	"ares-operator/conf"
	"ares-operator/observer"
	"ares-operator/observer/handler"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func NewObserverCommand() *cobra.Command {
	opts := NewOption()
	cmd := &cobra.Command{
		Use:   "observer",
		Short: "The Ares Observer",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Option = server.CommonOptions
			return run(opts, server.OperatorConfig)
		},
	}
	opts.Bind(cmd.Flags())
	return cmd
}

func run(opts *ObserverOption, config *conf.Configuration) error {
	setupLog := ctrl.Log.WithName("setup")
	setupLog.Info(" === [initializing...] === ")
	setupLog.Info(fmt.Sprintf("options: %s", opts.String()))
	// create manager
	restConfig, managerOpts := server.BuildManagerOptions(opts.Option, config)
	mgr, err := ctrl.NewManager(restConfig, managerOpts)
	if err != nil {
		setupLog.Error(err, "unable to start observer")
		return err
	}

	// create observer
	r := observer.NewObserver(mgr.GetClient())
	if err = r.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create observer")
		return err
	}
	r.Register()

	// add http server
	httpServer := func(ctx context.Context) error {
		svr := &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", opts.Port),
			Handler: handler.InitRouter(config.Cache.Observer),
		}
		return serve(ctx, svr, setupLog)
	}
	mgr.Add(manager.RunnableFunc(httpServer))

	// start
	setupLog.Info("starting observer")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running observer")
		return err
	}
	setupLog.Info(" === [terminated] === \n\n\n")
	return nil
}

func serve(ctx context.Context, svr *http.Server, log logr.Logger) error {
	errCh := make(chan error, 1)
	// start HTTP Server
	go func() {
		if err := svr.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "failed to ListenAndServe")
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}
