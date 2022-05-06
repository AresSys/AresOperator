package observer

import (
	"encoding/json"
	"fmt"

	"ares-operator/cmd/server"

	flag "github.com/spf13/pflag"
)

type ObserverOption struct {
	*server.Option `json:",inline"`
	Port           int `json:"port,omitempty"`
}

func NewOption() *ObserverOption {
	return &ObserverOption{
		Option: server.NewOption(),
	}
}

func (opts *ObserverOption) Bind(fs *flag.FlagSet) {
	fs.IntVar(&opts.Port, "port", 80,
		"Port of the server.")
}

func (opts *ObserverOption) String() string {
	if content, err := json.Marshal(opts); err == nil {
		return string(content)
	}
	return fmt.Sprintf("%#v", opts)
}
