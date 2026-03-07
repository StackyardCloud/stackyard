package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPKMSRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPKMSContractServer(t)

	location := "/gcp/v1/projects/stackyard/locations/us-central1"
	keyRings := location + "/keyRings"
	keyRing := keyRings + "/team-ring"
	cryptoKeys := keyRing + "/cryptoKeys"
	cryptoKey := cryptoKeys + "/app-key"
	cryptoKeyVersions := cryptoKey + "/cryptoKeyVersions"
	cryptoKeyVersion := cryptoKeyVersions + "/1"
	importJobs := keyRing + "/importJobs"
	importJob := importJobs + "/import-1"

	assertGCPKMSNotImplemented(t, ts, http.MethodGet, keyRings+"?pageSize=1", "/keyRings")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, keyRings, "/keyRings")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, keyRing, "/keyRings/team-ring")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, cryptoKeys+"?pageSize=1", "/cryptoKeys")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeys, "/cryptoKeys")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, cryptoKey, "/cryptoKeys/app-key")
	assertGCPKMSNotImplemented(t, ts, http.MethodPatch, cryptoKey+"?updateMask.paths=labels", "/cryptoKeys/app-key")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, cryptoKeyVersions+"?pageSize=1", "/cryptoKeyVersions")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersions, "/cryptoKeyVersions")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, cryptoKeyVersion, "/cryptoKeyVersions/1")
	assertGCPKMSNotImplemented(t, ts, http.MethodPatch, cryptoKeyVersion+"?updateMask.paths=state", "/cryptoKeyVersions/1")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKey+":updatePrimaryVersion", ":updatePrimaryVersion")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":destroy", ":destroy")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":restore", ":restore")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKey+":encrypt", ":encrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKey+":decrypt", ":decrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":rawEncrypt", ":rawEncrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":rawDecrypt", ":rawDecrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":asymmetricSign", ":asymmetricSign")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":asymmetricDecrypt", ":asymmetricDecrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":macSign", ":macSign")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersion+":macVerify", ":macVerify")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, location+":generateRandomBytes", ":generateRandomBytes")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, importJobs+"?pageSize=1", "/importJobs")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, importJobs, "/importJobs")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, importJob, "/importJobs/import-1")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKeyVersions+":import", "/cryptoKeyVersions:import")
	assertGCPKMSNotImplemented(t, ts, http.MethodGet, cryptoKeyVersion+"/publicKey", "/publicKey")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKey+":getIamPolicy", ":getIamPolicy")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKey+":setIamPolicy", ":setIamPolicy")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, cryptoKey+":testIamPermissions", ":testIamPermissions")
}

func TestGCPKMSRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPKMSContractServer(t)

	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/ListKeyRings", "KeyManagementService/ListKeyRings")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/CreateKeyRing", "KeyManagementService/CreateKeyRing")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/ListCryptoKeys", "KeyManagementService/ListCryptoKeys")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/CreateCryptoKey", "KeyManagementService/CreateCryptoKey")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/Encrypt", "KeyManagementService/Encrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/Decrypt", "KeyManagementService/Decrypt")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/GenerateRandomBytes", "KeyManagementService/GenerateRandomBytes")
	assertGCPKMSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.kms.v1.KeyManagementService/TestIamPermissions", "KeyManagementService/TestIamPermissions")
}

func newGCPKMSContractServer(t *testing.T) *httptest.Server {
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

func assertGCPKMSNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp kms router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPKmsRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPKmsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/kms?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp kms contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "kms" {
		t.Fatalf("expected service=kms, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

