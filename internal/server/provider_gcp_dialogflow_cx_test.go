package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDialogflowCXRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowCXContractServer(t)
	base := "/gcp/v3/projects/stackyard/locations/us-central1"

	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, base+"/agents?pageSize=1", nil, "agents")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, base+"/agents", []byte(`{"agent":{"displayName":"stackyard-cx-agent"}}`), "agents/agent-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, base+"/agents/agent-1", nil, "agent-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPatch, base+"/agents/agent-1", []byte(`{"agent":{"displayName":"updated"}}`), "agent-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, base+"/agents/agent-1:validate", []byte(`{}`), "validationErrors")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, base+"/agents/agent-1/validationResult", nil, "validationErrors")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, base+"/agents/agent-1:export", []byte(`{}`), "operations")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, base+"/agents/agent-1:restore", []byte(`{}`), "operations")

	agent := base + "/agents/agent-1"
	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, agent+"/flows?pageSize=1", nil, "flows")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, agent+"/flows", []byte(`{"flow":{"displayName":"stackyard-cx-flow"}}`), "flows/flow-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, agent+"/flows/flow-1", nil, "flow-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPatch, agent+"/flows/flow-1", []byte(`{"flow":{"displayName":"updated-flow"}}`), "flow-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, agent+"/flows/flow-1:validate", []byte(`{}`), "validationErrors")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, agent+"/flows/flow-1/validationResult", nil, "validationErrors")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, agent+"/flows/flow-1:train", []byte(`{}`), "operations")

	session := agent + "/sessions/session-1"
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, session+":detectIntent", []byte(`{"queryInput":{"text":{"text":"hello","languageCode":"en"}}}`), "queryResult")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, session+":matchIntent", []byte(`{"queryInput":{"text":{"text":"match","languageCode":"en"}}}`), "queryResult")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, session+":fulfillIntent", []byte(`{"matchIntentRequest":{"queryInput":{"text":{"text":"fulfill","languageCode":"en"}}}}`), "queryResult")

	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodGet, base+"/operations/op-1", nil, "op-1")
	assertGCPDialogflowCXSuccess(t, ts, http.MethodPost, base+"/operations/op-1:cancel", []byte(`{}`), "{}")
}

func TestGCPDialogflowCXRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowCXContractServer(t)
	assertGCPDialogflowCXNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.dialogflow.cx.v3.Sessions/DetectIntent", "google.cloud.dialogflow.cx.v3.Sessions/DetectIntent")
}

func TestGCPDialogflowCXRouter_ListAgentsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowCXContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/agents?pageSize=invalid", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dialogflow cx router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPDialogflowCXRouter_DetectIntentRequiresQueryInput(t *testing.T) {
	t.Parallel()

	ts := newGCPDialogflowCXContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1/agents/agent-1/sessions/session-1:detectIntent", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dialogflow cx router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPDialogflowCXContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDialogflowCXNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dialogflow cx router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPDialogflowCXSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dialogflow cx router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
