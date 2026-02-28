package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAppFlowStage12FlowLifecycleAndExecutions(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	flowName := "stage-flow-001"

	resp := appFlowRequest(t, ts, http.MethodPost, "/create-flow", []byte(`{"flowName":"`+flowName+`","description":"stage flow"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeAppFlowPayload(t, resp)
	if got := appFlowPayloadString(createPayload, "flowStatus"); got != "Active" {
		t.Fatalf("expected create flowStatus Active, got %q", got)
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/describe-flow", []byte(`{"flowName":"`+flowName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	describePayload := decodeAppFlowPayload(t, resp)
	if got := appFlowPayloadString(describePayload, "flowName"); got != flowName {
		t.Fatalf("expected describe flowName %q, got %q", flowName, got)
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/start-flow", []byte(`{"flowName":"`+flowName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeAppFlowPayload(t, resp)
	executionID := appFlowPayloadString(startPayload, "executionId")
	if executionID == "" {
		t.Fatalf("expected StartFlow executionId")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/describe-flow-execution-records", []byte(`{"flowName":"`+flowName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	recordsPayload := decodeAppFlowPayload(t, resp)
	records, _ := recordsPayload["flowExecutions"].([]any)
	if len(records) == 0 {
		t.Fatalf("expected at least one execution record")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/stop-flow", []byte(`{"flowName":"`+flowName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	stopPayload := decodeAppFlowPayload(t, resp)
	if got := appFlowPayloadString(stopPayload, "flowStatus"); got != "Suspended" {
		t.Fatalf("expected stop flowStatus Suspended, got %q", got)
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/cancel-flow-executions", []byte(`{"flowName":"`+flowName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/update-flow", []byte(`{"flowName":"`+flowName+`","description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	updatePayload := decodeAppFlowPayload(t, resp)
	if got := appFlowPayloadString(updatePayload, "flowArn"); got == "" {
		t.Fatalf("expected UpdateFlow flowArn")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/delete-flow", []byte(`{"flowName":"`+flowName+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestAppFlowStage345ConnectorProfilesAndRegistration(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	profileName := "stage-profile-001"
	connectorLabel := "CustomStageConnector"

	resp := appFlowRequest(t, ts, http.MethodPost, "/create-connector-profile", []byte(`{"connectorProfileName":"`+profileName+`","connectorType":"Salesforce"}`))
	assertStatus(t, resp, http.StatusOK)
	createProfilePayload := decodeAppFlowPayload(t, resp)
	if got := appFlowPayloadString(createProfilePayload, "connectorProfileArn"); got == "" {
		t.Fatalf("expected connectorProfileArn on create")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/describe-connector-profiles", []byte(`{"connectorProfileNames":["`+profileName+`"]}`))
	assertStatus(t, resp, http.StatusOK)
	describeProfilesPayload := decodeAppFlowPayload(t, resp)
	profiles, _ := describeProfilesPayload["connectorProfileDetails"].([]any)
	if len(profiles) == 0 {
		t.Fatalf("expected described connector profile")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/describe-connector", []byte(`{"connectorType":"Salesforce"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/describe-connector-entity", []byte(`{"entityName":"Account"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/list-connector-entities", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	connectorEntitiesPayload := decodeAppFlowPayload(t, resp)
	if _, ok := connectorEntitiesPayload["connectorEntityMap"].(map[string]any); !ok {
		t.Fatalf("expected connectorEntityMap in response")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/register-connector", []byte(`{"connectorLabel":"`+connectorLabel+`","connectorType":"CustomConnector"}`))
	assertStatus(t, resp, http.StatusOK)
	registerPayload := decodeAppFlowPayload(t, resp)
	if got := appFlowPayloadString(registerPayload, "connectorArn"); got == "" {
		t.Fatalf("expected connectorArn on register")
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/update-connector-registration", []byte(`{"connectorLabel":"`+connectorLabel+`","connectorType":"CustomConnector"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/list-connectors", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listConnectorsPayload := decodeAppFlowPayload(t, resp)
	connectors, _ := listConnectorsPayload["connectors"].([]any)
	found := false
	for _, item := range connectors {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(appFlowPayloadString(entry, "connectorLabel"), connectorLabel) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s in ListConnectors", connectorLabel)
	}

	resp = appFlowRequest(t, ts, http.MethodPost, "/describe-connectors", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/reset-connector-metadata-cache", []byte(`{"connectorLabel":"`+connectorLabel+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/unregister-connector", []byte(`{"connectorLabel":"`+connectorLabel+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodPost, "/delete-connector-profile", []byte(`{"connectorProfileName":"`+profileName+`"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestAppFlowStage6TaggingLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:appflow:us-east-1:123456789012:flow/stackyard-seed-flow"
	escapedARN := url.PathEscape(resourceARN)

	resp := appFlowRequest(t, ts, http.MethodPost, "/tags/"+escapedARN, []byte(`{"tags":{"env":"dev","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeAppFlowPayload(t, resp)
	tagMap, ok := listPayload["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected tags map in ListTagsForResource response")
	}
	if got := appFlowPayloadString(tagMap, "owner"); got != "qa" {
		t.Fatalf("expected owner tag qa, got %q", got)
	}

	resp = appFlowRequest(t, ts, http.MethodDelete, "/tags/"+escapedARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = appFlowRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload = decodeAppFlowPayload(t, resp)
	tagMap, _ = listPayload["tags"].(map[string]any)
	if _, exists := tagMap["owner"]; exists {
		t.Fatalf("expected owner tag removed")
	}
	if got := appFlowPayloadString(tagMap, "env"); got != "dev" {
		t.Fatalf("expected env tag dev after untagging owner, got %q", got)
	}
}

func decodeAppFlowPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func appFlowPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
