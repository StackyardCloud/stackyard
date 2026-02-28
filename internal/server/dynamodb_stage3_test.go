package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDynamoDBStage3TransactionsAndPartiQLCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage3-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "PutItem", []byte(`{"TableName":"stage3-table","Item":{"pk":{"S":"id#1"},"status":{"S":"NEW"}}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = dynamodbRequest(t, ts, "PutItem", []byte(`{"TableName":"stage3-table","Item":{"pk":{"S":"id#2"},"status":{"S":"OLD"}}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ExecuteStatement", []byte(`{"Statement":"SELECT * FROM stage3-table","Limit":1}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "BatchExecuteStatement", []byte(`{
		"Statements":[{"Statement":"SELECT * FROM stage3-table"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "ExecuteTransaction", []byte(`{
		"TransactStatements":[{"Statement":"SELECT * FROM stage3-table"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "TransactGetItems", []byte(`{
		"TransactItems":[{"Get":{"TableName":"stage3-table","Key":{"pk":{"S":"id#1"}}}}]
	}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		Responses []map[string]any `json:"Responses"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal transact get response: %v", err)
	}
	if len(getOut.Responses) != 1 {
		t.Fatalf("expected 1 transact get response, got %d", len(getOut.Responses))
	}

	resp = dynamodbRequest(t, ts, "TransactWriteItems", []byte(`{
		"TransactItems":[
			{"ConditionCheck":{"TableName":"stage3-table","Key":{"pk":{"S":"id#1"}}}},
			{"Update":{"TableName":"stage3-table","Key":{"pk":{"S":"id#1"}},"UpdateExpression":"SET #s = :s","ExpressionAttributeNames":{"#s":"status"},"ExpressionAttributeValues":{":s":{"S":"UPDATED"}}}},
			{"Delete":{"TableName":"stage3-table","Key":{"pk":{"S":"id#2"}}}},
			{"Put":{"TableName":"stage3-table","Item":{"pk":{"S":"id#3"},"status":{"S":"NEW"}}}}
		]
	}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestDynamoDBStage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage3-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "PutItem", []byte(`{"TableName":"stage3-implemented","Item":{"pk":{"S":"seed"}}}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "ExecuteStatement", body: []byte(`{"Statement":"SELECT * FROM stage3-implemented"}`)},
		{action: "BatchExecuteStatement", body: []byte(`{"Statements":[{"Statement":"SELECT * FROM stage3-implemented"}]}`)},
		{action: "ExecuteTransaction", body: []byte(`{"TransactStatements":[{"Statement":"SELECT * FROM stage3-implemented"}]}`)},
		{action: "TransactGetItems", body: []byte(`{"TransactItems":[{"Get":{"TableName":"stage3-implemented","Key":{"pk":{"S":"seed"}}}}]}`)},
		{action: "TransactWriteItems", body: []byte(`{"TransactItems":[{"Put":{"TableName":"stage3-implemented","Item":{"pk":{"S":"seed-2"}}}}]}`)},
	}

	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}
