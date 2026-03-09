package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureContentModeratorImageModerationURLWorkflow(t *testing.T) {
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
		"Content-Type":  "application/json",
	}
	urlBody := []byte(`{"DataRepresentation":"URL","Value":"https://example.com/safe-image.jpg"}`)

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=url", urlBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 evaluating URL image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	evaluate := providerContractJSONMap(t, resp)
	if statusCode := azureContentModeratorTestStatusCode(evaluate); statusCode != 3000 {
		t.Fatalf("expected status code 3000 in evaluate payload, got %#v", evaluate["Status"])
	}
	if _, ok := evaluate["AdultClassificationScore"].(float64); !ok {
		t.Fatalf("expected AdultClassificationScore in evaluate payload, got %#v", evaluate)
	}
	if strings.TrimSpace(azureContentModeratorTestString(evaluate["TrackingId"])) == "" {
		t.Fatalf("expected TrackingId in evaluate payload, got %#v", evaluate)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/FindFaces?overload=url", urlBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 finding faces from URL image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	findFaces := providerContractJSONMap(t, resp)
	if findFaces["Count"] == nil {
		t.Fatalf("expected Count in find-faces payload, got %#v", findFaces)
	}
	if faces, ok := findFaces["Faces"].([]any); !ok || len(faces) == 0 {
		t.Fatalf("expected non-empty Faces payload, got %#v", findFaces["Faces"])
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Match?overload=url&listId=12345", urlBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 matching URL image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	match := providerContractJSONMap(t, resp)
	if isMatch, _ := match["IsMatch"].(bool); !isMatch {
		t.Fatalf("expected IsMatch=true, got %#v", match)
	}
	if matches, ok := match["Matches"].([]any); !ok || len(matches) == 0 {
		t.Fatalf("expected Matches payload, got %#v", match["Matches"])
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=url&language=eng&enhanced=false", urlBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OCR from URL image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	ocr := providerContractJSONMap(t, resp)
	if got := azureContentModeratorTestString(ocr["Language"]); got != "eng" {
		t.Fatalf("expected OCR language eng, got %#v", ocr["Language"])
	}
	if strings.TrimSpace(azureContentModeratorTestString(ocr["Text"])) == "" {
		t.Fatalf("expected OCR text payload, got %#v", ocr)
	}
}

func TestAzureContentModeratorImageModerationStreamWorkflow(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=stream", []byte("adult-scene-content"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "image/jpeg",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 evaluating stream image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	evaluate := providerContractJSONMap(t, resp)
	if isAdult, _ := evaluate["IsImageAdultClassified"].(bool); !isAdult {
		t.Fatalf("expected IsImageAdultClassified=true for adult hint payload, got %#v", evaluate)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/FindFaces?overload=stream", []byte("noface-image"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "image/png",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 finding faces from stream image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	findFaces := providerContractJSONMap(t, resp)
	if count, _ := findFaces["Count"].(float64); count != 0 {
		t.Fatalf("expected zero faces for noface hint, got %#v", findFaces)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=stream&language=eng&enhanced=true", []byte("receipt-line-items"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "image/png",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OCR from stream image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	ocr := providerContractJSONMap(t, resp)
	if candidates, ok := ocr["Candidates"].([]any); !ok || len(candidates) == 0 {
		t.Fatalf("expected enhanced OCR candidates, got %#v", ocr)
	}
}

func TestAzureContentModeratorImageModerationValidationAndNotImplemented(t *testing.T) {
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
		"Content-Type":  "application/json",
	}

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=invalid", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid overload, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := providerContractJSONMap(t, resp); body["error"] != "InvalidRequest" {
		t.Fatalf("expected InvalidRequest for invalid overload, got %#v", body)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=url", []byte(`{"DataRepresentation":"URL"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing URL value, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Match?overload=url", []byte(`{"DataRepresentation":"URL","Value":"https://example.com/image.jpg"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing listId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=url", []byte(`{"DataRepresentation":"URL","Value":"https://example.com/image.jpg"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing language, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=stream&language=eng&enhanced=true", []byte("invoice"), map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "image/tiff",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for enhanced tiff OCR, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=url", nil, map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for unsupported method, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"azure"`) || !strings.Contains(body, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate") {
		t.Fatalf("unexpected not-implemented body: %s", body)
	}
}

func azureContentModeratorTestStatusCode(payload map[string]any) int {
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

func azureContentModeratorTestString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}
