package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func wellArchitectedRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "wellarchitected")
}

func TestWellArchitectedStage0CatalogCoverage(t *testing.T) {
	if len(wellArchitectedOperations) != 72 {
		t.Fatalf("expected 72 Well-Architected actions from docs, got %d", len(wellArchitectedOperations))
	}
	if len(wellArchitectedOperationByName) != len(wellArchitectedOperations) {
		t.Fatalf("expected unique Well-Architected action names")
	}

	requiredActions := []string{
		"CreateWorkload",
		"GetWorkload",
		"ListWorkloads",
		"UpdateWorkload",
		"DeleteWorkload",
		"ListLenses",
		"GetLens",
		"ListReviewTemplates",
		"ListNotifications",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := wellArchitectedOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(wellArchitectedDataTypes) != 65 {
		t.Fatalf("expected 65 Well-Architected data types from docs, got %d", len(wellArchitectedDataTypes))
	}
	if len(wellArchitectedDataTypeByName) != len(wellArchitectedDataTypes) {
		t.Fatalf("expected unique Well-Architected data type names")
	}

	requiredTypes := []string{
		"Workload",
		"WorkloadSummary",
		"Lens",
		"LensSummary",
		"ReviewTemplate",
		"ReviewTemplateSummary",
		"Profile",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := wellArchitectedDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWellArchitectedStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := wellArchitectedRequest(t, ts, http.MethodGet, "/wellarchitected/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWellArchitectedStage0KnownActionReturnsListLenses(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := wellArchitectedRequest(t, ts, http.MethodGet, "/lenses", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "LensSummaries") {
		t.Fatalf("expected ListLenses response body to include LensSummaries, got %q", body)
	}
}

func TestWellArchitectedStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"WorkloadId":        "workload-000001",
		"LensAlias":         "wellarchitected",
		"ProfileArn":        "arn:aws:wellarchitected:us-east-1:123456789012:profile/profile-000001",
		"TemplateArn":       "arn:aws:wellarchitected:us-east-1:123456789012:reviewTemplate/template-000001",
		"ReviewTemplateArn": "arn:aws:wellarchitected:us-east-1:123456789012:reviewTemplate/template-000001",
		"ShareId":           "share-000001",
		"QuestionId":        "security_1",
		"MilestoneNumber":   "1",
		"WorkloadArn":       "arn:aws:wellarchitected:us-east-1:123456789012:workload/workload-000001",
		"ShareInvitationId": "share-invitation-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range wellArchitectedOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.TrimSpace(strings.Trim(token, "{}"))
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		switch op.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			body = []byte(`{}`)
		default:
			body = nil
		}

		resp := wellArchitectedRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
