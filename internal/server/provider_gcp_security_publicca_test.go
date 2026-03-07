package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecurityPublicCARouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPublicCAContractServer(t)

	assertGCPSecurityPublicCASuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSecurityPublicCASuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global", nil, `"locationId":"global"`)
	assertGCPSecurityPublicCASuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/externalAccountKeys", []byte(`{"externalAccountKey":{"name":"projects/stackyard/locations/global/externalAccountKeys/eak-1"}}`), `"name":"projects/stackyard/locations/global/externalAccountKeys/eak-1"`)
}

func TestGCPSecurityPublicCARouter_ListLocationsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPublicCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "security-publicca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_publicca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPublicCARouter_CreateExternalAccountKeyRequiresPayload(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPublicCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/externalAccountKeys", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-publicca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_publicca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPublicCARouter_CreateExternalAccountKeyRejectsNonGlobalLocation(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPublicCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/externalAccountKeys", []byte(`{"externalAccountKey":{"name":"projects/stackyard/locations/us-central1/externalAccountKeys/eak-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-publicca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_publicca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPublicCARouter_CreateExternalAccountKeyNameMustMatchParent(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPublicCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/externalAccountKeys", []byte(`{"externalAccountKey":{"name":"projects/other/locations/global/externalAccountKeys/eak-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-publicca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_publicca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPublicCARouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/security_publicca?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp security_publicca contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "security_publicca" {
		t.Fatalf("expected service=security_publicca, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPSecurityPublicCARouter_CreateExternalAccountKeyOutputShape(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPublicCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/externalAccountKeys", []byte(`{"externalAccountKey":{"name":"projects/stackyard/locations/global/externalAccountKeys/eak-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-publicca",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp security_publicca create external account key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in create response, got %#v", body["name"])
	}
	if _, ok := body["keyId"].(string); !ok {
		t.Fatalf("expected typed keyId field in create response, got %#v", body["keyId"])
	}
	if _, ok := body["b64MacKey"].(string); !ok {
		t.Fatalf("expected typed b64MacKey field in create response, got %#v", body["b64MacKey"])
	}
}

func newGCPSecurityPublicCAContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecurityPublicCASuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "security-publicca",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp security_publicca router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
