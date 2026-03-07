package server

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSpannerAdapterRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	database := "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db"
	session := database + "/sessions/as-1"

	assertGCPSpannerAdapterSuccess(t, ts, http.MethodPost, session+":unknown", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1"}`), "unknown", http.StatusNotImplemented)
	assertGCPSpannerAdapterSuccess(t, ts, http.MethodPost, database+"/sessions:adapter", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1"}`), "/sessions/as-1", http.StatusOK)
	assertGCPSpannerAdapterSuccess(t, ts, http.MethodPost, session+":adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"pgwire","payload":"aGVsbG8=","attachments":{"trace":"t-1"}}`), "stateUpdates", http.StatusOK)
}

func TestGCPSpannerAdapterRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions:adapter", []byte(`{"name"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_CreateSessionRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions:adapter", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_CreateSessionNameMustMatchParent(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions:adapter", []byte(`{"name":"projects/stackyard/instances/other/databases/stackyard-db/sessions/as-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_CreateSessionNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/missing-db/sessions:adapter", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/missing-db/sessions/as-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessageRequiresProtocol(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessageNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-2","protocol":"pgwire"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessageAttachmentsMustBeStringMap(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"pgwire","attachments":{"trace":1}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessagePayloadMustBeBase64(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"pgwire","payload":"***"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessageUnsupportedProtocol(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"unsupported"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessageNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/missing-session:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/missing-session","protocol":"pgwire"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSpannerAdapterRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	database := "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db"
	session := database + "/sessions/as-1"

	createResp := providerContractRequest(t, ts, http.MethodPost, database+"/sessions:adapter", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from create session, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected session name string, got %#v", createBody["name"])
	}

	adaptResp := providerContractRequest(t, ts, http.MethodPost, session+":adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"pgwire","payload":"aGVsbG8=","attachments":{"trace":"t-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if adaptResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from adaptMessage, got %d body=%s", adaptResp.StatusCode, string(providerContractBody(t, adaptResp)))
	}
	adaptBody := providerContractJSONMap(t, adaptResp)
	if payload, _ := adaptBody["payload"].(string); payload == "" {
		t.Fatalf("expected payload string, got %#v", adaptBody["payload"])
	}
	updates, _ := adaptBody["stateUpdates"].(map[string]any)
	if _, ok := updates["protocol"].(string); !ok {
		t.Fatalf("expected stateUpdates.protocol string, got %#v", updates["protocol"])
	}
	if last, ok := adaptBody["last"].(bool); !ok || !last {
		t.Fatalf("expected last=true, got %#v", adaptBody["last"])
	}
}

func TestGCPSpannerAdapterRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/spanner_adapter?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner adapter contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "spanner_adapter" {
		t.Fatalf("expected service=spanner_adapter, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPSpannerAdapterRouter_AdaptMessagePayloadRoundTripBase64(t *testing.T) {
	t.Parallel()

	ts := newGCPSpannerAdapterContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1:adaptMessage", []byte(`{"name":"projects/stackyard/instances/stackyard-instance/databases/stackyard-db/sessions/as-1","protocol":"pgwire","payload":"aGVsbG8="}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "spanner-adapter",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp spanner adapter router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	encoded, _ := body["payload"].(string)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("payload should be base64, decode err=%v payload=%q", err, encoded)
	}
	if string(decoded) != "hello" {
		t.Fatalf("expected decoded payload to equal hello, got %q", string(decoded))
	}
}

func newGCPSpannerAdapterContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSpannerAdapterSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string, expectStatus int) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "spanner-adapter",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != expectStatus {
		t.Fatalf("expected %d from gcp spanner adapter router for %s %s, got %d body=%s", expectStatus, method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
