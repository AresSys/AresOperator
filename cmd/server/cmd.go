package server

import (
	goflag "flag"
	"fmt"

	aresv1 "ares-operator/api/v1"
	"ares-operator/conf"
	"ares-operator/utils"

	"github.com/bombsimon/logrusr"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

var (
	scheme = runtime.NewScheme()

	CommonOptions  *Option
	OperatorConfig *conf.Configuration
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(scheduling.AddToScheme(scheme))
	utilruntime.Must(aresv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func BuildManagerOptions(opts *Option, config *conf.Configuration) (*rest.Config, ctrl.Options) {
	restConfig := ctrl.GetConfigOrDie()
	ctrlOpts := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     opts.MetricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: opts.ProbeAddr,
		LeaderElectionID:       "963c7a0b.kuaishou.com",
		LeaderElection:         false,
		Namespace:              opts.Namespace,
	}
	return restConfig, ctrlOpts
}

func NewServerCommand() *cobra.Command {
	opts := NewOption()
	config := &conf.Configuration{}
	cmd := &cobra.Command{
		Use:           "ares [COMMAND]",
		Short:         "The Ares Family",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// logger
			logdir := "./log"
			utils.InitLogger(logdir)
			ctrl.SetLogger(logrusr.NewLogger(logrus.StandardLogger()))

			// config
			if err := config.Load(opts.ConfigPath); err != nil {
				return err
			}
			fmt.Printf("configurations: %#v\n", config)

			// assign
			CommonOptions = opts
			OperatorConfig = config
			return nil
		},
		Run: nil,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
		},
	}
	opts.Bind(cmd.PersistentFlags())
	//parse glog flags
	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	goflag.CommandLine.Parse([]string{})
	return cmd
}
