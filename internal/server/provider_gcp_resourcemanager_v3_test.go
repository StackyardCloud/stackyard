package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGCPResourceManagerV3Router_CreateProjectRejectsInvalidProjectID(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects", []byte(`{"projectId":"BAD","displayName":"Stackyard Prod","parent":"organizations/123456"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager v3 create project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerV3Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/resourcemanager_v3?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp resourcemanager_v3 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "resourcemanager_v3" {
		t.Fatalf("expected service=resourcemanager_v3, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
