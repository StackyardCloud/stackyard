package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func memorydbRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonMemoryDB." + action,
		},
		"memorydb",
	)
}

func TestMemoryDBStage0OperationCoverage(t *testing.T) {
	if len(memorydbOperations) != 43 {
		t.Fatalf("expected 43 MemoryDB operations from docs, got %d", len(memorydbOperations))
	}
	if len(memorydbOperationByName) != len(memorydbOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"CreateCluster",
		"DescribeClusters",
		"UpdateCluster",
		"DeleteCluster",
		"CreateUser",
		"CreateACL",
		"CreateSnapshot",
		"TagResource",
	}
	for _, name := range required {
		if _, ok := memorydbOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestMemoryDBStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMemoryDBStage0KnownActionIsImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "DescribeEngineVersions", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := mustBody(t, resp)
	if !strings.Contains(string(body), "EngineVersions") {
		t.Fatalf("expected engine versions response body, got %q", string(body))
	}
}
