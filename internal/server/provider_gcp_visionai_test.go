package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVisionAIRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	baseLocation := "/gcp/v1/projects/stackyard/locations/us-central1"
	operationPath := baseLocation + "/operations/createStream.stream-1"

	assertGCPVisionAISuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPVisionAISuccess(t, ts, http.MethodGet, baseLocation, nil, `"locationId":"us-central1"`)

	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.HealthCheckService/HealthCheck", []byte(`{
		"cluster":"projects/stackyard/locations/us-central1/clusters/cluster-1"
	}`), `"healthy":true`)

	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/ListStreams", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"streams":[`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/GetStream", []byte(`{
		"name":"projects/stackyard/locations/us-central1/streams/stream-1"
	}`), `/streams/stream-1`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/CreateStream", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"streamId":"stream-1",
		"stream":{"displayName":"Stream One"}
	}`), `/operations/createStream.stream-1`)

	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.AppPlatform/ListApplications", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"applications":[`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.LiveVideoAnalytics/ListPublicOperators", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"operators":[`)

	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.Warehouse/CreateCorpus", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"corpus":{"displayName":"Corpus One"}
	}`), `/operations/createCorpus.corpus-1`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.Warehouse/ListCorpora", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), `"corpora":[`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.Warehouse/GetCorpus", []byte(`{
		"name":"projects/stackyard/locations/us-central1/corpora/corpus-1"
	}`), `/corpora/corpus-1`)

	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamingService/AcquireLease", []byte(`{
		"series":"projects/stackyard/locations/us-central1/series/series-1",
		"owner":"owner-1"
	}`), `"series":"projects/stackyard/locations/us-central1/series/series-1"`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamingService/RenewLease", []byte(`{
		"id":"lease-1",
		"series":"projects/stackyard/locations/us-central1/series/series-1",
		"owner":"owner-1"
	}`), `"id":"lease-1"`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamingService/ReleaseLease", []byte(`{
		"id":"lease-1",
		"series":"projects/stackyard/locations/us-central1/series/series-1",
		"owner":"owner-1"
	}`), `{}`)

	assertGCPVisionAISuccess(t, ts, http.MethodGet, baseLocation+"/operations?pageSize=1", nil, `"operations":[`)
	assertGCPVisionAISuccess(t, ts, http.MethodGet, operationPath, nil, `/operations/createStream.stream-1`)
	assertGCPVisionAISuccess(t, ts, http.MethodPost, operationPath+":cancel", []byte(`{}`), `{}`)
	assertGCPVisionAISuccess(t, ts, http.MethodDelete, operationPath, nil, `{}`)
}

func TestGCPVisionAIRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/ListStreams", []byte(`{"parent"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp visionai invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionAIRouter_ListStreamsRequiresParent(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/ListStreams", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp visionai list streams missing parent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionAIRouter_GetStreamMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/GetStream", []byte(`{
		"name":"projects/stackyard/locations/us-central1/streams/missing-stream"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp visionai get stream missing, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionAIRouter_CreateStreamNameMustMatchParentAndID(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/CreateStream", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"streamId":"stream-1",
		"stream":{
			"name":"projects/stackyard/locations/us-central1/streams/stream-2",
			"displayName":"Stream One"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp visionai create stream name mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionAIRouter_HealthCheckRequiresClusterFormat(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.HealthCheckService/HealthCheck", []byte(`{
		"cluster":"projects/stackyard/locations/us-central1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp visionai health check invalid cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVisionAIRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	}

	createStreamResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/CreateStream", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"streamId":"stream-1",
		"stream":{"displayName":"Stream One"}
	}`), headers)
	if createStreamResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp visionai create stream, got %d body=%s", createStreamResp.StatusCode, string(providerContractBody(t, createStreamResp)))
	}
	createStreamBody := providerContractJSONMap(t, createStreamResp)
	if _, ok := createStreamBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createStreamBody["name"])
	}
	if _, ok := createStreamBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createStreamBody["done"])
	}
	metadata, ok := createStreamBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata object, got %#v", createStreamBody["metadata"])
	}
	if _, ok := metadata["@type"].(string); !ok {
		t.Fatalf("expected metadata @type string, got %#v", metadata["@type"])
	}
	response, ok := createStreamBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected response object, got %#v", createStreamBody["response"])
	}
	if _, ok := response["name"].(string); !ok {
		t.Fatalf("expected response.name string, got %#v", response["name"])
	}

	listStreamsResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/ListStreams", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), headers)
	if listStreamsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp visionai list streams, got %d body=%s", listStreamsResp.StatusCode, string(providerContractBody(t, listStreamsResp)))
	}
	listStreamsBody := providerContractJSONMap(t, listStreamsResp)
	streams, ok := listStreamsBody["streams"].([]any)
	if !ok || len(streams) == 0 {
		t.Fatalf("expected non-empty streams array, got %#v", listStreamsBody["streams"])
	}
	firstStream, ok := streams[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first stream object, got %#v", streams[0])
	}
	if _, ok := firstStream["name"].(string); !ok {
		t.Fatalf("expected stream.name string, got %#v", firstStream["name"])
	}
	if _, ok := firstStream["enableHlsPlayback"].(bool); !ok {
		t.Fatalf("expected stream.enableHlsPlayback bool, got %#v", firstStream["enableHlsPlayback"])
	}

	healthResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.HealthCheckService/HealthCheck", []byte(`{
		"cluster":"projects/stackyard/locations/us-central1/clusters/cluster-1"
	}`), headers)
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp visionai healthcheck, got %d body=%s", healthResp.StatusCode, string(providerContractBody(t, healthResp)))
	}
	healthBody := providerContractJSONMap(t, healthResp)
	if _, ok := healthBody["healthy"].(bool); !ok {
		t.Fatalf("expected healthy bool, got %#v", healthBody["healthy"])
	}
	clusterInfo, ok := healthBody["clusterInfo"].(map[string]any)
	if !ok {
		t.Fatalf("expected clusterInfo object, got %#v", healthBody["clusterInfo"])
	}
	if _, ok := clusterInfo["streamsCount"].(float64); !ok {
		t.Fatalf("expected clusterInfo.streamsCount number, got %#v", clusterInfo["streamsCount"])
	}
}

func TestGCPVisionAIRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/visionai?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp visionai contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "visionai" {
		t.Fatalf("expected service=visionai, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in contract probe response, got %#v", body["name"])
	}
	if gotProvider, _ := body["provider"].(string); gotProvider != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
}

func TestGCPVisionAIRouter_ContractProbeAliasSelector(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vision-ai?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vision-ai contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if _, ok := body["service"].(string); !ok {
		t.Fatalf("expected typed service in contract probe response, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in contract probe response, got %#v", body["name"])
	}
}

func TestGCPVisionAIRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVisionAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/visionai?stackyard_contract_probe=1&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "visionai",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp visionai contract probe invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPVisionAIContractServer(t *testing.T) *httptest.Server {
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

func assertGCPVisionAISuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "visionai",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp visionai router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
