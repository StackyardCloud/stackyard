package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func translateRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSShineFrontendService_20170701." + action,
		},
		"translate",
	)
}

func TestTranslateStage0CatalogCoverage(t *testing.T) {
	if len(translateOperations) != 19 {
		t.Fatalf("expected 19 Translate operations from docs, got %d", len(translateOperations))
	}
	if len(translateOperationByName) != len(translateOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"TranslateText",
		"TranslateDocument",
		"ListLanguages",
		"ImportTerminology",
		"CreateParallelData",
		"StartTextTranslationJob",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := translateOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(translateDataTypes) != 19 {
		t.Fatalf("expected 19 Translate data types from docs, got %d", len(translateDataTypes))
	}
	if len(translateDataTypeByName) != len(translateDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Language",
		"Term",
		"Tag",
		"TerminologyProperties",
		"ParallelDataProperties",
		"TextTranslationJobProperties",
		"TranslatedDocument",
	}
	for _, typeName := range requiredTypes {
		if _, ok := translateDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestTranslateStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := translateRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestTranslateStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := translateRequest(t, ts, "ListLanguages", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Languages") {
		t.Fatalf("expected ListLanguages response body to include Languages, got %q", body)
	}
}

func TestTranslateStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range translateOperations {
		resp := translateRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
