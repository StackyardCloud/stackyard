package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func quickSightRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "quicksight")
}

func TestQuickSightStage0CatalogCoverage(t *testing.T) {
	if len(quickSightOperations) != 230 {
		t.Fatalf("expected 230 QuickSight actions from docs, got %d", len(quickSightOperations))
	}
	if len(quickSightOperationByName) != len(quickSightOperations) {
		t.Fatalf("expected unique QuickSight action names")
	}

	requiredActions := []string{
		"DescribeAccountSettings",
		"CreateAnalysis",
		"DescribeAnalysis",
		"ListDashboards",
		"GenerateEmbedUrlForRegisteredUser",
		"GetDashboardEmbedUrl",
		"TagResource",
		"ListTagsForResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := quickSightOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(quickSightDataTypes) != 1040 {
		t.Fatalf("expected 1040 QuickSight data types from docs, got %d", len(quickSightDataTypes))
	}
	if len(quickSightDataTypeByName) != len(quickSightDataTypes) {
		t.Fatalf("expected unique QuickSight data type names")
	}

	requiredTypes := []string{
		"Analysis",
		"Dashboard",
		"DashboardSummary",
		"DataSet",
		"Template",
		"Theme",
		"VPCConnection",
	}
	for _, typeName := range requiredTypes {
		if _, ok := quickSightDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestQuickSightStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := quickSightRequest(t, ts, http.MethodPost, "/quicksight-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestQuickSightStage0KnownActionReturnsListDashboards(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := quickSightRequest(t, ts, http.MethodGet, "/accounts/123456789012/dashboards?max-results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DashboardSummaryList") {
		t.Fatalf("expected ListDashboards response to include DashboardSummaryList, got %q", body)
	}
}

func TestQuickSightStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"AwsAccountId":               "123456789012",
		"Namespace":                  "default",
		"Resolved":                   "true",
		"ActionConnectorId":          "action-connector-000001",
		"AnalysisId":                 "analysis-000001",
		"BrandId":                    "brand-000001",
		"DashboardId":                "dashboard-000001",
		"DataSetId":                  "dataset-000001",
		"DataSourceId":               "datasource-000001",
		"FolderId":                   "folder-000001",
		"MemberType":                 "DASHBOARD",
		"MemberId":                   "dashboard-000001",
		"GroupName":                  "stackyard-group",
		"MemberName":                 "stackyard-user",
		"AssignmentName":             "assignment-000001",
		"IngestionId":                "ingestion-000001",
		"ScheduleId":                 "schedule-000001",
		"Role":                       "AUTHOR",
		"TemplateId":                 "template-000001",
		"AliasName":                  "alias-000001",
		"ThemeId":                    "theme-000001",
		"TopicId":                    "topic-000001",
		"DatasetId":                  "dataset-000001",
		"UserName":                   "stackyard-user",
		"PrincipalId":                "stackyard-user",
		"VPCConnectionId":            "vpc-connection-000001",
		"CustomPermissionsName":      "custom-permissions-000001",
		"Service":                    "quicksight",
		"FlowId":                     "flow-000001",
		"ResourceArn":                "arn:aws:quicksight:us-east-1:123456789012:dashboard/dashboard-000001",
		"TagKeys":                    "env",
		"VersionNumber":              "1",
		"RecoveryWindowInDays":       "7",
		"ForceDeleteWithoutRecovery": "true",
		"MaxResults":                 "10",
		"NextToken":                  "token-000001",
		"Type":                       "QUICKSIGHT",
		"AdditionalDashboardIds":     "dashboard-000001",
		"IdentityType":               "IAM",
		"ResetDisabled":              "false",
		"SessionLifetimeInMinutes":   "30",
		"StatePersistenceEnabled":    "false",
		"UndoRedoDisabled":           "false",
		"UserArn":                    "arn:aws:quicksight:us-east-1:123456789012:user/default/stackyard-user",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range quickSightOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := quickSightRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
