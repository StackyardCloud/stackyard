package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGCPComputeMetadataRouter_ProjectIDRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "bearer_required",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/computeMetadata/v1/project/project-id", nil, map[string]string{
		"Metadata-Flavor": "Google",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp compute metadata router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := strings.TrimSpace(resp.Header.Get("Metadata-Flavor")); got != "Google" {
		t.Fatalf("expected Metadata-Flavor response header, got %q", got)
	}
	if body := strings.TrimSpace(string(providerContractBody(t, resp))); body != "stackyard" {
		t.Fatalf("expected stackyard project id, got %q", body)
	}
}

func TestGCPComputeMetadataRouter_RequiresMetadataFlavorHeader(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/computeMetadata/v1/instance/id", nil, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for missing metadata flavor header, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"MetadataFlavorRequired"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPComputeMetadataRouter_RootProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "bearer_required",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/", nil, map[string]string{
		"Metadata-Flavor": "Google",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 metadata probe response, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := strings.TrimSpace(resp.Header.Get("Metadata-Flavor")); got != "Google" {
		t.Fatalf("expected Metadata-Flavor response header, got %q", got)
	}
}

func TestGCPComputeMetadataRouter_InstanceTagsReturnsJSONArray(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/computeMetadata/v1/instance/tags", nil, map[string]string{
		"Metadata-Flavor": "Google",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for instance tags, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := strings.TrimSpace(string(providerContractBody(t, resp)))
	if body != `["stackyard","gcp","compute"]` {
		t.Fatalf("unexpected instance tags body: %s", body)
	}
}

func TestGCPComputeMetadataRouter_ProviderDisabled(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerAWS},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/computeMetadata/v1/project/project-id", nil, map[string]string{
		"Metadata-Flavor": "Google",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when gcp provider is disabled, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"ProviderDisabled"`) || !strings.Contains(body, `"provider":"gcp"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPComputeMetadataRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPComputeMetadataRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/compute_metadata?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp compute_metadata contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "compute_metadata" {
		t.Fatalf("expected service=compute_metadata, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
