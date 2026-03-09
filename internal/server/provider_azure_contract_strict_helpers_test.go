package server

import (
	"net/http"
	"strings"
	"testing"
)

func assertAzureInvalidAPIVersionContract(t *testing.T, method, route string, body []byte, contentType string) {
	t.Helper()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	invalidRoute := withInvalidAPIVersion(route)
	headers := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	}
	if strings.TrimSpace(contentType) != "" {
		headers["Content-Type"] = contentType
	} else if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}

	resp := providerContractRequest(t, ts, method, invalidRoute, body, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected %d for %s %s, got %d body=%s", http.StatusBadRequest, method, invalidRoute, resp.StatusCode, string(providerContractBody(t, resp)))
	}

	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure {
		t.Fatalf("expected provider azure in payload, got %#v", payload)
	}

	expectedPath := invalidRoute
	if idx := strings.Index(expectedPath, "?"); idx >= 0 {
		expectedPath = expectedPath[:idx]
	}
	if payload["path"] != expectedPath {
		t.Fatalf("expected path %q in payload, got %#v", expectedPath, payload["path"])
	}
}

func withInvalidAPIVersion(route string) string {
	const param = "api-version="
	idx := strings.Index(route, param)
	if idx >= 0 {
		start := idx + len(param)
		rest := route[start:]
		if next := strings.Index(rest, "&"); next >= 0 {
			return route[:start] + route[start+next:]
		}
		return route[:start]
	}
	if strings.Contains(route, "?") {
		return route + "&api-version="
	}
	return route + "?api-version="
}
