package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNovaActStage12WorkflowDefinitionAndRunLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	workflowName := "stage-workflow-001"
	workflowRunID := "workflow-run-101"

	resp := novaActRequest(t, ts, http.MethodPut, "/workflow-definitions", []byte(`{"workflowDefinitionName":"`+workflowName+`","description":"stage workflow"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodGet, "/workflow-definitions/"+workflowName, nil)
	assertStatus(t, resp, http.StatusOK)
	getDefinitionPayload := decodeNovaActPayload(t, resp)
	workflowDefinition, _ := getDefinitionPayload["workflowDefinition"].(map[string]any)
	if got := novaActPayloadString(workflowDefinition, "workflowDefinitionName"); got != workflowName {
		t.Fatalf("expected workflowDefinitionName %q, got %q", workflowName, got)
	}

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs", []byte(`{"workflowRunId":"`+workflowRunID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodGet, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID, nil)
	assertStatus(t, resp, http.StatusOK)
	getRunPayload := decodeNovaActPayload(t, resp)
	workflowRun, _ := getRunPayload["workflowRun"].(map[string]any)
	if got := novaActPayloadString(workflowRun, "workflowRunId"); got != workflowRunID {
		t.Fatalf("expected workflowRunId %q, got %q", workflowRunID, got)
	}

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID, []byte(`{"status":"SUCCEEDED"}`))
	assertStatus(t, resp, http.StatusOK)
	updateRunPayload := decodeNovaActPayload(t, resp)
	updatedRun, _ := updateRunPayload["workflowRun"].(map[string]any)
	if got := novaActPayloadString(updatedRun, "status"); got != "SUCCEEDED" {
		t.Fatalf("expected updated run status SUCCEEDED, got %q", got)
	}

	resp = novaActRequest(t, ts, http.MethodPost, "/workflow-definitions?maxResults=10", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	definitionListPayload := decodeNovaActPayload(t, resp)
	if _, ok := definitionListPayload["workflowDefinitionSummaries"].([]any); !ok {
		t.Fatalf("expected workflowDefinitionSummaries in ListWorkflowDefinitions response")
	}

	resp = novaActRequest(t, ts, http.MethodPost, "/workflow-definitions/"+workflowName+"/workflow-runs?maxResults=10", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	runListPayload := decodeNovaActPayload(t, resp)
	if _, ok := runListPayload["workflowRunSummaries"].([]any); !ok {
		t.Fatalf("expected workflowRunSummaries in ListWorkflowRuns response")
	}

	resp = novaActRequest(t, ts, http.MethodDelete, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodDelete, "/workflow-definitions/"+workflowName, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestNovaActStage345SessionActInvokeAndModelSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	workflowName := "stage-workflow-002"
	workflowRunID := "workflow-run-202"
	sessionID := "session-303"
	actID := "act-404"

	resp := novaActRequest(t, ts, http.MethodPut, "/workflow-definitions", []byte(`{"workflowDefinitionName":"`+workflowName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs", []byte(`{"workflowRunId":"`+workflowRunID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID+"/sessions", []byte(`{"sessionId":"`+sessionID+`"}`))
	assertStatus(t, resp, http.StatusOK)
	sessionPayload := decodeNovaActPayload(t, resp)
	session, _ := sessionPayload["session"].(map[string]any)
	if got := novaActPayloadString(session, "sessionId"); got != sessionID {
		t.Fatalf("expected sessionId %q, got %q", sessionID, got)
	}

	resp = novaActRequest(t, ts, http.MethodPost, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID+"?maxResults=10", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listSessionsPayload := decodeNovaActPayload(t, resp)
	if _, ok := listSessionsPayload["sessionSummaries"].([]any); !ok {
		t.Fatalf("expected sessionSummaries in ListSessions response")
	}

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID+"/sessions/"+sessionID+"/acts", []byte(`{"actId":"`+actID+`","name":"stage-act"}`))
	assertStatus(t, resp, http.StatusOK)
	createActPayload := decodeNovaActPayload(t, resp)
	act, _ := createActPayload["act"].(map[string]any)
	if got := novaActPayloadString(act, "actId"); got != actID {
		t.Fatalf("expected actId %q, got %q", actID, got)
	}

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID+"/sessions/"+sessionID+"/acts/"+actID, []byte(`{"status":"COMPLETED"}`))
	assertStatus(t, resp, http.StatusOK)
	updateActPayload := decodeNovaActPayload(t, resp)
	updatedAct, _ := updateActPayload["act"].(map[string]any)
	if got := novaActPayloadString(updatedAct, "status"); got != "COMPLETED" {
		t.Fatalf("expected updated act status COMPLETED, got %q", got)
	}

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs/"+workflowRunID+"/sessions/"+sessionID+"/acts/"+actID+"/invoke-step/", []byte(`{"step":{"name":"stage-step"}}`))
	assertStatus(t, resp, http.StatusOK)
	invokePayload := decodeNovaActPayload(t, resp)
	if got := novaActPayloadString(invokePayload, "status"); got != "SUCCEEDED" {
		t.Fatalf("expected invoke status SUCCEEDED, got %q", got)
	}
	if _, ok := invokePayload["traceLocation"].(map[string]any); !ok {
		t.Fatalf("expected traceLocation map in InvokeActStep response")
	}

	resp = novaActRequest(t, ts, http.MethodPost, "/workflow-definitions/"+workflowName+"/acts?maxResults=10&workflowRunId="+workflowRunID+"&sessionId="+sessionID, []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listActsPayload := decodeNovaActPayload(t, resp)
	if _, ok := listActsPayload["actSummaries"].([]any); !ok {
		t.Fatalf("expected actSummaries in ListActs response")
	}

	resp = novaActRequest(t, ts, http.MethodPost, "/models?clientCompatibilityVersion=1.0", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	modelsPayload := decodeNovaActPayload(t, resp)
	if _, ok := modelsPayload["models"].([]any); !ok {
		t.Fatalf("expected models list in ListModels response")
	}
	if _, ok := modelsPayload["compatibilityInformation"].(map[string]any); !ok {
		t.Fatalf("expected compatibilityInformation map in ListModels response")
	}
}

func TestNovaActStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	workflowName := "stage-workflow-idempotent"
	workflowRunID := "workflow-run-idempotent"

	resp := novaActRequest(t, ts, http.MethodPut, "/workflow-definitions", []byte(`{"workflowDefinitionName":"`+workflowName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions", []byte(`{"workflowDefinitionName":"`+workflowName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs", []byte(`{"workflowRunId":"`+workflowRunID+`"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions/"+workflowName+"/workflow-runs", []byte(`{"workflowRunId":"`+workflowRunID+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = novaActRequest(t, ts, http.MethodPost, "/workflow-definitions?maxResults=10", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listDefinitionsPayload := decodeNovaActPayload(t, resp)
	definitions, _ := listDefinitionsPayload["workflowDefinitionSummaries"].([]any)
	matches := 0
	for _, item := range definitions {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if novaActPayloadString(entry, "workflowDefinitionName") == workflowName {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("expected idempotent workflow definition create to keep exactly one definition, got %d", matches)
	}

	resp = novaActRequest(t, ts, http.MethodPost, "/workflow-definitions/"+workflowName+"/workflow-runs?maxResults=10", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listRunsPayload := decodeNovaActPayload(t, resp)
	runs, _ := listRunsPayload["workflowRunSummaries"].([]any)
	runMatches := 0
	for _, item := range runs {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if novaActPayloadString(entry, "workflowRunId") == workflowRunID {
			runMatches++
		}
	}
	if runMatches != 1 {
		t.Fatalf("expected idempotent workflow run create to keep exactly one run, got %d", runMatches)
	}

	resp = novaActRequest(t, ts, http.MethodPut, "/workflow-definitions", []byte(`{"workflowDefinitionName":"invalid-json"`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for invalid JSON body, got %q", body)
	}
}

func decodeNovaActPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func novaActPayloadString(payload map[string]any, key string) string {
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
