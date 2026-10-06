package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const settingsDocument = `{"site_id":"site-01","zero_downtime_deployment":true,"deploy_notification_email":null,"deployment_releases_retention":10,"shared_directories":["storage"],"shared_files":[".env"],"writeable_directories":[],"hook_before_updating_repository":"","hook_after_updating_repository":"","hook_before_making_current":"","hook_after_making_current":""}`

const settingsDocumentIndented = `{
    "site_id": "site-01",
    "zero_downtime_deployment": true,
    "deploy_notification_email": null,
    "deployment_releases_retention": 10,
    "shared_directories": [
        "storage"
    ],
    "shared_files": [
        ".env"
    ],
    "writeable_directories": [],
    "hook_before_updating_repository": "",
    "hook_after_updating_repository": "",
    "hook_before_making_current": "",
    "hook_after_making_current": ""
}
`

func settingsServer(t *testing.T) (*httptest.Server, *string, *string, *[]byte) {
	t.Helper()

	var gotMethod, gotPath string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, settingsDocument)
	}))
	t.Cleanup(server.Close)

	return server, &gotMethod, &gotPath, &gotBody
}

func TestSitesSettingsGetPrintsJsonByDefault(t *testing.T) {
	server, gotMethod, gotPath, _ := settingsServer(t)

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\nformat: table\n")
	out, err := runCLI(t, "--config", path, "sites", "settings", "get", "site-01")
	if err != nil {
		t.Fatalf("sites settings get: %v", err)
	}

	if *gotMethod != "GET" || *gotPath != "/api/v1/sites/site-01/settings" {
		t.Errorf("request = %s %s, want GET /api/v1/sites/site-01/settings", *gotMethod, *gotPath)
	}
	if out != settingsDocumentIndented {
		t.Errorf("output = %q, want the indented settings document", out)
	}
}

func TestSitesSettingsGetHonorsTableFormat(t *testing.T) {
	server, _, _, _ := settingsServer(t)

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	out, err := runCLI(t, "--config", path, "sites", "settings", "get", "site-01", "--format", "table")
	if err != nil {
		t.Fatalf("sites settings get: %v", err)
	}

	for _, want := range []string{"SITE_ID", "ZERO_DOWNTIME_DEPLOYMENT", "DEPLOY_NOTIFICATION_EMAIL", "site-01", "true"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "\"site_id\"") {
		t.Errorf("output looks like json:\n%s", out)
	}
}

func TestSitesSettingsSetSendsOnlyChangedFlags(t *testing.T) {
	server, gotMethod, gotPath, gotBody := settingsServer(t)

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\nformat: table\n")
	out, err := runCLI(t, "--config", path, "sites", "settings", "set", "site-01",
		"--retention", "20", "--shared-directory", "storage", "--shared-directory", "public",
		"--notification-email", "ops@example.test")
	if err != nil {
		t.Fatalf("sites settings set: %v", err)
	}

	if *gotMethod != "PATCH" || *gotPath != "/api/v1/sites/site-01/settings" {
		t.Errorf("request = %s %s, want PATCH /api/v1/sites/site-01/settings", *gotMethod, *gotPath)
	}

	var payload map[string]any
	if err := json.Unmarshal(*gotBody, &payload); err != nil {
		t.Fatalf("decode request body %s: %v", *gotBody, err)
	}
	want := map[string]any{
		"deployment_releases_retention": float64(20),
		"shared_directories":            []any{"storage", "public"},
		"deploy_notification_email":     "ops@example.test",
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload = %v, want %v (unchanged fields must be absent)", payload, want)
	}

	if !strings.Contains(out, "\"site_id\": \"site-01\"") || !strings.Contains(out, "\"deployment_releases_retention\": 10") {
		t.Errorf("output not the merged json document:\n%s", out)
	}
}

