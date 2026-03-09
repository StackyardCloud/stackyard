package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureAIServicesLanguageAnalyzeTextRouteReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	body := []byte(`{
		"kind":"SentimentAnalysis",
		"analysisInput":{
			"documents":[
				{"id":"1","language":"en","text":"The food was delicious and the staff was friendly."}
			]
		},
		"parameters":{"modelVersion":"latest"}
	}`)
	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/language/:analyze-text?api-version=2024-11-01&showStats=true", body, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for analyze text route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["status"] != "ok" || payload["provider"] != providerAzure {
		t.Fatalf("unexpected not-implemented payload: %#v", payload)
	}
	if payload["path"] != "/azure/language/:analyze-text" {
		t.Fatalf("expected path /azure/language/:analyze-text, got %#v", payload["path"])
	}
}

func TestAzureAIServicesLanguageAnalyzeTextUnsupportedMethodReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	path := "/azure/language/:analyze-text"
	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported method on analyze text route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, path) {
		t.Fatalf("unexpected payload: %s", body)
	}
}
