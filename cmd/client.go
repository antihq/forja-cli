package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type client struct {
	endpoint string
	apiKey   string
	team     string
	http     *http.Client
}

func newClient(s settings) *client {
	return &client{
		endpoint: s.Endpoint,
		apiKey:   s.APIKey,
		team:     s.Team,
		http:     &http.Client{Timeout: 60 * time.Second},
	}
}

// call performs one API request and returns the response body verbatim.
func (c *client) call(method, path string, query url.Values, payload any) (json.RawMessage, error) {
	if c.apiKey == "" {
		return nil, errors.New("api_key is required. Set api_key in ~/.forja.yaml, or FORJA_API_KEY, or --api-key")
	}

	endpoint := c.endpoint + "/api/v1" + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Accept", "application/json")
	if c.team != "" {
		request.Header.Set("X-Forja-Team", c.team)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", endpoint, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var probe any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&probe); err != nil {
		probe = nil
	}

	if response.StatusCode >= 400 {
		message := strings.TrimSpace(string(raw))
		if object, ok := probe.(map[string]any); ok {
			if text, ok := object["message"].(string); ok && text != "" {
				message = text
			}
		}
		if message == "" {
			message = "unknown error"
		}
		return nil, fmt.Errorf("%s (HTTP %d)", message, response.StatusCode)
	}

	if probe == nil {
		return nil, fmt.Errorf("unexpected response from %s", endpoint)
	}

	if object, ok := probe.(map[string]any); ok && len(object) == 1 {
		if _, ok := object["data"]; ok {
			return unwrapData(raw)
		}
	}

	return raw, nil
}

// unwrapData strips a top level {"data": ...} envelope, returning the inner
// document verbatim so key order survives for table rendering.
func unwrapData(raw json.RawMessage) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	first, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := first.(json.Delim); !ok || delimiter != '{' {
		return raw, nil
	}

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, _ := keyToken.(string)

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		if key == "data" {
			return value, nil
		}
	}

	return raw, nil
}
