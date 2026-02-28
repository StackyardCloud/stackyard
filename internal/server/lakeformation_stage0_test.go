package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLakeFormationStage0CatalogCoverage(t *testing.T) {
	if len(lakeFormationOperations) != 61 {
		t.Fatalf("expected 61 Lake Formation actions from docs, got %d", len(lakeFormationOperations))
	}
	if len(lakeFormationOperationByName) != len(lakeFormationOperations) {
		t.Fatalf("expected unique Lake Formation action names")
	}

	requiredActions := []string{
		"RegisterResource",
		"ListResources",
		"CreateLFTag",
		"CreateDataCellsFilter",
		"StartTransaction",
		"GetTemporaryDataLocationCredentials",
		"AssumeDecoratedRoleWithSAML",
	}
	for _, action := range requiredActions {
		if _, ok := lakeFormationOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(lakeFormationDataTypes) != 55 {
		t.Fatalf("expected 55 Lake Formation data types from docs, got %d", len(lakeFormationDataTypes))
	}
	if len(lakeFormationDataTypeByName) != len(lakeFormationDataTypes) {
		t.Fatalf("expected unique Lake Formation data type names")
	}

	requiredTypes := []string{
		"DataLakeSettings",
		"DataCellsFilter",
		"LFTag",
		"TemporaryCredentials",
		"TransactionDescription",
		"WorkUnitRange",
	}
	for _, typeName := range requiredTypes {
		if _, ok := lakeFormationDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func lakeFormationRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/"+action,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"lakeformation",
	)
}

func TestLakeFormationStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lakeFormationRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestLakeFormationStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lakeFormationRequest(t, ts, "ListResources", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ResourceInfoList") {
		t.Fatalf("expected ListResources response to include ResourceInfoList, got %q", body)
	}
}

func TestLakeFormationStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range lakeFormationOperations {
		action := strings.TrimPrefix(op.URI, "/")
		resp := lakeFormationRequest(t, ts, action, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
