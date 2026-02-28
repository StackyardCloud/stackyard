package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func groundStationRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "groundstation")
}

func groundStationPathForOperation(op groundStationOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{contactId}":               "contact-00000001",
		"{configType}":              "antenna-downlink",
		"{configId}":                "cfg-00000001",
		"{dataflowEndpointGroupId}": "deg-00000001",
		"{ephemerisId}":             "eph-00000001",
		"{missionProfileId}":        "mp-00000001",
		"{agentId}":                 "agent-00000001",
		"{satelliteId}":             "25544",
		"{resourceArn}":             url.PathEscape("arn:aws:groundstation:us-east-1:123456789012:mission-profile/mp-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestGroundStationStage0CatalogCoverage(t *testing.T) {
	if len(groundStationOperations) != 35 {
		t.Fatalf("expected 35 Ground Station operations from docs, got %d", len(groundStationOperations))
	}
	if len(groundStationOperationByName) != len(groundStationOperations) {
		t.Fatalf("expected unique Ground Station operation names")
	}

	requiredActions := []string{
		"CreateDataflowEndpointGroupV2",
		"GetAgentTaskResponseUrl",
		"CreateMissionProfile",
		"ReserveContact",
		"GetMinuteUsage",
		"ListGroundStations",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := groundStationOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(groundStationDataTypes) != 78 {
		t.Fatalf("expected 78 Ground Station data types from docs, got %d", len(groundStationDataTypes))
	}
	if len(groundStationDataTypeByName) != len(groundStationDataTypes) {
		t.Fatalf("expected unique Ground Station data type names")
	}
}

func TestGroundStationStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := groundStationRequest(t, ts, http.MethodGet, "/not-a-real-groundstation-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestGroundStationStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := groundStationRequest(t, ts, http.MethodGet, "/groundstation", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "groundStationList") {
		t.Fatalf("expected ListGroundStations response body to include groundStationList, got %q", body)
	}
}

func TestGroundStationStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range groundStationOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := groundStationRequest(t, ts, op.Method, groundStationPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
