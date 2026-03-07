package server

import (
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

func TestGCPConfidentialComputingRouter_CreateChallengeRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/challenges", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp confidential computing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, ok := body["name"].(string); !ok || got != "projects/stackyard/locations/us-central1/challenges/ch-1" {
		t.Fatalf("unexpected challenge name: %#v", body["name"])
	}
	if used, ok := body["used"].(bool); !ok || used {
		t.Fatalf("expected used=false in create challenge response, got %#v", body["used"])
	}
}

func TestGCPConfidentialComputingRouter_VerifyAttestationRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/challenges/ch-1:verifyAttestation", []byte(`{"attester":"confidential-space"}`), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp confidential computing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	token, ok := body["oidcClaimsToken"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		t.Fatalf("expected oidcClaimsToken in verify response, got %#v", body["oidcClaimsToken"])
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected stackyard token format, got %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	if !strings.Contains(string(payload), "attester=confidential-space") {
		t.Fatalf("unexpected verify payload %q", string(payload))
	}
}

func TestGCPConfidentialComputingRouter_VerifyConfidentialSpaceRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/challenges/ch-1:verifyConfidentialSpace", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp confidential computing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	token, ok := body["oidcClaimsToken"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		t.Fatalf("expected oidcClaimsToken in verify response, got %#v", body["oidcClaimsToken"])
	}
}

func TestGCPConfidentialComputingRouter_VerifyConfidentialGkeRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/challenges/ch-1:verifyConfidentialGke", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp confidential computing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	token, ok := body["oidcClaimsToken"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		t.Fatalf("expected oidcClaimsToken in verify response, got %#v", body["oidcClaimsToken"])
	}
}

func TestGCPConfidentialComputingRouter_ListLocationsRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "confidentialcomputing",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp confidential computing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	items, ok := body["locations"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one location in page, got %#v", body["locations"])
	}
	if next, ok := body["nextPageToken"].(string); !ok || next != "1" {
		t.Fatalf("expected nextPageToken=1, got %#v", body["nextPageToken"])
	}
}

func TestGCPConfidentialComputingRouter_GetLocationRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "confidentialcomputing",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp confidential computing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, ok := body["name"].(string); !ok || got != "projects/stackyard/locations/us-central1" {
		t.Fatalf("unexpected location name %#v", body["name"])
	}
}

func TestGCPConfidentialComputingRouter_ListLocationsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "confidentialcomputing",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPConfidentialComputingRouter_ListLocationsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageToken=abc", nil, map[string]string{
		"X-Stackyard-GCP-Service": "confidentialcomputing",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPConfidentialComputingRouter_ListLocationsOutOfRangePageToken(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageToken=9", nil, map[string]string{
		"X-Stackyard-GCP-Service": "confidentialcomputing",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-range pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}
