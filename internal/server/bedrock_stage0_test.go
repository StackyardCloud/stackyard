package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bedrockRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Bedrock_20230420." + action,
		},
		"bedrock",
	)
}

func TestBedrockStage0CatalogCoverage(t *testing.T) {
	if len(bedrockOperations) != 98 {
		t.Fatalf("expected 98 Bedrock operations from docs, got %d", len(bedrockOperations))
	}
	if len(bedrockOperationByName) != len(bedrockOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"ListFoundationModels",
		"GetFoundationModel",
		"CreateGuardrail",
		"CreateModelCustomizationJob",
		"StopModelInvocationJob",
		"UpdateProvisionedModelThroughput",
	}
	for _, action := range requiredActions {
		if _, ok := bedrockOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(bedrockDataTypes) != 227 {
		t.Fatalf("expected 227 Bedrock data types from docs, got %d", len(bedrockDataTypes))
	}
	if len(bedrockDataTypeByName) != len(bedrockDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Tag",
		"VpcConfig",
		"ValidationDetails",
		"GuardrailSummary",
		"ModelCopyJobSummary",
		"ProvisionedModelSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := bedrockDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestBedrockStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestBedrockKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := bedrockRequest(t, ts, "ListFoundationModels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "modelSummaries") {
		t.Fatalf("expected ListFoundationModels response body to include modelSummaries, got %q", body)
	}
}
