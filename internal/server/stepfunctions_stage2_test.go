package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStepFunctionsASLChoiceExecutionLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	definition := `{
		"StartAt":"Route",
		"States":{
			"Route":{
				"Type":"Choice",
				"Choices":[{"Variable":"$.route","StringEquals":"left","Next":"Left"}],
				"Default":"Right"
			},
			"Left":{"Type":"Pass","Result":{"branch":"left"},"End":true},
			"Right":{"Type":"Pass","Result":{"branch":"right"},"End":true}
		}
	}`

	create := stepFunctionsRequest(t, ts, "CreateStateMachine", `{
		"name":"asl-choice-sm",
		"definition":`+quoteJSON(definition)+`,
		"roleArn":"arn:aws:iam::123456789012:role/stackyard-step-functions-role"
	}`)
	assertStatus(t, create, http.StatusOK)
	createBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, create), &createBody); err != nil {
		t.Fatalf("failed to parse CreateStateMachine response: %v", err)
	}
	stateMachineARN, _ := createBody["stateMachineArn"].(string)
	if stateMachineARN == "" {
		t.Fatalf("expected stateMachineArn in CreateStateMachine response")
	}

	start := stepFunctionsRequest(t, ts, "StartExecution", `{
		"stateMachineArn":`+quoteJSON(stateMachineARN)+`,
		"name":"asl-choice-exec",
		"input":"{\"route\":\"left\"}"
	}`)
	assertStatus(t, start, http.StatusOK)
	startBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, start), &startBody); err != nil {
		t.Fatalf("failed to parse StartExecution response: %v", err)
	}
	execARN, _ := startBody["executionArn"].(string)
	if execARN == "" {
		t.Fatalf("expected executionArn in StartExecution response")
	}

	describe := stepFunctionsRequest(t, ts, "DescribeExecution", `{"executionArn":`+quoteJSON(execARN)+`}`)
	assertStatus(t, describe, http.StatusOK)
	describeBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, describe), &describeBody); err != nil {
		t.Fatalf("failed to parse DescribeExecution response: %v", err)
	}
	if status, _ := describeBody["status"].(string); status != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED execution status, got %v", describeBody["status"])
	}
	outputText, _ := describeBody["output"].(string)
	output := map[string]any{}
	if err := json.Unmarshal([]byte(outputText), &output); err != nil {
		t.Fatalf("failed to parse execution output %q: %v", outputText, err)
	}
	if got, _ := output["branch"].(string); got != "left" {
		t.Fatalf("expected branch=left output, got %v", output)
	}

	history := stepFunctionsRequest(t, ts, "GetExecutionHistory", `{"executionArn":`+quoteJSON(execARN)+`}`)
	assertStatus(t, history, http.StatusOK)
	historyBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, history), &historyBody); err != nil {
		t.Fatalf("failed to parse GetExecutionHistory response: %v", err)
	}
	events, _ := historyBody["events"].([]any)
	if len(events) < 3 {
		t.Fatalf("expected execution history events, got %v", historyBody["events"])
	}
	hasChoice := false
	hasSucceeded := false
	for _, raw := range events {
		event, _ := raw.(map[string]any)
		typ, _ := event["type"].(string)
		if typ == "ChoiceStateEntered" {
			hasChoice = true
		}
		if typ == "ExecutionSucceeded" {
			hasSucceeded = true
		}
	}
	if !hasChoice || !hasSucceeded {
		t.Fatalf("expected ChoiceStateEntered and ExecutionSucceeded in history, got %v", events)
	}
}

func TestStepFunctionsASLStartSyncExecutionFailure(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	definition := `{
		"StartAt":"Boom",
		"States":{
			"Boom":{"Type":"Fail","Error":"MyCustomError","Cause":"simulated failure"}
		}
	}`

	create := stepFunctionsRequest(t, ts, "CreateStateMachine", `{
		"name":"asl-fail-sm",
		"definition":`+quoteJSON(definition)+`,
		"roleArn":"arn:aws:iam::123456789012:role/stackyard-step-functions-role"
	}`)
	assertStatus(t, create, http.StatusOK)
	createBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, create), &createBody); err != nil {
		t.Fatalf("failed to parse CreateStateMachine response: %v", err)
	}
	stateMachineARN, _ := createBody["stateMachineArn"].(string)
	if stateMachineARN == "" {
		t.Fatalf("expected stateMachineArn in CreateStateMachine response")
	}

	startSync := stepFunctionsRequest(t, ts, "StartSyncExecution", `{
		"stateMachineArn":`+quoteJSON(stateMachineARN)+`,
		"name":"asl-fail-sync",
		"input":"{}"
	}`)
	assertStatus(t, startSync, http.StatusOK)
	respBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, startSync), &respBody); err != nil {
		t.Fatalf("failed to parse StartSyncExecution response: %v", err)
	}
	if status, _ := respBody["status"].(string); status != "FAILED" {
		t.Fatalf("expected FAILED sync execution status, got %v", respBody["status"])
	}
	if errCode, _ := respBody["error"].(string); errCode != "MyCustomError" {
		t.Fatalf("expected MyCustomError in sync response, got %v", respBody["error"])
	}
	if cause, _ := respBody["cause"].(string); cause != "simulated failure" {
		t.Fatalf("expected failure cause in sync response, got %v", respBody["cause"])
	}
}

func quoteJSON(s string) string {
	encoded, _ := json.Marshal(s)
	return string(encoded)
}
