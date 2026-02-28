package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCodeCatalystStage1SpaceAndProjectLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPatch, "/v1/spaces/stackyard-space", []byte(`{"description":"stage1-updated-space"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPut, "/v1/spaces/stackyard-space/projects", []byte(`{"name":"stage1-project","description":"stage1-description"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space/projects/stage1-project", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPatch, "/v1/spaces/stackyard-space/projects/stage1-project", []byte(`{"description":"stage1-updated-project"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces/stackyard-space/projects", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodDelete, "/v1/spaces/stackyard-space/projects/stage1-project", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodDelete, "/v1/spaces/stackyard-space", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestCodeCatalystStage2RepositoryAndBranchLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories/stage2-repo",
		[]byte(`{"description":"stage2-repo"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories/stage2-repo", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories/stage2-repo/branches/feature-1",
		[]byte(`{}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories/stage2-repo/branches",
		[]byte(`{"maxResults":10}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(
		t,
		ts,
		http.MethodGet,
		"/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories/stage2-repo/cloneUrls",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodDelete, "/v1/spaces/stackyard-space/projects/stackyard-project/sourceRepositories/stage2-repo", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestCodeCatalystStage3DevEnvironmentLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments",
		[]byte(`{"alias":"stage3-dev-env"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeCodeCatalystPayload(t, resp)
	devEnvID := codeCatalystPayloadString(createPayload, "id")
	if devEnvID == "" {
		t.Fatalf("expected CreateDevEnvironment to return id")
	}

	escapedDevEnvID := url.PathEscape(devEnvID)
	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/"+escapedDevEnvID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(
		t,
		ts,
		http.MethodPatch,
		"/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/"+escapedDevEnvID,
		[]byte(`{"alias":"stage3-dev-env-updated"}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPut, "/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/"+escapedDevEnvID+"/start", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPut, "/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/"+escapedDevEnvID+"/stop", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces/stackyard-space/devEnvironments", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodDelete, "/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/"+escapedDevEnvID, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestCodeCatalystStage4SessionLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(
		t,
		ts,
		http.MethodPut,
		"/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/dev-env-000001/session",
		[]byte(`{"sessionType":"SSH"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeCodeCatalystPayload(t, resp)
	sessionID := codeCatalystPayloadString(startPayload, "id")
	if sessionID == "" {
		t.Fatalf("expected StartDevEnvironmentSession to return id")
	}

	resp = codeCatalystRequest(
		t,
		ts,
		http.MethodPost,
		"/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/dev-env-000001/sessions",
		[]byte(`{"maxResults":10}`),
	)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(
		t,
		ts,
		http.MethodDelete,
		"/v1/spaces/stackyard-space/projects/stackyard-project/devEnvironments/dev-env-000001/session/"+url.PathEscape(sessionID),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
}

func TestCodeCatalystStage5WorkflowAndEvents(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces/stackyard-space/projects/stackyard-project/workflows", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
	workflowList := decodeCodeCatalystPayload(t, resp)
	items, ok := workflowList["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected ListWorkflows to return items")
	}

	workflow, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first workflow item to be an object")
	}
	workflowID := codeCatalystPayloadString(workflow, "id")
	if workflowID == "" {
		workflowID = "workflow-000001"
	}

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space/projects/stackyard-project/workflows/"+url.PathEscape(workflowID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPut, "/v1/spaces/stackyard-space/projects/stackyard-project/workflowRuns", []byte(`{"workflowId":"`+workflowID+`"}`))
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeCodeCatalystPayload(t, resp)
	workflowRunID := codeCatalystPayloadString(startPayload, "id")
	if workflowRunID == "" {
		t.Fatalf("expected StartWorkflowRun to return id")
	}

	resp = codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces/stackyard-space/projects/stackyard-project/workflowRuns", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space/projects/stackyard-project/workflowRuns/"+url.PathEscape(workflowRunID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces/stackyard-space/eventLogs", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestCodeCatalystStage6AccessTokenIdentityAndValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(t, ts, http.MethodPut, "/v1/accessTokens", []byte(`{"name":"stage6-token"}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeCodeCatalystPayload(t, resp)
	accessTokenID := codeCatalystPayloadString(createPayload, "id")
	if accessTokenID == "" {
		t.Fatalf("expected CreateAccessToken to return id")
	}

	resp = codeCatalystRequest(t, ts, http.MethodPost, "/v1/accessTokens", []byte(`{"maxResults":10}`))
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodDelete, "/v1/accessTokens/"+url.PathEscape(accessTokenID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = codeCatalystRequest(t, ts, http.MethodDelete, "/v1/accessTokens/"+url.PathEscape(accessTokenID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/v1/spaces/stackyard-space/subscription", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/userDetails", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/session", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = codeCatalystRequest(t, ts, http.MethodGet, "/unknown-codecatalyst-route", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPut,
		ts.URL+"/v1/accessTokens",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"codecatalyst",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeCodeCatalystPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func codeCatalystPayloadString(payload map[string]any, key string) string {
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
