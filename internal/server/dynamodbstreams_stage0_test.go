package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dynamodbstreamsRequest(t *testing.T, ts *httptest.Server, action, body string) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.0",
		"X-Amz-Target": dynamodbstreamsTargetPrefix + action,
	}, "dynamodb")
}

func newDynamoDBStreamsTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestDynamoDBStreamsStage0OperationCoverage(t *testing.T) {
	if len(dynamodbstreamsOperations) != 4 || len(dynamodbstreamsOperationByName) != 4 {
		t.Fatalf("expected 4 unique DynamoDB Streams operations, got %d", len(dynamodbstreamsOperations))
	}
	for _, name := range []string{"DescribeStream", "GetRecords", "GetShardIterator", "ListStreams"} {
		if _, ok := dynamodbstreamsOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestDynamoDBStreamsStage0RoutingAndErrors(t *testing.T) {
	ts := newDynamoDBStreamsTestServer(t)

	resp := dynamodbstreamsRequest(t, ts, "Nope", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "UnknownOperationException") {
		t.Fatalf("expected UnknownOperationException, got %s", body)
	}

	resp = dynamodbstreamsRequest(t, ts, "ListStreams", `{not json`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "SerializationException") {
		t.Fatalf("expected SerializationException, got %s", body)
	}

	// Unsigned: SigV4 rejects it before any action runs.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", dynamodbstreamsTargetPrefix+"ListStreams")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if body := string(mustBody(t, resp)); resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Fatalf("expected a 4xx SigV4 rejection for an unsigned request, got %d: %s", resp.StatusCode, body)
	}

	// The DynamoDB router still serves its own targets next to the streams router.
	resp = dynamodbRequest(t, ts, "ListTables", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var tables struct{ TableNames []string }
	if err := json.Unmarshal(mustBody(t, resp), &tables); err != nil {
		t.Fatalf("ListTables: %v", err)
	}

	resp = dynamodbstreamsRequest(t, ts, "ListStreams", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, `"Streams":[]`) {
		t.Fatalf("expected an empty stream list, got %s", body)
	}
}
