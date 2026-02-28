package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func deviceFarmRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "DeviceFarm_20150623." + action,
		},
		"devicefarm",
	)
}

func TestDeviceFarmStage0CatalogCoverage(t *testing.T) {
	if len(deviceFarmOperations) != 77 {
		t.Fatalf("expected 77 Device Farm actions from docs, got %d", len(deviceFarmOperations))
	}
	if len(deviceFarmOperationByName) != len(deviceFarmOperations) {
		t.Fatalf("expected unique Device Farm action names")
	}

	requiredActions := []string{
		"CreateProject",
		"CreateUpload",
		"ScheduleRun",
		"ListProjects",
		"GetRun",
		"ListTestGridSessions",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := deviceFarmOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(deviceFarmDataTypes) != 56 {
		t.Fatalf("expected 56 Device Farm data types from docs, got %d", len(deviceFarmDataTypes))
	}
	if len(deviceFarmDataTypeByName) != len(deviceFarmDataTypes) {
		t.Fatalf("expected unique Device Farm data type names")
	}

	requiredTypes := []string{
		"DevicePool",
		"Run",
		"Suite",
		"TestGridProject",
		"Upload",
		"VPCEConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := deviceFarmDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDeviceFarmStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := deviceFarmRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDeviceFarmStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := deviceFarmRequest(t, ts, "ListProjects", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "projects") {
		t.Fatalf("expected ListProjects response body to include projects, got %q", body)
	}
}

func TestDeviceFarmStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range deviceFarmOperations {
		resp := deviceFarmRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
