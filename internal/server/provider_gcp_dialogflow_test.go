package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDialogflowRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowContractServer(t)

	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/agent", nil, "Stackyard Agent")
	assertGCPDialogflowSuccess(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/agent", []byte(`{"agent":{"displayName":"Updated Agent"}}`), "Stackyard Agent")
	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/-/agent:search?pageSize=1", nil, "agents")
	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/agent/validationResult", nil, "validationErrors")
	assertGCPDialogflowSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/agent:train", []byte(`{}`), "operations")

	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/agent/intents?pageSize=1", nil, "intents")
	assertGCPDialogflowSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/agent/intents", []byte(`{"intent":{"displayName":"orders.intent"}}`), "orders.intent")
	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/agent/intents/intent-1", nil, "intent-1")
	assertGCPDialogflowSuccess(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/agent/intents/intent-1", []byte(`{"intent":{"displayName":"intent-1"}}`), "intent-1")
	assertGCPDialogflowSuccess(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/agent/intents/intent-1", nil, "{}")
	assertGCPDialogflowSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/agent/sessions/session-1:detectIntent", []byte(`{"queryInput":{"text":{"text":"hello","languageCode":"en"}}}`), "queryResult")

	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/operations?pageSize=1", nil, "operations")
	assertGCPDialogflowSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/operations/op-1", nil, "op-1")
	assertGCPDialogflowSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/operations/op-1:cancel", []byte(`{}`), "{}")
}

func TestGCPDialogflowRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowContractServer(t)

	assertGCPDialogflowNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.dialogflow.v2.Sessions/DetectIntent", "google.cloud.dialogflow.v2.Sessions/DetectIntent")
}

func TestGCPDialogflowRouter_ListIntentsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/agent/intents?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dialogflow router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPDialogflowRouter_DetectIntentRequiresQueryInput(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/agent/sessions/session-1:detectIntent", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dialogflow router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPDialogflowContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDialogflowNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dialogflow router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPDialogflowSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dialogflow router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
