package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureContentModeratorTextModerationDetectLanguageAndScreen(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "text/plain",
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessText/DetectLanguage", []byte("hola equipo gracias"), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 detect language, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	detect := providerContractJSONMap(t, resp)
	if got := textModerationString(detect["DetectedLanguage"]); got != "spa" {
		t.Fatalf("expected detected language spa, got %#v", detect)
	}
	if code := textModerationStatusCode(detect); code != 3000 {
		t.Fatalf("expected status code 3000, got %#v", detect)
	}

	screenPath := "/azure/contentmoderator/moderate/v1.0/ProcessText/Screen/?language=eng&autocorrect=true&PII=true&listId=42&classify=true"
	screenBody := []byte("teh damn invoice was sent to ops@example.com")
	resp = providerContractRequest(t, ts, http.MethodPost, screenPath, screenBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 screen text, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	screen := providerContractJSONMap(t, resp)
	if got := textModerationString(screen["Language"]); got != "eng" {
		t.Fatalf("expected language eng, got %#v", screen)
	}
	if got := textModerationString(screen["OriginalText"]); !strings.Contains(got, "teh") {
		t.Fatalf("expected original text with typo, got %#v", screen)
	}
	if got := textModerationString(screen["NormalizedText"]); !strings.Contains(got, "the") {
		t.Fatalf("expected autocorrected normalized text, got %#v", screen)
	}
	terms, ok := screen["Terms"].([]any)
	if !ok || len(terms) == 0 {
		t.Fatalf("expected non-empty Terms, got %#v", screen)
	}
	if _, ok := screen["PII"].(map[string]any); !ok {
		t.Fatalf("expected PII object in screen response, got %#v", screen)
	}
	classification, ok := screen["Classification"].(map[string]any)
	if !ok {
		t.Fatalf("expected Classification object, got %#v", screen)
	}
	if review, _ := classification["ReviewRecommended"].(bool); !review {
		t.Fatalf("expected ReviewRecommended=true, got %#v", classification)
	}
}

func TestAzureContentModeratorTextModerationValidationAndNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessText/DetectLanguage", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing detect-language body, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := providerContractJSONMap(t, resp); body["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest for missing body, got %#v", body)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessText/Screen/?autocorrect=not-bool", []byte("hello"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid autocorrect, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessText/Screen/?PII=true&listId=-1", []byte("hello"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "text/plain",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid listId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessText/Screen/", []byte("hello"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported content type, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/contentmoderator/moderate/v1.0/ProcessText/DetectLanguage", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported method, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, "/azure/contentmoderator/moderate/v1.0/ProcessText/DetectLanguage") {
		t.Fatalf("unexpected not-implemented payload: %s", body)
	}
}

func textModerationStatusCode(payload map[string]any) int {
	status, ok := payload["Status"].(map[string]any)
	if !ok {
		return 0
	}
	code, ok := status["Code"].(float64)
	if !ok {
		return 0
	}
	return int(code)
}

func textModerationString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
