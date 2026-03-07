package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPGeminiDataAnalyticsRouter_DataAgentRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents?pageSize=1", nil, "dataAgents")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents?dataAgentId=analytics-agent", []byte(`{
		"displayName":"analytics-agent",
		"description":"stackyard sample data agent",
		"dataAnalyticsAgent":{}
	}`), `"done":true`)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents/analytics-agent", nil, "displayName")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPatch, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents/analytics-agent?updateMask=display_name", []byte(`{
		"name":"projects/stackyard/locations/us-central1/dataAgents/analytics-agent",
		"displayName":"analytics-agent-updated"
	}`), `"done":true`)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodDelete, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents/analytics-agent", nil, `"done":true`)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents:listAccessible?pageSize=1", nil, "dataAgents")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents/analytics-agent:getIamPolicy", []byte(`{"resource":"projects/stackyard/locations/us-central1/dataAgents/analytics-agent"}`), "bindings")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents/analytics-agent:setIamPolicy", []byte(`{
		"resource":"projects/stackyard/locations/us-central1/dataAgents/analytics-agent",
		"policy":{"bindings":[]}
	}`), "bindings")
}

func TestGCPGeminiDataAnalyticsRouter_DataChatRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1:chat", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"messages":[{"userMessage":{"text":"show monthly revenue"}}]
	}`), "systemMessage")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1/conversations?conversationId=conv-1", []byte(`{
		"agents":["projects/stackyard/locations/us-central1/dataAgents/analytics-agent"]
	}`), "conversations/conv-1")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/conversations?pageSize=1", nil, "conversations")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/conversations/conv-1", nil, "conversations/conv-1")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/conversations/conv-1/messages?pageSize=1", nil, "messages")
}

func TestGCPGeminiDataAnalyticsRouter_LocationsAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, "operations")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/operations/op-1", nil, "operations/op-1")
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1/operations/op-1:cancel", []byte(`{"name":"projects/stackyard/locations/us-central1/operations/op-1"}`), `"cancelled":true`)
	assertGCPGeminiDataAnalyticsSuccess(t, ts, http.MethodDelete, "/gcp/v1beta/projects/stackyard/locations/us-central1/operations/op-1", nil, `"deleted":true`)
}

func TestGCPGeminiDataAnalyticsRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	assertGCPGeminiDataAnalyticsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.geminidataanalytics.v1beta.DataAgentService/ListDataAgents", "DataAgentService/ListDataAgents")
	assertGCPGeminiDataAnalyticsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.geminidataanalytics.v1beta.DataAgentService/CreateDataAgent", "DataAgentService/CreateDataAgent")
	assertGCPGeminiDataAnalyticsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.geminidataanalytics.v1beta.DataAgentService/ListAccessibleDataAgents", "DataAgentService/ListAccessibleDataAgents")
	assertGCPGeminiDataAnalyticsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.geminidataanalytics.v1beta.DataChatService/Chat", "DataChatService/Chat")
	assertGCPGeminiDataAnalyticsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.geminidataanalytics.v1beta.DataChatService/CreateConversation", "DataChatService/CreateConversation")
}

func TestGCPGeminiDataAnalyticsRouter_ListDataAgentsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp geminidataanalytics router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPGeminiDataAnalyticsRouter_CreateDataAgentInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1/dataAgents", []byte(`{"displayName"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp geminidataanalytics router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPGeminiDataAnalyticsRouter_ChatRequiresMessages(t *testing.T) {
	t.Parallel()

	ts := newGCPGeminiDataAnalyticsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1beta/projects/stackyard/locations/us-central1:chat", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"messages":[]
	}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp geminidataanalytics router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPGeminiDataAnalyticsContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPGeminiDataAnalyticsNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp geminidataanalytics router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPGeminiDataAnalyticsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp geminidataanalytics router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bodyBytes := providerContractBody(t, resp)
	body := string(bodyBytes)
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}

	if strings.HasSuffix(path, ":chat") {
		var streamPayload []map[string]any
		if err := json.Unmarshal(bodyBytes, &streamPayload); err != nil {
			t.Fatalf("decode chat stream payload: %v", err)
		}
		if len(streamPayload) == 0 {
			t.Fatalf("expected chat stream payload items")
		}
	}
}
