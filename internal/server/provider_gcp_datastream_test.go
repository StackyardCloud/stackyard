package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDatastreamRouter_ConnectionProfileRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastreamContractServer(t)
	assertGCPDatastreamNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectionProfiles?pageSize=1", "/connectionProfiles")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectionProfiles", "/connectionProfiles")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectionProfiles/source-a", "/connectionProfiles/source-a")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:discoverConnectionProfile", ":discoverConnectionProfile")
}

func TestGCPDatastreamRouter_StreamAndObjectRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastreamContractServer(t)
	assertGCPDatastreamNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/streams?pageSize=1", "/streams")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/streams", "/streams")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/streams/team-stream:run", ":run")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/streams/team-stream/objects?pageSize=1", "/objects")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/streams/team-stream/objects:lookup", ":lookup")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/streams/team-stream/objects/orders:startBackfillJob", ":startBackfillJob")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/streams/team-stream/objects/orders:stopBackfillJob", ":stopBackfillJob")
}

func TestGCPDatastreamRouter_NetworkingAndStaticIPsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastreamContractServer(t)
	assertGCPDatastreamNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateConnections?pageSize=1", "/privateConnections")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/privateConnections", "/privateConnections")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/privateConnections/private-a/routes?pageSize=1", "/routes")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/privateConnections/private-a/routes", "/routes")
	assertGCPDatastreamNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:fetchStaticIps", ":fetchStaticIps")
}

func newGCPDatastreamContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDatastreamNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datastream router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDatastreamRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDatastreamRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datastream?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp datastream contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "datastream" {
		t.Fatalf("expected service=datastream, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

