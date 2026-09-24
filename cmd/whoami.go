package cmd

import "github.com/spf13/cobra"

func newWhoamiCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the signed-in user and teams",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return rt.run(cmd, "GET", "/me", nil)
		},
	}
}
