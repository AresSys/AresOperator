package operator

import (
	"encoding/json"
	"fmt"

	"ares-operator/cmd/server"

	flag "github.com/spf13/pflag"
)

type OperatorOption struct {
	*server.Option       `json:",inline"`
	EnableLeaderElection bool `json:"enableLeaderElection,omitempty"`
}

func NewOption() *OperatorOption {
	return &OperatorOption{
		Option: server.NewOption(),
	}
}

func (opts *OperatorOption) Bind(fs *flag.FlagSet) {
	fs.BoolVar(&opts.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
}

func (opts *OperatorOption) String() string {
	if content, err := json.Marshal(opts); err == nil {
		return string(content)
	}
	return fmt.Sprintf("%#v", opts)
}
