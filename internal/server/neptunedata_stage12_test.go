package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func neptuneDataRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{
		"Accept": "application/json",
	}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "neptune-db")
}

func neptuneDataCall(t *testing.T, ts *httptest.Server, method, path string, body []byte) (int, string) {
	t.Helper()
	resp := neptuneDataRequest(t, ts, method, path, body)
	return resp.StatusCode, string(mustBody(t, resp))
}

func TestNeptuneDataStage1ReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodGet, "/status", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetEngineStatus 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"status\"") {
		t.Fatalf("expected engine status payload, got: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/gremlin/status", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListGremlinQueries 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"acceptedQueryCount\"") {
		t.Fatalf("expected gremlin list payload, got: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/opencypher/status", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListOpenCypherQueries 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"runningQueryCount\"") {
		t.Fatalf("expected openCypher list payload, got: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/propertygraph/statistics/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetPropertygraphSummary 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"graphSummary\"") {
		t.Fatalf("expected propertygraph summary payload, got: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/rdf/statistics/summary", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetRDFGraphSummary 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"graphSummary\"") {
		t.Fatalf("expected rdf summary payload, got: %s", body)
	}
}

func TestNeptuneDataStage2QueryExecutionAndCancellation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodPost, "/gremlin", []byte(`{"gremlin":"g.V().limit(1)"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteGremlinQuery 200, got %d: %s", status, body)
	}
	gremlinQueryID := jsonTagValue(body, "queryId")
	if gremlinQueryID == "" {
		t.Fatalf("expected gremlin query id in execute response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/gremlin/explain", []byte(`{"gremlin":"g.V().limit(1)"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteGremlinExplainQuery 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/gremlin/profile", []byte(`{"gremlin":"g.V().limit(1)"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteGremlinProfileQuery 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/gremlin/status/"+gremlinQueryID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetGremlinQueryStatus 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, gremlinQueryID) {
		t.Fatalf("expected gremlin query id in status response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/gremlin/status/"+gremlinQueryID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelGremlinQuery 200, got %d: %s", status, body)
	}
	if !strings.Contains(strings.ToLower(body), "cancel") {
		t.Fatalf("expected cancel marker in gremlin cancel response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/opencypher", []byte(`{"query":"MATCH (n) RETURN n LIMIT 1"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteOpenCypherQuery 200, got %d: %s", status, body)
	}
	openCypherQueryID := jsonTagValue(body, "queryId")
	if openCypherQueryID == "" {
		t.Fatalf("expected openCypher query id in execute response: %s", body)
	}

	status, body = neptuneDataCall(
		t,
		ts,
		http.MethodPost,
		"/opencypher/explain",
		[]byte(`{"query":"MATCH (n) RETURN n LIMIT 1","explain":"details"}`),
	)
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteOpenCypherExplainQuery 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/opencypher/status/"+openCypherQueryID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetOpenCypherQueryStatus 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, openCypherQueryID) {
		t.Fatalf("expected openCypher query id in status response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/opencypher/status/"+openCypherQueryID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelOpenCypherQuery 200, got %d: %s", status, body)
	}
	if !strings.Contains(strings.ToLower(body), "cancel") {
		t.Fatalf("expected cancel marker in openCypher cancel response: %s", body)
	}
}

func TestNeptuneDataStage12ImplementedRoutesDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/status"},
		{method: http.MethodGet, path: "/gremlin/status"},
		{method: http.MethodGet, path: "/gremlin/status/stackyard-query"},
		{method: http.MethodDelete, path: "/gremlin/status/stackyard-query"},
		{method: http.MethodPost, path: "/gremlin", body: []byte(`{"gremlin":"g.V().limit(1)"}`)},
		{method: http.MethodPost, path: "/gremlin/explain", body: []byte(`{"gremlin":"g.V().limit(1)"}`)},
		{method: http.MethodPost, path: "/gremlin/profile", body: []byte(`{"gremlin":"g.V().limit(1)"}`)},
		{method: http.MethodGet, path: "/opencypher/status"},
		{method: http.MethodGet, path: "/opencypher/status/stackyard-query"},
		{method: http.MethodDelete, path: "/opencypher/status/stackyard-query"},
		{method: http.MethodPost, path: "/opencypher", body: []byte(`{"query":"MATCH (n) RETURN n LIMIT 1"}`)},
		{method: http.MethodPost, path: "/opencypher/explain", body: []byte(`{"query":"MATCH (n) RETURN n LIMIT 1","explain":"details"}`)},
		{method: http.MethodGet, path: "/propertygraph/statistics/summary"},
		{method: http.MethodGet, path: "/rdf/statistics/summary"},
	}

	for _, tc := range cases {
		status, body := neptuneDataCall(t, ts, tc.method, tc.path, tc.body)
		if status == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s %s unexpectedly returned not implemented: status=%d body=%s", tc.method, tc.path, status, body)
		}
	}
}

func jsonTagValue(payload, key string) string {
	marker := `"` + key + `":"`
	start := strings.Index(payload, marker)
	if start == -1 {
		return ""
	}
	start += len(marker)
	end := strings.Index(payload[start:], `"`)
	if end == -1 {
		return ""
	}
	return payload[start : start+end]
}
