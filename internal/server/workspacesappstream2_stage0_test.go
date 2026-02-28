package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func workspacesAppStream2Request(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "PhotonAdminProxyService." + action,
		},
		"appstream",
	)
}

func TestWorkSpacesAppStream2Stage0CatalogCoverage(t *testing.T) {
	if len(workspacesAppStream2Operations) != 88 {
		t.Fatalf("expected 88 WorkSpaces Applications actions from docs, got %d", len(workspacesAppStream2Operations))
	}
	if len(workspacesAppStream2OperationByName) != len(workspacesAppStream2Operations) {
		t.Fatalf("expected unique WorkSpaces Applications action names")
	}

	requiredActions := []string{
		"CreateFleet",
		"DescribeFleets",
		"CreateStack",
		"ListAssociatedFleets",
		"CreateApplication",
		"DescribeApplications",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := workspacesAppStream2OperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(workspacesAppStream2Types) != 52 {
		t.Fatalf("expected 52 WorkSpaces Applications data types from docs, got %d", len(workspacesAppStream2Types))
	}
	if len(workspacesAppStream2TypeByName) != len(workspacesAppStream2Types) {
		t.Fatalf("expected unique WorkSpaces Applications data type names")
	}

	requiredTypes := []string{
		"Fleet",
		"Stack",
		"Image",
		"ImageBuilder",
		"Application",
		"DirectoryConfig",
		"Entitlement",
		"User",
	}
	for _, typeName := range requiredTypes {
		if _, ok := workspacesAppStream2TypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWorkSpacesAppStream2Stage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesAppStream2Request(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWorkSpacesAppStream2Stage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesAppStream2Request(t, ts, "DescribeFleets", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Fleets") {
		t.Fatalf("expected DescribeFleets response body to include Fleets, got %q", body)
	}
}

func TestWorkSpacesAppStream2Stage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range workspacesAppStream2Operations {
		resp := workspacesAppStream2Request(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
