package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func omicsRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"omics",
	)
}

func TestOmicsStage0CatalogCoverage(t *testing.T) {
	if len(omicsOperations) != 96 {
		t.Fatalf("expected 96 Omics operations from docs, got %d", len(omicsOperations))
	}
	if len(omicsOperationByName) != len(omicsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateSequenceStore",
		"ListSequenceStores",
		"GetRun",
		"StartRun",
		"ListWorkflows",
		"TagResource",
		"UploadReadSetPart",
	}
	for _, action := range requiredActions {
		if _, ok := omicsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(omicsDataTypes) != 80 {
		t.Fatalf("expected 80 Omics data types from docs, got %d", len(omicsDataTypes))
	}
	if len(omicsDataTypeByName) != len(omicsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"SequenceStoreDetail",
		"RunListItem",
		"WorkflowListItem",
		"ReferenceStoreDetail",
		"VariantStoreItem",
		"AnnotationStoreItem",
	}
	for _, typeName := range requiredTypes {
		if _, ok := omicsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestOmicsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := omicsRequest(t, ts, http.MethodPost, "/unknown-omics-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestOmicsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := omicsRequest(t, ts, http.MethodPost, "/sequencestores?maxResults=1", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "sequenceStores") {
		t.Fatalf("expected ListSequenceStores response body to include sequenceStores, got %q", body)
	}
}
