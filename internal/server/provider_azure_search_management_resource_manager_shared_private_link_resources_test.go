package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSearchManagementResourceManagerSharedPrivateLinkResourcesRoutesReturnNotImplemented(t *testing.T) {
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
		body   []byte
	}{
		{
			name:   "create or update",
			method: http.MethodPut,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/sharedPrivateLinkResources/sql?api-version=2025-05-01",
			body:   []byte(`{"properties":{"privateLinkResourceId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-target/providers/Microsoft.Sql/servers/sqlsrv","groupId":"sqlServer","requestMessage":"please approve"}}`),
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/sharedPrivateLinkResources/sql?api-version=2025-05-01",
		},
		{
			name:   "get",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/sharedPrivateLinkResources/sql?api-version=2025-05-01",
		},
		{
			name:   "list by search service",
			method: http.MethodGet,
			path:   "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-search/providers/Microsoft.Search/searchServices/my-search/sharedPrivateLinkResources?api-version=2025-05-01",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization": "SharedKey devstoreaccount1:signature",
			}
			if tt.body != nil {
				headers["Content-Type"] = "application/json"
			}

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
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
