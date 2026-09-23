package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveSettingsPrecedenceFlagOverEnvOverFile(t *testing.T) {
	path := writeConfig(t, "api_key: filekey\nendpoint: http://file\nformat: table\n")
	t.Setenv("FORJA_FORMAT", "json")

	flagKey := "flagkey"
	s, err := resolveSettings(path, map[string]*string{"api_key": &flagKey})
	if err != nil {
		t.Fatalf("resolveSettings: %v", err)
	}

	if s.APIKey != "flagkey" {
		t.Errorf("api key = %q, want flagkey (flags beat env)", s.APIKey)
	}
	if s.Format != "json" {
		t.Errorf("format = %q, want json (env beats file)", s.Format)
	}
	if s.Endpoint != "http://file" {
		t.Errorf("endpoint = %q, want http://file", s.Endpoint)
	}
}

func TestResolveSettingsDefaultsAndMissingFile(t *testing.T) {
	s, err := resolveSettings(filepath.Join(t.TempDir(), "absent.yaml"), nil)
	if err != nil {
		t.Fatalf("resolveSettings: %v", err)
	}

	if s.Endpoint != defaultEndpoint {
		t.Errorf("endpoint = %q, want %q", s.Endpoint, defaultEndpoint)
	}
	if s.Format != defaultFormat {
		t.Errorf("format = %q, want %q", s.Format, defaultFormat)
	}
	if s.APIKey != "" {
		t.Errorf("api key = %q, want empty", s.APIKey)
	}
}

func TestResolveSettingsRejectsMalformedFile(t *testing.T) {
	path := writeConfig(t, "api_key: [unclosed")
	if _, err := resolveSettings(path, nil); err == nil {
		t.Fatal("expected an error for a malformed config file, got nil")
	}
}

func TestResolveSettingsValidatesFormat(t *testing.T) {
	path := writeConfig(t, "format: xml\n")
	if _, err := resolveSettings(path, nil); err == nil {
		t.Fatal("expected an error for format xml, got nil")
	}
}

func TestResolveSettingsTrimsEndpointSlash(t *testing.T) {
	path := writeConfig(t, "endpoint: http://x/\n")
	s, err := resolveSettings(path, nil)
	if err != nil {
		t.Fatalf("resolveSettings: %v", err)
	}
	if s.Endpoint != "http://x" {
		t.Errorf("endpoint = %q, want http://x", s.Endpoint)
	}
}
