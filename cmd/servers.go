package cmd

import (
	"github.com/spf13/cobra"
)

func newServersCommand(rt *runtime) *cobra.Command {
	command := &cobra.Command{
		Use:   "servers",
		Short: "Manage servers",
	}

	command.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List servers",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return rt.run(cmd, "GET", "/servers", nil)
			},
		},
		&cobra.Command{
			Use:   "get <id>",
			Short: "Show one server",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return rt.run(cmd, "GET", "/servers/"+args[0], nil)
			},
		},
	)

	return command
}
