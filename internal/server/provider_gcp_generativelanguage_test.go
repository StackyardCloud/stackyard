package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGCPGenerativeLanguageRouter_ListModelsRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/models", nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp generativelanguage router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, `/gcp/v1/models`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPGenerativeLanguageRouter_GenerateContentRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	payload := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/models/gemini-2.0-flash:generateContent", payload, map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp generativelanguage router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, `:generateContent`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}
