package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	var out, errOut bytes.Buffer
	root := newRootCLI(&out, &errOut)
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
