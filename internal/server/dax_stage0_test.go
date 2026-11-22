package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newDAXTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func daxRequest(t *testing.T, ts *httptest.Server, op, body string) (int, []byte) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": daxTargetPrefix + op,
	}, "dax")
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, raw
}

func daxOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, raw := daxRequest(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %s", op, body, status, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: decode %s: %v", op, raw, err)
	}
	return out
}

func daxFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, raw := daxRequest(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s %s: status %d (want %d) %s", op, body, status, http.StatusBadRequest, raw)
	}
	var e struct {
		Type string `json:"__type"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("%s: decode error %s: %v", op, raw, err)
	}
	return e.Type
}

func TestDAXStage0CatalogMatchesHandlers(t *testing.T) {
	if len(daxOperations) != 21 || len(daxOperationByName) != 21 {
		t.Fatalf("expected 21 DAX operations, got %d/%d", len(daxOperations), len(daxOperationByName))
	}
	for _, op := range daxOperations {
		if daxHandlers[op.Name] == nil {
			t.Fatalf("operation %s has no handler", op.Name)
		}
	}
	if len(daxHandlers) != len(daxOperations) {
		t.Fatalf("handlers (%d) and operations (%d) differ", len(daxHandlers), len(daxOperations))
	}
}

func TestDAXStage0RoutingAndProtocol(t *testing.T) {
	_, ts := newDAXTestServer(t)
	if code := daxFail(t, ts, "Nope", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	if code := daxFail(t, ts, "DescribeClusters", `{"Bogus":1}`); code != "SerializationException" {
		t.Fatalf("unknown member: %s", code)
	}
	if code := daxFail(t, ts, "DescribeClusters", `{"MaxResults":"x"}`); code != "SerializationException" {
		t.Fatalf("bad type: %s", code)
	}
	// Signed for another service: refused.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": daxTargetPrefix + "DescribeClusters",
	}, "dynamodb")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	// Chunked bodies of unknown length are refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", daxTargetPrefix+"DescribeClusters")
	cresp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	cresp.Body.Close()
	if cresp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", cresp.StatusCode)
	}
	// Plain DynamoDB traffic still reaches DynamoDB.
	dresp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "DynamoDB_20120810.ListTables",
	}, "dynamodb")
	body := mustBody(t, dresp)
	if dresp.StatusCode != http.StatusOK || !strings.Contains(string(body), "TableNames") {
		t.Fatalf("dynamodb ListTables: %d %s", dresp.StatusCode, body)
	}
}