func TestSitesSettingsSetEmptyValuesClearFields(t *testing.T) {
	server, _, _, gotBody := settingsServer(t)

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	if _, err := runCLI(t, "--config", path, "sites", "settings", "set", "site-01",
		"--notification-email", "", "--shared-file", ""); err != nil {
		t.Fatalf("sites settings set: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(*gotBody, &payload); err != nil {
		t.Fatalf("decode request body %s: %v", *gotBody, err)
	}
	want := map[string]any{
		"deploy_notification_email": nil,
		"shared_files":              []any{},
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload = %v, want %v (empty values must clear)", payload, want)
	}
}

func TestSitesSettingsSetReadsHookFromFile(t *testing.T) {
	server, _, _, gotBody := settingsServer(t)

	script := filepath.Join(t.TempDir(), "build.sh")
	if err := os.WriteFile(script, []byte("php artisan config:clear\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	if _, err := runCLI(t, "--config", path, "sites", "settings", "set", "site-01",
		"--hook-before-making-current", "@"+script); err != nil {
		t.Fatalf("sites settings set: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(*gotBody, &payload); err != nil {
		t.Fatalf("decode request body %s: %v", *gotBody, err)
	}
	if payload["hook_before_making_current"] != "php artisan config:clear\n" {
		t.Errorf("hook = %v, want the file contents", payload["hook_before_making_current"])
	}
}

func TestSitesSettingsSetMissingHookFileFailsBeforeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made when the @file is unreadable")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "sites", "settings", "set", "site-01",
		"--hook-before-updating-repository", "@/nonexistent/nope.sh")

	if err == nil {
		t.Fatal("expected a missing-file error, got nil")
	}
	if !strings.Contains(err.Error(), "--hook-before-updating-repository") {
		t.Errorf("error = %q, want it to name the flag", err.Error())
	}
}

func TestSitesSettingsSetRequiresFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made without flags")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "sites", "settings", "set", "site-01")

	want := "sites settings set requires --stdin or at least one settings flag. Example: forja sites settings set 01abc --retention 20"
	if err == nil {
		t.Fatal("expected a missing-flags error, got nil")
	}
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSitesSettingsSetStdinRejectsFieldFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made when --stdin is combined with field flags")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "sites", "settings", "set", "site-01",
		"--stdin", "--retention", "20", "--notification-email", "ops@example.test")

	if err == nil {
		t.Fatal("expected a --stdin conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "--stdin") || !strings.Contains(err.Error(), "--notification-email, --retention") {
		t.Errorf("error = %q, want it to name --stdin and the field flags", err.Error())
	}
}

func TestSitesSettingsSetStdinSendsBody(t *testing.T) {
	server, gotMethod, gotPath, gotBody := settingsServer(t)

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	if _, err := runCLIStdin(t, `{"deployment_releases_retention": 20, "hook_before_making_current": "php artisan down"}`,
		"--config", path, "sites", "settings", "set", "site-01", "--stdin"); err != nil {
		t.Fatalf("sites settings set --stdin: %v", err)
	}

	if *gotMethod != "PATCH" || *gotPath != "/api/v1/sites/site-01/settings" {
		t.Errorf("request = %s %s, want PATCH /api/v1/sites/site-01/settings", *gotMethod, *gotPath)
	}

	var payload map[string]any
	if err := json.Unmarshal(*gotBody, &payload); err != nil {
		t.Fatalf("decode request body %s: %v", *gotBody, err)
	}
	want := map[string]any{
		"deployment_releases_retention": float64(20),
		"hook_before_making_current":    "php artisan down",
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload = %v, want %v", payload, want)
	}
}

func TestSitesSettingsSetStdinRejectsUnknownKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made for unknown settings keys")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLIStdin(t, `{"zero_downtime_deployment": false}`,
		"--config", path, "sites", "settings", "set", "site-01", "--stdin")

	want := "--stdin has unknown settings keys: zero_downtime_deployment. Allowed keys: deploy_notification_email, deployment_releases_retention, hook_after_making_current, hook_after_updating_repository, hook_before_making_current, hook_before_updating_repository, shared_directories, shared_files, writeable_directories"
	if err == nil {
		t.Fatal("expected an unknown-keys error, got nil")
	}
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSitesSettingsSetStdinRejectsNonJson(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made for a non-json body")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLIStdin(t, "not json", "--config", path, "sites", "settings", "set", "site-01", "--stdin")

	if err == nil {
		t.Fatal("expected a json error, got nil")
	}
	if !strings.Contains(err.Error(), "--stdin is not valid json") {
		t.Errorf("error = %q, want it to name --stdin", err.Error())
	}
}
