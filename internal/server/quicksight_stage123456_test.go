package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestQuickSightStage12AnalysisLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := "123456789012"
	analysisID := "stage-analysis-001"

	resp := quickSightRequest(t, ts, http.MethodPost, "/accounts/"+accountID+"/analyses/"+analysisID, []byte(`{"Name":"stage-analysis"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeQuickSightPayload(t, resp)
	if quickSightPayloadString(createPayload, "Arn") == "" {
		t.Fatalf("expected CreateAnalysis to return Arn")
	}

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/analyses/"+analysisID, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Analysis") {
		t.Fatalf("expected DescribeAnalysis response to include Analysis, got %q", body)
	}

	resp = quickSightRequest(t, ts, http.MethodPut, "/accounts/"+accountID+"/analyses/"+analysisID, []byte(`{"Name":"stage-analysis-updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/analyses?max-results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "AnalysisSummaryList") {
		t.Fatalf("expected ListAnalyses response to include AnalysisSummaryList, got %q", body)
	}

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/analyses/"+analysisID+"/permissions", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Permissions") {
		t.Fatalf("expected DescribeAnalysisPermissions response to include Permissions, got %q", body)
	}

	resp = quickSightRequest(
		t,
		ts,
		http.MethodPut,
		"/accounts/"+accountID+"/analyses/"+analysisID+"/permissions",
		[]byte(`{"Permissions":[{"Principal":"arn:aws:iam::123456789012:root","Actions":["quicksight:DescribeAnalysis"]}]}`),
	)
	assertStatus(t, resp, http.StatusOK)
	updatePermsPayload := decodeQuickSightPayload(t, resp)
	if _, ok := updatePermsPayload["Permissions"].([]any); !ok {
		t.Fatalf("expected UpdateAnalysisPermissions to return Permissions")
	}

	resp = quickSightRequest(
		t,
		ts,
		http.MethodDelete,
		"/accounts/"+accountID+"/analyses/"+analysisID+"?force-delete-without-recovery=true&recovery-window-in-days=7",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
}

func TestQuickSightStage34NamespaceEmbedAndFlowSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	accountID := "123456789012"
	namespace := "default"
	groupName := "stage-group-001"
	userName := "stage-user-001"

	resp := quickSightRequest(
		t,
		ts,
		http.MethodPost,
		"/accounts/"+accountID+"/namespaces/"+namespace+"/groups",
		[]byte(`{"GroupName":"`+groupName+`","Description":"stage group"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = quickSightRequest(
		t,
		ts,
		http.MethodPost,
		"/accounts/"+accountID+"/namespaces/"+namespace+"/users",
		[]byte(`{"UserName":"`+userName+`","Email":"stage-user@example.com","IdentityType":"IAM","UserRole":"AUTHOR"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/namespaces/"+namespace+"/users?max-results=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "UserList") {
		t.Fatalf("expected ListUsers response to include UserList, got %q", body)
	}

	resp = quickSightRequest(t, ts, http.MethodPost, "/accounts/"+accountID+"/embed-url/registered-user", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	embedPayload := decodeQuickSightPayload(t, resp)
	if quickSightPayloadString(embedPayload, "EmbedUrl") == "" {
		t.Fatalf("expected GenerateEmbedUrlForRegisteredUser to return EmbedUrl")
	}

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/dashboards/dashboard-000001/embed-url", nil)
	assertStatus(t, resp, http.StatusOK)
	dashboardEmbedPayload := decodeQuickSightPayload(t, resp)
	if quickSightPayloadString(dashboardEmbedPayload, "EmbedUrl") == "" {
		t.Fatalf("expected GetDashboardEmbedUrl to return EmbedUrl")
	}

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/flows/flow-000001/metadata", nil)
	assertStatus(t, resp, http.StatusOK)
	flowMetadataPayload := decodeQuickSightPayload(t, resp)
	if quickSightPayloadString(flowMetadataPayload, "FlowId") == "" {
		t.Fatalf("expected GetFlowMetadata to return FlowId")
	}

	resp = quickSightRequest(t, ts, http.MethodGet, "/accounts/"+accountID+"/flows/flow-000001/permissions", nil)
	assertStatus(t, resp, http.StatusOK)
	flowPermsPayload := decodeQuickSightPayload(t, resp)
	if _, ok := flowPermsPayload["Permissions"].([]any); !ok {
		t.Fatalf("expected GetFlowPermissions to return Permissions")
	}

	resp = quickSightRequest(t, ts, http.MethodDelete, "/accounts/"+accountID+"/namespaces/"+namespace+"/users/"+userName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = quickSightRequest(t, ts, http.MethodDelete, "/accounts/"+accountID+"/namespaces/"+namespace+"/groups/"+groupName, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestQuickSightStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:quicksight:us-east-1:123456789012:dashboard/dashboard-000001"
	escapedARN := url.PathEscape(resourceARN)

	resp := quickSightRequest(t, ts, http.MethodPost, "/resources/"+escapedARN+"/tags", []byte(`{"Tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = quickSightRequest(t, ts, http.MethodGet, "/resources/"+escapedARN+"/tags", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected ListTagsForResource response to include owner tag, got %q", body)
	}

	resp = quickSightRequest(t, ts, http.MethodDelete, "/resources/"+escapedARN+"/tags?keys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = quickSightRequest(t, ts, http.MethodGet, "/resources/"+escapedARN+"/tags", nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected owner tag removed, got %q", body)
	}
	if !strings.Contains(body, "\"env\"") {
		t.Fatalf("expected env tag to remain after untag, got %q", body)
	}

	resp = quickSightRequest(t, ts, http.MethodPost, "/quicksight/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body = string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/accounts/123456789012/embed-url/registered-user",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"quicksight",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body = string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeQuickSightPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func quickSightPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
