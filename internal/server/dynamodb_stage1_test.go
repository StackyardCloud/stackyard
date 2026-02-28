package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestDynamoDBStage1TableLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tableName := "stage1-table"
	createPayload := []byte(`{
		"TableName":"stage1-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`)
	resp := dynamodbRequest(t, ts, "CreateTable", createPayload)
	assertStatus(t, resp, http.StatusOK)

	var createOut struct {
		TableDescription struct {
			TableName   string `json:"TableName"`
			TableArn    string `json:"TableArn"`
			TableStatus string `json:"TableStatus"`
		} `json:"TableDescription"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create table response: %v", err)
	}
	if createOut.TableDescription.TableName != tableName {
		t.Fatalf("expected table name %q, got %q", tableName, createOut.TableDescription.TableName)
	}
	if createOut.TableDescription.TableArn == "" {
		t.Fatalf("expected table arn")
	}
	if createOut.TableDescription.TableStatus == "" {
		t.Fatalf("expected table status")
	}

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage1-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ListTables", []byte(`{"Limit":10}`))
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		TableNames []string `json:"TableNames"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list tables response: %v", err)
	}
	if !slices.Contains(listOut.TableNames, tableName) {
		t.Fatalf("expected ListTables to include %q, got %v", tableName, listOut.TableNames)
	}

	resp = dynamodbRequest(t, ts, "UpdateTable", []byte(`{"TableName":"stage1-table","BillingMode":"PAY_PER_REQUEST"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeLimits", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DeleteTable", []byte(`{"TableName":"stage1-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage1-table"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException after delete")
	}
}

func TestDynamoDBStage1ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_ = dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage1-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "CreateTable", body: []byte(`{"TableName":"stage1-implemented-2","AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],"BillingMode":"PAY_PER_REQUEST"}`)},
		{action: "DescribeTable", body: []byte(`{"TableName":"stage1-implemented"}`)},
		{action: "ListTables", body: []byte(`{"Limit":10}`)},
		{action: "UpdateTable", body: []byte(`{"TableName":"stage1-implemented","BillingMode":"PAY_PER_REQUEST"}`)},
		{action: "DeleteTable", body: []byte(`{"TableName":"stage1-implemented"}`)},
		{action: "DescribeLimits", body: []byte(`{}`)},
	}
	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
