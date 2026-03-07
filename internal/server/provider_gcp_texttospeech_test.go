package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTextToSpeechRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)

	assertGCPTextToSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPTextToSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `"locationId":"us-central1"`)

	assertGCPTextToSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/voices?languageCode=en-US", nil, "en-US-Standard-A")
	assertGCPTextToSpeechSuccess(t, ts, http.MethodPost, "/gcp/v1/text:synthesize", []byte(`{
		"input":{"text":"stackyard says hello"},
		"voice":{"languageCode":"en-US","ssmlGender":"FEMALE"},
		"audioConfig":{"audioEncoding":"MP3","sampleRateHertz":24000}
	}`), `"audioContent":"`)
	assertGCPTextToSpeechSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio", []byte(`{
		"input":{"text":"stackyard long audio"},
		"audioConfig":{"audioEncoding":"LINEAR16"},
		"outputGcsUri":"gs://stackyard/output.wav"
	}`), `operations/synthesizeLongAudio.op-1`)

	assertGCPTextToSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, `"operations":[`)
	assertGCPTextToSpeechSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/synthesizeLongAudio.op-1", nil, `"done":true`)

	assertGCPTextToSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.texttospeech.v1.TextToSpeech/ListVoices", []byte(`{"languageCode":"en-US"}`), "en-US-Standard-A")
	assertGCPTextToSpeechSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/ListOperations", []byte(`{"name":"projects/stackyard/locations/us-central1","pageSize":1}`), `"operations":[`)

	notImplementedResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.texttospeech.v1.TextToSpeech/UnknownMethod", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if notImplementedResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp texttospeech unknown route, got %d body=%s", notImplementedResp.StatusCode, string(providerContractBody(t, notImplementedResp)))
	}
}

func TestGCPTextToSpeechRouter_ParseLongAudioPath(t *testing.T) {
	t.Parallel()

	project, location, ok := parseGCPTextToSpeechSynthesizeLongAudioPath("/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio")
	if !ok {
		t.Fatalf("expected long audio path to parse")
	}
	if project != "stackyard" || location != "us-central1" {
		t.Fatalf("unexpected parse result project=%q location=%q", project, location)
	}
}

func TestGCPTextToSpeechRouter_IsPathLongAudio(t *testing.T) {
	t.Parallel()

	path := "/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio"
	if !isGCPTextToSpeechPath(path, true) {
		t.Fatalf("expected texttospeech long audio path to be recognized")
	}
}

func TestGCPTextToSpeechRouter_SynthesizeRequiresInput(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/text:synthesize", []byte(`{
		"audioConfig":{"audioEncoding":"MP3"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech synthesize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_SynthesizeRejectsTextAndSSMLTogether(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/text:synthesize", []byte(`{
		"input":{"text":"hello","ssml":"<speak>hello</speak>"},
		"audioConfig":{"audioEncoding":"MP3"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech synthesize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_SynthesizeRejectsInvalidSampleRate(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/text:synthesize", []byte(`{
		"input":{"text":"hello"},
		"audioConfig":{"audioEncoding":"MP3","sampleRateHertz":0}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech synthesize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_LongAudioRequiresOutputGCSURI(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio", []byte(`{
		"input":{"text":"long audio"},
		"audioConfig":{"audioEncoding":"LINEAR16"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech synthesizeLongAudio, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_LongAudioRejectsNonGCSOutputURI(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio", []byte(`{
		"input":{"text":"long audio"},
		"audioConfig":{"audioEncoding":"LINEAR16"},
		"outputGcsUri":"https://example.com/output.wav"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech synthesizeLongAudio, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_ListOperationsRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech list operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_GetOperationMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/missing-op", nil, map[string]string{
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp texttospeech get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_GRPCListOperationsRequiresName(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/ListOperations", []byte(`{"pageSize":1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp texttospeech grpc list operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTextToSpeechRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPTextToSpeechContractServer(t)

	voicesResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/voices?languageCode=en-US", nil, map[string]string{
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if voicesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp texttospeech list voices, got %d body=%s", voicesResp.StatusCode, string(providerContractBody(t, voicesResp)))
	}
	voicesBody := providerContractJSONMap(t, voicesResp)
	voices, _ := voicesBody["voices"].([]any)
	if len(voices) == 0 {
		t.Fatalf("expected non-empty voices array, got %#v", voicesBody["voices"])
	}
	firstVoice, _ := voices[0].(map[string]any)
	if _, ok := firstVoice["name"].(string); !ok {
		t.Fatalf("expected voice.name string, got %#v", firstVoice["name"])
	}
	if _, ok := firstVoice["ssmlGender"].(string); !ok {
		t.Fatalf("expected voice.ssmlGender string, got %#v", firstVoice["ssmlGender"])
	}
	if _, ok := firstVoice["naturalSampleRateHertz"].(float64); !ok {
		t.Fatalf("expected voice.naturalSampleRateHertz number, got %#v", firstVoice["naturalSampleRateHertz"])
	}

	synthResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/text:synthesize", []byte(`{
		"input":{"text":"hello"},
		"audioConfig":{"audioEncoding":"MP3"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if synthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp texttospeech synthesize, got %d body=%s", synthResp.StatusCode, string(providerContractBody(t, synthResp)))
	}
	synthBody := providerContractJSONMap(t, synthResp)
	if _, ok := synthBody["audioContent"].(string); !ok {
		t.Fatalf("expected audioContent string, got %#v", synthBody["audioContent"])
	}

	longAudioResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:synthesizeLongAudio", []byte(`{
		"input":{"text":"long audio"},
		"audioConfig":{"audioEncoding":"LINEAR16"},
		"outputGcsUri":"gs://stackyard/output.wav"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if longAudioResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp texttospeech synthesizeLongAudio, got %d body=%s", longAudioResp.StatusCode, string(providerContractBody(t, longAudioResp)))
	}
	longAudioBody := providerContractJSONMap(t, longAudioResp)
	if _, ok := longAudioBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", longAudioBody["name"])
	}
	if _, ok := longAudioBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", longAudioBody["done"])
	}
	metadata, _ := longAudioBody["metadata"].(map[string]any)
	if _, ok := metadata["startTime"].(string); !ok {
		t.Fatalf("expected operation metadata.startTime string, got %#v", metadata["startTime"])
	}

	operationsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "texttospeech",
	})
	if operationsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp texttospeech list operations, got %d body=%s", operationsResp.StatusCode, string(providerContractBody(t, operationsResp)))
	}
	operationsBody := providerContractJSONMap(t, operationsResp)
	operations, _ := operationsBody["operations"].([]any)
	if len(operations) == 0 {
		t.Fatalf("expected operations array, got %#v", operationsBody["operations"])
	}
	firstOperation, _ := operations[0].(map[string]any)
	if _, ok := firstOperation["name"].(string); !ok {
		t.Fatalf("expected operation.name string, got %#v", firstOperation["name"])
	}
}

func TestGCPTextToSpeechRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/texttospeech?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp texttospeech contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "texttospeech" {
		t.Fatalf("expected service=texttospeech, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPTextToSpeechContractServer(t *testing.T) *httptest.Server {
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

func assertGCPTextToSpeechSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "texttospeech",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp texttospeech router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
