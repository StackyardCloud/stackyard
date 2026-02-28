package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPCSStage12ClusterLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := pcsRequest(t, ts, "CreateCluster", `{"clientToken":"stage-pcs-create-cluster-token-000001","clusterName":"stage-pcs-cluster","size":"MEDIUM"}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodePCSPayload(t, resp)
	cluster := pcsPayloadMap(payload, "cluster")
	clusterID := pcsPayloadString(cluster, "id")
	if clusterID == "" {
		t.Fatalf("expected CreateCluster to return cluster.id")
	}

	resp = pcsRequest(t, ts, "GetCluster", `{"clusterIdentifier":"`+clusterID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = pcsRequest(t, ts, "ListClusters", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-pcs-cluster") {
		t.Fatalf("expected ListClusters to include stage-pcs-cluster, got %q", body)
	}

	resp = pcsRequest(t, ts, "UpdateCluster", `{"clusterIdentifier":"`+clusterID+`","size":"LARGE","slurmConfiguration":{"scaleDownIdleTimeInSeconds":300}}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "LARGE") {
		t.Fatalf("expected UpdateCluster response to include updated size, got %q", body)
	}

	resp = pcsRequest(t, ts, "DeleteCluster", `{"clusterIdentifier":"`+clusterID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestPCSStage3InstanceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterID := "pcs_cluster_000001"

	resp := pcsRequest(t, ts, "CreateComputeNodeGroup", `{"clientToken":"stage-pcs-create-cng-token-000001","clusterIdentifier":"`+clusterID+`","computeNodeGroupName":"stage-cng"}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodePCSPayload(t, resp)
	nodeGroup := pcsPayloadMap(payload, "computeNodeGroup")
	nodeGroupID := pcsPayloadString(nodeGroup, "id")
	if nodeGroupID == "" {
		t.Fatalf("expected CreateComputeNodeGroup to return computeNodeGroup.id")
	}

	resp = pcsRequest(t, ts, "GetComputeNodeGroup", `{"clusterIdentifier":"`+clusterID+`","computeNodeGroupIdentifier":"`+nodeGroupID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = pcsRequest(t, ts, "ListComputeNodeGroups", `{"clusterIdentifier":"`+clusterID+`","maxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-cng") {
		t.Fatalf("expected ListComputeNodeGroups to include stage-cng, got %q", body)
	}

	resp = pcsRequest(t, ts, "UpdateComputeNodeGroup", `{"clusterIdentifier":"`+clusterID+`","computeNodeGroupIdentifier":"`+nodeGroupID+`","purchaseOption":"SPOT"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = pcsRequest(t, ts, "RegisterComputeNodeGroupInstance", `{"clusterIdentifier":"`+clusterID+`","bootstrapId":"i-stage000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "sharedSecret") {
		t.Fatalf("expected RegisterComputeNodeGroupInstance to include sharedSecret, got %q", body)
	}

	resp = pcsRequest(t, ts, "DeleteComputeNodeGroup", `{"clusterIdentifier":"`+clusterID+`","computeNodeGroupIdentifier":"`+nodeGroupID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestPCSStage45ParameterGroupsAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterID := "pcs_cluster_000001"

	resp := pcsRequest(t, ts, "CreateQueue", `{"clientToken":"stage-pcs-create-queue-token-000001","clusterIdentifier":"`+clusterID+`","queueName":"stage-queue","computeNodeGroupConfigurations":[{"computeNodeGroupId":"pcs_cng_000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodePCSPayload(t, resp)
	queue := pcsPayloadMap(payload, "queue")
	queueID := pcsPayloadString(queue, "id")
	queueARN := pcsPayloadString(queue, "arn")
	if queueID == "" || queueARN == "" {
		t.Fatalf("expected CreateQueue to return queue.id and queue.arn")
	}

	resp = pcsRequest(t, ts, "GetQueue", `{"clusterIdentifier":"`+clusterID+`","queueIdentifier":"`+queueID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = pcsRequest(t, ts, "ListQueues", `{"clusterIdentifier":"`+clusterID+`","maxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-queue") {
		t.Fatalf("expected ListQueues to include stage-queue, got %q", body)
	}

	resp = pcsRequest(t, ts, "UpdateQueue", `{"clusterIdentifier":"`+clusterID+`","queueIdentifier":"`+queueID+`","slurmConfiguration":{"slurmCustomSettings":[{"parameterName":"PriorityType","parameterValue":"priority/multifactor"}]}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = pcsRequest(t, ts, "TagResource", `{"resourceArn":"`+queueARN+`","tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)

	resp = pcsRequest(t, ts, "ListTagsForResource", `{"resourceArn":"`+queueARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = pcsRequest(t, ts, "UntagResource", `{"resourceArn":"`+queueARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = pcsRequest(t, ts, "DeleteQueue", `{"clusterIdentifier":"`+clusterID+`","queueIdentifier":"`+queueID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestPCSStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := "stage-pcs-idempotent-token-000001"
	resp := pcsRequest(t, ts, "CreateCluster", `{"clientToken":"`+token+`","clusterName":"stage-idempotent-cluster"}`)
	assertStatus(t, resp, http.StatusOK)
	first := decodePCSPayload(t, resp)
	firstID := pcsPayloadString(pcsPayloadMap(first, "cluster"), "id")
	if firstID == "" {
		t.Fatalf("expected first CreateCluster response to include cluster.id")
	}

	resp = pcsRequest(t, ts, "CreateCluster", `{"clientToken":"`+token+`","clusterName":"different-name-ignored-by-idempotency"}`)
	assertStatus(t, resp, http.StatusOK)
	second := decodePCSPayload(t, resp)
	secondID := pcsPayloadString(pcsPayloadMap(second, "cluster"), "id")
	if firstID != secondID {
		t.Fatalf("expected idempotent CreateCluster to return same id: %s != %s", firstID, secondID)
	}

	resp = pcsRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AWSParallelComputingService.ListClusters",
		},
		"pcs",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodePCSPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func pcsPayloadMap(payload map[string]any, key string) map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func pcsPayloadString(payload map[string]any, key string) string {
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
