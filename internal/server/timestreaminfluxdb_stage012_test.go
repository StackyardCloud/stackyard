package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func timestreamInfluxDBRequest(t *testing.T, ts *httptest.Server, action string, payload map[string]any) *http.Response {
	t.Helper()
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = encoded
	}

	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AmazonTimestreamInfluxDB." + action,
		},
		"timestream-influxdb",
	)
}

func decodeBodyJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	data := mustBody(t, resp)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var out map[string]any
	if err := decoder.Decode(&out); err != nil {
		t.Fatalf("decode JSON body: %v body=%s", err, string(data))
	}
	return out
}

func TestTimestreamInfluxDBStage0CatalogCoverage(t *testing.T) {
	if len(timestreamInfluxDBOperations) != 19 {
		t.Fatalf("expected 19 Timestream for InfluxDB operations from docs, got %d", len(timestreamInfluxDBOperations))
	}
	if len(timestreamInfluxDBOperationByName) != len(timestreamInfluxDBOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateDbCluster",
		"GetDbCluster",
		"ListDbClusters",
		"CreateDbInstance",
		"ListDbInstances",
		"TagResource",
		"UntagResource",
	}
	for _, name := range requiredActions {
		if _, ok := timestreamInfluxDBOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(timestreamInfluxDBDataTypes) != 12 {
		t.Fatalf("expected 12 Timestream for InfluxDB data types from docs, got %d", len(timestreamInfluxDBDataTypes))
	}
	if len(timestreamInfluxDBDataTypeByName) != len(timestreamInfluxDBDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"DbClusterSummary",
		"DbInstanceSummary",
		"InfluxDBv2Parameters",
		"InfluxDBv3EnterpriseParameters",
		"LogDeliveryConfiguration",
		"S3Configuration",
	}
	for _, name := range requiredTypes {
		if _, ok := timestreamInfluxDBDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestTimestreamInfluxDBStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := timestreamInfluxDBRequest(t, ts, "TotallyUnknownAction", map[string]any{})
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestTimestreamInfluxDBStage12ClusterLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Stage 1: parameter groups are listable/readable.
	listPGResp := timestreamInfluxDBRequest(t, ts, "ListDbParameterGroups", map[string]any{})
	assertStatus(t, listPGResp, http.StatusOK)
	listPGBody := decodeBodyJSON(t, listPGResp)
	itemsRaw, ok := listPGBody["items"].([]any)
	if !ok || len(itemsRaw) == 0 {
		t.Fatalf("expected parameter group items, got %#v", listPGBody["items"])
	}
	firstPG, ok := itemsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected parameter group item type")
	}
	pgID, ok := firstPG["id"].(string)
	if !ok || strings.TrimSpace(pgID) == "" {
		t.Fatalf("expected parameter group id")
	}

	getPGResp := timestreamInfluxDBRequest(t, ts, "GetDbParameterGroup", map[string]any{
		"identifier": pgID,
	})
	assertStatus(t, getPGResp, http.StatusOK)
	getPGBody := decodeBodyJSON(t, getPGResp)
	if gotID, _ := getPGBody["id"].(string); gotID != pgID {
		t.Fatalf("expected parameter group id %q, got %#v", pgID, getPGBody["id"])
	}

	// Stage 2: cluster lifecycle.
	createResp := timestreamInfluxDBRequest(t, ts, "CreateDbCluster", map[string]any{
		"name":                "stage12-cluster",
		"dbInstanceType":      "db.influx.medium",
		"vpcSubnetIds":        []string{"subnet-12345678", "subnet-87654321"},
		"vpcSecurityGroupIds": []string{"sg-12345678"},
		"tags": map[string]string{
			"env": "test",
		},
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := decodeBodyJSON(t, createResp)
	clusterID, ok := createBody["dbClusterId"].(string)
	if !ok || strings.TrimSpace(clusterID) == "" {
		t.Fatalf("expected dbClusterId in create response, got %#v", createBody)
	}

	getClusterResp := timestreamInfluxDBRequest(t, ts, "GetDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	assertStatus(t, getClusterResp, http.StatusOK)
	getClusterBody := decodeBodyJSON(t, getClusterResp)
	if gotID, _ := getClusterBody["id"].(string); gotID != clusterID {
		t.Fatalf("expected cluster id %q, got %#v", clusterID, getClusterBody["id"])
	}
	clusterARN, ok := getClusterBody["arn"].(string)
	if !ok || strings.TrimSpace(clusterARN) == "" {
		t.Fatalf("expected cluster arn")
	}

	listClusterResp := timestreamInfluxDBRequest(t, ts, "ListDbClusters", map[string]any{})
	assertStatus(t, listClusterResp, http.StatusOK)
	listClusterBody := decodeBodyJSON(t, listClusterResp)
	clusterItems, ok := listClusterBody["items"].([]any)
	if !ok || len(clusterItems) == 0 {
		t.Fatalf("expected cluster items, got %#v", listClusterBody["items"])
	}

	updateResp := timestreamInfluxDBRequest(t, ts, "UpdateDbCluster", map[string]any{
		"dbClusterId":    clusterID,
		"dbInstanceType": "db.influx.large",
	})
	assertStatus(t, updateResp, http.StatusOK)

	rebootResp := timestreamInfluxDBRequest(t, ts, "RebootDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	assertStatus(t, rebootResp, http.StatusOK)

	listInstancesResp := timestreamInfluxDBRequest(t, ts, "ListDbInstancesForCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	assertStatus(t, listInstancesResp, http.StatusOK)
	listInstancesBody := decodeBodyJSON(t, listInstancesResp)
	instanceItems, ok := listInstancesBody["items"].([]any)
	if !ok || len(instanceItems) == 0 {
		t.Fatalf("expected cluster instances, got %#v", listInstancesBody["items"])
	}

	listTagsResp := timestreamInfluxDBRequest(t, ts, "ListTagsForResource", map[string]any{
		"resourceArn": clusterARN,
	})
	assertStatus(t, listTagsResp, http.StatusOK)
	listTagsBody := decodeBodyJSON(t, listTagsResp)
	tagsRaw, ok := listTagsBody["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected tags object, got %#v", listTagsBody["tags"])
	}
	if got, _ := tagsRaw["env"].(string); got != "test" {
		t.Fatalf("expected env=test tag, got %#v", tagsRaw["env"])
	}

	deleteResp := timestreamInfluxDBRequest(t, ts, "DeleteDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	assertStatus(t, deleteResp, http.StatusOK)

	getAfterDeleteResp := timestreamInfluxDBRequest(t, ts, "GetDbCluster", map[string]any{
		"dbClusterId": clusterID,
	})
	assertStatus(t, getAfterDeleteResp, http.StatusNotFound)
	body := string(mustBody(t, getAfterDeleteResp))
	if !strings.Contains(body, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException response body, got %q", body)
	}
}
