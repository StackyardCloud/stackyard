package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKeyspacesStage6CompatibilityHardening(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{
		"keyspaceName":"stage6_ks",
		"tags":[{"key":"env","value":"stage6"},{"key":"team","value":"platform"}]
	}`))
	assertStatus(t, resp, http.StatusOK)
	var createKeyspaceOut struct {
		ResourceARN string `json:"resourceArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createKeyspaceOut); err != nil {
		t.Fatalf("create keyspace unmarshal: %v", err)
	}
	if createKeyspaceOut.ResourceARN == "" {
		t.Fatalf("expected keyspace resourceArn")
	}

	resp = keyspacesRequest(t, ts, "CreateType", []byte(`{
		"keyspaceName":"stage6_ks",
		"typeName":"address_type",
		"fieldDefinitions":[{"name":"street","type":"text"},{"name":"zip","type":"int"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "CreateType", []byte(`{
		"keyspaceName":"stage6_ks",
		"typeName":"customer_type",
		"fieldDefinitions":[{"name":"address","type":"frozen<address_type>"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "CreateTable", []byte(`{
		"keyspaceName":"stage6_ks",
		"tableName":"customers",
		"schemaDefinition":{
			"allColumns":[{"name":"pk","type":"text"},{"name":"profile","type":"frozen<address_type>"}],
			"partitionKeys":[{"name":"pk"}]
		}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "GetType", []byte(`{"keyspaceName":"stage6_ks","typeName":"address_type"}`))
	assertStatus(t, resp, http.StatusOK)
	var getTypeOut struct {
		DirectReferringTables []string `json:"directReferringTables"`
		DirectParentTypes     []string `json:"directParentTypes"`
		MaxNestingDepth       int      `json:"maxNestingDepth"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getTypeOut); err != nil {
		t.Fatalf("get type unmarshal: %v", err)
	}
	if len(getTypeOut.DirectReferringTables) != 1 || getTypeOut.DirectReferringTables[0] != "customers" {
		t.Fatalf("expected customers to directly refer address_type, got %+v", getTypeOut.DirectReferringTables)
	}
	if len(getTypeOut.DirectParentTypes) != 1 || getTypeOut.DirectParentTypes[0] != "customer_type" {
		t.Fatalf("expected customer_type to be direct parent, got %+v", getTypeOut.DirectParentTypes)
	}
	if getTypeOut.MaxNestingDepth < 1 {
		t.Fatalf("expected non-zero maxNestingDepth, got %d", getTypeOut.MaxNestingDepth)
	}

	resp = keyspacesRequest(t, ts, "DeleteType", []byte(`{"keyspaceName":"stage6_ks","typeName":"address_type"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ConflictException") {
		t.Fatalf("expected conflict when deleting referenced type")
	}

	resp = keyspacesRequest(t, ts, "TagResource", []byte(`{
		"resourceArn":"`+createKeyspaceOut.ResourceARN+`",
		"tags":[]
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for empty tag set")
	}

	resp = keyspacesRequest(t, ts, "UntagResource", []byte(`{
		"resourceArn":"`+createKeyspaceOut.ResourceARN+`",
		"tags":[]
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for empty untag set")
	}

	resp = keyspacesRequest(t, ts, "TagResource", []byte(`{
		"resourceArn":"`+createKeyspaceOut.ResourceARN+`",
		"tags":[{"key":"owner","value":"qa"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "ListTagsForResource", []byte(`{
		"resourceArn":"`+createKeyspaceOut.ResourceARN+`",
		"maxResults":1
	}`))
	assertStatus(t, resp, http.StatusOK)
	var listTagsOut struct {
		NextToken string `json:"nextToken"`
		Tags      []struct {
			Key string `json:"key"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsOut); err != nil {
		t.Fatalf("list tags unmarshal: %v", err)
	}
	if len(listTagsOut.Tags) != 1 || strings.TrimSpace(listTagsOut.NextToken) == "" {
		t.Fatalf("expected paginated tags response, got %+v", listTagsOut)
	}

	resp = keyspacesRequest(t, ts, "ListTagsForResource", []byte(`{
		"resourceArn":"`+createKeyspaceOut.ResourceARN+`",
		"maxResults":1,
		"nextToken":"bad-token"
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid nextToken")
	}

	resp = keyspacesRequest(t, ts, "ListTypes", []byte(`{
		"keyspaceName":"stage6_ks",
		"maxResults":1,
		"nextToken":"bad-token"
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for invalid nextToken in list types")
	}

	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	resp = keyspacesRequest(t, ts, "RestoreTable", []byte(`{
		"sourceKeyspaceName":"stage6_ks",
		"sourceTableName":"customers",
		"targetKeyspaceName":"stage6_ks",
		"targetTableName":"customers_restore",
		"restoreTimestamp":"`+future+`"
	}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected validation exception for future restore timestamp")
	}

	resp = keyspacesRequest(t, ts, "DeleteTable", []byte(`{"keyspaceName":"stage6_ks","tableName":"customers"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "DeleteType", []byte(`{"keyspaceName":"stage6_ks","typeName":"customer_type"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "DeleteType", []byte(`{"keyspaceName":"stage6_ks","typeName":"address_type"}`))
	assertStatus(t, resp, http.StatusOK)
}
