package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func connectRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "connect")
}

func TestConnectStage0CatalogCoverage(t *testing.T) {
	if len(connectOperations) != 364 {
		t.Fatalf("expected 364 Connect operations from docs, got %d", len(connectOperations))
	}
	if len(connectOperationByName) != len(connectOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateInstance",
		"DescribeInstance",
		"ListInstances",
		"CreateUser",
		"ListUsers",
		"CreateQueue",
		"ListQueues",
		"CreateContactFlow",
		"ListContactFlows",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := connectOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(connectTypes) != 523 {
		t.Fatalf("expected 523 Connect data types from docs, got %d", len(connectTypes))
	}
	if len(connectTypeByName) != len(connectTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Instance",
		"InstanceSummary",
		"User",
		"Queue",
		"ContactFlow",
		"TagSet",
	}
	for _, typeName := range requiredTypes {
		if _, ok := connectTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestConnectStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := connectRequest(t, ts, http.MethodPost, "/connect-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestConnectStage0KnownActionReturnsListInstances(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := connectRequest(t, ts, http.MethodGet, "/instance", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "InstanceSummaryList") {
		t.Fatalf("expected ListInstances response body to include InstanceSummaryList, got %q", body)
	}
}

func TestConnectStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"InstanceId":                 "instance-000001",
		"ContactId":                  "contact-000001",
		"EvaluationFormId":           "evaluation-form-000001",
		"EvaluationId":               "evaluation-000001",
		"ContactFlowId":              "contact-flow-000001",
		"ContactFlowVersion":         "1",
		"ContactFlowModuleId":        "contact-flow-module-000001",
		"ContactFlowModuleVersion":   "1",
		"AliasId":                    "alias-000001",
		"DataTableId":                "data-table-000001",
		"AttributeName":              "attribute-000001",
		"EmailAddressId":             "email-address-000001",
		"HoursOfOperationId":         "hours-000001",
		"HoursOfOperationOverrideId": "hours-override-000001",
		"AssociationId":              "association-000001",
		"NotificationId":             "notification-000001",
		"PhoneNumberId":              "phone-number-000001",
		"PromptId":                   "prompt-000001",
		"QueueId":                    "queue-000001",
		"QuickConnectId":             "quick-connect-000001",
		"RoutingProfileId":           "routing-profile-000001",
		"RuleId":                     "rule-000001",
		"SecurityProfileId":          "security-profile-000001",
		"TaskTemplateId":             "task-template-000001",
		"TestCaseId":                 "test-case-000001",
		"TestCaseExecutionId":        "test-case-execution-000001",
		"TrafficDistributionGroupId": "traffic-distribution-group-000001",
		"IntegrationAssociationId":   "integration-association-000001",
		"UseCaseId":                  "use-case-000001",
		"UserId":                     "user-000001",
		"HierarchyGroupId":           "hierarchy-group-000001",
		"ViewId":                     "view-000001",
		"ViewVersion":                "1",
		"VocabularyId":               "vocabulary-000001",
		"WorkspaceId":                "workspace-000001",
		"Page":                       "page-000001",
		"LanguageCode":               "en-US",
		"Name":                       "name-000001",
		"Id":                         "id-000001",
		"ResourceId":                 "resource-000001",
		"ResourceType":               "QUEUE",
		"InitialContactId":           "initial-contact-000001",
		"AuthenticationProfileId":    "auth-profile-000001",
		"AttributeType":              "INBOUND_CALLS",
		"FileId":                     "file-000001",
		"RegistrationId":             "registration-000001",
		"resourceArn":                "arn:aws:connect:us-east-1:123456789012:instance/instance-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range connectOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}

		resp := connectRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
