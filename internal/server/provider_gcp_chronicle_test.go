package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPChronicleRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPChronicleContractServer(t)
	base := "/gcp/v1/projects/stackyard/locations/us/instances/default"

	assertGCPChronicleSuccess(t, ts, http.MethodGet, base, nil, "instances/default")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/dataAccessLabels?pageSize=1", nil, "dataAccessLabels")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/dataAccessLabels/team-label", nil, "team-label")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/dataAccessScopes?pageSize=1", nil, "dataAccessScopes")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/referenceLists?pageSize=1", nil, "referenceLists")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules?pageSize=1", nil, "rules")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules/team-rule", nil, "team-rule")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules/team-rule:listRevisions?pageSize=1", nil, "ruleRevisions")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules/-/deployments?pageSize=1", nil, "ruleDeployments")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules/team-rule/deployment", nil, "deployment")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules/team-rule/retrohunts?pageSize=1", nil, "retrohunts")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/rules/team-rule/retrohunts/team-retrohunt", nil, "team-retrohunt")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/watchlists?pageSize=1", nil, "watchlists")
	assertGCPChronicleSuccess(t, ts, http.MethodGet, base+"/watchlists/high-risk-entities", nil, "high-risk-entities")
}

func TestGCPChronicleRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPChronicleContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.chronicle.v1.RuleService/ListRules", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp chronicle router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, "RuleService/ListRules") {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPChronicleRouter_ListRulesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPChronicleContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us/instances/default/rules?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp chronicle router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPChronicleContractServer(t *testing.T) *httptest.Server {
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

func assertGCPChronicleSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp chronicle router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
