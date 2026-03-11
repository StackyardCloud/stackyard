package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureLuisRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "get slot prediction post",
			method: http.MethodPost,
			path:   "/azure/luis/prediction/v3.0/apps/4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123/slots/staging/predict?verbose=true&show-all-intents=true&log=true",
			body: []byte(`{
  "query":"forward to frank 30 dollars through HSBC",
  "options":{"datetimeReference":"2015-02-13T13:15:00.000Z"},
  "externalEntities":[{"entityName":"Bank","startIndex":36,"entityLength":4,"resolution":{"text":"International Bank"}}],
  "dynamicLists":[{"listEntityName":"Employees","requestLists":[{"name":"Management","canonicalForm":"Frank","synonyms":[]}]}]
}`),
		},
		{
			name:   "get slot prediction get",
			method: http.MethodGet,
			path:   "/azure/luis/prediction/v3.0/apps/4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123/slots/staging/predict?query=forward%20to%20frank%2030%20dollars%20through%20HSBC&verbose=true&show-all-intents=true&log=true",
		},
		{
			name:   "get version prediction post",
			method: http.MethodPost,
			path:   "/azure/luis/prediction/v3.0/apps/4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123/versions/0.1/predict?verbose=true&show-all-intents=true&log=true",
			body: []byte(`{
  "query":"forward to frank 30 dollars through HSBC",
  "options":{"datetimeReference":"2015-02-13T13:15:00.000Z"}
}`),
		},
		{
			name:   "get version prediction get",
			method: http.MethodGet,
			path:   "/azure/luis/prediction/v3.0/apps/4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123/versions/0.1/predict?query=forward%20to%20frank%2030%20dollars%20through%20HSBC&verbose=true&show-all-intents=true&log=true",
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

func TestAzureLuisInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/luis/prediction/v3.0/apps/4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123/slots/staging/predict?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/luis/prediction/v3.0/apps/4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123/slots/staging/predict" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureLuisUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/luis/prediction/v3.0/custom/preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown luis nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
