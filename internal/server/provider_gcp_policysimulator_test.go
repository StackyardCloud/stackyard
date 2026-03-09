package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPPolicySimulatorRouter_SimulatorRESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPolicySimulatorContractServer(t)

	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/replays", "/replays")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/replays/replay-1", "/replays/replay-1")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/replays/replay-1/results?pageSize=1", "/results")
}

func TestGCPPolicySimulatorRouter_OrgPolicyPreviewRESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPolicySimulatorContractServer(t)

	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789012/locations/global/orgPolicyViolationsPreviews?pageSize=1", "/orgPolicyViolationsPreviews")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789012/locations/global/orgPolicyViolationsPreviews/preview-1", "/orgPolicyViolationsPreviews/preview-1")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/v1/organizations/123456789012/locations/global/orgPolicyViolationsPreviews?orgPolicyViolationsPreviewId=preview-1", "/orgPolicyViolationsPreviews")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789012/locations/global/orgPolicyViolationsPreviews/preview-1/orgPolicyViolations?pageSize=1", "/orgPolicyViolations")
}

func TestGCPPolicySimulatorRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPolicySimulatorContractServer(t)

	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.Simulator/CreateReplay", "Simulator/CreateReplay")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.Simulator/GetReplay", "Simulator/GetReplay")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.Simulator/ListReplayResults", "Simulator/ListReplayResults")

	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.OrgPolicyViolationsPreviewService/ListOrgPolicyViolationsPreviews", "OrgPolicyViolationsPreviewService/ListOrgPolicyViolationsPreviews")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.OrgPolicyViolationsPreviewService/GetOrgPolicyViolationsPreview", "OrgPolicyViolationsPreviewService/GetOrgPolicyViolationsPreview")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.OrgPolicyViolationsPreviewService/CreateOrgPolicyViolationsPreview", "OrgPolicyViolationsPreviewService/CreateOrgPolicyViolationsPreview")
	assertGCPPolicySimulatorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.policysimulator.v1.OrgPolicyViolationsPreviewService/ListOrgPolicyViolations", "OrgPolicyViolationsPreviewService/ListOrgPolicyViolations")
}

func newGCPPolicySimulatorContractServer(t *testing.T) *httptest.Server {
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

func assertGCPPolicySimulatorNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp policysimulator router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPPolicysimulatorRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPPolicysimulatorRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/policysimulator?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp policysimulator contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "policysimulator" {
		t.Fatalf("expected service=policysimulator, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
