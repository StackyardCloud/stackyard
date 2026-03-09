package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSearchManagementResourceManagerOperationsRoutesReturnNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.Search/operations?api-version=2025-05-01"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
	}

	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" {
		t.Fatalf("expected success payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure {
		t.Fatalf("expected provider azure in payload, got %#v", payload)
	}

	expectedPath := path
	if idx := strings.Index(expectedPath, "?"); idx >= 0 {
		expectedPath = expectedPath[:idx]
	}
	if payload["path"] != expectedPath {
		t.Fatalf("expected path %q in payload, got %#v", expectedPath, payload["path"])
	}
}
