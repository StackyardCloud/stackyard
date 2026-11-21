package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAppConfigDataTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func appConfigDataRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	if headers == nil {
		headers = map[string]string{}
	}
	if body != nil && headers["Content-Type"] == "" {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "appconfig")
}

func TestAppConfigDataStage0OperationCoverage(t *testing.T) {
	if len(appConfigDataOperations) != 2 || len(appConfigDataOperationByName) != 2 {
		t.Fatalf("expected 2 unique AppConfig Data operations, got %d", len(appConfigDataOperations))
	}
	for _, name := range []string{"StartConfigurationSession", "GetLatestConfiguration"} {
		if _, ok := appConfigDataOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestAppConfigDataStage0RoutesAheadOfAppConfig(t *testing.T) {
	_, ts := newAppConfigDataTestServer(t)

	// Signed "appconfig" like the control plane, so routing order decides.
	resp := appConfigDataRequest(t, ts, http.MethodPost, "/configurationsessions", []byte(`{}`), nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ApplicationIdentifier") {
		t.Fatalf("expected AppConfig Data validation error, got %q", body)
	}
	if got := resp.Header.Get("X-Amzn-ErrorType"); got != "BadRequestException" {
		t.Fatalf("expected BadRequestException, got %q", got)
	}

	// The control plane still answers its own routes.
	resp = appConfigDataRequest(t, ts, http.MethodGet, "/applications", nil, nil)
	assertStatus(t, resp, http.StatusOK)

	// Wrong signing service is refused.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/configurationsessions", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "appconfigdata")
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusBadRequest && strings.Contains(string(mustBody(t, resp)), "ApplicationIdentifier") {
		t.Fatalf("expected a request signed for another service not to reach AppConfig Data, got %d", resp.StatusCode)
	}
}
