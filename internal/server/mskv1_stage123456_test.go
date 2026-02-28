package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMSKV1Stage12ReplicatorLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskv1Request(
		t,
		ts,
		http.MethodPost,
		"/replication/v1/replicators",
		[]byte(`{
			"replicatorName":"stage-mskv1-replicator",
			"description":"stage replicator",
			"serviceExecutionRoleArn":"arn:aws:iam::123456789012:role/stackyard-msk-replicator",
			"kafkaClusters":[
				{"amazonMskCluster":{"mskClusterArn":"arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-source/01234567-89ab-cdef-0123-456789abcdef-1"},"vpcConfig":{"subnetIds":["subnet-0123456789abcdef0"],"securityGroupIds":["sg-0123456789abcdef0"]}},
				{"amazonMskCluster":{"mskClusterArn":"arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-target/01234567-89ab-cdef-0123-456789abcdef-2"},"vpcConfig":{"subnetIds":["subnet-0123456789abcdef0"],"securityGroupIds":["sg-0123456789abcdef0"]}}
			],
			"replicationInfoList":[
				{"sourceKafkaClusterArn":"arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-source/01234567-89ab-cdef-0123-456789abcdef-1","targetKafkaClusterArn":"arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-target/01234567-89ab-cdef-0123-456789abcdef-2","targetCompressionType":"NONE","consumerGroupReplication":{"consumerGroupsToReplicate":[".*"]},"topicReplication":{"topicsToReplicate":[".*"]}}
			]
		}`),
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMSKV1Payload(t, resp)
	replicatorArn := mskv1PayloadString(createPayload, "replicatorArn")
	if replicatorArn == "" {
		t.Fatalf("expected CreateReplicator to return replicatorArn")
	}

	resp = mskv1Request(t, ts, http.MethodGet, "/replication/v1/replicators/"+url.PathEscape(replicatorArn), nil)
	assertStatus(t, resp, http.StatusOK)
	describePayload := decodeMSKV1Payload(t, resp)
	if got := mskv1PayloadString(describePayload, "replicatorArn"); got == "" {
		t.Fatalf("expected DescribeReplicator to include replicatorArn")
	}

	resp = mskv1Request(t, ts, http.MethodGet, "/replication/v1/replicators?replicatorNameFilter=stage-mskv1&maxResults=20&nextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeMSKV1Payload(t, resp)
	items, ok := listPayload["replicators"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListReplicators to return replicators")
	}

	resp = mskv1Request(t, ts, http.MethodDelete, "/replication/v1/replicators/"+url.PathEscape(replicatorArn), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestMSKV1Stage34UpdateReplicationInfoSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replicatorArn := mskv1SeedReplicatorARN

	resp := mskv1Request(
		t,
		ts,
		http.MethodPut,
		"/replication/v1/replicators/"+url.PathEscape(replicatorArn)+"/replication-info",
		[]byte(`{
			"currentVersion":"1",
			"sourceKafkaClusterArn":"arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-source/01234567-89ab-cdef-0123-456789abcdef-1",
			"targetKafkaClusterArn":"arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-target/01234567-89ab-cdef-0123-456789abcdef-2",
			"targetCompressionType":"GZIP",
			"consumerGroupReplication":{"consumerGroupsToReplicate":[".*"]},
			"topicReplication":{"topicsToReplicate":[".*"]}
		}`),
	)
	assertStatus(t, resp, http.StatusOK)
	updatePayload := decodeMSKV1Payload(t, resp)
	if got := mskv1PayloadString(updatePayload, "replicatorArn"); got == "" {
		t.Fatalf("expected UpdateReplicationInfo to include replicatorArn")
	}

	resp = mskv1Request(t, ts, http.MethodGet, "/replication/v1/replicators/"+url.PathEscape(replicatorArn), nil)
	assertStatus(t, resp, http.StatusOK)
	describePayload := decodeMSKV1Payload(t, resp)
	replicationInfoList, ok := describePayload["replicationInfoList"].([]any)
	if !ok || len(replicationInfoList) == 0 {
		t.Fatalf("expected DescribeReplicator to include replicationInfoList after update")
	}
}

func TestMSKV1Stage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskv1Request(t, ts, http.MethodPost, "/replication/v1/mskv1-unknown-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/replication/v1/replicators",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"kafka",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = mskv1Request(t, ts, http.MethodDelete, "/replication/v1/replicators/"+url.PathEscape(mskv1SeedReplicatorARN), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = mskv1Request(t, ts, http.MethodDelete, "/replication/v1/replicators/"+url.PathEscape(mskv1SeedReplicatorARN), nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeMSKV1Payload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mskv1PayloadString(payload map[string]any, key string) string {
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
