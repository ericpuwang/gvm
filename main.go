package main

import (
	"github.com/periky/gvm/command"
	"github.com/spf13/cobra"
)

func main() {
	cmd := cobra.Command{
		Use:  "gvm",
		Long: "GVM is the Go Version Manager",
	}

	cmd.AddCommand(command.List())
	cmd.AddCommand(command.Install())
	cmd.AddCommand(command.Use())
	cmd.AddCommand(command.UnInstall())

	cmd.Execute()
}
