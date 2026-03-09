package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSearchServiceDataPlaneDataSourcesRoutesReturnNotImplemented(t *testing.T) {
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
			name:   "create",
			method: http.MethodPost,
			path:   "/azure/datasources?api-version=2025-09-01",
			body: []byte(`{
				"name":"hotels-ds",
				"type":"azuresql",
				"credentials":{"connectionString":"Server=tcp:local,1433;Database=db;User Id=u;Password=p;"},
				"container":{"name":"hotels"}
			}`),
		},
		{
			name:   "create or update",
			method: http.MethodPut,
			path:   "/azure/datasources('hotels-ds')?api-version=2025-09-01&allowIndexDowntime=true",
			body: []byte(`{
				"name":"hotels-ds",
				"type":"azuresql",
				"credentials":{"connectionString":"Server=tcp:local,1433;Database=db;User Id=u;Password=p;"},
				"container":{"name":"hotels"}
			}`),
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			path:   "/azure/datasources('hotels-ds')?api-version=2025-09-01",
		},
		{
			name:   "get",
			method: http.MethodGet,
			path:   "/azure/datasources('hotels-ds')?api-version=2025-09-01",
		},
		{
			name:   "list",
			method: http.MethodGet,
			path:   "/azure/datasources?api-version=2025-09-01&$select=name&$top=10",
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

func TestAzureSearchServiceDataPlaneDataSourcesUnsupportedNestedPathReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/datasources('hotels-ds')/unsupported/segment"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown data sources nested path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
