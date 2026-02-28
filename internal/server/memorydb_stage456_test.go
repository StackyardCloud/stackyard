package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemoryDBStage4Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "DescribeEngineVersions", []byte(`{"MaxResults":1}`))
	assertStatus(t, resp, http.StatusOK)
	var versionsOut struct {
		EngineVersions []map[string]any `json:"EngineVersions"`
		NextToken      string           `json:"NextToken"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &versionsOut); err != nil {
		t.Fatalf("describe engine versions unmarshal: %v", err)
	}
	if len(versionsOut.EngineVersions) != 1 || strings.TrimSpace(versionsOut.NextToken) == "" {
		t.Fatalf("expected one engine version and next token, got %+v", versionsOut)
	}

	resp = memorydbRequest(t, ts, "ListAllowedNodeTypeUpdates", []byte(`{"ClusterName":"stage4-cluster"}`))
	assertStatus(t, resp, http.StatusOK)
	var allowedOut struct {
		ScaleUpNodeTypes   []string `json:"ScaleUpNodeTypes"`
		ScaleDownNodeTypes []string `json:"ScaleDownNodeTypes"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &allowedOut); err != nil {
		t.Fatalf("list allowed node type updates unmarshal: %v", err)
	}
	if len(allowedOut.ScaleUpNodeTypes) == 0 || len(allowedOut.ScaleDownNodeTypes) == 0 {
		t.Fatalf("expected scale up/down node type options")
	}

	resp = memorydbRequest(t, ts, "DescribeReservedNodesOfferings", []byte(`{"MaxResults":1}`))
	assertStatus(t, resp, http.StatusOK)
	var offeringsOut struct {
		ReservedNodesOfferings []struct {
			ReservedNodesOfferingID string `json:"ReservedNodesOfferingId"`
		} `json:"ReservedNodesOfferings"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &offeringsOut); err != nil {
		t.Fatalf("describe reserved nodes offerings unmarshal: %v", err)
	}
	if len(offeringsOut.ReservedNodesOfferings) != 1 {
		t.Fatalf("expected one offering, got %d", len(offeringsOut.ReservedNodesOfferings))
	}
	offeringID := offeringsOut.ReservedNodesOfferings[0].ReservedNodesOfferingID
	if strings.TrimSpace(offeringID) == "" {
		t.Fatalf("expected non-empty reserved nodes offering id")
	}

	resp = memorydbRequest(t, ts, "PurchaseReservedNodesOffering", []byte(`{
		"ReservedNodesOfferingId":"`+offeringID+`",
		"ReservationId":"stage4-reservation",
		"NodeCount":2
	}`))
	assertStatus(t, resp, http.StatusOK)
	var purchaseOut struct {
		ReservedNode struct {
			ReservationID string `json:"ReservationId"`
		} `json:"ReservedNode"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &purchaseOut); err != nil {
		t.Fatalf("purchase reserved nodes offering unmarshal: %v", err)
	}
	if purchaseOut.ReservedNode.ReservationID != "stage4-reservation" {
		t.Fatalf("expected reservation id stage4-reservation, got %q", purchaseOut.ReservedNode.ReservationID)
	}

	resp = memorydbRequest(t, ts, "DescribeReservedNodes", []byte(`{"ReservationId":"stage4-reservation"}`))
	assertStatus(t, resp, http.StatusOK)
	var reservedOut struct {
		ReservedNodes []struct {
			ReservationID string `json:"ReservationId"`
		} `json:"ReservedNodes"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &reservedOut); err != nil {
		t.Fatalf("describe reserved nodes unmarshal: %v", err)
	}
	if len(reservedOut.ReservedNodes) != 1 || reservedOut.ReservedNodes[0].ReservationID != "stage4-reservation" {
		t.Fatalf("unexpected describe reserved nodes output: %+v", reservedOut.ReservedNodes)
	}
}

func TestMemoryDBStage5Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:memorydb:us-east-1:123456789012:cluster/stage5-cluster"

	resp := memorydbRequest(t, ts, "TagResource", []byte(`{
		"ResourceArn":"`+resourceARN+`",
		"Tags":[{"Key":"env","Value":"test"},{"Key":"team","Value":"platform"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "ListTags", []byte(`{
		"ResourceArn":"`+resourceARN+`",
		"MaxResults":1
	}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		TagList   []struct{ Key string } `json:"TagList"`
		NextToken string                 `json:"NextToken"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("list tags unmarshal: %v", err)
	}
	if len(listOut.TagList) != 1 || strings.TrimSpace(listOut.NextToken) == "" {
		t.Fatalf("expected paginated tag list, got %+v", listOut)
	}

	resp = memorydbRequest(t, ts, "UntagResource", []byte(`{
		"ResourceArn":"`+resourceARN+`",
		"TagKeys":["team"]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "ListTags", []byte(`{
		"ResourceArn":"`+resourceARN+`",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)
	var finalTagsOut struct {
		TagList []struct{ Key string } `json:"TagList"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &finalTagsOut); err != nil {
		t.Fatalf("final list tags unmarshal: %v", err)
	}
	if len(finalTagsOut.TagList) != 1 || finalTagsOut.TagList[0].Key != "env" {
		t.Fatalf("expected only env tag after untag, got %+v", finalTagsOut.TagList)
	}

	resp = memorydbRequest(t, ts, "DescribeEvents", []byte(`{
		"SourceName":"stage5-cluster",
		"SourceType":"cluster",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)
	var eventsOut struct {
		Events []map[string]any `json:"Events"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &eventsOut); err != nil {
		t.Fatalf("describe events unmarshal: %v", err)
	}
	if len(eventsOut.Events) == 0 {
		t.Fatalf("expected event entries")
	}

	resp = memorydbRequest(t, ts, "DescribeServiceUpdates", []byte(`{
		"ClusterNames":["stage5-cluster"],
		"Status":["available"],
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)
	var updatesOut struct {
		ServiceUpdates []map[string]any `json:"ServiceUpdates"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updatesOut); err != nil {
		t.Fatalf("describe service updates unmarshal: %v", err)
	}
	if len(updatesOut.ServiceUpdates) == 0 {
		t.Fatalf("expected service update entries")
	}
}

func TestMemoryDBStage6CompatibilityHardening(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "CreateACL", []byte(`{"ACLName":"stage6-acl"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "CreateCluster", []byte(`{
		"ClusterName":"stage6-cluster",
		"NodeType":"db.r6g.large",
		"ACLName":"stage6-acl"
	}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "CreateParameterGroup", []byte(`{
		"ParameterGroupName":"stage6-params",
		"Family":"memorydb_redis7"
	}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "CreateSnapshot", []byte(`{
		"ClusterName":"stage6-cluster",
		"SnapshotName":"stage6-snapshot"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:memorydb:us-east-1:123456789012:cluster/stage6-cluster"
	resp = memorydbRequest(t, ts, "TagResource", []byte(`{
		"ResourceArn":"`+resourceARN+`",
		"Tags":[{"Key":"env","Value":"stage6"},{"Key":"team","Value":"qa"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeParameters", []byte(`{
		"ParameterGroupName":"stage6-params",
		"NextToken":"stackyard",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeSnapshots", []byte(`{
		"ClusterName":"stage6-cluster",
		"NextToken":"stackyard",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeEngineVersions", []byte(`{
		"NextToken":"stackyard",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "ListTags", []byte(`{
		"ResourceArn":"`+resourceARN+`",
		"NextToken":"stackyard",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeParameters", []byte(`{
		"ParameterGroupName":"stage6-params",
		"NextToken":"bad-token",
		"MaxResults":10
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "InvalidParameterValueException") {
		t.Fatalf("expected InvalidParameterValueException for invalid next token")
	}
}
