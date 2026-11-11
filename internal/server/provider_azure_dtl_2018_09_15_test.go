package server

import (
	"net/http"
	"testing"
)

func TestAzureDtl20180915Stage1Routing(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "provider operations list",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.DevTestLab/operations?api-version=2018-09-15",
			wantStatus: http.StatusOK,
		},
		{
			name:       "labs list by subscription",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.DevTestLab/labs?api-version=2018-09-15",
			wantStatus: http.StatusOK,
		},
		{
			name:       "vm start action",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-dtl/providers/Microsoft.DevTestLab/labs/lab-a/virtualmachines/vm-a/start?api-version=2018-09-15",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unsupported method on known route",
			method:     http.MethodPut,
			path:       "/azure/providers/Microsoft.DevTestLab/operations?api-version=2018-09-15",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "unknown dtl route",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-dtl/providers/Microsoft.DevTestLab/labs/lab-a/unsupportedCollection?api-version=2018-09-15",
			wantStatus: http.StatusNotFound,
			wantError:  "NotFound",
		},
		{
			name:       "missing api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.DevTestLab/operations",
			wantStatus: http.StatusBadRequest,
			wantError:  "InvalidRequest",
		},
		{
			name:       "invalid api version value",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.DevTestLab/operations?api-version=bad",
			wantStatus: http.StatusBadRequest,
			wantError:  "InvalidRequest",
		},
		{
			name:       "duplicate api version parameters",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.DevTestLab/operations?api-version=2018-09-15&api-version=2018-09-15",
			wantStatus: http.StatusBadRequest,
			wantError:  "InvalidRequest",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := providerContractRequest(t, ts, tt.method, tt.path, nil, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d for %s, got %d body=%s", tt.wantStatus, tt.name, resp.StatusCode, string(providerContractBody(t, resp)))
			}

			if tt.wantStatus == http.StatusMethodNotAllowed {
				return
			}

			payload := providerContractJSONMap(t, resp)
			if tt.wantError != "" {
				if payload["error"] != tt.wantError {
					t.Fatalf("expected error %q, got %#v", tt.wantError, payload)
				}
				if payload["message"] == "" {
					t.Fatalf("expected non-empty message, got %#v", payload)
				}
				return
			}

			if payload["status"] != "ok" {
				t.Fatalf("expected status=ok, got %#v", payload)
			}
			if payload["provider"] != providerAzure {
				t.Fatalf("expected provider=%q, got %#v", providerAzure, payload)
			}
			if payload["path"] == "" {
				t.Fatalf("expected response path field, got %#v", payload)
			}
		})
	}
}
