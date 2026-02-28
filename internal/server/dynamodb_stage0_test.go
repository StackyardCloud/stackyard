package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dynamodbRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "DynamoDB_20120810." + action,
		},
		"dynamodb",
	)
}

func TestDynamoDBStage0OperationCoverage(t *testing.T) {
	if len(dynamodbOperations) != 57 {
		t.Fatalf("expected 57 DynamoDB operations from docs, got %d", len(dynamodbOperations))
	}
	if len(dynamodbOperationByName) != len(dynamodbOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateTable",
		"DescribeTable",
		"ListTables",
		"UpdateTable",
		"DeleteTable",
		"DescribeLimits",
		"PutItem",
		"GetItem",
		"UpdateItem",
		"DeleteItem",
		"BatchWriteItem",
		"BatchGetItem",
		"Query",
		"Scan",
	}
	for _, name := range required {
		if _, ok := dynamodbOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestDynamoDBStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDynamoDBStage0KnownCatalogActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "PutResourcePolicy", []byte(`{"ResourceArn":"arn:aws:dynamodb:us-east-1:123456789012:table/demo","Policy":"{}"}`))
	if resp.StatusCode == http.StatusNotImplemented {
		body := string(mustBody(t, resp))
		t.Fatalf("expected implemented handler for PutResourcePolicy, got NotImplemented: %q", body)
	}
}
