package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func arcZonalShiftRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ArcZonalShiftService." + action,
		},
		"arc-zonal-shift",
	)
}

func TestARCZonalShiftStage0CatalogCoverage(t *testing.T) {
	if len(arcZonalShiftOperations) != 15 {
		t.Fatalf("expected 15 ARC Zonal Shift operations from docs, got %d", len(arcZonalShiftOperations))
	}
	if len(arcZonalShiftOperationByName) != len(arcZonalShiftOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"ListManagedResources",
		"GetManagedResource",
		"StartZonalShift",
		"UpdateZonalShift",
		"CancelZonalShift",
		"StartPracticeRun",
		"CancelPracticeRun",
		"UpdateZonalAutoshiftConfiguration",
	}
	for _, action := range requiredActions {
		if _, ok := arcZonalShiftOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(arcZonalShiftDataTypes) != 7 {
		t.Fatalf("expected 7 ARC Zonal Shift data types from docs, got %d", len(arcZonalShiftDataTypes))
	}
	if len(arcZonalShiftDataTypeByName) != len(arcZonalShiftDataTypes) {
		t.Fatalf("expected unique data type names")
	}
}

func TestARCZonalShiftStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := arcZonalShiftRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestARCZonalShiftStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := arcZonalShiftRequest(t, ts, "ListManagedResources", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "items") {
		t.Fatalf("expected ListManagedResources response body to include items, got %q", body)
	}
}

func TestARCZonalShiftStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range arcZonalShiftOperations {
		resp := arcZonalShiftRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
