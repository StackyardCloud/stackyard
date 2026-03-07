package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGCPAiplatformRouter_RESTRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/aiplatform/v1/projects/stackyard/locations/us-central1/datasets", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from aiplatform router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if _, ok := body["datasets"].([]any); !ok {
		t.Fatalf("expected datasets field in response body, got %#v", body)
	}
}

func TestGCPAiplatformRouter_GRPCRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/google.cloud.aiplatform.v1.DatasetService/ListDatasets", nil, map[string]string{
		"Content-Type": "application/grpc",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 gRPC foundation response from aiplatform route, got %d body=%q", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := resp.Header.Get("Content-Type"); got != "application/grpc" {
		t.Fatalf("expected gRPC content type, got %q", got)
	}
	body := providerContractBody(t, resp)
	if len(body) < 5 {
		t.Fatalf("expected framed gRPC payload, got %d bytes", len(body))
	}
}

func TestGCPAiplatformRouter_RESTInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/aiplatform/v1/projects/stackyard/locations/us-central1/models?pageSize=abc", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAiplatformRouter_RESTInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/aiplatform/v1/projects/stackyard/locations/us-central1/customJobs?pageToken=oops", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAiplatformRouter_RESTOutOfRangePageToken(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/aiplatform/v1/projects/stackyard/locations/us-central1/endpoints?pageToken=9", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-range pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}
