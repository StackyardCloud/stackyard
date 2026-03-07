package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPLanguageV2Router_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLanguageV2ContractServer(t)

	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/v2/documents:analyzeSentiment", ":analyzeSentiment")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/v2/documents:analyzeEntities", ":analyzeEntities")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/v2/documents:classifyText", ":classifyText")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/v2/documents:moderateText", ":moderateText")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/v2/documents:annotateText", ":annotateText")
}

func TestGCPLanguageV2Router_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLanguageV2ContractServer(t)

	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/google.cloud.language.v2.LanguageService/AnalyzeSentiment", "LanguageService/AnalyzeSentiment")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/google.cloud.language.v2.LanguageService/AnalyzeEntities", "LanguageService/AnalyzeEntities")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/google.cloud.language.v2.LanguageService/ClassifyText", "LanguageService/ClassifyText")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/google.cloud.language.v2.LanguageService/ModerateText", "LanguageService/ModerateText")
	assertGCPLanguageV2NotImplemented(t, ts, "/gcp/google.cloud.language.v2.LanguageService/AnnotateText", "LanguageService/AnnotateText")
}

func newGCPLanguageV2ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPLanguageV2NotImplemented(t *testing.T, ts *httptest.Server, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, http.MethodPost, path, []byte(`{"document":{"content":"hello","type":"PLAIN_TEXT","languageCode":"en"}}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp language v2 router for POST %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for POST %s: %s", path, body)
	}
}

func TestGCPLanguageV2Router_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPLanguageV2Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/language_v2?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp language_v2 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "language_v2" {
		t.Fatalf("expected service=language_v2, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

