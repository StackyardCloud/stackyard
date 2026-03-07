package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpeechRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)

	assertGCPSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)

	notImplementedResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/UnknownMethod", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if notImplementedResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp speech unknown route, got %d body=%s", notImplementedResp.StatusCode, string(providerContractBody(t, notImplementedResp)))
	}

	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/Recognize", []byte(`{
		"config":{"languageCode":"en-US","encoding":"LINEAR16","sampleRateHertz":16000},
		"audio":{"content":"c3RhY2t5YXJk"}
	}`), `"results":[`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/LongRunningRecognize", []byte(`{
		"config":{"languageCode":"en-US"},
		"audio":{"content":"c3RhY2t5YXJk"}
	}`), `"done":true`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/StreamingRecognize", []byte(`{
		"streamingConfig":{"config":{"languageCode":"en-US","sampleRateHertz":16000}},
		"audioContent":"c3RyZWFt"
	}`), `"speechEventType":"END_OF_SINGLE_UTTERANCE"`)

	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/CreatePhraseSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"phraseSetId":"phrase-set-1",
		"phraseSet":{"phrases":[{"value":"stackyard"}],"boost":10}
	}`), `phraseSets/phrase-set-1`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/GetPhraseSet", []byte(`{
		"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1"
	}`), `phraseSets/phrase-set-1`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/ListPhraseSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"phraseSets":[`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/UpdatePhraseSet", []byte(`{
		"phraseSet":{"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1","boost":15},
		"updateMask":"boost"
	}`), `"boost":15`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/DeletePhraseSet", []byte(`{
		"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1"
	}`), `{}`)

	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/CreateCustomClass", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"customClassId":"custom-class-1",
		"customClass":{"items":[{"value":"stackyard"}]}
	}`), `customClasses/custom-class-1`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/GetCustomClass", []byte(`{
		"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1"
	}`), `customClasses/custom-class-1`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/ListCustomClasses", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"customClasses":[`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/UpdateCustomClass", []byte(`{
		"customClass":{"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1","items":[{"value":"speech"}]},
		"updateMask":"items"
	}`), `"items":[{"value":"speech"}]`)
	assertGCPSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/DeleteCustomClass", []byte(`{
		"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1"
	}`), `{}`)
}

func TestGCPSpeechRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/Recognize", []byte(`{"config"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_RecognizeRequiresConfig(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/Recognize", []byte(`{
		"audio":{"content":"c3RhY2t5YXJk"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech recognize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_RecognizeRequiresSingleAudioSource(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/Recognize", []byte(`{
		"config":{"languageCode":"en-US"},
		"audio":{"content":"c3RhY2t5YXJk","uri":"gs://stackyard/audio.raw"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech recognize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_StreamingRecognizeRequiresConfigFirst(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/StreamingRecognize", []byte(`{
		"audioContent":"c3RhY2t5YXJk"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech streaming recognize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_CreatePhraseSetRequiresParent(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/CreatePhraseSet", []byte(`{
		"phraseSetId":"phrase-set-1",
		"phraseSet":{"phrases":[{"value":"stackyard"}]}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech create phrase set, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_CreatePhraseSetAlreadyExists(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/CreatePhraseSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"phraseSetId":"existing-phrase-set",
		"phraseSet":{"phrases":[{"value":"stackyard"}]}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp speech create phrase set, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"AlreadyExists"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_GetPhraseSetMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/GetPhraseSet", []byte(`{
		"name":"projects/stackyard/locations/us-central1/phraseSets/missing-phrase-set"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp speech get phrase set, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_UpdatePhraseSetImmutableNameMaskFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/UpdatePhraseSet", []byte(`{
		"phraseSet":{"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1","boost":10},
		"updateMask":"name"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech update phrase set, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_UpdateCustomClassRequiresMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/UpdateCustomClass", []byte(`{
		"customClass":{"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1","items":[{"value":"speech"}]}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech update custom class, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_ListPhraseSetRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/ListPhraseSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageToken":"bad"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech list phrase sets, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)

	recognizeResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/Recognize", []byte(`{
		"config":{"languageCode":"en-US","encoding":"LINEAR16","sampleRateHertz":16000},
		"audio":{"content":"c3RhY2t5YXJk"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if recognizeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech recognize, got %d body=%s", recognizeResp.StatusCode, string(providerContractBody(t, recognizeResp)))
	}
	recognizeBody := providerContractJSONMap(t, recognizeResp)
	results, _ := recognizeBody["results"].([]any)
	if len(results) == 0 {
		t.Fatalf("expected recognize results array, got %#v", recognizeBody["results"])
	}
	firstResult, _ := results[0].(map[string]any)
	alternatives, _ := firstResult["alternatives"].([]any)
	if len(alternatives) == 0 {
		t.Fatalf("expected recognize alternatives array, got %#v", firstResult["alternatives"])
	}
	firstAlternative, _ := alternatives[0].(map[string]any)
	if _, ok := firstAlternative["transcript"].(string); !ok {
		t.Fatalf("expected transcript string, got %#v", firstAlternative["transcript"])
	}
	if _, ok := firstAlternative["confidence"].(float64); !ok {
		t.Fatalf("expected confidence number, got %#v", firstAlternative["confidence"])
	}

	lroResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/LongRunningRecognize", []byte(`{
		"config":{"languageCode":"en-US"},
		"audio":{"content":"c3RhY2t5YXJk"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if lroResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech long running recognize, got %d body=%s", lroResp.StatusCode, string(providerContractBody(t, lroResp)))
	}
	lroBody := providerContractJSONMap(t, lroResp)
	if _, ok := lroBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", lroBody["name"])
	}
	if _, ok := lroBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", lroBody["done"])
	}
	lroResponse, _ := lroBody["response"].(map[string]any)
	lroResults, _ := lroResponse["results"].([]any)
	if len(lroResults) == 0 {
		t.Fatalf("expected long running response results array, got %#v", lroResponse["results"])
	}

	streamingResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Speech/StreamingRecognize", []byte(`{
		"streamingConfig":{"config":{"languageCode":"en-US","sampleRateHertz":16000}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if streamingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech streaming recognize, got %d body=%s", streamingResp.StatusCode, string(providerContractBody(t, streamingResp)))
	}
	streamingBody := providerContractJSONMap(t, streamingResp)
	streamingResults, _ := streamingBody["results"].([]any)
	if len(streamingResults) == 0 {
		t.Fatalf("expected streaming results array, got %#v", streamingBody["results"])
	}
	streamingFirstResult, _ := streamingResults[0].(map[string]any)
	streamingAlternatives, _ := streamingFirstResult["alternatives"].([]any)
	if len(streamingAlternatives) == 0 {
		t.Fatalf("expected streaming alternatives array, got %#v", streamingFirstResult["alternatives"])
	}
	if _, ok := streamingFirstResult["isFinal"].(bool); !ok {
		t.Fatalf("expected streaming isFinal bool, got %#v", streamingFirstResult["isFinal"])
	}
	if _, ok := streamingFirstResult["stability"].(float64); !ok {
		t.Fatalf("expected streaming stability number, got %#v", streamingFirstResult["stability"])
	}
	if _, ok := streamingFirstResult["resultEndTime"].(string); !ok {
		t.Fatalf("expected streaming resultEndTime string, got %#v", streamingFirstResult["resultEndTime"])
	}

	phraseSetResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/ListPhraseSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if phraseSetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech list phrase sets, got %d body=%s", phraseSetResp.StatusCode, string(providerContractBody(t, phraseSetResp)))
	}
	phraseSetBody := providerContractJSONMap(t, phraseSetResp)
	phraseSets, _ := phraseSetBody["phraseSets"].([]any)
	if len(phraseSets) == 0 {
		t.Fatalf("expected phraseSets array, got %#v", phraseSetBody["phraseSets"])
	}
	firstPhraseSet, _ := phraseSets[0].(map[string]any)
	if _, ok := firstPhraseSet["name"].(string); !ok {
		t.Fatalf("expected phrase set name string, got %#v", firstPhraseSet["name"])
	}
	phrases, _ := firstPhraseSet["phrases"].([]any)
	if len(phrases) == 0 {
		t.Fatalf("expected phrase list in phrase set, got %#v", firstPhraseSet["phrases"])
	}
	if token, _ := phraseSetBody["nextPageToken"].(string); token == "" {
		t.Fatalf("expected non-empty nextPageToken for paginated phrase set response, got %#v", phraseSetBody["nextPageToken"])
	}

	customClassResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v1.Adaptation/ListCustomClasses", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech",
	})
	if customClassResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech list custom classes, got %d body=%s", customClassResp.StatusCode, string(providerContractBody(t, customClassResp)))
	}
	customClassBody := providerContractJSONMap(t, customClassResp)
	customClasses, _ := customClassBody["customClasses"].([]any)
	if len(customClasses) == 0 {
		t.Fatalf("expected customClasses array, got %#v", customClassBody["customClasses"])
	}
	firstClass, _ := customClasses[0].(map[string]any)
	items, _ := firstClass["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected custom class items array, got %#v", firstClass["items"])
	}
	firstItem, _ := items[0].(map[string]any)
	if _, ok := firstItem["value"].(string); !ok {
		t.Fatalf("expected custom class item value string, got %#v", firstItem["value"])
	}
	if token, _ := customClassBody["nextPageToken"].(string); token == "" {
		t.Fatalf("expected non-empty nextPageToken for paginated custom class response, got %#v", customClassBody["nextPageToken"])
	}
}

func TestGCPSpeechRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/speech?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "speech" {
		t.Fatalf("expected service=speech, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in probe response, got %#v", body["name"])
	}
}

func newGCPSpeechContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSpeechSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "speech",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
