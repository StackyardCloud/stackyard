package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMSKStage12ClusterLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskRequest(
		t,
		ts,
		http.MethodPost,
		"/api/v2/clusters",
		[]byte(`{"clusterName":"stage-msk-cluster","provisioned":{"numberOfBrokerNodes":3,"kafkaVersion":"3.7.x"}}`),
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMSKPayload(t, resp)
	clusterArn := mskPayloadString(createPayload, "clusterArn")
	if clusterArn == "" {
		t.Fatalf("expected CreateClusterV2 to return clusterArn")
	}

	resp = mskRequest(t, ts, http.MethodGet, "/api/v2/clusters/"+url.PathEscape(clusterArn), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = mskRequest(t, ts, http.MethodGet, "/api/v2/clusters?maxResults=20", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeMSKPayload(t, resp)
	items, ok := listPayload["clusterInfoList"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListClustersV2 to return clusterInfoList")
	}
}

func TestMSKStage34ClusterOperationReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskRequest(
		t,
		ts,
		http.MethodPost,
		"/api/v2/clusters",
		[]byte(`{"clusterName":"stage-msk-cluster-ops","provisioned":{"numberOfBrokerNodes":3,"kafkaVersion":"3.7.x"}}`),
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMSKPayload(t, resp)
	clusterArn := mskPayloadString(createPayload, "clusterArn")
	if clusterArn == "" {
		t.Fatalf("expected CreateClusterV2 to return clusterArn")
	}

	resp = mskRequest(t, ts, http.MethodGet, "/api/v2/clusters/"+url.PathEscape(clusterArn)+"/operations?maxResults=20", nil)
	assertStatus(t, resp, http.StatusOK)
	opsPayload := decodeMSKPayload(t, resp)
	opItems, ok := opsPayload["clusterOperationInfoList"].([]any)
	if !ok || len(opItems) == 0 {
		t.Fatalf("expected ListClusterOperationsV2 to return clusterOperationInfoList")
	}

	first, ok := opItems[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first operation item to be an object")
	}
	opArn := mskPayloadString(first, "operationArn")
	if opArn == "" {
		t.Fatalf("expected ListClusterOperationsV2 operation to include operationArn")
	}

	resp = mskRequest(t, ts, http.MethodGet, "/api/v2/operations/"+url.PathEscape(opArn), nil)
	assertStatus(t, resp, http.StatusOK)
	describePayload := decodeMSKPayload(t, resp)
	info, _ := describePayload["clusterOperationInfo"].(map[string]any)
	if mskPayloadString(info, "operationState") == "" {
		t.Fatalf("expected DescribeClusterOperationV2 to include operationState")
	}
}

func TestMSKStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskRequest(t, ts, http.MethodPost, "/api/v2/unknown-msk-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/api/v2/clusters",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"kafka",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	clusterArn := "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-msk-v2-cluster/01234567-89ab-cdef-0123-456789abcdef-7"
	resp = mskRequest(t, ts, http.MethodGet, "/api/v2/clusters/"+url.PathEscape(clusterArn), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskRequest(t, ts, http.MethodGet, "/api/v2/clusters/"+url.PathEscape(clusterArn), nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeMSKPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mskPayloadString(payload map[string]any, key string) string {
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
