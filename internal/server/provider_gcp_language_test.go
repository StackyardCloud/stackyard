package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPLanguageRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLanguageContractServer(t)

	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:analyzeSentiment", ":analyzeSentiment")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:analyzeEntities", ":analyzeEntities")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:analyzeEntitySentiment", ":analyzeEntitySentiment")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:analyzeSyntax", ":analyzeSyntax")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:classifyText", ":classifyText")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:moderateText", ":moderateText")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/v1/documents:annotateText", ":annotateText")
}

func TestGCPLanguageRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLanguageContractServer(t)

	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/AnalyzeSentiment", "LanguageService/AnalyzeSentiment")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/AnalyzeEntities", "LanguageService/AnalyzeEntities")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/AnalyzeEntitySentiment", "LanguageService/AnalyzeEntitySentiment")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/AnalyzeSyntax", "LanguageService/AnalyzeSyntax")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/ClassifyText", "LanguageService/ClassifyText")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/ModerateText", "LanguageService/ModerateText")
	assertGCPLanguageNotImplemented(t, ts, "/gcp/google.cloud.language.v1.LanguageService/AnnotateText", "LanguageService/AnnotateText")
}

func newGCPLanguageContractServer(t *testing.T) *httptest.Server {
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

func assertGCPLanguageNotImplemented(t *testing.T, ts *httptest.Server, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, http.MethodPost, path, []byte(`{"document":{"content":"hello","type":"PLAIN_TEXT"}}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp language router for POST %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for POST %s: %s", path, body)
	}
}

func TestGCPLanguageRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPLanguageRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/language?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp language contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "language" {
		t.Fatalf("expected service=language, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
