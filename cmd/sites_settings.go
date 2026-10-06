package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// settingsFields is the single list of editable deployment settings. It
// drives the set flags, the --stdin key whitelist, and the guard messages.
var settingsFields = []struct {
	flag string
	key  string
	kind string
}{
	{"shared-directory", "shared_directories", "list"},
	{"shared-file", "shared_files", "list"},
	{"writeable-directory", "writeable_directories", "list"},
	{"hook-before-updating-repository", "hook_before_updating_repository", "hook"},
	{"hook-after-updating-repository", "hook_after_updating_repository", "hook"},
	{"hook-before-making-current", "hook_before_making_current", "hook"},
	{"hook-after-making-current", "hook_after_making_current", "hook"},
	{"notification-email", "deploy_notification_email", "email"},
	{"retention", "deployment_releases_retention", "int"},
}

func newSitesSettingsCommand(rt *runtime) *cobra.Command {
	command := &cobra.Command{
		Use:   "settings",
		Short: "Show or update site deployment settings",
	}

	get := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one site's deployment settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := defaultJSONFormat(cmd); err != nil {
				return err
			}

			return rt.run(cmd, "GET", "/sites/"+args[0]+"/settings", nil)
		},
	}

	var stdin bool

	set := &cobra.Command{
		Use:   "set <id>",
		Short: "Update deployment settings, applied on the next deploy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := defaultJSONFormat(cmd); err != nil {
				return err
			}

			payload, err := settingsPayload(rt, cmd, stdin)
			if err != nil {
				return err
			}

			return rt.run(cmd, "PATCH", "/sites/"+args[0]+"/settings", nil, payload)
		},
	}
	for _, field := range settingsFields {
		switch field.kind {
		case "list":
			set.Flags().StringArray(field.flag, nil, "replaces the stored list; repeat for each entry, \"\" clears it")
		case "hook":
			set.Flags().String(field.flag, "", "hook script; a value starting with @ reads the file, e.g. @scripts/build.sh")
		case "email":
			set.Flags().String(field.flag, "", "deployment notification email; \"\" clears it")
		case "int":
			set.Flags().Int(field.flag, 0, "deployment releases retention, 1-50")
		}
	}
	set.Flags().BoolVar(&stdin, "stdin", false, "read the whole settings body as json from stdin")

	command.AddCommand(get, set)

	return command
}

// defaultJSONFormat pins json output when the user did not choose a format;
// a settings table truncates hooks and lists to fit the six-column cap.
func defaultJSONFormat(cmd *cobra.Command) error {
	if cmd.Flags().Changed("format") {
		return nil
	}

	return cmd.Flags().Set("format", "json")
}

// settingsPayload builds the PATCH body. Field flags contribute only when
// changed, which is what keeps the call patch-shaped; --stdin replaces the
// body wholesale.
func settingsPayload(rt *runtime, cmd *cobra.Command, stdin bool) (map[string]any, error) {
	if stdin {
		changed := changedSettingsFlags(cmd)
		if len(changed) > 0 {
			return nil, fmt.Errorf("sites settings set cannot combine --stdin with %s. Example: forja sites settings set 01abc --retention 20", strings.Join(changed, ", "))
		}

		return stdinPayload(rt.stdin)
	}

	payload := map[string]any{}
	for _, field := range settingsFields {
		flag := cmd.Flags().Lookup(field.flag)
		if flag == nil || !flag.Changed {
			continue
		}

		switch field.kind {
		case "list":
			values, _ := cmd.Flags().GetStringArray(field.flag)
			if len(values) == 1 && values[0] == "" {
				payload[field.key] = []string{}
				continue
			}
			payload[field.key] = values
		case "hook":
			value, _ := cmd.Flags().GetString(field.flag)
			content, err := expandHookFile(field.flag, value)
			if err != nil {
				return nil, err
			}
			payload[field.key] = content
		case "email":
			value, _ := cmd.Flags().GetString(field.flag)
			if value == "" {
				payload[field.key] = nil
				continue
			}
			payload[field.key] = value
		case "int":
			value, _ := cmd.Flags().GetInt(field.flag)
			payload[field.key] = value
		}
	}

	if len(payload) == 0 {
		return nil, fmt.Errorf("sites settings set requires --stdin or at least one settings flag. Example: forja sites settings set 01abc --retention 20")
	}

	return payload, nil
}

func changedSettingsFlags(cmd *cobra.Command) []string {
	var changed []string
	for _, field := range settingsFields {
		if flag := cmd.Flags().Lookup(field.flag); flag != nil && flag.Changed {
			changed = append(changed, "--"+field.flag)
		}
	}

	return changed
}

// expandHookFile reads the file when the hook value starts with @. The read
// happens before any HTTP, so a bad path fails without a request.
func expandHookFile(flag, value string) (string, error) {
	if !strings.HasPrefix(value, "@") {
		return value, nil
	}

	content, err := os.ReadFile(value[1:])
	if err != nil {
		return "", fmt.Errorf("--%s cannot read %s: %v. Example: forja sites settings set 01abc --%s @scripts/build.sh", flag, value, err, flag)
	}

	return string(content), nil
}

func stdinPayload(stdin io.Reader) (map[string]any, error) {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("reading --stdin: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("--stdin is not valid json: %v", err)
	}

	allowed := make([]string, 0, len(settingsFields))
	permitted := map[string]bool{}
	for _, field := range settingsFields {
		allowed = append(allowed, field.key)
		permitted[field.key] = true
	}
	sort.Strings(allowed)

	var unknown []string
	for key := range body {
		if !permitted[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return nil, fmt.Errorf("--stdin has unknown settings keys: %s. Allowed keys: %s", strings.Join(unknown, ", "), strings.Join(allowed, ", "))
	}

	return body, nil
}
