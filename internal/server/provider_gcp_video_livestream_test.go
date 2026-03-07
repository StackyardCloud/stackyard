package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVideoLivestreamRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	channel := base + "/channels/channel-1"
	input := base + "/inputs/input-1"
	operation := base + "/operations/createChannel.channel-1"

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `locations`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `us-central1`)

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, base+"/channels?channelId=channel-1", []byte(`{"channel":{"name":"projects/stackyard/locations/us-central1/channels/channel-1"}}`), `createChannel.channel-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, base+"/channels?pageSize=1", nil, `channels`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, channel, nil, `channels/channel-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPatch, channel+"?updateMask=output", []byte(`{"channel":{"name":"projects/stackyard/locations/us-central1/channels/channel-1"}}`), `updateChannel.channel-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+":start", []byte(`{}`), `startChannel.channel-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+":stop", []byte(`{}`), `stopChannel.channel-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+"/distributions/distribution-1:start", []byte(`{}`), `startDistribution.channel-1.distribution-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+"/distributions/distribution-1:stop", []byte(`{}`), `stopDistribution.channel-1.distribution-1`)

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, base+"/inputs?inputId=input-1", []byte(`{"input":{"name":"projects/stackyard/locations/us-central1/inputs/input-1"}}`), `createInput.input-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, base+"/inputs?pageSize=1", nil, `inputs`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, input+":preview", []byte(`{}`), `preview.example.com`)

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+"/events?eventId=event-1", []byte(`{"event":{"name":"projects/stackyard/locations/us-central1/channels/channel-1/events/event-1"}}`), `events/event-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, channel+"/events?pageSize=1", nil, `events`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+"/clips?clipId=clip-1", []byte(`{"clip":{"name":"projects/stackyard/locations/us-central1/channels/channel-1/clips/clip-1"}}`), `createClip.clip-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, channel+"/clips?pageSize=1", nil, `clips`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, channel+"/dvrSessions?dvrSessionId=dvr-session-1", []byte(`{"dvrSession":{"name":"projects/stackyard/locations/us-central1/channels/channel-1/dvrSessions/dvr-session-1"}}`), `createDvrSession.dvr-session-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, channel+"/dvrSessions?pageSize=1", nil, `dvrSessions`)

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, base+"/assets?assetId=asset-1", []byte(`{"asset":{"name":"projects/stackyard/locations/us-central1/assets/asset-1"}}`), `createAsset.asset-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, base+"/assets?pageSize=1", nil, `assets`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, base+"/pools/default", nil, `pools/default`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPatch, base+"/pools/default?updateMask=networkConfig", []byte(`{"pool":{"name":"projects/stackyard/locations/us-central1/pools/default"}}`), `updatePool.default`)

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, `operations`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodGet, operation, nil, `createChannel.channel-1`)
	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), `{}`)

	assertGCPVideoLivestreamSuccess(t, ts, http.MethodPost, gcpVideoLivestreamGRPCPathPrefix+"CreateChannel", []byte(`{"parent":"projects/stackyard/locations/us-central1","channelId":"channel-2","channel":{"name":"projects/stackyard/locations/us-central1/channels/channel-2"}}`), `createChannel.channel-2`)
}

func TestGCPVideoLivestreamRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/channels?channelId=channel-1", []byte(`{"channel"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video livestream invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoLivestreamRouter_CreateChannelRequiresChannelID(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/channels", []byte(`{"channel":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video livestream create channel missing id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoLivestreamRouter_UpdateInputRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/inputs/input-1", []byte(`{"input":{"name":"projects/stackyard/locations/us-central1/inputs/input-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video livestream update input missing updateMask, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoLivestreamRouter_GetChannelMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/channels/missing-channel", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video livestream get missing channel, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoLivestreamRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)

	listChannelsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/channels?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if listChannelsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video livestream list channels, got %d body=%s", listChannelsResp.StatusCode, string(providerContractBody(t, listChannelsResp)))
	}
	listBody := providerContractJSONMap(t, listChannelsResp)
	channels, ok := listBody["channels"].([]any)
	if !ok || len(channels) == 0 {
		t.Fatalf("expected channels array, got %#v", listBody["channels"])
	}
	firstChannel, _ := channels[0].(map[string]any)
	if _, ok := firstChannel["name"].(string); !ok {
		t.Fatalf("expected channel name string, got %#v", firstChannel["name"])
	}

	createChannelResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/channels?channelId=channel-1", []byte(`{"channel":{"name":"projects/stackyard/locations/us-central1/channels/channel-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if createChannelResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video livestream create channel, got %d body=%s", createChannelResp.StatusCode, string(providerContractBody(t, createChannelResp)))
	}
	createBody := providerContractJSONMap(t, createChannelResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}

	previewResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/inputs/input-1:preview", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if previewResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video livestream preview input, got %d body=%s", previewResp.StatusCode, string(providerContractBody(t, previewResp)))
	}
	previewBody := providerContractJSONMap(t, previewResp)
	if _, ok := previewBody["uri"].(string); !ok {
		t.Fatalf("expected preview uri string, got %#v", previewBody["uri"])
	}
}

func TestGCPVideoLivestreamRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/video_livestream?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video livestream contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "video_livestream" {
		t.Fatalf("expected service=video_livestream, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPVideoLivestreamRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoLivestreamContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/video_livestream?stackyard_contract_probe=1&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video livestream contract probe invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func assertGCPVideoLivestreamSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, contains string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "video-livestream",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video livestream %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if contains != "" {
		body := string(providerContractBody(t, resp))
		if !strings.Contains(body, contains) {
			t.Fatalf("expected body to contain %q, got %s", contains, body)
		}
	}
}

func newGCPVideoLivestreamContractServer(t *testing.T) *httptest.Server {
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
