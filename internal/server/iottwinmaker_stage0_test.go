package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func iotTwinMakerRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"iottwinmaker",
	)
}

func TestIoTTwinMakerStage0CatalogCoverage(t *testing.T) {
	if len(iotTwinMakerOperations) != 40 {
		t.Fatalf("expected 40 IoT TwinMaker operations from docs, got %d", len(iotTwinMakerOperations))
	}
	if len(iotTwinMakerOperationByName) != len(iotTwinMakerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateWorkspace",
		"GetWorkspace",
		"ListWorkspaces",
		"CreateEntity",
		"UpdateEntity",
		"DeleteEntity",
		"ExecuteQuery",
	}
	for _, action := range requiredActions {
		if _, ok := iotTwinMakerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iotTwinMakerDataTypes) != 72 {
		t.Fatalf("expected 72 IoT TwinMaker data types from docs, got %d", len(iotTwinMakerDataTypes))
	}
	if len(iotTwinMakerDataTypeByName) != len(iotTwinMakerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"WorkspaceSummary",
		"EntitySummary",
		"ComponentResponse",
		"SceneSummary",
		"MetadataTransferJobSummary",
		"PropertyValue",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotTwinMakerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIoTTwinMakerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotTwinMakerRequest(t, ts, http.MethodPost, "/unknown-iottwinmaker-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTTwinMakerKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotTwinMakerRequest(t, ts, http.MethodPost, "/workspaces-list", `{"maxResults":1}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "workspaceSummaries") {
		t.Fatalf("expected ListWorkspaces response body to include workspaceSummaries, got %q", body)
	}
}
