package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpeechV2Router_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)

	assertGCPSpeechV2Success(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPSpeechV2Success(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)

	notImplementedResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.speech.v2.Speech/UnknownMethod", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if notImplementedResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp speech v2 unknown route, got %d body=%s", notImplementedResp.StatusCode, string(providerContractBody(t, notImplementedResp)))
	}

	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2CreateRecognizerPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"recognizerId":"recognizer-1",
		"recognizer":{"displayName":"Stackyard Recognizer One"}
	}`), `createRecognizer.recognizer-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2ListRecognizersPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"recognizers":[`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2GetRecognizerPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/recognizers/recognizer-1"
	}`), `recognizers/recognizer-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UpdateRecognizerPath, []byte(`{
		"recognizer":{
			"name":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
			"displayName":"Stackyard Recognizer One Updated",
			"etag":"W/\"speech-v2-recognizer-1\""
		},
		"updateMask":"display_name"
	}`), `updateRecognizer.recognizer-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2DeleteRecognizerPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"etag":"W/\"speech-v2-recognizer-1\""
	}`), `deleteRecognizer.recognizer-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UndeleteRecognizerPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"etag":"W/\"speech-v2-recognizer-1\""
	}`), `undeleteRecognizer.recognizer-1`)

	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2RecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"config":{"languageCodes":["en-US"]},
		"content":"c3RhY2t5YXJk"
	}`), `"results":[`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2StreamingRecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"streamingConfig":{"config":{"languageCodes":["en-US"]}}
	}`), `"speechEventType":"END_OF_SINGLE_UTTERANCE"`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2BatchRecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"files":[{"uri":"gs://stackyard/audio-1.wav"}],
		"recognitionOutputConfig":{"inlineResponseConfig":{}}
	}`), `batchRecognize.stackyard`)

	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2GetConfigPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/config"
	}`), `"kmsKeyName":"projects/stackyard/locations/us-central1/keyRings/stackyard/cryptoKeys/speech-v2"`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UpdateConfigPath, []byte(`{
		"config":{"name":"projects/stackyard/locations/us-central1/config","kmsKeyName":"projects/stackyard/locations/us-central1/keyRings/stackyard/cryptoKeys/custom"},
		"updateMask":"kms_key_name"
	}`), `"kmsKeyName":"projects/stackyard/locations/us-central1/keyRings/stackyard/cryptoKeys/custom"`)

	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2CreateCustomClassPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"customClassId":"custom-class-1",
		"customClass":{"items":[{"value":"stackyard"}]}
	}`), `createCustomClass.custom-class-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2ListCustomClassesPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"customClasses":[`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2GetCustomClassPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1"
	}`), `customClasses/custom-class-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UpdateCustomClassPath, []byte(`{
		"customClass":{
			"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1",
			"displayName":"Updated",
			"etag":"W/\"speech-v2-custom-class-1\"",
			"items":[{"value":"speech"}]
		},
		"updateMask":"items"
	}`), `updateCustomClass.custom-class-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2DeleteCustomClassPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1",
		"etag":"W/\"speech-v2-custom-class-1\""
	}`), `deleteCustomClass.custom-class-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UndeleteCustomClassPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/customClasses/custom-class-1",
		"etag":"W/\"speech-v2-custom-class-1\""
	}`), `undeleteCustomClass.custom-class-1`)

	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2CreatePhraseSetPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"phraseSetId":"phrase-set-1",
		"phraseSet":{"phrases":[{"value":"stackyard"}],"boost":10}
	}`), `createPhraseSet.phrase-set-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2ListPhraseSetsPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"phraseSets":[`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2GetPhraseSetPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1"
	}`), `phraseSets/phrase-set-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UpdatePhraseSetPath, []byte(`{
		"phraseSet":{
			"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1",
			"boost":13.5,
			"etag":"W/\"speech-v2-phrase-set-1\""
		},
		"updateMask":"boost"
	}`), `updatePhraseSet.phrase-set-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2DeletePhraseSetPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1",
		"etag":"W/\"speech-v2-phrase-set-1\""
	}`), `deletePhraseSet.phrase-set-1`)
	assertGCPSpeechV2Success(t, ts, http.MethodPost, gcpSpeechV2UndeletePhraseSetPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/phraseSets/phrase-set-1",
		"etag":"W/\"speech-v2-phrase-set-1\""
	}`), `undeletePhraseSet.phrase-set-1`)
}

