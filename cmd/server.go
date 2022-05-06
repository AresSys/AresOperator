package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ares-operator/cmd/observer"
	"ares-operator/cmd/operator"
	"ares-operator/cmd/server"
)

func main() {
	commands := []*cobra.Command{
		operator.NewOperatorCommand(),
		observer.NewObserverCommand(),
	}
	command := server.NewServerCommand()
	for _, cmd := range commands {
		command.AddCommand(cmd)
	}
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error occurred: %+v\n", err)
		os.Exit(1)
	}
}
