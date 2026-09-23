package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallSendsAuthTeamAndAcceptHeaders(t *testing.T) {
	var gotAuthorization, gotTeam, gotAccept, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		gotTeam = r.Header.Get("X-Forja-Team")
		gotAccept = r.Header.Get("Accept")
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `{"id":"1"}`)
	}))
	defer server.Close()

	api := newClient(settings{APIKey: "secret", Endpoint: server.URL, Team: "7"})
	raw, err := api.call("GET", "/servers", nil, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if gotAuthorization != "Bearer secret" {
		t.Errorf("authorization = %q, want Bearer secret", gotAuthorization)
	}
	if gotTeam != "7" {
		t.Errorf("team header = %q, want 7", gotTeam)
	}
	if gotAccept != "application/json" {
		t.Errorf("accept = %q, want application/json", gotAccept)
	}
	if gotPath != "/api/v1/servers" {
		t.Errorf("path = %q, want /api/v1/servers", gotPath)
	}
	if string(raw) != `{"id":"1"}` {
		t.Errorf("body = %s, want the response verbatim", raw)
	}
}

func TestCallUnwrapsSoleDataEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"id":"1","name":"web1"}}`)
	}))
	defer server.Close()

	api := newClient(settings{APIKey: "k", Endpoint: server.URL})
	raw, err := api.call("GET", "/me", nil, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if string(raw) != `{"id":"1","name":"web1"}` {
		t.Errorf("body = %s, want the unwrapped object", raw)
	}
}

func TestCallKeepsObjectWithDataSiblingKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":"a","name":"b"}`)
	}))
	defer server.Close()

	api := newClient(settings{APIKey: "k", Endpoint: server.URL})
	raw, err := api.call("GET", "/me", nil, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if string(raw) != `{"data":"a","name":"b"}` {
		t.Errorf("body = %s, want the object verbatim", raw)
	}
}

func TestCallMapsErrorMessages(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		want   string
	}{
		"401": {401, `{"message":"Unauthenticated."}`, "Unauthenticated. (HTTP 401)"},
		"422": {422, `{"message":"Ya hay un deploy en curso."}`, "Ya hay un deploy en curso. (HTTP 422)"},
		"500": {500, `boom`, "boom (HTTP 500)"},
		"501": {501, ``, "unknown error (HTTP 501)"},
	}
	for name, testCase := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(testCase.status)
			_, _ = io.WriteString(w, testCase.body)
		}))

		api := newClient(settings{APIKey: "k", Endpoint: server.URL})
		_, err := api.call("GET", "/x", nil, nil)
		server.Close()

		if err == nil {
			t.Fatalf("%s: expected an error, got nil", name)
		}
		if err.Error() != testCase.want {
			t.Errorf("%s: error = %q, want %q", name, err.Error(), testCase.want)
		}
	}
}

func TestCallRejectsMissingAPIKey(t *testing.T) {
	api := newClient(settings{Endpoint: "http://127.0.0.1:1"})
	_, err := api.call("GET", "/me", nil, nil)

	if err == nil {
		t.Fatal("expected an error for a missing api key, got nil")
	}
	if want := "api_key is required"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestCallUnexpectedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "not json")
	}))
	defer server.Close()

	api := newClient(settings{APIKey: "k", Endpoint: server.URL})
	if _, err := api.call("GET", "/me", nil, nil); err == nil {
		t.Fatal("expected an error for a non-json 200, got nil")
	}
}

func TestDeploySendsPostToDeployPath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `{"id":"d1","status":"pending"}`)
	}))
	defer server.Close()

	api := newClient(settings{APIKey: "k", Endpoint: server.URL})
	raw, err := api.call("POST", "/sites/01abc/deploy", nil, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/v1/sites/01abc/deploy" {
		t.Errorf("path = %q, want /api/v1/sites/01abc/deploy", gotPath)
	}

	var deployment map[string]any
	if err := json.Unmarshal(raw, &deployment); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if deployment["status"] != "pending" {
		t.Errorf("status = %v, want pending", deployment["status"])
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
