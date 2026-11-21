package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func apiGatewayManagementRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(t, method, ts.URL+path, body, nil, "execute-api")
}

func TestAPIGatewayManagementStage0Catalog(t *testing.T) {
	if len(apiGatewayManagementOperations) != 3 {
		t.Fatalf("expected 3 operations, got %d", len(apiGatewayManagementOperations))
	}
	for _, name := range []string{"DeleteConnection", "GetConnection", "PostToConnection"} {
		if _, ok := apiGatewayManagementOperationByName[name]; !ok {
			t.Fatalf("missing operation %s", name)
		}
	}
	if len(apiGatewayManagementOperationByMethod) != 3 {
		t.Fatalf("expected one operation per method")
	}
}

func TestAPIGatewayManagementStage0Routing(t *testing.T) {
	cases := []struct {
		path      string
		stage, id string
		ok        bool
	}{
		{"/@connections/abc=", "", "abc=", true},
		{"/prod/@connections/abc%3D", "prod", "abc%3D", true},
		{"/@connections", "", "", false},
		{"/a/b/@connections/x", "", "", false},
		{"/prod/connections/x", "", "", false},
	}
	for _, tc := range cases {
		stage, id, ok := parseAPIGatewayManagementPath(tc.path)
		if stage != tc.stage || id != tc.id || ok != tc.ok {
			t.Fatalf("%s: got (%q,%q,%v)", tc.path, stage, id, ok)
		}
	}
	if got := apiGatewayManagementHostAPIID("AbC123.execute-api.us-east-1.amazonaws.com"); got != "abc123" {
		t.Fatalf("host api id = %q", got)
	}
}

func TestAPIGatewayManagementStage0UnsupportedMethodAndAuth(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := apiGatewayManagementRequest(t, ts, http.MethodPut, "/@connections/abc=", nil)
	assertStatus(t, resp, http.StatusNotImplemented)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("unexpected body %s", body)
	}

	// An unsigned bare @connections request is claimed and rejected by SigV4.
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/@connections/abc=", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected an auth error, got %d", resp.StatusCode)
	}
	_ = mustBody(t, resp)
}
