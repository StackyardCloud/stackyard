package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSearchServiceDataPlaneDocumentsRoutesReturnNotImplemented(t *testing.T) {
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
			name:   "autocomplete get",
			method: http.MethodGet,
			path:   "/azure/indexes('hotels')/docs/search.autocomplete?api-version=2025-09-01&search=washington&suggesterName=sg",
		},
		{
			name:   "autocomplete post",
			method: http.MethodPost,
			path:   "/azure/indexes('hotels')/docs/search.post.autocomplete?api-version=2025-09-01",
			body:   []byte(`{"search":"washington","suggesterName":"sg","autocompleteMode":"oneTermWithContext"}`),
		},
		{
			name:   "count",
			method: http.MethodGet,
			path:   "/azure/indexes('hotels')/docs/$count?api-version=2025-09-01",
		},
		{
			name:   "get document",
			method: http.MethodGet,
			path:   "/azure/indexes('hotels')/docs('1')?api-version=2025-09-01&select=hotelName",
		},
		{
			name:   "index",
			method: http.MethodPost,
			path:   "/azure/indexes('hotels')/docs/search.index?api-version=2025-09-01",
			body:   []byte(`{"value":[{"@search.action":"upload","id":"1","hotelName":"Stackyard Inn"}]}`),
		},
		{
			name:   "search get",
			method: http.MethodGet,
			path:   "/azure/indexes('hotels')/docs?api-version=2025-09-01&search=pool&$top=5",
		},
		{
			name:   "search post",
			method: http.MethodPost,
			path:   "/azure/indexes('hotels')/docs/search.post.search?api-version=2025-09-01",
			body:   []byte(`{"search":"pool","top":5,"queryType":"simple"}`),
		},
		{
			name:   "suggest get",
			method: http.MethodGet,
			path:   "/azure/indexes('hotels')/docs/search.suggest?api-version=2025-09-01&search=stack&suggesterName=sg",
		},
		{
			name:   "suggest post",
			method: http.MethodPost,
			path:   "/azure/indexes('hotels')/docs/search.post.suggest?api-version=2025-09-01",
			body:   []byte(`{"search":"stack","suggesterName":"sg","top":5}`),
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

func TestAzureSearchServiceDataPlaneDocumentsUnsupportedNestedPathReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/indexes('hotels')/docs/unsupported/segment"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown search documents nested path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
}
