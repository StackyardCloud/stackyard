package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStepFunctionsASLDataFlowPathsAndParameters(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	definition := `{
		"StartAt":"Prepare",
		"States":{
			"Prepare":{
				"Type":"Pass",
				"InputPath":"$.payload",
				"Parameters":{"copied.$":"$.value","kind":"fixed"},
				"Next":"Work"
			},
			"Work":{
				"Type":"Task",
				"Result":{"status":"ok","answer":42},
				"ResultPath":"$.taskResult",
				"OutputPath":"$.taskResult",
				"End":true
			}
		}
	}`

	create := stepFunctionsRequest(t, ts, "CreateStateMachine", `{
		"name":"asl-dataflow-sm",
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
		"name":"asl-dataflow-exec",
		"input":"{\"payload\":{\"value\":\"hello\",\"count\":3},\"ignored\":true}"
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
	if got, _ := output["status"].(string); got != "ok" {
		t.Fatalf("expected status=ok output, got %v", output)
	}
	if got, _ := output["answer"].(float64); got != 42 {
		t.Fatalf("expected answer=42 output, got %v", output)
	}

	history := stepFunctionsRequest(t, ts, "GetExecutionHistory", `{"executionArn":`+quoteJSON(execARN)+`}`)
	assertStatus(t, history, http.StatusOK)
	historyBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, history), &historyBody); err != nil {
		t.Fatalf("failed to parse GetExecutionHistory response: %v", err)
	}
	events, _ := historyBody["events"].([]any)
	if len(events) == 0 {
		t.Fatalf("expected execution history events")
	}
	foundTaskEntered := false
	for _, raw := range events {
		event, _ := raw.(map[string]any)
		if typ, _ := event["type"].(string); typ == "TaskStateEntered" {
			details, _ := event["stateEnteredEventDetails"].(map[string]any)
			stateInputText, _ := details["input"].(string)
			stateInput := map[string]any{}
			if err := json.Unmarshal([]byte(stateInputText), &stateInput); err != nil {
				t.Fatalf("failed to parse task state input %q: %v", stateInputText, err)
			}
			if got, _ := stateInput["copied"].(string); got != "hello" {
				t.Fatalf("expected copied=hello in TaskStateEntered input, got %v", stateInput)
			}
			if got, _ := stateInput["kind"].(string); got != "fixed" {
				t.Fatalf("expected kind=fixed in TaskStateEntered input, got %v", stateInput)
			}
			foundTaskEntered = true
			break
		}
	}
	if !foundTaskEntered {
		t.Fatalf("expected TaskStateEntered event in history")
	}
}

func TestStepFunctionsASLMapAndParallelExecution(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	mapDefinition := `{
		"StartAt":"FanOut",
		"States":{
			"FanOut":{
				"Type":"Map",
				"ItemsPath":"$.items",
				"Iterator":{
					"StartAt":"Project",
					"States":{
						"Project":{"Type":"Pass","Parameters":{"item.$":"$.v"},"End":true}
					}
				},
				"End":true
			}
		}
	}`

	mapCreate := stepFunctionsRequest(t, ts, "CreateStateMachine", `{
		"name":"asl-map-sm",
		"definition":`+quoteJSON(mapDefinition)+`,
		"roleArn":"arn:aws:iam::123456789012:role/stackyard-step-functions-role"
	}`)
	assertStatus(t, mapCreate, http.StatusOK)
	mapCreateBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, mapCreate), &mapCreateBody); err != nil {
		t.Fatalf("failed to parse map CreateStateMachine response: %v", err)
	}
	mapStateMachineARN, _ := mapCreateBody["stateMachineArn"].(string)
	if mapStateMachineARN == "" {
		t.Fatalf("expected map stateMachineArn in CreateStateMachine response")
	}

	mapSync := stepFunctionsRequest(t, ts, "StartSyncExecution", `{
		"stateMachineArn":`+quoteJSON(mapStateMachineARN)+`,
		"name":"asl-map-sync",
		"input":"{\"items\":[{\"v\":\"a\"},{\"v\":\"b\"}]}"
	}`)
	assertStatus(t, mapSync, http.StatusOK)
	mapSyncBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, mapSync), &mapSyncBody); err != nil {
		t.Fatalf("failed to parse map StartSyncExecution response: %v", err)
	}
	if status, _ := mapSyncBody["status"].(string); status != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED map sync execution status, got %v", mapSyncBody["status"])
	}
	mapOutputText, _ := mapSyncBody["output"].(string)
	mapOutput := []map[string]any{}
	if err := json.Unmarshal([]byte(mapOutputText), &mapOutput); err != nil {
		t.Fatalf("failed to parse map execution output %q: %v", mapOutputText, err)
	}
	if len(mapOutput) != 2 {
		t.Fatalf("expected 2 map output items, got %v", mapOutput)
	}
	if got, _ := mapOutput[0]["item"].(string); got != "a" {
		t.Fatalf("expected first map item=a, got %v", mapOutput)
	}
	if got, _ := mapOutput[1]["item"].(string); got != "b" {
		t.Fatalf("expected second map item=b, got %v", mapOutput)
	}

	parallelDefinition := `{
		"StartAt":"FanOut",
		"States":{
			"FanOut":{
				"Type":"Parallel",
				"Branches":[
					{"StartAt":"Left","States":{"Left":{"Type":"Pass","Result":{"branch":"left"},"End":true}}},
					{"StartAt":"Right","States":{"Right":{"Type":"Pass","Result":{"branch":"right"},"End":true}}}
				],
				"ResultPath":"$.parallel",
				"End":true
			}
		}
	}`

	parallelCreate := stepFunctionsRequest(t, ts, "CreateStateMachine", `{
		"name":"asl-parallel-sm",
		"definition":`+quoteJSON(parallelDefinition)+`,
		"roleArn":"arn:aws:iam::123456789012:role/stackyard-step-functions-role"
	}`)
	assertStatus(t, parallelCreate, http.StatusOK)
	parallelCreateBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, parallelCreate), &parallelCreateBody); err != nil {
		t.Fatalf("failed to parse parallel CreateStateMachine response: %v", err)
	}
	parallelStateMachineARN, _ := parallelCreateBody["stateMachineArn"].(string)
	if parallelStateMachineARN == "" {
		t.Fatalf("expected parallel stateMachineArn in CreateStateMachine response")
	}

	parallelSync := stepFunctionsRequest(t, ts, "StartSyncExecution", `{
		"stateMachineArn":`+quoteJSON(parallelStateMachineARN)+`,
		"name":"asl-parallel-sync",
		"input":"{\"seed\":\"x\"}"
	}`)
	assertStatus(t, parallelSync, http.StatusOK)
	parallelSyncBody := map[string]any{}
	if err := json.Unmarshal(mustBody(t, parallelSync), &parallelSyncBody); err != nil {
		t.Fatalf("failed to parse parallel StartSyncExecution response: %v", err)
	}
	if status, _ := parallelSyncBody["status"].(string); status != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED parallel sync execution status, got %v", parallelSyncBody["status"])
	}
	parallelOutputText, _ := parallelSyncBody["output"].(string)
	parallelOutput := map[string]any{}
	if err := json.Unmarshal([]byte(parallelOutputText), &parallelOutput); err != nil {
		t.Fatalf("failed to parse parallel execution output %q: %v", parallelOutputText, err)
	}
	if seed, _ := parallelOutput["seed"].(string); seed != "x" {
		t.Fatalf("expected seed=x in parallel output, got %v", parallelOutput)
	}
	branchResults, _ := parallelOutput["parallel"].([]any)
	if len(branchResults) != 2 {
		t.Fatalf("expected 2 parallel branch results, got %v", parallelOutput)
	}
	first, _ := branchResults[0].(map[string]any)
	second, _ := branchResults[1].(map[string]any)
	if got, _ := first["branch"].(string); got != "left" {
		t.Fatalf("expected first branch result=left, got %v", branchResults)
	}
	if got, _ := second["branch"].(string); got != "right" {
		t.Fatalf("expected second branch result=right, got %v", branchResults)
	}
}
