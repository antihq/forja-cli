package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeploymentsCommand(rt *runtime) *cobra.Command {
	var siteID string

	command := &cobra.Command{
		Use:   "deployments",
		Short: "Manage deployments",
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List deployments for a site",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if siteID == "" {
				return fmt.Errorf("deployments list requires --site <id>. Example: forja deployments list --site 01abc")
			}

			return rt.run(cmd, "GET", "/sites/"+siteID+"/deployments", nil)
		},
	}
	list.Flags().StringVar(&siteID, "site", "", "site id to list deployments for")

	get := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rt.run(cmd, "GET", "/deployments/"+args[0], nil)
		},
	}

	command.AddCommand(list, get)

	return command
}
