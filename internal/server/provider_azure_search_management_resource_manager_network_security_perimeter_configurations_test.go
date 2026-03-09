package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSearchManagementResourceManagerNetworkSecurityPerimeterConfigurationsRoutesReturnNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "list by search service",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/networkSecurityPerimeterConfigurations?api-version=2025-05-01",
		},
		{
			name:   "get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/networkSecurityPerimeterConfigurations/default?api-version=2025-05-01",
		},
		{
			name:   "reconcile",
			method: http.MethodPost,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/networkSecurityPerimeterConfigurations/default/reconcile?api-version=2025-05-01",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := providerContractRequest(t, ts, tt.method, tt.path, nil, map[string]string{
				"Authorization": "SharedKey devstoreaccount1:signature",
			})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for %s %s, got %d body=%s", tt.method, tt.path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			payload := providerContractJSONMap(t, resp)
			if payload["status"] != "ok" {
				t.Fatalf("expected success payload, got %#v", payload)
			}
			if payload["provider"] != providerAzure {
				t.Fatalf("expected provider azure in payload, got %#v", payload)
			}

			expectedPath := tt.path
			if idx := strings.Index(expectedPath, "?"); idx >= 0 {
				expectedPath = expectedPath[:idx]
			}
			if payload["path"] != expectedPath {
				t.Fatalf("expected path %q in payload, got %#v", expectedPath, payload["path"])
			}
		})
	}
}
