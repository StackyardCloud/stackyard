package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func TestQBusinessStage0CatalogCoverage(t *testing.T) {
	if len(qBusinessOperations) != 83 {
		t.Fatalf("expected 83 Amazon Q Business actions from docs, got %d", len(qBusinessOperations))
	}
	if len(qBusinessOperationByName) != len(qBusinessOperations) {
		t.Fatalf("expected unique Amazon Q Business action names")
	}

	requiredActions := []string{
		"CreateApplication",
		"CreateIndex",
		"CreateDataSource",
		"Chat",
		"ChatSync",
		"ListApplications",
		"ListChatResponseConfigurations",
		"UpdateSubscription",
	}
	for _, action := range requiredActions {
		if _, ok := qBusinessOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(qBusinessDataTypes) != 152 {
		t.Fatalf("expected 152 Amazon Q Business data types from docs, got %d", len(qBusinessDataTypes))
	}
	if len(qBusinessDataTypeByName) != len(qBusinessDataTypes) {
		t.Fatalf("expected unique Amazon Q Business data type names")
	}

	requiredTypes := []string{
		"Application",
		"Index",
		"DataSource",
		"Conversation",
		"Message",
		"Subscription",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := qBusinessDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func qBusinessRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "qbusiness")
}

func TestQBusinessStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := qBusinessRequest(t, ts, http.MethodPost, "/unknown-qbusiness-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestQBusinessStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := qBusinessRequest(t, ts, http.MethodGet, "/applications", ``)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "applications") {
		t.Fatalf("expected ListApplications response body to include applications, got %q", body)
	}
}

func TestQBusinessStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range qBusinessOperations {
		path := qBusinessRenderTestURI(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			payload = `{}`
		}

		resp := qBusinessRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s (path=%s)", op.Name, resp.StatusCode, body, path)
		}
	}
}

var qBusinessURIPlaceholderPattern = regexp.MustCompile(`\{([^}]+)\}`)

func qBusinessRenderTestURI(uriTemplate string) string {
	return qBusinessURIPlaceholderPattern.ReplaceAllStringFunc(uriTemplate, func(raw string) string {
		placeholder := strings.TrimSuffix(strings.Trim(raw, "{}"), "+")
		switch strings.ToLower(placeholder) {
		case "applicationid":
			return "app-000001"
		case "indexid":
			return "idx-000001"
		case "datasourceid":
			return "ds-000001"
		case "dataaccessorid":
			return "da-000001"
		case "pluginid":
			return "plugin-000001"
		case "retrieverid":
			return "retriever-000001"
		case "userid":
			return "user-000001"
		case "conversationid":
			return "conv-000001"
		case "messageid":
			return "msg-000001"
		case "mediaid":
			return "media-000001"
		case "attachmentid":
			return "att-000001"
		case "subscriptionid":
			return "sub-000001"
		case "chatresponseconfigurationid":
			return "crc-000001"
		case "statementid":
			return "statement-000001"
		case "groupname":
			return "engineering"
		case "documentid":
			return "doc-000001"
		case "plugintype":
			return "CUSTOM"
		case "resourcearn":
			return url.QueryEscape("arn:aws:qbusiness:us-east-1:123456789012:application/app-000001")
		case "clienttoken":
			return "token-000001"
		case "parentmessageid":
			return "msg-000001"
		case "maxresults":
			return "10"
		case "nexttoken":
			return "token-000001"
		case "starttime":
			return "2026-01-01T00:00:00Z"
		case "endtime":
			return "2026-01-02T00:00:00Z"
		case "statusfilter":
			return "SUCCEEDED"
		case "tagkeys":
			return "env"
		case "outputformat":
			return "MARKDOWN"
		case "datasourceids":
			return "ds-000001"
		case "updatedearlierthan":
			return "2026-01-02T00:00:00Z"
		case "usergroups":
			return "engineering"
		default:
			return "stackyard"
		}
	})
}
