package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func incidentManagerRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	if incidentManagerIsContactAction(action) {
		return signedRequestWithService(
			t,
			http.MethodPost,
			ts.URL+"/",
			[]byte(payload),
			map[string]string{
				"Content-Type": "application/x-amz-json-1.1",
				"X-Amz-Target": "SSMContacts." + action,
			},
			"ssm-contacts",
		)
	}

	path := "/" + strings.ToLower(action[:1]) + action[1:]
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"ssm-incidents",
	)
}

func TestIncidentManagerStage0CatalogCoverage(t *testing.T) {
	if len(incidentManagerOperations) != 67 {
		t.Fatalf("expected 67 Incident Manager operations from docs, got %d", len(incidentManagerOperations))
	}
	if len(incidentManagerOperationByName) != len(incidentManagerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"ListIncidentRecords",
		"StartIncident",
		"CreateResponsePlan",
		"CreateContact",
		"ListContacts",
		"StartEngagement",
	}
	for _, action := range requiredActions {
		if _, ok := incidentManagerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(incidentManagerDataTypes) != 66 {
		t.Fatalf("expected 66 Incident Manager data types from docs, got %d", len(incidentManagerDataTypes))
	}
	if len(incidentManagerDataTypeByName) != len(incidentManagerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"IncidentRecord",
		"ResponsePlanSummary",
		"TimelineEvent",
		"Contact",
		"Rotation",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := incidentManagerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIncidentManagerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/doesNotExist",
		[]byte(`{}`),
		map[string]string{"Content-Type": "application/json"},
		"ssm-incidents",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIncidentManagerKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := incidentManagerRequest(t, ts, "ListIncidentRecords", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") || strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "IncidentRecordSummaries") {
		t.Fatalf("expected ListIncidentRecords response body to include IncidentRecordSummaries, got %q", body)
	}
}

func TestIncidentManagerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range incidentManagerOperations {
		resp := incidentManagerRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
