package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMWAAServerlessStage12WorkflowLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mwaaServerlessRequest(t, ts, "CreateWorkflow", `{
		"Name":"stage-mwaa-workflow",
		"Description":"stage workflow",
		"RoleArn":"arn:aws:iam::123456789012:role/stackyard-mwaa-serverless-role",
		"DefinitionS3Location":{"Bucket":"stackyard-mwaa-serverless","ObjectKey":"workflows/stage.yaml","VersionId":"1"},
		"EngineVersion":2.10,
		"TriggerMode":"ON_DEMAND"
	}`)
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeMWAAServerlessPayload(t, resp)
	workflowArn := mwaaServerlessStringFromMap(createPayload, "WorkflowArn")
	if workflowArn == "" {
		t.Fatalf("expected CreateWorkflow to return WorkflowArn")
	}

	resp = mwaaServerlessRequest(t, ts, "GetWorkflow", `{"WorkflowArn":"`+workflowArn+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "ListWorkflows", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "UpdateWorkflow", `{
		"WorkflowArn":"`+workflowArn+`",
		"Description":"updated workflow",
		"EngineVersion":2.11
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "ListWorkflowVersions", `{"WorkflowArn":"`+workflowArn+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestMWAAServerlessStage34RunAndTaskSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	workflowArn := "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/stackyard-workflow"

	resp := mwaaServerlessRequest(t, ts, "StartWorkflowRun", `{
		"WorkflowArn":"`+workflowArn+`",
		"WorkflowVersion":"1"
	}`)
	assertStatus(t, resp, http.StatusOK)
	startPayload := decodeMWAAServerlessPayload(t, resp)
	runID := mwaaServerlessStringFromMap(startPayload, "RunId")
	if runID == "" {
		t.Fatalf("expected StartWorkflowRun to return RunId")
	}

	resp = mwaaServerlessRequest(t, ts, "GetWorkflowRun", `{
		"WorkflowArn":"`+workflowArn+`",
		"WorkflowVersion":"1",
		"RunId":"`+runID+`"
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "ListWorkflowRuns", `{
		"WorkflowArn":"`+workflowArn+`",
		"WorkflowVersion":"1"
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "ListTaskInstances", `{
		"WorkflowArn":"`+workflowArn+`",
		"WorkflowVersion":"1",
		"RunId":"`+runID+`"
	}`)
	assertStatus(t, resp, http.StatusOK)
	tasksPayload := decodeMWAAServerlessPayload(t, resp)
	taskInstances := mwaaServerlessArrayFromMap(tasksPayload, "TaskInstances")
	taskID := "task-000001"
	if len(taskInstances) > 0 {
		if first, ok := taskInstances[0].(map[string]any); ok {
			if id := mwaaServerlessStringFromMap(first, "TaskInstanceId"); id != "" {
				taskID = id
			}
		}
	}

	resp = mwaaServerlessRequest(t, ts, "GetTaskInstance", `{
		"WorkflowArn":"`+workflowArn+`",
		"WorkflowVersion":"1",
		"RunId":"`+runID+`",
		"TaskInstanceId":"`+taskID+`"
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "StopWorkflowRun", `{
		"WorkflowArn":"`+workflowArn+`",
		"WorkflowVersion":"1",
		"RunId":"`+runID+`"
	}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestMWAAServerlessStage5TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	workflowArn := "arn:aws:mwaa-serverless:us-east-1:123456789012:workflow/stackyard-workflow"

	resp := mwaaServerlessRequest(t, ts, "TagResource", `{
		"ResourceArn":"`+workflowArn+`",
		"Tags":{"owner":"qa","env":"stage"}
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "ListTagsForResource", `{"ResourceArn":"`+workflowArn+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = mwaaServerlessRequest(t, ts, "UntagResource", `{
		"ResourceArn":"`+workflowArn+`",
		"TagKeys":["owner"]
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = mwaaServerlessRequest(t, ts, "ListTagsForResource", `{"ResourceArn":"`+workflowArn+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = mwaaServerlessRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonMWAAServerless.ListWorkflows",
		},
		"mwaa-serverless",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeMWAAServerlessPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func mwaaServerlessStringFromMap(payload map[string]any, key string) string {
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

func mwaaServerlessArrayFromMap(payload map[string]any, key string) []any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	value, ok := raw.([]any)
	if !ok {
		return nil
	}
	return value
}
