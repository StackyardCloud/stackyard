package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVideoStitcherRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	cdnKey := base + "/cdnKeys/cdn-key-1"
	slate := base + "/slates/slate-1"
	liveConfig := base + "/liveConfigs/live-config-1"
	vodConfig := base + "/vodConfigs/vod-config-1"
	vodSession := base + "/vodSessions/vod-session-1"
	liveSession := base + "/liveSessions/live-session-1"
	operation := base + "/operations/createCdnKey.cdn-key-1"

	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `locations`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `us-central1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, base+"/cdnKeys?cdnKeyId=cdn-key-1", []byte(`{"cdnKey":{"name":"projects/stackyard/locations/us-central1/cdnKeys/cdn-key-1"}}`), `createCdnKey.cdn-key-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, base+"/cdnKeys?pageSize=1", nil, `cdnKeys`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, cdnKey, nil, `cdnKeys/cdn-key-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodPatch, cdnKey+"?updateMask=hostname", []byte(`{"cdnKey":{"name":"projects/stackyard/locations/us-central1/cdnKeys/cdn-key-1"}}`), `updateCdnKey.cdn-key-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodDelete, cdnKey, nil, `deleteCdnKey.cdn-key-1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, base+"/vodSessions", []byte(`{"vodSession":{"name":"projects/stackyard/locations/us-central1/vodSessions/vod-session-1","vodConfig":"projects/stackyard/locations/us-central1/vodConfigs/vod-config-1"}}`), `vodSessions/vod-session-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, vodSession, nil, `vodSessions/vod-session-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, vodSession+"/vodStitchDetails?pageSize=1", nil, `vodStitchDetails`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, vodSession+"/vodStitchDetails/stitch-1", nil, `vodStitchDetails/stitch-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, vodSession+"/vodAdTagDetails?pageSize=1", nil, `vodAdTagDetails`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, vodSession+"/vodAdTagDetails/adtag-1", nil, `vodAdTagDetails/adtag-1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, base+"/slates?slateId=slate-1", []byte(`{"slate":{"name":"projects/stackyard/locations/us-central1/slates/slate-1","uri":"https://cdn.example.com/slates/slate-1.mp4"}}`), `createSlate.slate-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, base+"/slates?pageSize=1", nil, `slates`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, slate, nil, `slates/slate-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodPatch, slate+"?updateMask=uri", []byte(`{"slate":{"name":"projects/stackyard/locations/us-central1/slates/slate-1","uri":"https://cdn.example.com/slates/slate-1.mp4"}}`), `updateSlate.slate-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodDelete, slate, nil, `deleteSlate.slate-1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, base+"/liveSessions", []byte(`{"liveSession":{"name":"projects/stackyard/locations/us-central1/liveSessions/live-session-1","liveConfig":"projects/stackyard/locations/us-central1/liveConfigs/live-config-1"}}`), `liveSessions/live-session-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, liveSession, nil, `liveSessions/live-session-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, liveSession+"/liveAdTagDetails?pageSize=1", nil, `liveAdTagDetails`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, liveSession+"/liveAdTagDetails/adtag-1", nil, `liveAdTagDetails/adtag-1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, base+"/liveConfigs?liveConfigId=live-config-1", []byte(`{"liveConfig":{"name":"projects/stackyard/locations/us-central1/liveConfigs/live-config-1","sourceUri":"https://origin.example.com/live/live-config-1.m3u8"}}`), `createLiveConfig.live-config-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, base+"/liveConfigs?pageSize=1", nil, `liveConfigs`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, liveConfig, nil, `liveConfigs/live-config-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodPatch, liveConfig+"?updateMask=sourceUri", []byte(`{"liveConfig":{"name":"projects/stackyard/locations/us-central1/liveConfigs/live-config-1"}}`), `updateLiveConfig.live-config-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodDelete, liveConfig, nil, `deleteLiveConfig.live-config-1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, base+"/vodConfigs?vodConfigId=vod-config-1", []byte(`{"vodConfig":{"name":"projects/stackyard/locations/us-central1/vodConfigs/vod-config-1","sourceUri":"https://origin.example.com/vod/vod-config-1.m3u8","adTagUri":"https://ads.example.com/vod/vod-config-1"}}`), `createVodConfig.vod-config-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, base+"/vodConfigs?pageSize=1", nil, `vodConfigs`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, vodConfig, nil, `vodConfigs/vod-config-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodPatch, vodConfig+"?updateMask=adTagUri", []byte(`{"vodConfig":{"name":"projects/stackyard/locations/us-central1/vodConfigs/vod-config-1"}}`), `updateVodConfig.vod-config-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodDelete, vodConfig, nil, `deleteVodConfig.vod-config-1`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, `operations`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodGet, operation, nil, `createCdnKey.cdn-key-1`)
	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, operation+":cancel", []byte(`{}`), `{}`)

	assertGCPVideoStitcherSuccess(t, ts, http.MethodPost, gcpVideoStitcherGRPCPathPrefix+"CreateCdnKey", []byte(`{"parent":"projects/stackyard/locations/us-central1","cdnKeyId":"cdn-key-2","cdnKey":{"name":"projects/stackyard/locations/us-central1/cdnKeys/cdn-key-2"}}`), `createCdnKey.cdn-key-2`)
}

func TestGCPVideoStitcherRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys?cdnKeyId=cdn-key-1", []byte(`{"cdnKey"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video stitcher invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoStitcherRouter_CreateCdnKeyRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys", []byte(`{"cdnKey":{"hostname":"example.com"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video stitcher create cdn key missing id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoStitcherRouter_GetCdnKeyMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys/missing-cdn-key", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video stitcher get missing cdn key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoStitcherRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video stitcher list cdn keys, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	cdnKeys, ok := listBody["cdnKeys"].([]any)
	if !ok || len(cdnKeys) == 0 {
		t.Fatalf("expected cdnKeys array, got %#v", listBody["cdnKeys"])
	}
	firstCdnKey, _ := cdnKeys[0].(map[string]any)
	if _, ok := firstCdnKey["name"].(string); !ok {
		t.Fatalf("expected cdn key name string, got %#v", firstCdnKey["name"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys?cdnKeyId=cdn-key-1", []byte(`{"cdnKey":{"name":"projects/stackyard/locations/us-central1/cdnKeys/cdn-key-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video stitcher create cdn key, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}
}

func TestGCPVideoStitcherRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/video_stitcher?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video stitcher contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "video_stitcher" {
		t.Fatalf("expected service=video_stitcher, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPVideoStitcherRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoStitcherContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/video_stitcher?stackyard_contract_probe=1&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video stitcher contract probe invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func assertGCPVideoStitcherSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, contains string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video stitcher %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if contains != "" {
		body := string(providerContractBody(t, resp))
		if !strings.Contains(body, contains) {
			t.Fatalf("expected body to contain %q, got %s", contains, body)
		}
	}
}

func newGCPVideoStitcherContractServer(t *testing.T) *httptest.Server {
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
