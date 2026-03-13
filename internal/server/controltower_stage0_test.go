package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func controlTowerRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "controltower")
}

func controlTowerPathForTest(template string) string {
	resourceARN := "arn:aws:controltower:us-east-1:123456789012:landingzone/lz-000001"
	out := template
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape(resourceARN))
	return out
}

func TestControlTowerStage0CatalogCoverage(t *testing.T) {
	if len(controlTowerOperations) != 28 {
		t.Fatalf("expected 28 Control Tower operations from docs, got %d", len(controlTowerOperations))
	}
	if len(controlTowerOperationByName) != len(controlTowerOperations) {
		t.Fatalf("expected unique operation names")
	}
	if len(controlTowerDataTypes) != 31 {
		t.Fatalf("expected 31 Control Tower data types from docs, got %d", len(controlTowerDataTypes))
	}
	if len(controlTowerDataTypeByName) != len(controlTowerDataTypes) {
		t.Fatalf("expected unique data type names")
	}
}

func TestControlTowerUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := controlTowerRequest(t, ts, http.MethodPost, "/controltower-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestControlTowerAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range controlTowerOperations {
		path := controlTowerPathForTest(op.URI)
		var body []byte
		switch op.Name {
		case "TagResource":
			body = []byte(`{"tags":{"stackyard":"true"}}`)
		default:
			if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
				body = []byte(`{}`)
			}
		}

		resp := controlTowerRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}

func TestControlTowerStoreGetResponsesMatchModeledShapes(t *testing.T) {
	store := newControlTowerStore()

	baseline := store.Handle(
		"GetBaseline",
		map[string]any{"baselineIdentifier": "arn:aws:controltower:us-east-1::baseline/aws-baseline/default"},
		nil,
		nil,
	)
	if _, ok := baseline["baseline"]; ok {
		t.Fatalf("GetBaseline unexpectedly returned nested baseline wrapper: %#v", baseline)
	}
	if baseline["arn"] == nil || baseline["name"] == nil {
		t.Fatalf("GetBaseline missing modeled root fields: %#v", baseline)
	}

	enabledBaseline := store.Handle(
		"GetEnabledBaseline",
		map[string]any{"enabledBaselineIdentifier": "arn:aws:controltower:us-east-1:123456789012:enabledbaseline/ebl-000001"},
		nil,
		nil,
	)
	if _, ok := enabledBaseline["enabledBaseline"]; ok {
		t.Fatalf("GetEnabledBaseline unexpectedly returned legacy enabledBaseline wrapper: %#v", enabledBaseline)
	}
	enabledBaselineDetails, ok := enabledBaseline["enabledBaselineDetails"].(map[string]any)
	if !ok {
		t.Fatalf("GetEnabledBaseline missing enabledBaselineDetails object: %#v", enabledBaseline)
	}
	driftSummary, ok := enabledBaselineDetails["driftStatusSummary"].(map[string]any)
	if !ok {
		t.Fatalf("GetEnabledBaseline missing driftStatusSummary object: %#v", enabledBaselineDetails)
	}
	driftTypes, ok := driftSummary["types"].(map[string]any)
	if !ok {
		t.Fatalf("GetEnabledBaseline drift types should be an object, got %#v", driftSummary["types"])
	}
	inheritance, ok := driftTypes["inheritance"].(map[string]any)
	if !ok || inheritance["status"] != "IN_SYNC" {
		t.Fatalf("GetEnabledBaseline inheritance drift summary mismatch: %#v", driftTypes)
	}

	enabledControl := store.Handle(
		"GetEnabledControl",
		map[string]any{"enabledControlIdentifier": "arn:aws:controltower:us-east-1:123456789012:enabledcontrol/ec-000001"},
		nil,
		nil,
	)
	if _, ok := enabledControl["enabledControl"]; ok {
		t.Fatalf("GetEnabledControl unexpectedly returned legacy enabledControl wrapper: %#v", enabledControl)
	}
	enabledControlDetails, ok := enabledControl["enabledControlDetails"].(map[string]any)
	if !ok {
		t.Fatalf("GetEnabledControl missing enabledControlDetails object: %#v", enabledControl)
	}
	controlDriftSummary, ok := enabledControlDetails["driftStatusSummary"].(map[string]any)
	if !ok || controlDriftSummary["driftStatus"] != "IN_SYNC" {
		t.Fatalf("GetEnabledControl drift summary mismatch: %#v", enabledControlDetails)
	}
	if _, ok := controlDriftSummary["types"]; ok {
		t.Fatalf("GetEnabledControl drift summary unexpectedly returned legacy types object: %#v", controlDriftSummary)
	}

	landingZone := store.Handle(
		"GetLandingZone",
		map[string]any{"landingZoneIdentifier": "arn:aws:controltower:us-east-1:123456789012:landingzone/lz-000001"},
		nil,
		nil,
	)
	landingZoneDetails, ok := landingZone["landingZone"].(map[string]any)
	if !ok {
		t.Fatalf("GetLandingZone missing landingZone object: %#v", landingZone)
	}
	if _, ok := landingZoneDetails["latestDriftStatus"]; ok {
		t.Fatalf("GetLandingZone unexpectedly returned latestDriftStatus: %#v", landingZoneDetails)
	}
	if _, ok := landingZoneDetails["createdAt"]; ok {
		t.Fatalf("GetLandingZone unexpectedly returned createdAt: %#v", landingZoneDetails)
	}
	manifest, ok := landingZoneDetails["manifest"].(map[string]any)
	if !ok {
		t.Fatalf("GetLandingZone manifest should remain a document object: %#v", landingZoneDetails["manifest"])
	}
	if _, ok := manifest["governedRegions"].([]any); !ok {
		t.Fatalf("GetLandingZone manifest missing governedRegions: %#v", manifest)
	}
}

