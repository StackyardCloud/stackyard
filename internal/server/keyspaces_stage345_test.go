package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKeyspacesStage3RestoreAndAutoScaling(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "CreateTable", []byte(`{
		"keyspaceName":"demo_ks",
		"tableName":"source_tbl",
		"schemaDefinition":{"allColumns":[{"name":"pk","type":"text"}],"partitionKeys":[{"name":"pk"}]},
		"autoScalingSpecification":{"readCapacityAutoScaling":{"minimumUnits":1,"maximumUnits":5}},
		"replicaSpecifications":[{"region":"us-east-1","readCapacityAutoScaling":{"minimumUnits":1,"maximumUnits":3}}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "GetTableAutoScalingSettings", []byte(`{"keyspaceName":"demo_ks","tableName":"source_tbl"}`))
	assertStatus(t, resp, http.StatusOK)
	var autoScalingOut struct {
		KeyspaceName             string `json:"keyspaceName"`
		TableName                string `json:"tableName"`
		ResourceARN              string `json:"resourceArn"`
		AutoScalingSpecification struct {
			ReadCapacityAutoScaling map[string]any `json:"readCapacityAutoScaling"`
		} `json:"autoScalingSpecification"`
		ReplicaSpecifications []struct {
			Region                   string `json:"region"`
			AutoScalingSpecification struct {
				ReadCapacityAutoScaling map[string]any `json:"readCapacityAutoScaling"`
			} `json:"autoScalingSpecification"`
		} `json:"replicaSpecifications"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &autoScalingOut); err != nil {
		t.Fatalf("get table autoscaling unmarshal: %v", err)
	}
	if autoScalingOut.KeyspaceName != "demo_ks" || autoScalingOut.TableName != "source_tbl" {
		t.Fatalf("unexpected autoscaling identity: %+v", autoScalingOut)
	}
	if autoScalingOut.ResourceARN == "" {
		t.Fatalf("expected autoscaling resourceArn")
	}
	if autoScalingOut.AutoScalingSpecification.ReadCapacityAutoScaling == nil {
		t.Fatalf("expected table autoscaling specification")
	}
	if len(autoScalingOut.ReplicaSpecifications) != 1 {
		t.Fatalf("expected one replica autoscaling specification, got %d", len(autoScalingOut.ReplicaSpecifications))
	}

	resp = keyspacesRequest(t, ts, "RestoreTable", []byte(`{
		"sourceKeyspaceName":"demo_ks",
		"sourceTableName":"source_tbl",
		"targetKeyspaceName":"demo_ks",
		"targetTableName":"restored_tbl",
		"tagsOverride":[{"key":"env","value":"stage3"}],
		"autoScalingSpecification":{"readCapacityAutoScaling":{"minimumUnits":2,"maximumUnits":4}}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var restoreOut struct {
		RestoredTableARN string `json:"restoredTableARN"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &restoreOut); err != nil {
		t.Fatalf("restore table unmarshal: %v", err)
	}
	if restoreOut.RestoredTableARN == "" || !strings.Contains(restoreOut.RestoredTableARN, "/table/restored_tbl") {
		t.Fatalf("unexpected restored table arn: %q", restoreOut.RestoredTableARN)
	}

	resp = keyspacesRequest(t, ts, "GetTable", []byte(`{"keyspaceName":"demo_ks","tableName":"restored_tbl"}`))
	assertStatus(t, resp, http.StatusOK)
	var restoredTableOut struct {
		TableName                string `json:"tableName"`
		AutoScalingSpecification struct {
			ReadCapacityAutoScaling map[string]any `json:"readCapacityAutoScaling"`
		} `json:"autoScalingSpecification"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &restoredTableOut); err != nil {
		t.Fatalf("get restored table unmarshal: %v", err)
	}
	if restoredTableOut.TableName != "restored_tbl" {
		t.Fatalf("expected restored_tbl, got %q", restoredTableOut.TableName)
	}
	if restoredTableOut.AutoScalingSpecification.ReadCapacityAutoScaling == nil {
		t.Fatalf("expected restored table autoscaling override")
	}
}

func TestKeyspacesStage4TypeLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "CreateType", []byte(`{
		"keyspaceName":"demo_ks",
		"typeName":"address_type",
		"fieldDefinitions":[{"name":"street","type":"text"},{"name":"zip","type":"int"}]
	}`))
	assertStatus(t, resp, http.StatusOK)
	var createTypeOut struct {
		KeyspaceARN string `json:"keyspaceArn"`
		TypeName    string `json:"typeName"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createTypeOut); err != nil {
		t.Fatalf("create type unmarshal: %v", err)
	}
	if createTypeOut.KeyspaceARN == "" || createTypeOut.TypeName != "address_type" {
		t.Fatalf("unexpected create type output: %+v", createTypeOut)
	}

	resp = keyspacesRequest(t, ts, "GetType", []byte(`{"keyspaceName":"demo_ks","typeName":"address_type"}`))
	assertStatus(t, resp, http.StatusOK)
	var getTypeOut struct {
		KeyspaceName          string `json:"keyspaceName"`
		TypeName              string `json:"typeName"`
		KeyspaceARN           string `json:"keyspaceArn"`
		Status                string `json:"status"`
		MaxNestingDepth       int    `json:"maxNestingDepth"`
		FieldDefinitions      []any  `json:"fieldDefinitions"`
		LastModifiedTimestamp any    `json:"lastModifiedTimestamp"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getTypeOut); err != nil {
		t.Fatalf("get type unmarshal: %v", err)
	}
	if getTypeOut.KeyspaceName != "demo_ks" || getTypeOut.TypeName != "address_type" {
		t.Fatalf("unexpected get type identity: %+v", getTypeOut)
	}
	if getTypeOut.KeyspaceARN == "" || getTypeOut.Status == "" || getTypeOut.MaxNestingDepth != 1 {
		t.Fatalf("unexpected get type metadata: %+v", getTypeOut)
	}
	if len(getTypeOut.FieldDefinitions) != 2 || getTypeOut.LastModifiedTimestamp == nil {
		t.Fatalf("expected field definitions and lastModifiedTimestamp, got %+v", getTypeOut)
	}

	resp = keyspacesRequest(t, ts, "ListTypes", []byte(`{"keyspaceName":"demo_ks","maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listTypesOut struct {
		Types []string `json:"types"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTypesOut); err != nil {
		t.Fatalf("list types unmarshal: %v", err)
	}
	if len(listTypesOut.Types) != 1 || listTypesOut.Types[0] != "address_type" {
		t.Fatalf("expected address_type in list types, got %+v", listTypesOut.Types)
	}

	resp = keyspacesRequest(t, ts, "DeleteType", []byte(`{"keyspaceName":"demo_ks","typeName":"address_type"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestKeyspacesStage5TaggingLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := keyspacesRequest(t, ts, "CreateKeyspace", []byte(`{"keyspaceName":"demo_ks"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "CreateTable", []byte(`{
		"keyspaceName":"demo_ks",
		"tableName":"tagged_tbl",
		"schemaDefinition":{"allColumns":[{"name":"pk","type":"text"}],"partitionKeys":[{"name":"pk"}]}
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

	resp = keyspacesRequest(t, ts, "TagResource", []byte(`{
		"resourceArn":"`+createTableOut.ResourceARN+`",
		"tags":[{"key":"owner","value":"stackyard"},{"key":"env","value":"test"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "ListTagsForResource", []byte(`{
		"resourceArn":"`+createTableOut.ResourceARN+`",
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
	if len(listTagsOut.Tags) != 1 || listTagsOut.NextToken == "" {
		t.Fatalf("expected paginated tags response, got %+v", listTagsOut)
	}

	resp = keyspacesRequest(t, ts, "UntagResource", []byte(`{
		"resourceArn":"`+createTableOut.ResourceARN+`",
		"tags":[{"key":"owner","value":"stackyard"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = keyspacesRequest(t, ts, "ListTagsForResource", []byte(`{
		"resourceArn":"`+createTableOut.ResourceARN+`",
		"maxResults":10
	}`))
	assertStatus(t, resp, http.StatusOK)
	var finalTagsOut struct {
		Tags []struct {
			Key string `json:"key"`
		} `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &finalTagsOut); err != nil {
		t.Fatalf("final list tags unmarshal: %v", err)
	}
	if len(finalTagsOut.Tags) != 1 || finalTagsOut.Tags[0].Key != "env" {
		t.Fatalf("expected only env tag after untag, got %+v", finalTagsOut.Tags)
	}
}
