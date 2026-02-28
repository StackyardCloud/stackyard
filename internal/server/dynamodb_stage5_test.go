package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestDynamoDBStage5GlobalTableAndReplicaAutoScalingCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage5-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "CreateGlobalTable", []byte(`{
		"GlobalTableName":"stage5-global",
		"ReplicationGroup":[{"RegionName":"us-east-1"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeGlobalTable", []byte(`{"GlobalTableName":"stage5-global"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListGlobalTables", []byte(`{"Limit":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		GlobalTables []struct {
			GlobalTableName string `json:"GlobalTableName"`
		} `json:"GlobalTables"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list global tables response: %v", err)
	}
	names := make([]string, 0, len(listOut.GlobalTables))
	for _, item := range listOut.GlobalTables {
		names = append(names, item.GlobalTableName)
	}
	if !slices.Contains(names, "stage5-global") {
		t.Fatalf("expected stage5-global in ListGlobalTables output, got %v", names)
	}

	resp = dynamodbRequest(t, ts, "UpdateGlobalTable", []byte(`{
		"GlobalTableName":"stage5-global",
		"ReplicaUpdates":[{"Create":{"RegionName":"us-west-2"}}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeGlobalTableSettings", []byte(`{"GlobalTableName":"stage5-global"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "UpdateGlobalTableSettings", []byte(`{
		"GlobalTableName":"stage5-global",
		"ReplicaSettingsUpdate":[{"RegionName":"us-east-1"},{"RegionName":"us-west-2"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTableReplicaAutoScaling", []byte(`{"TableName":"stage5-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "UpdateTableReplicaAutoScaling", []byte(`{
		"TableName":"stage5-table",
		"ProvisionedWriteCapacityAutoScalingUpdate":{"MinimumUnits":1,"MaximumUnits":10},
		"ProvisionedReadCapacityAutoScalingUpdate":{"MinimumUnits":1,"MaximumUnits":10}
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestDynamoDBStage5ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage5-implemented-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "CreateGlobalTable", []byte(`{
		"GlobalTableName":"stage5-implemented-global",
		"ReplicationGroup":[{"RegionName":"us-east-1"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "CreateGlobalTable", body: []byte(`{"GlobalTableName":"stage5-implemented-global-2","ReplicationGroup":[{"RegionName":"us-east-1"}]}`)},
		{action: "DescribeGlobalTable", body: []byte(`{"GlobalTableName":"stage5-implemented-global"}`)},
		{action: "ListGlobalTables", body: []byte(`{"Limit":10}`)},
		{action: "UpdateGlobalTable", body: []byte(`{"GlobalTableName":"stage5-implemented-global","ReplicaUpdates":[{"Create":{"RegionName":"us-west-1"}}]}`)},
		{action: "DescribeGlobalTableSettings", body: []byte(`{"GlobalTableName":"stage5-implemented-global"}`)},
		{action: "UpdateGlobalTableSettings", body: []byte(`{"GlobalTableName":"stage5-implemented-global","ReplicaSettingsUpdate":[{"RegionName":"us-east-1"}]}`)},
		{action: "DescribeTableReplicaAutoScaling", body: []byte(`{"TableName":"stage5-implemented-table"}`)},
		{action: "UpdateTableReplicaAutoScaling", body: []byte(`{"TableName":"stage5-implemented-table","ProvisionedReadCapacityAutoScalingUpdate":{"MinimumUnits":1,"MaximumUnits":10}}`)},
	}

	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