func TestGCPSpeechV2Router_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2RecognizePath, []byte(`{"recognizer"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_RecognizeRequiresRecognizer(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2RecognizePath, []byte(`{
		"content":"c3RhY2t5YXJk"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 recognize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_RecognizeRequiresSingleAudioSource(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2RecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"content":"c3RhY2t5YXJk",
		"uri":"gs://stackyard/audio.raw"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 recognize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_CreateRecognizerRequiresParent(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2CreateRecognizerPath, []byte(`{
		"recognizerId":"recognizer-1",
		"recognizer":{"displayName":"missing parent"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 create recognizer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_UpdateRecognizerImmutableNameMaskFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2UpdateRecognizerPath, []byte(`{
		"recognizer":{"name":"projects/stackyard/locations/us-central1/recognizers/recognizer-1"},
		"updateMask":"name"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 update recognizer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_DeleteRecognizerRejectsEtagMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2DeleteRecognizerPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"etag":"bad"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 delete recognizer, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_ListRecognizersRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2ListRecognizersPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageToken":"bad"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 list recognizers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_BatchRecognizeRequiresOutputConfig(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2BatchRecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"files":[{"uri":"gs://stackyard/audio-1.wav"}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp speech v2 batch recognize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpeechV2Router_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)

	recognizeResp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2RecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"config":{"languageCodes":["en-US"]},
		"content":"c3RhY2t5YXJk"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if recognizeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 recognize, got %d body=%s", recognizeResp.StatusCode, string(providerContractBody(t, recognizeResp)))
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

	batchResp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2BatchRecognizePath, []byte(`{
		"recognizer":"projects/stackyard/locations/us-central1/recognizers/recognizer-1",
		"files":[{"uri":"gs://stackyard/audio-1.wav"}],
		"recognitionOutputConfig":{"inlineResponseConfig":{}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 batch recognize, got %d body=%s", batchResp.StatusCode, string(providerContractBody(t, batchResp)))
	}
	batchBody := providerContractJSONMap(t, batchResp)
	if _, ok := batchBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", batchBody["name"])
	}
	if _, ok := batchBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", batchBody["done"])
	}
	batchResponse, _ := batchBody["response"].(map[string]any)
	resultMap, _ := batchResponse["results"].(map[string]any)
	if len(resultMap) == 0 {
		t.Fatalf("expected batch response results map, got %#v", batchResponse["results"])
	}

	listRecognizersResp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2ListRecognizersPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1,
		"showDeleted":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if listRecognizersResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 list recognizers, got %d body=%s", listRecognizersResp.StatusCode, string(providerContractBody(t, listRecognizersResp)))
	}
	listRecognizersBody := providerContractJSONMap(t, listRecognizersResp)
	recognizers, _ := listRecognizersBody["recognizers"].([]any)
	if len(recognizers) == 0 {
		t.Fatalf("expected recognizers array, got %#v", listRecognizersBody["recognizers"])
	}
	firstRecognizer, _ := recognizers[0].(map[string]any)
	if _, ok := firstRecognizer["name"].(string); !ok {
		t.Fatalf("expected recognizer name string, got %#v", firstRecognizer["name"])
	}
	if token, _ := listRecognizersBody["nextPageToken"].(string); token == "" {
		t.Fatalf("expected non-empty nextPageToken for paginated recognizer response, got %#v", listRecognizersBody["nextPageToken"])
	}

	getConfigResp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2GetConfigPath, []byte(`{
		"name":"projects/stackyard/locations/us-central1/config"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if getConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 get config, got %d body=%s", getConfigResp.StatusCode, string(providerContractBody(t, getConfigResp)))
	}
	getConfigBody := providerContractJSONMap(t, getConfigResp)
	if _, ok := getConfigBody["name"].(string); !ok {
		t.Fatalf("expected config name string, got %#v", getConfigBody["name"])
	}
	if _, ok := getConfigBody["kmsKeyName"].(string); !ok {
		t.Fatalf("expected config kmsKeyName string, got %#v", getConfigBody["kmsKeyName"])
	}

	listCustomClassesResp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2ListCustomClassesPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if listCustomClassesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 list custom classes, got %d body=%s", listCustomClassesResp.StatusCode, string(providerContractBody(t, listCustomClassesResp)))
	}
	listCustomClassesBody := providerContractJSONMap(t, listCustomClassesResp)
	customClasses, _ := listCustomClassesBody["customClasses"].([]any)
	if len(customClasses) == 0 {
		t.Fatalf("expected customClasses array, got %#v", listCustomClassesBody["customClasses"])
	}
	firstCustomClass, _ := customClasses[0].(map[string]any)
	items, _ := firstCustomClass["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected custom class items array, got %#v", firstCustomClass["items"])
	}

	listPhraseSetsResp := providerContractRequest(t, ts, http.MethodPost, gcpSpeechV2ListPhraseSetsPath, []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "speech-apiv2",
	})
	if listPhraseSetsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 list phrase sets, got %d body=%s", listPhraseSetsResp.StatusCode, string(providerContractBody(t, listPhraseSetsResp)))
	}
	listPhraseSetsBody := providerContractJSONMap(t, listPhraseSetsResp)
	phraseSets, _ := listPhraseSetsBody["phraseSets"].([]any)
	if len(phraseSets) == 0 {
		t.Fatalf("expected phraseSets array, got %#v", listPhraseSetsBody["phraseSets"])
	}
	firstPhraseSet, _ := phraseSets[0].(map[string]any)
	phrases, _ := firstPhraseSet["phrases"].([]any)
	if len(phrases) == 0 {
		t.Fatalf("expected phrase set phrases array, got %#v", firstPhraseSet["phrases"])
	}
}

func TestGCPSpeechV2Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPSpeechV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/speech_v2?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "speech_v2" {
		t.Fatalf("expected service=speech_v2, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in probe response, got %#v", body["name"])
	}
}

func newGCPSpeechV2ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSpeechV2Success(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "speech-apiv2",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp speech v2 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
