package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNeptuneAnalyticsStage0CatalogCoverage(t *testing.T) {
	if len(neptuneAnalyticsOperations) != 34 {
		t.Fatalf("expected 34 Neptune Analytics API operations from docs, got %d", len(neptuneAnalyticsOperations))
	}
	if len(neptuneAnalyticsOperationByName) != len(neptuneAnalyticsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredOps := []string{
		"CreateGraph",
		"ListGraphs",
		"ExecuteQuery",
		"GetGraphSummary",
		"StartImportTask",
		"TagResource",
		"StartGraph",
		"StopGraph",
	}
	for _, name := range requiredOps {
		if _, ok := neptuneAnalyticsOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(neptuneAnalyticsDataTypes) != 17 {
		t.Fatalf("expected 17 Neptune Analytics API data types from docs, got %d", len(neptuneAnalyticsDataTypes))
	}
	if len(neptuneAnalyticsDataTypeByName) != len(neptuneAnalyticsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"GraphSummary",
		"GraphDataSummary",
		"ImportTaskDetails",
		"ExportTaskDetails",
		"PrivateGraphEndpointSummary",
		"VectorSearchConfiguration",
	}
	for _, name := range requiredTypes {
		if _, ok := neptuneAnalyticsDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestNeptuneAnalyticsStage0KnownRouteReturnsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/__stackyard_unimplemented_neptuneanalytics_route__",
		nil,
		map[string]string{"Content-Type": "application/json"},
		"neptune-graph",
	)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected NotImplementedException response body, got %q", body)
	}
}
