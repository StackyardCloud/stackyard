package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMediaTranslationRouter_StreamingTranslateSpeechRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMediaTranslationContractServer(t)
	assertGCPMediaTranslationSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/StreamingTranslateSpeech", []byte(`{
		"streamingConfig":{
			"audioConfig":{
				"audioEncoding":"linear16",
				"sourceLanguageCode":"en-US",
				"targetLanguageCode":"es-ES",
				"sampleRateHertz":16000
			},
			"singleUtterance":true
		}
	}`), "textTranslationResult")
}

func TestGCPMediaTranslationRouter_UnknownServiceMethodNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMediaTranslationContractServer(t)
	assertGCPMediaTranslationNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/UnknownMethod", "UnknownMethod")
}

func TestGCPMediaTranslationRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPMediaTranslationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/StreamingTranslateSpeech", []byte(`{"streamingConfig"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp mediatranslation router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMediaTranslationRouter_StreamingConfigRequired(t *testing.T) {
	t.Parallel()

	ts := newGCPMediaTranslationContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/StreamingTranslateSpeech", []byte(`{"audioContent":"c3RhY2t5YXJk"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp mediatranslation router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMediaTranslationContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMediaTranslationNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp mediatranslation router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMediaTranslationSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp mediatranslation router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
