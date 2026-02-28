package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func rekognitionRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "RekognitionService." + action,
		},
		"rekognition",
	)
}

func TestRekognitionStage0CatalogCoverage(t *testing.T) {
	if len(rekognitionOperations) != 75 {
		t.Fatalf("expected 75 Rekognition operations from docs, got %d", len(rekognitionOperations))
	}
	if len(rekognitionOperationByName) != len(rekognitionOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCollection",
		"ListCollections",
		"DetectLabels",
		"CreateProject",
		"StartLabelDetection",
		"GetLabelDetection",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := rekognitionOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(rekognitionDataTypes) != 150 {
		t.Fatalf("expected 150 Rekognition data types from docs, got %d", len(rekognitionDataTypes))
	}
	if len(rekognitionDataTypeByName) != len(rekognitionDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"BoundingBox",
		"FaceDetail",
		"ProjectDescription",
		"StreamProcessor",
		"User",
	}
	for _, typeName := range requiredTypes {
		if _, ok := rekognitionDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestRekognitionStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rekognitionRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRekognitionKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rekognitionRequest(t, ts, "ListCollections", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "CollectionIds") {
		t.Fatalf("expected ListCollections response body to include CollectionIds, got %q", body)
	}
}

func TestRekognitionStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range rekognitionOperations {
		resp := rekognitionRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
