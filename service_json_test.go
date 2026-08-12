package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestServiceManifestUsesCanonicalEndpoints(t *testing.T) {
	raw, err := os.ReadFile("service.json")
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}

	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(raw, &topLevel); err != nil {
		t.Fatalf("decode service.json: %v", err)
	}
	for _, legacyField := range []string{"ports", "portmapping", "urls"} {
		if _, exists := topLevel[legacyField]; exists {
			t.Fatalf("service.json must not author legacy top-level %q", legacyField)
		}
	}

	var manifest struct {
		Env       map[string]string `json:"env"`
		Endpoints []struct {
			ID        string `json:"id"`
			Kind      string `json:"kind"`
			Label     string `json:"label"`
			Target    string `json:"target"`
			Transport string `json:"transport"`
			Protocol  string `json:"protocol"`
			Bind      string `json:"bind"`
			URL       string `json:"url"`
			Port      struct {
				Default  int    `json:"default"`
				Strategy string `json:"strategy"`
			} `json:"port"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode endpoint contract: %v", err)
	}

	expectedEnv := map[string]string{
		"ECHO_PORT":             "${endpoint.service.port}",
		"ECHO_HTTP_HEALTH_PORT": "${endpoint.http_health.port}",
		"ECHO_TCP_PORT":         "${endpoint.tcp_health.port}",
	}
	for key, expected := range expectedEnv {
		if actual := manifest.Env[key]; actual != expected {
			t.Errorf("%s = %q, want %q", key, actual, expected)
		}
	}

	type networkExpectation struct {
		port     int
		protocol string
	}
	expectedNetworks := map[string]networkExpectation{
		"service":     {port: 4010, protocol: "http"},
		"http_health": {port: 4011, protocol: "http"},
		"tcp_health":  {port: 4012, protocol: "tcp"},
	}
	expectedURLs := map[string]struct {
		label  string
		target string
		url    string
	}{
		"ui": {
			label:  "ui",
			target: "service",
			url:    "http://${endpoint.service.bind}:${endpoint.service.port}/",
		},
		"service_health": {
			label:  "service",
			target: "service",
			url:    "http://${endpoint.service.bind}:${endpoint.service.port}/health",
		},
		"http_health_url": {
			label:  "http-health",
			target: "http_health",
			url:    "http://${endpoint.http_health.bind}:${endpoint.http_health.port}/health",
		},
	}

	seen := make(map[string]bool, len(manifest.Endpoints))
	for _, endpoint := range manifest.Endpoints {
		if endpoint.ID == "" {
			t.Fatal("endpoint id must not be empty")
		}
		if seen[endpoint.ID] {
			t.Fatalf("duplicate endpoint id %q", endpoint.ID)
		}
		seen[endpoint.ID] = true

		switch endpoint.Kind {
		case "network":
			expected, ok := expectedNetworks[endpoint.ID]
			if !ok {
				t.Errorf("unexpected network endpoint %q", endpoint.ID)
				continue
			}
			if endpoint.Bind != "127.0.0.1" || endpoint.Transport != "tcp" || endpoint.Protocol != expected.protocol {
				t.Errorf("network endpoint %q has unexpected binding: bind=%q transport=%q protocol=%q", endpoint.ID, endpoint.Bind, endpoint.Transport, endpoint.Protocol)
			}
			if endpoint.Port.Default != expected.port || endpoint.Port.Strategy != "preferred" {
				t.Errorf("network endpoint %q has port default=%d strategy=%q, want %d/preferred", endpoint.ID, endpoint.Port.Default, endpoint.Port.Strategy, expected.port)
			}
		case "url":
			expected, ok := expectedURLs[endpoint.ID]
			if !ok {
				t.Errorf("unexpected URL endpoint %q", endpoint.ID)
				continue
			}
			if endpoint.Label != expected.label || endpoint.Target != expected.target || endpoint.URL != expected.url {
				t.Errorf("URL endpoint %q = label %q target %q url %q, want label %q target %q url %q", endpoint.ID, endpoint.Label, endpoint.Target, endpoint.URL, expected.label, expected.target, expected.url)
			}
		default:
			t.Errorf("unexpected endpoint kind %q for %q", endpoint.Kind, endpoint.ID)
		}
	}

	for id := range expectedNetworks {
		if !seen[id] {
			t.Errorf("missing network endpoint %q", id)
		}
	}
	for id := range expectedURLs {
		if !seen[id] {
			t.Errorf("missing URL endpoint %q", id)
		}
	}
}
