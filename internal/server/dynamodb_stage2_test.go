package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDynamoDBStage2ItemAndAccessPathCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage2-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"},{"AttributeName":"sk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"},{"AttributeName":"sk","KeyType":"RANGE"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "PutItem", []byte(`{
		"TableName":"stage2-table",
		"Item":{
			"pk":{"S":"acct#1"},
			"sk":{"S":"order#1"},
			"status":{"S":"PENDING"},
			"amount":{"N":"100"}
		}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "GetItem", []byte(`{
		"TableName":"stage2-table",
		"Key":{"pk":{"S":"acct#1"},"sk":{"S":"order#1"}}
	}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		Item map[string]any `json:"Item"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal get item response: %v", err)
	}
	if len(getOut.Item) == 0 {
		t.Fatalf("expected item in GetItem output")
	}

	resp = dynamodbRequest(t, ts, "UpdateItem", []byte(`{
		"TableName":"stage2-table",
		"Key":{"pk":{"S":"acct#1"},"sk":{"S":"order#1"}},
		"AttributeUpdates":{
			"status":{"Value":{"S":"PAID"},"Action":"PUT"}
		},
		"ReturnValues":"ALL_NEW"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "Query", []byte(`{
		"TableName":"stage2-table",
		"KeyConditionExpression":"pk = :pk",
		"ExpressionAttributeValues":{":pk":{"S":"acct#1"}}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "Scan", []byte(`{"TableName":"stage2-table"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "BatchWriteItem", []byte(`{
		"RequestItems":{
			"stage2-table":[
				{"PutRequest":{"Item":{"pk":{"S":"acct#1"},"sk":{"S":"order#2"},"status":{"S":"PENDING"}}}}
			]
		}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "BatchGetItem", []byte(`{
		"RequestItems":{
			"stage2-table":{
				"Keys":[
					{"pk":{"S":"acct#1"},"sk":{"S":"order#1"}},
					{"pk":{"S":"acct#1"},"sk":{"S":"order#2"}}
				]
			}
		}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DeleteItem", []byte(`{
		"TableName":"stage2-table",
		"Key":{"pk":{"S":"acct#1"},"sk":{"S":"order#2"}}
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestDynamoDBStage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage2-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "PutItem", body: []byte(`{"TableName":"stage2-implemented","Item":{"pk":{"S":"k1"},"status":{"S":"NEW"}}}`)},
		{action: "GetItem", body: []byte(`{"TableName":"stage2-implemented","Key":{"pk":{"S":"k1"}}}`)},
		{action: "UpdateItem", body: []byte(`{"TableName":"stage2-implemented","Key":{"pk":{"S":"k1"}},"AttributeUpdates":{"status":{"Value":{"S":"UPDATED"},"Action":"PUT"}}}`)},
		{action: "DeleteItem", body: []byte(`{"TableName":"stage2-implemented","Key":{"pk":{"S":"k1"}}}`)},
		{action: "BatchWriteItem", body: []byte(`{"RequestItems":{"stage2-implemented":[{"PutRequest":{"Item":{"pk":{"S":"k2"}}}}]}}`)},
		{action: "BatchGetItem", body: []byte(`{"RequestItems":{"stage2-implemented":{"Keys":[{"pk":{"S":"k2"}}]}}}`)},
		{action: "Query", body: []byte(`{"TableName":"stage2-implemented","KeyConditionExpression":"pk = :pk","ExpressionAttributeValues":{":pk":{"S":"k2"}}}`)},
		{action: "Scan", body: []byte(`{"TableName":"stage2-implemented"}`)},
	}

	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
