package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAmplifyStage1AppLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyRequest(t, ts, http.MethodPost, "/apps", []byte(`{"name":"stage-app","repository":"https://example.com/stage.git"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeAmplifyPayload(t, resp)
	app := amplifyPayloadMap(createPayload, "app")
	appID := amplifyPayloadString(app, "appId")
	if appID == "" {
		t.Fatalf("expected CreateApp to return app.appId")
	}

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+url.PathEscape(appID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+url.PathEscape(appID), []byte(`{"name":"stage-app-updated"}`))
	assertStatus(t, resp, http.StatusOK)
	updatePayload := decodeAmplifyPayload(t, resp)
	updated := amplifyPayloadMap(updatePayload, "app")
	if got := amplifyPayloadString(updated, "name"); got != "stage-app-updated" {
		t.Fatalf("expected updated app name, got %q", got)
	}

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps", nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeAmplifyPayload(t, resp)
	apps, ok := listPayload["apps"].([]any)
	if !ok || len(apps) == 0 {
		t.Fatalf("expected ListApps to return apps")
	}

	resp = amplifyRequest(t, ts, http.MethodDelete, "/apps/"+url.PathEscape(appID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestAmplifyStage2BranchAndBackendLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	appID := "d1234567890"
	branchName := "stage2-branch"
	environmentName := "stage2"

	resp := amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/branches", []byte(`{"branchName":"`+branchName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	createBranchPayload := decodeAmplifyPayload(t, resp)
	branch := amplifyPayloadMap(createBranchPayload, "branch")
	if got := amplifyPayloadString(branch, "branchName"); got != branchName {
		t.Fatalf("expected created branch %q, got %q", branchName, got)
	}

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/branches/"+url.PathEscape(branchName), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/branches/"+url.PathEscape(branchName), []byte(`{"displayName":"Stage Two Branch"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/branches", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/backendenvironments", []byte(`{"environmentName":"`+environmentName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	createBackendPayload := decodeAmplifyPayload(t, resp)
	backend := amplifyPayloadMap(createBackendPayload, "backendEnvironment")
	if got := amplifyPayloadString(backend, "environmentName"); got != environmentName {
		t.Fatalf("expected created backend environment %q, got %q", environmentName, got)
	}

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/backendenvironments/"+url.PathEscape(environmentName), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/backendenvironments", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodDelete, "/apps/"+appID+"/backendenvironments/"+url.PathEscape(environmentName), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodDelete, "/apps/"+appID+"/branches/"+url.PathEscape(branchName), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestAmplifyStage34JobsDomainsWebhooksAndAccessLogs(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	appID := "d1234567890"
	branchName := "main"

	resp := amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/branches/"+branchName+"/deployments", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createDeploymentPayload := decodeAmplifyPayload(t, resp)
	jobID := amplifyPayloadString(createDeploymentPayload, "jobId")
	if jobID == "" {
		t.Fatalf("expected CreateDeployment to return jobId")
	}

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/branches/"+branchName+"/deployments/start", []byte(`{"jobId":"`+jobID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/branches/"+branchName+"/jobs", []byte(`{"jobType":"RELEASE"}`))
	assertStatus(t, resp, http.StatusOK)
	startJobPayload := decodeAmplifyPayload(t, resp)
	jobSummary := amplifyPayloadMap(startJobPayload, "jobSummary")
	startedJobID := amplifyPayloadString(jobSummary, "jobId")
	if startedJobID == "" {
		t.Fatalf("expected StartJob to return jobSummary.jobId")
	}

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/branches/"+branchName+"/jobs/"+startedJobID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/branches/"+branchName+"/jobs", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodDelete, "/apps/"+appID+"/branches/"+branchName+"/jobs/"+startedJobID+"/stop", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodDelete, "/apps/"+appID+"/branches/"+branchName+"/jobs/"+startedJobID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/branches/"+branchName+"/jobs/"+jobID+"/artifacts", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/artifacts/artifact-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/domains", []byte(`{"domainName":"stage.example.com"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/domains/stage.example.com", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/domains/stage.example.com", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/domains", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodDelete, "/apps/"+appID+"/domains/stage.example.com", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/webhooks", []byte(`{"branchName":"main"}`))
	assertStatus(t, resp, http.StatusOK)
	createWebhookPayload := decodeAmplifyPayload(t, resp)
	webhook := amplifyPayloadMap(createWebhookPayload, "webhook")
	webhookID := amplifyPayloadString(webhook, "webhookId")
	if webhookID == "" {
		t.Fatalf("expected CreateWebhook to return webhook.webhookId")
	}

	resp = amplifyRequest(t, ts, http.MethodGet, "/webhooks/"+url.PathEscape(webhookID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/webhooks/"+url.PathEscape(webhookID), []byte(`{"description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/apps/"+appID+"/webhooks", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodDelete, "/webhooks/"+url.PathEscape(webhookID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodPost, "/apps/"+appID+"/accesslogs", []byte(`{"domainName":"example.com"}`))
	assertStatus(t, resp, http.StatusOK)
	accessLogsPayload := decodeAmplifyPayload(t, resp)
	if got := amplifyPayloadString(accessLogsPayload, "logUrl"); got == "" {
		t.Fatalf("expected GenerateAccessLogs to return logUrl")
	}
}

func TestAmplifyStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:amplify:us-east-1:123456789012:apps/d1234567890"
	escapedARN := url.PathEscape(resourceARN)

	resp := amplifyRequest(t, ts, http.MethodPost, "/tags/"+escapedARN, []byte(`{"tags":{"env":"dev","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload := decodeAmplifyPayload(t, resp)
	tags := amplifyPayloadMap(listPayload, "tags")
	if got := amplifyPayloadString(tags, "owner"); got != "qa" {
		t.Fatalf("expected owner tag qa, got %q", got)
	}

	resp = amplifyRequest(t, ts, http.MethodDelete, "/tags/"+escapedARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = amplifyRequest(t, ts, http.MethodGet, "/tags/"+escapedARN, nil)
	assertStatus(t, resp, http.StatusOK)
	listPayload = decodeAmplifyPayload(t, resp)
	tags = amplifyPayloadMap(listPayload, "tags")
	if _, exists := tags["owner"]; exists {
		t.Fatalf("expected owner tag removed")
	}
	if got := amplifyPayloadString(tags, "env"); got != "dev" {
		t.Fatalf("expected env tag to remain after untag")
	}

	resp = amplifyRequest(t, ts, http.MethodPost, "/unknown-amplify-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}
}

func decodeAmplifyPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func amplifyPayloadMap(payload map[string]any, key string) map[string]any {
	value, _ := payload[key].(map[string]any)
	return value
}

func amplifyPayloadString(payload map[string]any, key string) string {
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
