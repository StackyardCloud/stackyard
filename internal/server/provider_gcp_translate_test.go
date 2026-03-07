package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTranslateRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)

	assertGCPTranslateSuccess(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPTranslateSuccess(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:translateText", []byte(`{
		"contents":["hello from stackyard"],
		"targetLanguageCode":"es"
	}`), `"translations":[`)
	assertGCPTranslateSuccess(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/supportedLanguages", nil, `"languages":[`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1/glossaries", []byte(`{
		"glossary":{"displayName":"Stackyard Glossary"}
	}`), `"name":"projects/stackyard/locations/us-central1/operations/createGlossary.glossary-1"`)
	assertGCPTranslateSuccess(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/glossaries?pageSize=1", nil, `"glossaries":[`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1/datasets", []byte(`{
		"dataset":{"displayName":"Stackyard Dataset"}
	}`), `"name":"projects/stackyard/locations/us-central1/operations/createDataset.dataset-1"`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1/adaptiveMtDatasets", []byte(`{
		"adaptiveMtDataset":{"displayName":"Adaptive Dataset"}
	}`), `"name":"projects/stackyard/locations/us-central1/adaptiveMtDatasets/adaptive-dataset-1"`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1/models", []byte(`{
		"model":{"displayName":"Stackyard Model"}
	}`), `"name":"projects/stackyard/locations/us-central1/operations/createModel.model-1"`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:batchTranslateText", []byte(`{
		"sourceLanguageCode":"en",
		"targetLanguageCodes":["es"],
		"inputConfigs":[{"mimeType":"text/plain"}],
		"outputConfig":{"gcsDestination":{"outputUriPrefix":"gs://stackyard/output/"}}
	}`), `"name":"projects/stackyard/locations/us-central1/operations/batchTranslateText.stackyard"`)
	assertGCPTranslateSuccess(t, ts, http.MethodPost, gcpTranslateGRPCPathPrefix+"TranslateText", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"contents":["hello grpc json"],
		"targetLanguageCode":"es"
	}`), `"translations":[`)
}

func TestGCPTranslateRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:translateText", []byte(`{"contents"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate-apiv3",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp translate invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTranslateRouter_TranslateTextRequiresTargetLanguage(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:translateText", []byte(`{
		"contents":["hello"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp translate translateText missing target, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTranslateRouter_ListRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/glossaries?pageToken=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "translate",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp translate list glossaries invalid page token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTranslateRouter_GetGlossaryMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/glossaries/missing-glossary", nil, map[string]string{
		"X-Stackyard-GCP-Service": "translate",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp translate get glossary missing, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTranslateRouter_CreateModelRequiresModel(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1/models", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp translate create model missing model, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTranslateRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)

	translateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:translateText", []byte(`{
		"contents":["shape check"],
		"targetLanguageCode":"es"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate",
	})
	if translateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp translate translateText, got %d body=%s", translateResp.StatusCode, string(providerContractBody(t, translateResp)))
	}
	translateBody := providerContractJSONMap(t, translateResp)
	translations, ok := translateBody["translations"].([]any)
	if !ok || len(translations) == 0 {
		t.Fatalf("expected translations array, got %#v", translateBody["translations"])
	}
	firstTranslation, _ := translations[0].(map[string]any)
	if _, ok := firstTranslation["translatedText"].(string); !ok {
		t.Fatalf("expected translatedText string, got %#v", firstTranslation["translatedText"])
	}

	listGlossariesResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/glossaries?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "translate",
	})
	if listGlossariesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp translate listGlossaries, got %d body=%s", listGlossariesResp.StatusCode, string(providerContractBody(t, listGlossariesResp)))
	}
	listGlossariesBody := providerContractJSONMap(t, listGlossariesResp)
	glossaries, ok := listGlossariesBody["glossaries"].([]any)
	if !ok || len(glossaries) == 0 {
		t.Fatalf("expected glossaries array, got %#v", listGlossariesBody["glossaries"])
	}
	firstGlossary, _ := glossaries[0].(map[string]any)
	if _, ok := firstGlossary["name"].(string); !ok {
		t.Fatalf("expected glossary name string, got %#v", firstGlossary["name"])
	}

	batchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/stackyard/locations/us-central1:batchTranslateText", []byte(`{
		"sourceLanguageCode":"en",
		"targetLanguageCodes":["es"],
		"inputConfigs":[{"mimeType":"text/plain"}],
		"outputConfig":{"gcsDestination":{"outputUriPrefix":"gs://stackyard/output/"}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "translate",
	})
	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp translate batchTranslateText, got %d body=%s", batchResp.StatusCode, string(providerContractBody(t, batchResp)))
	}
	batchBody := providerContractJSONMap(t, batchResp)
	if _, ok := batchBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", batchBody["name"])
	}
	if _, ok := batchBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", batchBody["done"])
	}
}

func TestGCPTranslateRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPTranslateContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/us-central1/translate?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "translate",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp translate contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "translate" {
		t.Fatalf("expected service=translate, got %#v", body["service"])
	}
	methods, ok := body["methods"].([]any)
	if !ok || len(methods) == 0 {
		t.Fatalf("expected methods array in contract probe response, got %#v", body["methods"])
	}
}

func assertGCPTranslateSuccess(t *testing.T, ts *httptest.Server, method, path string, body []byte, contains string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "translate-apiv3",
	}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, body, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp translate %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if contains != "" {
		payload := string(providerContractBody(t, resp))
		if !strings.Contains(payload, contains) {
			t.Fatalf("expected body to contain %q, got %s", contains, payload)
		}
	}
}

func newGCPTranslateContractServer(t *testing.T) *httptest.Server {
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
