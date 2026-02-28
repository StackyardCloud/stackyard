package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComprehendMedicalStage0CatalogCoverage(t *testing.T) {
	if len(comprehendMedicalOperations) != 25 {
		t.Fatalf("expected 25 Comprehend Medical operations from docs, got %d", len(comprehendMedicalOperations))
	}
	if len(comprehendMedicalOperationByName) != len(comprehendMedicalOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"DetectEntitiesV2",
		"DetectPHI",
		"InferICD10CM",
		"StartEntitiesDetectionV2Job",
		"ListSNOMEDCTInferenceJobs",
		"StopRxNormInferenceJob",
	}
	for _, action := range requiredActions {
		if _, ok := comprehendMedicalOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(comprehendMedicalDataTypes) != 22 {
		t.Fatalf("expected 22 Comprehend Medical data types from docs, got %d", len(comprehendMedicalDataTypes))
	}
	if len(comprehendMedicalDataTypeByName) != len(comprehendMedicalDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Entity",
		"Trait",
		"ComprehendMedicalAsyncJobProperties",
		"ICD10CMEntity",
		"RxNormEntity",
		"SNOMEDCTDetails",
	}
	for _, typeName := range requiredTypes {
		if _, ok := comprehendMedicalDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func comprehendMedicalRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ComprehendMedical_20181030." + action,
		},
		"comprehendmedical",
	)
}

func TestComprehendMedicalStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := comprehendMedicalRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestComprehendMedicalKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := comprehendMedicalRequest(t, ts, "DetectPHI", `{"Text":"Patient Jane Doe has diabetes."}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "Entities") {
		t.Fatalf("expected DetectPHI response body to include Entities, got %q", body)
	}
}
