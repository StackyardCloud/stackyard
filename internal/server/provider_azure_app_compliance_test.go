package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAppComplianceRoutesReturnFoundationSuccess(t *testing.T) {
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
			name:   "list operations",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/operations?api-version=2024-06-27",
		},
		{
			name:   "check name availability",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/checkNameAvailability?api-version=2024-06-27",
			body:   []byte(`{"name":"report-a","type":"Microsoft.AppComplianceAutomation/reports"}`),
		},
		{
			name:   "create report",
			method: http.MethodPut,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a?api-version=2024-06-27",
			body:   []byte(`{"properties":{"offerGuid":"11111111-1111-1111-1111-111111111111","timeZone":"UTC"}}`),
		},
		{
			name:   "update report",
			method: http.MethodPatch,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a?api-version=2024-06-27",
			body:   []byte(`{"properties":{"timeZone":"UTC"}}`),
		},
		{
			name:   "verify report",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/verify?api-version=2024-06-27",
			body:   []byte(`{"trigger":"manual"}`),
		},
		{
			name:   "create evidence",
			method: http.MethodPut,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/evidences/evidence-a?api-version=2024-06-27",
			body:   []byte(`{"properties":{"displayName":"Evidence A"}}`),
		},
		{
			name:   "download evidence",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/evidences/evidence-a/download?api-version=2024-06-27",
			body:   []byte(`{}`),
		},
		{
			name:   "create scoping configuration",
			method: http.MethodPut,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/scopingConfigurations/scoping-a?api-version=2024-06-27",
			body:   []byte(`{"properties":{"answers":[{"questionId":"q1","answers":["a1"]}]}}`),
		},
		{
			name:   "list snapshots",
			method: http.MethodGet,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/snapshots?api-version=2024-06-27",
		},
		{
			name:   "download snapshot",
			method: http.MethodPost,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/snapshots/snapshot-a/download?api-version=2024-06-27",
			body:   []byte(`{}`),
		},
		{
			name:   "create webhook",
			method: http.MethodPut,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/webhooks/webhook-a?api-version=2024-06-27",
			body:   []byte(`{"properties":{"uri":"https://example.com/hook","events":["ReportUpdated"]}}`),
		},
		{
			name:   "delete webhook",
			method: http.MethodDelete,
			path:   "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/webhooks/webhook-a?api-version=2024-06-27",
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

func TestAzureAppComplianceInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.AppComplianceAutomation/operations?api-version="
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
	if payload["provider"] != providerAzure || payload["path"] != "/azure/providers/Microsoft.AppComplianceAutomation/operations" {
		t.Fatalf("unexpected payload for invalid api-version: %#v", payload)
	}
}

func TestAzureAppComplianceUnknownNestedRouteReturnsFoundationSuccess(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/providers/Microsoft.AppComplianceAutomation/reports/report-a/custom-preview-route"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unknown app compliance nested route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure || payload["path"] != path {
		t.Fatalf("unexpected payload for unknown nested route: %#v", payload)
	}
}
