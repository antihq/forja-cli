package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var (
	cfgFile      string
	flagAPIKey   string
	flagEndpoint string
	flagTeam     string
	flagFormat   string
)

type runtime struct {
	stdout io.Writer
	stderr io.Writer
}

// run resolves settings, performs one API call, and renders the response.
func (rt *runtime) run(cmd *cobra.Command, method, path string, query map[string]string) error {
	resolved, api, err := currentSettings(cmd.Root())
	if err != nil {
		return err
	}

	raw, err := api.call(method, path, queryString(query), nil)
	if err != nil {
		return err
	}

	return renderData(rt.stdout, raw, resolved.Format)
}

func queryString(params map[string]string) (query url.Values) {
	query = url.Values{}
	for key, value := range params {
		if value != "" {
			query.Set(key, value)
		}
	}

	return query
}

func currentSettings(root *cobra.Command) (settings, *client, error) {
	resolved, err := resolveSettings(cfgFile, flagOverrides(root))
	if err != nil {
		return settings{}, nil, err
	}

	return resolved, newClient(resolved), nil
}

func flagOverrides(root *cobra.Command) map[string]*string {
	overrides := map[string]*string{}
	collect := func(key, name string) {
		if root.PersistentFlags().Changed(name) {
			value, _ := root.PersistentFlags().GetString(name)
			overrides[key] = &value
		}
	}

	collect("api_key", "api-key")
	collect("endpoint", "endpoint")
	collect("team", "team")
	collect("format", "format")

	return overrides
}

func newRootCLI(out, errOut io.Writer) *cobra.Command {
	rt := &runtime{stdout: out, stderr: errOut}

	root := &cobra.Command{
		Use:   "forja",
		Short: "Manage Forja from the command line",
		Long: `A command line interface for managing Forja servers, sites, and deployments.

Authenticate with a personal API key generated in Forja under Settings > API.

Configuration lives in ~/.forja.yaml:

  api_key: your-personal-api-key
  endpoint: https://forja.example.com
  team: 1
  format: json

Environment variables FORJA_API_KEY, FORJA_ENDPOINT, FORJA_TEAM, and
FORJA_FORMAT override the file. Flags override everything.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := root.PersistentFlags()
	flags.StringVar(&cfgFile, "config", "", "config file (default is ~/.forja.yaml)")
	flags.StringVar(&flagAPIKey, "api-key", "", "Forja personal API key")
	flags.StringVar(&flagEndpoint, "endpoint", "", "Forja base URL (default "+defaultEndpoint+")")
	flags.StringVar(&flagTeam, "team", "", "team id (default your personal team)")
	flags.StringVar(&flagFormat, "format", "", "output format: table or json")

	root.SetOut(out)
	root.SetErr(errOut)

	root.AddCommand(
		newVersionCommand(),
		newWhoamiCommand(rt),
		newServersCommand(rt),
		newSitesCommand(rt),
		newDeploymentsCommand(rt),
	)

	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "forja %s\n", version)
			return err
		},
	}
}

func Execute() {
	root := newRootCLI(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "forja:", err)
		os.Exit(1)
	}
}
