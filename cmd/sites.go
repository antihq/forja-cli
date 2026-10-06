package cmd

import (
	"fmt"
	"strings"

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

	var (
		server       string
		address      string
		phpVersion   string
		siteType     string
		webFolder    string
		zeroDowntime bool
		repository   string
		branch       string
		deployKey    bool
	)

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a site on a server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			required := []struct{ placeholder, value string }{
				{"--server <id>", server},
				{"--address <domain>", address},
				{"--php-version <version>", phpVersion},
				{"--type <type>", siteType},
			}
			var missing []string
			for _, each := range required {
				if each.value == "" {
					missing = append(missing, each.placeholder)
				}
			}
			if len(missing) > 0 {
				return fmt.Errorf("sites create requires %s. Example: forja sites create --server 01abc --address example.com --php-version php85 --type laravel", strings.Join(missing, ", "))
			}

			payload := map[string]any{
				"address":                  address,
				"php_version":              phpVersion,
				"type":                     siteType,
				"web_folder":               webFolder,
				"zero_downtime_deployment": zeroDowntime,
			}
			if repository != "" {
				payload["repository_url"] = repository
			}
			if branch != "" {
				payload["repository_branch"] = branch
			}
			if deployKey {
				payload["use_deploy_key"] = true
			}

			return rt.run(cmd, "POST", "/servers/"+server+"/sites", nil, payload)
		},
	}
	create.Flags().StringVar(&server, "server", "", "server id to create the site on")
	create.Flags().StringVar(&address, "address", "", "site domain, e.g. example.com")
	create.Flags().StringVar(&phpVersion, "php-version", "", "php version id, e.g. php85")
	create.Flags().StringVar(&siteType, "type", "", "site type: generic, laravel, static, or wordpress")
	create.Flags().StringVar(&webFolder, "web-folder", "public", "web folder inside the site root")
	create.Flags().BoolVar(&zeroDowntime, "zero-downtime", true, "use zero-downtime deployment")
	create.Flags().StringVar(&repository, "repository", "", "repository url to clone")
	create.Flags().StringVar(&branch, "branch", "", "repository branch to deploy")
	create.Flags().BoolVar(&deployKey, "deploy-key", false, "mint a deploy key instead of starting the first deploy")

	command.AddCommand(list, get, deploy, create, newSitesSettingsCommand(rt))

	return command
}
