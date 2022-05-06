package server

import (
	flag "github.com/spf13/pflag"
)

type Option struct {
	MetricsAddr string `json:"metricsAddr,omitempty"`
	ProbeAddr   string `json:"probeAddr,omitempty"`
	ConfigPath  string `json:"configPath,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

func NewOption() *Option {
	return &Option{}
}

func (opts *Option) Bind(fs *flag.FlagSet) {
	fs.StringVar(&opts.MetricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&opts.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.StringVar(&opts.ConfigPath, "config", "./conf/staging.yaml", "The path to configuration file.")
	fs.StringVar(&opts.Namespace, "namespace", "", "Specific namespace to manage or observer.")
}