func TestControlTowerStoreOperationReadsReturnModeledWrappers(t *testing.T) {
	store := newControlTowerStore()

	updateLandingZone := store.Handle(
		"UpdateLandingZone",
		map[string]any{"landingZoneIdentifier": "arn:aws:controltower:us-east-1:123456789012:landingzone/lz-000001"},
		nil,
		nil,
	)
	landingZoneOpID := ctString(updateLandingZone["operationIdentifier"], "")
	landingZoneOp := store.Handle(
		"GetLandingZoneOperation",
		map[string]any{"operationIdentifier": landingZoneOpID},
		nil,
		nil,
	)
	if _, ok := landingZoneOp["landingZoneOperation"]; ok {
		t.Fatalf("GetLandingZoneOperation unexpectedly returned legacy wrapper: %#v", landingZoneOp)
	}
	operationDetails, ok := landingZoneOp["operationDetails"].(map[string]any)
	if !ok || operationDetails["operationIdentifier"] != landingZoneOpID {
		t.Fatalf("GetLandingZoneOperation details mismatch: %#v", landingZoneOp)
	}

	enableControl := store.Handle(
		"EnableControl",
		map[string]any{
			"controlIdentifier": ctDefaultControlIdentifier(),
			"targetIdentifier":  "ou-0000-example",
		},
		nil,
		nil,
	)
	controlOpID := ctString(enableControl["operationIdentifier"], "")
	controlOp := store.Handle(
		"GetControlOperation",
		map[string]any{"operationIdentifier": controlOpID},
		nil,
		nil,
	)
	controlDetails, ok := controlOp["controlOperation"].(map[string]any)
	if !ok {
		t.Fatalf("GetControlOperation missing controlOperation object: %#v", controlOp)
	}
	if controlDetails["operationIdentifier"] != controlOpID {
		t.Fatalf("GetControlOperation operationIdentifier mismatch: %#v", controlDetails)
	}
}
