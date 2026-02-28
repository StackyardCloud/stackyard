package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMSKConnectStage12ConnectorLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskConnectRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/connectors",
		[]byte(`{
			"connectorName":"stage-mskconnect-connector",
			"kafkaConnectVersion":"2.7.1",
			"serviceExecutionRoleArn":"arn:aws:iam::123456789012:role/stackyard-msk-connect"
		}`),
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMSKConnectPayload(t, resp)
	connectorARN := mskConnectPayloadString(createPayload, "connectorArn")
	if connectorARN == "" {
		t.Fatalf("expected CreateConnector to return connectorArn")
	}

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/connectors/"+url.PathEscape(connectorARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/connectors?connectorNamePrefix=stage-mskconnect&maxResults=20&nextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeMSKConnectPayload(t, resp)
	items, ok := listPayload["connectors"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListConnectors to return connectors")
	}

	resp = mskConnectRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/connectors/"+url.PathEscape(connectorARN)+"?currentVersion=1",
		[]byte(`{"capacity":{"provisionedCapacity":{"workerCount":1,"mcuCount":1}}}`),
	)
	assertStatus(t, resp, http.StatusOK)
	updatePayload := decodeMSKConnectPayload(t, resp)
	if mskConnectPayloadString(updatePayload, "currentVersion") == "" {
		t.Fatalf("expected UpdateConnector to return currentVersion")
	}

	resp = mskConnectRequest(t, ts, http.MethodDelete, "/v1/connectors/"+url.PathEscape(connectorARN)+"?currentVersion=2", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestMSKConnectStage34PluginWorkerAndOperationSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskConnectRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/custom-plugins",
		[]byte(`{"name":"stage-mskconnect-plugin","contentType":"ZIP"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	customPluginARN := mskConnectPayloadString(decodeMSKConnectPayload(t, resp), "customPluginArn")
	if customPluginARN == "" {
		t.Fatalf("expected CreateCustomPlugin to return customPluginArn")
	}

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/custom-plugins/"+url.PathEscape(customPluginARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/custom-plugins?namePrefix=stage-mskconnect&maxResults=20&nextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskConnectRequest(t, ts, http.MethodDelete, "/v1/custom-plugins/"+url.PathEscape(customPluginARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mskConnectRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/worker-configurations",
		[]byte(`{"name":"stage-mskconnect-worker","propertiesFileContent":"offset.flush.interval.ms=1000"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	workerConfigurationARN := mskConnectPayloadString(decodeMSKConnectPayload(t, resp), "workerConfigurationArn")
	if workerConfigurationARN == "" {
		t.Fatalf("expected CreateWorkerConfiguration to return workerConfigurationArn")
	}

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/worker-configurations/"+url.PathEscape(workerConfigurationARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/worker-configurations?namePrefix=stage-mskconnect&maxResults=20&nextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskConnectRequest(t, ts, http.MethodDelete, "/v1/worker-configurations/"+url.PathEscape(workerConfigurationARN), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/connectors/"+url.PathEscape(mskConnectSeedConnectorARN)+"/operations?maxResults=20&nextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	opsPayload := decodeMSKConnectPayload(t, resp)
	opItems, ok := opsPayload["connectorOperations"].([]any)
	if !ok || len(opItems) == 0 {
		t.Fatalf("expected ListConnectorOperations to return connectorOperations")
	}
	first, ok := opItems[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first connector operation item to be an object")
	}
	connectorOperationARN := mskConnectPayloadString(first, "connectorOperationArn")
	if connectorOperationARN == "" {
		t.Fatalf("expected ListConnectorOperations item to include connectorOperationArn")
	}

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/connectorOperations/"+url.PathEscape(connectorOperationARN), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestMSKConnectStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := mskConnectSeedConnectorARN
	escapedResourceARN := url.PathEscape(resourceARN)

	resp := mskConnectRequest(t, ts, http.MethodPost, "/v1/tags/"+escapedResourceARN, []byte(`{"tags":{"owner":"qa","env":"stage"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/tags/"+escapedResourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload := decodeMSKConnectPayload(t, resp)
	tags, ok := tagPayload["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected ListTagsForResource to return tags map")
	}
	if got := mskConnectPayloadString(tags, "owner"); got != "qa" {
		t.Fatalf("expected owner tag qa, got %q", got)
	}

	resp = mskConnectRequest(t, ts, http.MethodDelete, "/v1/tags/"+escapedResourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mskConnectRequest(t, ts, http.MethodGet, "/v1/tags/"+escapedResourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"owner"`) {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = mskConnectRequest(t, ts, http.MethodPost, "/v1/mskconnect-unknown-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/v1/connectors",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"kafkaconnect",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = mskConnectRequest(t, ts, http.MethodDelete, "/v1/custom-plugins/"+url.PathEscape(mskConnectSeedCustomPluginARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskConnectRequest(t, ts, http.MethodDelete, "/v1/custom-plugins/"+url.PathEscape(mskConnectSeedCustomPluginARN), nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeMSKConnectPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mskConnectPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
