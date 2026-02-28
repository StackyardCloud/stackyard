package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func inspectorV2Request(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "inspector2")
}

func inspectorV2PathForOperation(op inspectorV2Operation) string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	out := re.ReplaceAllStringFunc(op.URI, func(match string) string {
		name := strings.Trim(match, "{}")
		switch name {
		case "resourceArn":
			return url.PathEscape("arn:aws:inspector2:us-east-1:123456789012:target/stackyard")
		case "tagKeys":
			return "env"
		case "resourceType":
			return "EC2"
		case "scanType":
			return "NETWORK"
		case "maxResults":
			return "10"
		case "nextToken":
			return "token"
		default:
			return "stackyard"
		}
	})
	return strings.ReplaceAll(out, "?tagKeys=env", "?tagKeys=env")
}

func TestInspectorV2RouterUnknownAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := inspectorV2Request(t, ts, http.MethodPost, "/inspector/unknown", "{}")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestInspectorV2RouterKnownActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "ListFindings", method: http.MethodPost, path: "/findings/list", body: `{"maxResults":10}`},
		{name: "CreateFilter", method: http.MethodPost, path: "/filters/create", body: `{"name":"stackyard-filter","action":"NONE","filterCriteria":{}}`},
		{name: "ListTagsForResource", method: http.MethodGet, path: "/tags/arn%3Aaws%3Ainspector2%3Aus-east-1%3A123456789012%3Atarget%2Fstackyard", body: ""},
	}

	for _, tc := range cases {
		resp := inspectorV2Request(t, ts, tc.method, tc.path, tc.body)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s %s unexpectedly returned NotImplemented: status=%d body=%s", tc.method, tc.path, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s %s unexpectedly returned status=%d body=%s", tc.method, tc.path, resp.StatusCode, body)
		}
	}
}

func TestInspectorV2AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range inspectorV2Operations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		path := inspectorV2PathForOperation(op)
		resp := inspectorV2Request(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
