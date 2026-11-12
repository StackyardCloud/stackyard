package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureContentModeratorRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "term lists create",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/lists/v1.0/termlists",
			body:   []byte(`{"Name":"blocked-terms","Description":"terms list"}`),
		},
		{
			name:   "term lists get all",
			method: http.MethodGet,
			path:   "/azure/contentmoderator/lists/v1.0/termlists",
		},
		{
			name:   "terms add",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/lists/v1.0/termlists/123/terms?language=eng",
			body:   []byte(`{"Term":"forbidden"}`),
		},
		{
			name:   "terms get all",
			method: http.MethodGet,
			path:   "/azure/contentmoderator/lists/v1.0/termlists/123/terms?language=eng",
		},
		{
			name:   "reviews create",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/review/v1.0/teams/local-team/reviews",
			body:   []byte(`{"Type":"Image","Content":"https://example.com/image.jpg"}`),
		},
		{
			name:   "reviews get",
			method: http.MethodGet,
			path:   "/azure/contentmoderator/review/v1.0/teams/local-team/reviews/review-1",
		},
		{
			name:   "reviews publish",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/review/v1.0/teams/local-team/reviews/review-1/publish",
		},
		{
			name:   "jobs create",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/review/v1.0/teams/local-team/jobs",
			body:   []byte(`{"Type":"Image","Content":"https://example.com/image.jpg"}`),
		},
		{
			name:   "jobs get",
			method: http.MethodGet,
			path:   "/azure/contentmoderator/review/v1.0/teams/local-team/jobs/job-1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
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

func TestAzureContentModeratorInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/contentmoderator/review/v1.0/teams/local-team/reviews?api-version="
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest error payload, got %#v", payload)
	}
	if payload["provider"] != providerAzure || payload["path"] != "/azure/contentmoderator/review/v1.0/teams/local-team/reviews" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureContentModeratorUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/contentmoderator/unknown-resource"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown content-moderator route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
