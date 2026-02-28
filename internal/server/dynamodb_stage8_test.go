package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDynamoDBStage8ResourcePolicyCore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage8-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage8-table"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe table response: %v", err)
	}
	tableArn := describeOut.Table.TableArn
	if tableArn == "" {
		t.Fatalf("expected table arn")
	}

	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"dynamodb:*","Resource":"` + tableArn + `","Principal":"*"}]}`
	resp = dynamodbRequest(t, ts, "PutResourcePolicy", []byte(`{"ResourceArn":"`+tableArn+`","Policy":`+strconvQuote(policy)+`}`))
	assertStatus(t, resp, http.StatusOK)
	var putOut struct {
		RevisionID string `json:"RevisionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putOut); err != nil {
		t.Fatalf("unmarshal put resource policy response: %v", err)
	}
	if strings.TrimSpace(putOut.RevisionID) == "" {
		t.Fatalf("expected policy revision id")
	}

	resp = dynamodbRequest(t, ts, "GetResourcePolicy", []byte(`{"ResourceArn":"`+tableArn+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var getOut struct {
		Policy     string `json:"Policy"`
		RevisionID string `json:"RevisionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getOut); err != nil {
		t.Fatalf("unmarshal get resource policy response: %v", err)
	}
	if !strings.Contains(getOut.Policy, `"Statement"`) {
		t.Fatalf("expected policy document in GetResourcePolicy response, got %q", getOut.Policy)
	}
	if strings.TrimSpace(getOut.RevisionID) == "" {
		t.Fatalf("expected revision id in GetResourcePolicy response")
	}

	resp = dynamodbRequest(t, ts, "DeleteResourcePolicy", []byte(`{"ResourceArn":"`+tableArn+`","ExpectedRevisionId":"`+getOut.RevisionID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "GetResourcePolicy", []byte(`{"ResourceArn":"`+tableArn+`"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if !strings.Contains(string(mustBody(t, resp)), "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException after delete")
	}
}

func TestDynamoDBStage8ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"stage8-implemented",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeTable", []byte(`{"TableName":"stage8-implemented"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Table struct {
			TableArn string `json:"TableArn"`
		} `json:"Table"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe response: %v", err)
	}
	tableArn := describeOut.Table.TableArn

	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"dynamodb:*","Resource":"` + tableArn + `","Principal":"*"}]}`
	resp = dynamodbRequest(t, ts, "PutResourcePolicy", []byte(`{"ResourceArn":"`+tableArn+`","Policy":`+strconvQuote(policy)+`}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		action string
		body   []byte
	}{
		{action: "PutResourcePolicy", body: []byte(`{"ResourceArn":"` + tableArn + `","Policy":` + strconvQuote(policy) + `}`)},
		{action: "GetResourcePolicy", body: []byte(`{"ResourceArn":"` + tableArn + `"}`)},
		{action: "DeleteResourcePolicy", body: []byte(`{"ResourceArn":"` + tableArn + `"}`)},
	}
	for _, tc := range actions {
		resp := dynamodbRequest(t, ts, tc.action, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s returned NotImplemented", tc.action)
		}
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
