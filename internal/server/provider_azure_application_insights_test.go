package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAzureApplicationInsightsRoutes(t *testing.T) {
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
		name          string
		method        string
		path          string
		body          []byte
		wantStatus    int
		contentType   string
		checkJSONBody func(t *testing.T, payload map[string]any)
		checkRawBody  func(t *testing.T, body string)
	}{
		{
			name:       "query get",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/query?query=requests%20%7C%20take%201&timespan=PT30M",
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				tables, ok := payload["tables"].([]any)
				if !ok || len(tables) != 1 {
					t.Fatalf("expected query tables payload, got %#v", payload)
				}
			},
		},
		{
			name:       "query post",
			method:     http.MethodPost,
			path:       "/azure/application-insights/v1/apps/app-123/query",
			body:       []byte(`{"query":"requests | summarize count()","timespan":"PT1H","applications":["app-123","app-456"]}`),
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				tables, ok := payload["tables"].([]any)
				if !ok || len(tables) == 0 {
					t.Fatalf("expected query result tables, got %#v", payload)
				}
			},
		},
		{
			name:       "events by type",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/events/requests?$count=true",
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["@odata.count"] != float64(1) {
					t.Fatalf("expected @odata.count=1, got %#v", payload)
				}
				events, ok := payload["value"].([]any)
				if !ok || len(events) != 1 {
					t.Fatalf("expected event list payload, got %#v", payload)
				}
			},
		},
		{
			name:       "event get",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/events/requests/event-123",
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["eventType"] != "requests" {
					t.Fatalf("expected requests event payload, got %#v", payload)
				}
			},
		},
		{
			name:        "events metadata xml",
			method:      http.MethodGet,
			path:        "/azure/application-insights/v1/apps/app-123/events/$metadata",
			wantStatus:  http.StatusOK,
			contentType: "application/xml;charset=utf-8",
			checkRawBody: func(t *testing.T, body string) {
				t.Helper()
				if !strings.Contains(body, "<edmx:Edmx") || !strings.Contains(body, "<EntityType Name=\"event\">") {
					t.Fatalf("unexpected XML metadata body: %s", body)
				}
			},
		},
		{
			name:       "metrics metadata",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/metrics/metadata",
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				value, ok := payload["value"].([]any)
				if !ok || len(value) == 0 {
					t.Fatalf("expected metrics metadata payload, got %#v", payload)
				}
			},
		},
		{
			name:       "metric get",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/metrics/requests/count?aggregation=sum&segment=request/name",
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if payload["id"] != "requests/count" {
					t.Fatalf("expected requests/count metric payload, got %#v", payload)
				}
			},
		},
		{
			name:       "metrics get multiple",
			method:     http.MethodPost,
			path:       "/azure/application-insights/v1/apps/app-123/metrics",
			body:       []byte(`[{"id":"requests/count"},{"id":"exceptions/count"}]`),
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				value, ok := payload["value"].([]any)
				if !ok || len(value) != 2 {
					t.Fatalf("expected batched metric results, got %#v", payload)
				}
			},
		},
		{
			name:       "metadata get",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/metadata",
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if _, ok := payload["tables"].([]any); !ok {
					t.Fatalf("expected metadata tables, got %#v", payload)
				}
			},
		},
		{
			name:       "metadata post",
			method:     http.MethodPost,
			path:       "/azure/application-insights/v1/apps/app-123/metadata",
			body:       []byte(`{"includeTables":["requests"]}`),
			wantStatus: http.StatusOK,
			checkJSONBody: func(t *testing.T, payload map[string]any) {
				t.Helper()
				if _, ok := payload["applications"].([]any); !ok {
					t.Fatalf("expected applications metadata, got %#v", payload)
				}
			},
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
			body := providerContractBody(t, resp)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d for %s %s, got %d body=%s", tt.wantStatus, tt.method, tt.path, resp.StatusCode, string(body))
			}
			if tt.contentType != "" && resp.Header.Get("Content-Type") != tt.contentType {
				t.Fatalf("expected Content-Type %q, got %q body=%s", tt.contentType, resp.Header.Get("Content-Type"), string(body))
			}
			if tt.checkRawBody != nil {
				tt.checkRawBody(t, string(body))
				return
			}
			payload := mustDecodeJSONMap(t, body)
			if tt.checkJSONBody != nil {
				tt.checkJSONBody(t, payload)
			}
		})
	}
}

func TestAzureApplicationInsightsValidation(t *testing.T) {
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
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid api version query",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/metadata?api-version=",
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "query get missing query",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/query",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadArgumentError",
		},
		{
			name:       "query post invalid json",
			method:     http.MethodPost,
			path:       "/azure/application-insights/v1/apps/app-123/query",
			body:       []byte(`{`),
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadArgumentError",
		},
		{
			name:       "unsupported event type",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/events/not-real",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadArgumentError",
		},
		{
			name:       "unknown metric",
			method:     http.MethodGet,
			path:       "/azure/application-insights/v1/apps/app-123/metrics/not/a/metric",
			wantStatus: http.StatusNotFound,
			wantCode:   "PathNotFoundError",
		},
		{
			name:       "metrics batch missing body",
			method:     http.MethodPost,
			path:       "/azure/application-insights/v1/apps/app-123/metrics",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadArgumentError",
		},
		{
			name:       "metadata post invalid json",
			method:     http.MethodPost,
			path:       "/azure/application-insights/v1/apps/app-123/metadata",
			body:       []byte(`{`),
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadArgumentError",
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
			body := providerContractBody(t, resp)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d for %s, got %d body=%s", tt.wantStatus, tt.path, resp.StatusCode, string(body))
			}

			payload := mustDecodeJSONMap(t, body)
			switch tt.wantCode {
			case "InvalidRequest":
				if payload["error"] != "InvalidRequest" {
					t.Fatalf("expected InvalidRequest payload, got %#v", payload)
				}
			default:
				errPayload, ok := payload["error"].(map[string]any)
				if !ok {
					t.Fatalf("expected nested error payload, got %#v", payload)
				}
				if errPayload["code"] != tt.wantCode {
					t.Fatalf("expected error code %q, got %#v", tt.wantCode, payload)
				}
			}
		})
	}
}

func mustDecodeJSONMap(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal response JSON: %v (body=%s)", err, string(body))
	}
	return payload
}
