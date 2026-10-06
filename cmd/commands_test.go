package cmd

import (
	"bytes"
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

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runCLIStdin(t, "", args...)
}

func runCLIStdin(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()

	var out, errOut bytes.Buffer
	root := newRootCLI(&out, &errOut, strings.NewReader(input))
	root.SetArgs(args)
	err := root.Execute()

	return out.String(), err
}

func configFor(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "forja.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWhoamiCommandRendersTable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":1,"name":"Verify Agent","email":"verify@example.test","teams":[{"id":1,"name":"T","personal_team":true}]}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	out, err := runCLI(t, "--config", path, "whoami")
	if err != nil {
		t.Fatalf("whoami: %v", err)
	}

	for _, want := range []string{"ID", "NAME", "EMAIL", "TEAMS", "Verify Agent", "verify@example.test"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestWhoamiCommandJsonFormatFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":1,"name":"Verify Agent"}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\nformat: table\n")
	out, err := runCLI(t, "--config", path, "--format", "json", "whoami")
	if err != nil {
		t.Fatalf("whoami: %v", err)
	}

	if want := "{\n    \"id\": 1,\n    \"name\": \"Verify Agent\"\n}\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestSitesDeployCommandPostsAndRenders(t *testing.T) {
	var gotPath, gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_, _ = io.WriteString(w, `{"id":"d1","site_id":"01abc","status":"pending"}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	out, err := runCLI(t, "--config", path, "sites", "deploy", "01abc")
	if err != nil {
		t.Fatalf("sites deploy: %v", err)
	}

	if gotMethod != "POST" || gotPath != "/api/v1/sites/01abc/deploy" {
		t.Errorf("request = %s %s, want POST /api/v1/sites/01abc/deploy", gotMethod, gotPath)
	}
	if !strings.Contains(out, "d1") || !strings.Contains(out, "pending") {
		t.Errorf("output missing deployment fields:\n%s", out)
	}
}

func TestSitesListServerFilter(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	if _, err := runCLI(t, "--config", path, "sites", "list", "--server", "srv1"); err != nil {
		t.Fatalf("sites list: %v", err)
	}

	if gotQuery != "server_id=srv1" {
		t.Errorf("query = %q, want server_id=srv1", gotQuery)
	}
}

func TestSitesCreatePostsPayloadAndRenders(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"id":"site-11","server_id":"srv1","address":"example.com","url":"https://example.com","type":"laravel","php_version":"php85","tls_setting":"auto","repository_url":null,"repository_branch":null,"zero_downtime_deployment":true,"installed_at":null,"created_at":"2026-10-05T09:00:00Z","deployment":{"id":"dep-300","site_id":"site-11","user_id":1,"status":"pending","git_hash":null,"short_git_hash":null,"created_at":"2026-10-05T09:00:00Z","updated_at":"2026-10-05T09:00:00Z"}}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	out, err := runCLI(t, "--config", path, "sites", "create",
		"--server", "srv1", "--address", "example.com", "--php-version", "php85", "--type", "laravel")
	if err != nil {
		t.Fatalf("sites create: %v", err)
	}

	if gotMethod != "POST" || gotPath != "/api/v1/servers/srv1/sites" {
		t.Errorf("request = %s %s, want POST /api/v1/servers/srv1/sites", gotMethod, gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("content type = %q, want application/json", gotContentType)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("decode request body %s: %v", gotBody, err)
	}
	want := map[string]any{
		"address":                  "example.com",
		"php_version":              "php85",
		"type":                     "laravel",
		"web_folder":               "public",
		"zero_downtime_deployment": true,
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload = %v, want %v (empty optionals must be absent)", payload, want)
	}

	if !strings.Contains(out, "site-11") || !strings.Contains(out, "example.com") {
		t.Errorf("output missing created site fields:\n%s", out)
	}
}

func TestSitesCreateDeployKeySendsUseDeployKey(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"id":"site-11","server_id":"srv1","address":"example.com","url":"https://example.com","type":"laravel","php_version":"php85","tls_setting":"auto","repository_url":"git@github.com:acme/site.git","repository_branch":"main","zero_downtime_deployment":true,"installed_at":null,"created_at":"2026-10-05T09:00:00Z","deploy_key_public":"ssh-ed25519 AAAA verify"}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	if _, err := runCLI(t, "--config", path, "sites", "create",
		"--server", "srv1", "--address", "example.com", "--php-version", "php85", "--type", "laravel",
		"--repository", "git@github.com:acme/site.git", "--branch", "main", "--deploy-key"); err != nil {
		t.Fatalf("sites create: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("decode request body %s: %v", gotBody, err)
	}
	want := map[string]any{
		"address":                  "example.com",
		"php_version":              "php85",
		"type":                     "laravel",
		"web_folder":               "public",
		"zero_downtime_deployment": true,
		"repository_url":           "git@github.com:acme/site.git",
		"repository_branch":        "main",
		"use_deploy_key":           true,
	}
	if !reflect.DeepEqual(payload, want) {
		t.Errorf("payload = %v, want %v", payload, want)
	}
}

func TestSitesCreateRequiresFlags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made without all required flags")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "sites", "create", "--address", "example.com")

	want := "sites create requires --server <id>, --php-version <version>, --type <type>. Example: forja sites create --server 01abc --address example.com --php-version php85 --type laravel"
	if err == nil {
		t.Fatal("expected a missing-flags error, got nil")
	}
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSitesCreateRendersValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"message":"El campo address ya ha sido tomado. (and 1 more error)","errors":{"repository_branch":["El campo repository branch es obligatorio."],"address":["El campo address ya ha sido tomado."]}}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "sites", "create",
		"--server", "srv1", "--address", "acme.com", "--php-version", "php85", "--type", "laravel")

	want := "El campo address ya ha sido tomado. (and 1 more error); address: El campo address ya ha sido tomado.; repository_branch: El campo repository branch es obligatorio. (HTTP 422)"
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestSitesCreateJsonFormatPrintsCreatedObjectVerbatim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"site-11","server_id":"srv1","address":"example.com","url":"https://example.com","type":"laravel","php_version":"php85","tls_setting":"auto","repository_url":"git@github.com:acme/site.git","repository_branch":"main","zero_downtime_deployment":true,"installed_at":null,"created_at":"2026-10-05T09:00:00Z","deploy_key_public":"ssh-ed25519 AAAA verify"}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\nformat: table\n")
	out, err := runCLI(t, "--config", path, "--format", "json", "sites", "create",
		"--server", "srv1", "--address", "example.com", "--php-version", "php85", "--type", "laravel",
		"--repository", "git@github.com:acme/site.git", "--branch", "main", "--deploy-key")
	if err != nil {
		t.Fatalf("sites create: %v", err)
	}

	want := `{
    "id": "site-11",
    "server_id": "srv1",
    "address": "example.com",
    "url": "https://example.com",
    "type": "laravel",
    "php_version": "php85",
    "tls_setting": "auto",
    "repository_url": "git@github.com:acme/site.git",
    "repository_branch": "main",
    "zero_downtime_deployment": true,
    "installed_at": null,
    "created_at": "2026-10-05T09:00:00Z",
    "deploy_key_public": "ssh-ed25519 AAAA verify"
}
`
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestDeploymentsListRequiresSiteFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("no request should be made without --site")
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "deployments", "list")

	if err == nil {
		t.Fatal("expected an error without --site, got nil")
	}
	if !strings.Contains(err.Error(), "requires --site") {
		t.Errorf("error = %q, want it to mention --site", err.Error())
	}
}

func TestServersGetUsesPathArgument(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `{"id":"srv1","name":"web1","status":"running"}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: k\nendpoint: "+server.URL+"\n")
	out, err := runCLI(t, "--config", path, "servers", "get", "srv1")
	if err != nil {
		t.Fatalf("servers get: %v", err)
	}

	if gotPath != "/api/v1/servers/srv1" {
		t.Errorf("path = %q, want /api/v1/servers/srv1", gotPath)
	}
	if !strings.Contains(out, "web1") {
		t.Errorf("output missing server name:\n%s", out)
	}
}

func TestVersionCommand(t *testing.T) {
	out, err := runCLI(t, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if out != "forja "+version+"\n" {
		t.Errorf("output = %q, want forja %s", out, version)
	}
}

func TestErrorsSurfaceWithoutUsageDump(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"Unauthenticated."}`)
	}))
	defer server.Close()

	path := configFor(t, "api_key: wrong\nendpoint: "+server.URL+"\n")
	_, err := runCLI(t, "--config", path, "whoami")

	if err == nil {
		t.Fatal("expected an auth error, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("error = %q, want HTTP 401 in it", err.Error())
	}
}
