package cmd

import (
	"github.com/spf13/cobra"
)

func newSitesCommand(rt *runtime) *cobra.Command {
	var serverID string

	command := &cobra.Command{
		Use:   "sites",
		Short: "Manage sites",
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List sites, optionally for one server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return rt.run(cmd, "GET", "/sites", map[string]string{"server_id": serverID})
		},
	}
	list.Flags().StringVar(&serverID, "server", "", "list sites of this server id only")

	get := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one site",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.run(cmd, "GET", "/sites/"+args[0], nil)
		},
	}

	deploy := &cobra.Command{
		Use:   "deploy <id>",
		Short: "Trigger a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.run(cmd, "POST", "/sites/"+args[0]+"/deploy", nil)
		},
	}

	command.AddCommand(list, get, deploy)

	return command
}
