package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPGenerativeLanguageRouter_ListModelsRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/models?pageSize=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp generativelanguage router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	models, ok := body["models"].([]any)
	if !ok || len(models) != 1 {
		t.Fatalf("expected one model in response, got %#v", body["models"])
	}
	if next, ok := body["nextPageToken"].(string); !ok || next != "1" {
		t.Fatalf("expected nextPageToken=1, got %#v", body["nextPageToken"])
	}
}

func TestGCPGenerativeLanguageRouter_GetModelRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/models/gemini-2.0-flash", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp generativelanguage router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, ok := body["name"].(string); !ok || got != "models/gemini-2.0-flash" {
		t.Fatalf("unexpected model name: %#v", body["name"])
	}
}

func TestGCPGenerativeLanguageRouter_ModelActionsReturnTypedResponses(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	payload := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/models/gemini-2.0-flash:generateContent", payload, map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for generateContent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	candidates, ok := body["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		t.Fatalf("expected candidates in generateContent response, got %#v", body["candidates"])
	}

	streamResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/models/gemini-2.0-flash:streamGenerateContent", payload, map[string]string{
		"Content-Type": "application/json",
	})
	if streamResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for streamGenerateContent, got %d body=%s", streamResp.StatusCode, string(providerContractBody(t, streamResp)))
	}
	var streamPayload []map[string]any
	if err := json.Unmarshal(providerContractBody(t, streamResp), &streamPayload); err != nil {
		t.Fatalf("decode stream response: %v", err)
	}
	if len(streamPayload) == 0 {
		t.Fatalf("expected stream response payload items")
	}

	countResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/models/gemini-2.0-flash:countTokens", payload, map[string]string{
		"Content-Type": "application/json",
	})
	if countResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for countTokens, got %d body=%s", countResp.StatusCode, string(providerContractBody(t, countResp)))
	}
	countBody := providerContractJSONMap(t, countResp)
	if _, ok := countBody["totalTokens"].(float64); !ok {
		t.Fatalf("expected numeric totalTokens, got %#v", countBody["totalTokens"])
	}
}

func TestGCPGenerativeLanguageRouter_ListModelsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/models?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPGenerativeLanguageRouter_ListModelsOutOfRangePageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/models?pageToken=9", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for out-of-range pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPGenerativeLanguageRouter_GenerateContentInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/models/gemini-2.0-flash:generateContent", []byte(`{"contents"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPGenerativeLanguageRouter_GenerateContentMissingContents(t *testing.T) {
	t.Parallel()

	ts := newGCPGenerativeLanguageContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/models/gemini-2.0-flash:generateContent", []byte(`{"contents":[]}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty contents, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPGenerativeLanguageContractServer(t *testing.T) *httptest.Server {
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
