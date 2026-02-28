package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNeptuneDataStage0OperationCoverage(t *testing.T) {
	if len(neptuneDataOperations) != 43 {
		t.Fatalf("expected 43 Neptune Data API operations from docs, got %d", len(neptuneDataOperations))
	}
	if len(neptuneDataOperationByName) != len(neptuneDataOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"GetEngineStatus",
		"ExecuteGremlinQuery",
		"ExecuteOpenCypherQuery",
		"StartLoaderJob",
		"GetPropertygraphSummary",
		"GetRDFGraphSummary",
	}
	for _, name := range required {
		if _, ok := neptuneDataOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestNeptuneDataStage0UnimplementedActionReturnsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodGet,
		ts.URL+"/neptunedata-not-implemented",
		nil,
		map[string]string{"Content-Type": "application/json", "Accept": "application/json"},
		"neptune-db",
	)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected NotImplementedException response body, got %q", body)
	}
}
