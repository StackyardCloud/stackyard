package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeyspacesStage12KeyspaceLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)
	var createOut struct {
		ResourceARN string `json:"resourceArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("create keyspace unmarshal: %v", err)
	}
	if createOut.ResourceARN == "" {
		t.Fatalf("expected resourceArn")
	}

	resp = keyspacesRequest(t, ts, "GetKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		KeyspaceName        string   `json:"keyspaceName"`
		ReplicationStrategy string   `json:"replicationStrategy"`
		ReplicationRegions  []string `json:"replicationRegions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("get keyspace unmarshal: %v", err)
	}
	if getOut.KeyspaceName != "demo_ks" {
		t.Fatalf("expected keyspaceName demo_ks, got %q", getOut.KeyspaceName)
	}
	if getOut.ReplicationStrategy == "" {
		t.Fatalf("expected replicationStrategy")
	}

	resp = keyspacesRequest(t, ts, "ListKeyspaces", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		Keyspaces []struct {
			KeyspaceName string `json:"keyspaceName"`
		} `json:"keyspaces"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("list keyspaces unmarshal: %v", err)
	}
	if len(listOut.Keyspaces) == 0 {
		t.Fatalf("expected keyspaces in list")
	}

	resp = keyspacesRequest(t, ts, "UpdateKeyspace", []byte(`{
		"keyspaceName":"demo_ks",
		"replicationSpecification":{"replicationStrategy":"MULTI_REGION","regionList":["us-east-1","us-west-2"]}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "GetKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)
	var updatedOut struct {
		ReplicationStrategy string   `json:"replicationStrategy"`
		ReplicationRegions  []string `json:"replicationRegions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updatedOut); err != nil {
		t.Fatalf("get updated keyspace unmarshal: %v", err)
	}
	if updatedOut.ReplicationStrategy != "MULTI_REGION" {
		t.Fatalf("expected MULTI_REGION replication strategy, got %q", updatedOut.ReplicationStrategy)
	}
	if len(updatedOut.ReplicationRegions) < 2 {
		t.Fatalf("expected at least 2 replication regions, got %v", updatedOut.ReplicationRegions)
	}

	resp = keyspacesRequest(t, ts, "DeleteKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestKeyspacesStage12TableLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "CreateTable", []byte(`{
		"keyspaceName":"demo_ks",
		"tableName":"demo_tbl",
		"schemaDefinition":{
			"allColumns":[{"name":"pk","type":"text"}],
			"partitionKeys":[{"name":"pk"}]
		}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var createTableOut struct {
		ResourceARN string `json:"resourceArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createTableOut); err != nil {
		t.Fatalf("create table unmarshal: %v", err)
	}
	if createTableOut.ResourceARN == "" {
		t.Fatalf("expected table resourceArn")
	}

	resp = keyspacesRequest(t, ts, "GetTable", []byte(`{"keyspaceName":"demo_ks","tableName":"demo_tbl"}`))
	assertStatus(t, resp, http.StatusOK)
	var getTableOut struct {
		KeyspaceName string `json:"keyspaceName"`
		TableName    string `json:"tableName"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getTableOut); err != nil {
		t.Fatalf("get table unmarshal: %v", err)
	}
	if getTableOut.KeyspaceName != "demo_ks" || getTableOut.TableName != "demo_tbl" {
		t.Fatalf("unexpected table identity: %+v", getTableOut)
	}
	if getTableOut.Status == "" {
		t.Fatalf("expected table status")
	}

	resp = keyspacesRequest(t, ts, "ListTables", []byte(`{"keyspaceName":"demo_ks","maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listTablesOut struct {
		Tables []struct {
			TableName string `json:"tableName"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTablesOut); err != nil {
		t.Fatalf("list tables unmarshal: %v", err)
	}
	if len(listTablesOut.Tables) != 1 || listTablesOut.Tables[0].TableName != "demo_tbl" {
		t.Fatalf("expected demo_tbl in list tables, got %+v", listTablesOut.Tables)
	}

	resp = keyspacesRequest(t, ts, "UpdateTable", []byte(`{
		"keyspaceName":"demo_ks",
		"tableName":"demo_tbl",
		"addColumns":[{"name":"sk","type":"text"}],
		"defaultTimeToLive":86400
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "GetTable", []byte(`{"keyspaceName":"demo_ks","tableName":"demo_tbl"}`))
	assertStatus(t, resp, http.StatusOK)
	var updatedTableOut struct {
		DefaultTimeToLive int `json:"defaultTimeToLive"`
		SchemaDefinition  struct {
			AllColumns []struct {
				Name string `json:"name"`
			} `json:"allColumns"`
		} `json:"schemaDefinition"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updatedTableOut); err != nil {
		t.Fatalf("updated get table unmarshal: %v", err)
	}
	if updatedTableOut.DefaultTimeToLive != 86400 {
		t.Fatalf("expected defaultTimeToLive 86400, got %d", updatedTableOut.DefaultTimeToLive)
	}
	if len(updatedTableOut.SchemaDefinition.AllColumns) < 2 {
		t.Fatalf("expected added column, got %+v", updatedTableOut.SchemaDefinition.AllColumns)
	}

	resp = keyspacesRequest(t, ts, "DeleteTable", []byte(`{"keyspaceName":"demo_ks","tableName":"demo_tbl"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "ListTables", []byte(`{"keyspaceName":"demo_ks","maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	var finalListTablesOut struct {
		Tables []struct {
			TableName string `json:"tableName"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &finalListTablesOut); err != nil {
		t.Fatalf("final list tables unmarshal: %v", err)
	}
	if len(finalListTablesOut.Tables) != 0 {
		t.Fatalf("expected no tables after delete, got %+v", finalListTablesOut.Tables)
	}
}
