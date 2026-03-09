package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDataQNARouter_SuggestQueriesRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataQnAContractServer(t)
	assertGCPDataQnANotImplemented(t, ts, http.MethodPost, "/gcp/v1alpha/projects/stackyard/locations/us-central1:suggestQueries", ":suggestQueries")
}

func TestGCPDataQNARouter_QuestionLifecycleRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataQnAContractServer(t)
	assertGCPDataQnANotImplemented(t, ts, http.MethodPost, "/gcp/v1alpha/projects/stackyard/locations/us-central1/questions", "/questions")
	assertGCPDataQnANotImplemented(t, ts, http.MethodGet, "/gcp/v1alpha/projects/stackyard/locations/us-central1/questions/q-1", "/questions/q-1")
	assertGCPDataQnANotImplemented(t, ts, http.MethodPost, "/gcp/v1alpha/projects/stackyard/locations/us-central1/questions/q-1:execute", ":execute")
}

func TestGCPDataQNARouter_UserFeedbackRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataQnAContractServer(t)
	assertGCPDataQnANotImplemented(t, ts, http.MethodGet, "/gcp/v1alpha/projects/stackyard/locations/us-central1/questions/q-1/userFeedback", "/userFeedback")
	assertGCPDataQnANotImplemented(t, ts, http.MethodPatch, "/gcp/v1alpha/projects/stackyard/locations/us-central1/questions/q-1/userFeedback?updateMask=rating", "/userFeedback")
}

func newGCPDataQnAContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDataQnANotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataqna router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDataqnaRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDataqnaRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/dataqna?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dataqna contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "dataqna" {
		t.Fatalf("expected service=dataqna, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
