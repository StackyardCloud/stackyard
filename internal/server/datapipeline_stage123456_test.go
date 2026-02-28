package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDataPipelineStage12PipelineLifecycleAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataPipelineRequest(t, ts, "CreatePipeline", `{"name":"stage-pipeline","uniqueId":"stage-unique-001"}`)
	assertStatus(t, resp, http.StatusOK)
	body := mustBody(t, resp)

	var createOut map[string]any
	if err := json.Unmarshal(body, &createOut); err != nil {
		t.Fatalf("decode CreatePipeline response: %v", err)
	}
	pipelineID, _ := createOut["pipelineId"].(string)
	if strings.TrimSpace(pipelineID) == "" {
		t.Fatalf("expected pipelineId in CreatePipeline response, got %q", string(body))
	}

	resp = dataPipelineRequest(t, ts, "AddTags", `{"pipelineId":"`+pipelineID+`","tags":[{"key":"env","value":"stage"},{"key":"owner","value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = dataPipelineRequest(t, ts, "DescribePipelines", `{"pipelineIds":["`+pipelineID+`"]}`)
	assertStatus(t, resp, http.StatusOK)
	if out := string(mustBody(t, resp)); !strings.Contains(out, "pipelineDescriptionList") {
		t.Fatalf("expected DescribePipelines output to include pipelineDescriptionList, got %q", out)
	}

	resp = dataPipelineRequest(t, ts, "ActivatePipeline", `{"pipelineId":"`+pipelineID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = dataPipelineRequest(t, ts, "DeactivatePipeline", `{"pipelineId":"`+pipelineID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = dataPipelineRequest(t, ts, "RemoveTags", `{"pipelineId":"`+pipelineID+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDataPipelineStage34DefinitionObjectAndTaskRunnerSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataPipelineRequest(t, ts, "PutPipelineDefinition", `{
		"pipelineId":"df-000001",
		"pipelineObjects":[
			{"id":"DefaultActivity","name":"DefaultActivity","fields":[{"key":"type","stringValue":"ShellCommandActivity"}]},
			{"id":"DefaultSchedule","name":"DefaultSchedule","fields":[{"key":"type","stringValue":"Schedule"}]}
		],
		"parameterObjects":[{"id":"myParam","attributes":[{"key":"type","stringValue":"String"}]}],
		"parameterValues":[{"id":"myParam","stringValue":"value"}]
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = dataPipelineRequest(t, ts, "ValidatePipelineDefinition", `{"pipelineId":"df-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = dataPipelineRequest(t, ts, "GetPipelineDefinition", `{"pipelineId":"df-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if out := string(mustBody(t, resp)); !strings.Contains(out, "pipelineObjects") {
		t.Fatalf("expected GetPipelineDefinition output to include pipelineObjects, got %q", out)
	}

	resp = dataPipelineRequest(t, ts, "QueryObjects", `{"pipelineId":"df-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if out := string(mustBody(t, resp)); !strings.Contains(out, "ids") {
		t.Fatalf("expected QueryObjects output to include ids, got %q", out)
	}

	resp = dataPipelineRequest(t, ts, "DescribeObjects", `{"pipelineId":"df-000001","objectIds":["DefaultActivity"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = dataPipelineRequest(t, ts, "EvaluateExpression", `{"pipelineId":"df-000001","expression":"#{@scheduledStartTime}"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = dataPipelineRequest(t, ts, "PollForTask", `{"workerGroup":"default-worker-group","hostname":"stage-host","instanceIdentity":{"document":"abc","signature":"def"}}`)
	assertStatus(t, resp, http.StatusOK)
	if out := string(mustBody(t, resp)); !strings.Contains(out, "taskObject") {
		t.Fatalf("expected PollForTask output to include taskObject, got %q", out)
	}

	resp = dataPipelineRequest(t, ts, "ReportTaskRunnerHeartbeat", `{"pipelineId":"df-000001","taskId":"task-000001","workerGroup":"default-worker-group"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = dataPipelineRequest(t, ts, "ReportTaskProgress", `{"pipelineId":"df-000001","taskId":"task-000001","fields":[{"key":"percentComplete","stringValue":"50"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = dataPipelineRequest(t, ts, "SetTaskStatus", `{"pipelineId":"df-000001","taskId":"task-000001","taskStatus":"FINISHED"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = dataPipelineRequest(t, ts, "SetStatus", `{"pipelineId":"df-000001","objectIds":["DefaultActivity"],"status":"FINISHED"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDataPipelineStage56ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataPipelineRequest(t, ts, "TotallyUnknownAction", `{}`)
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
			"X-Amz-Target": "DataPipeline.ListPipelines",
		},
		"datapipeline",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	resp = dataPipelineRequest(t, ts, "CreatePipeline", `{"name":"stage-idempotent","uniqueId":"stage-idempotent-unique"}`)
	assertStatus(t, resp, http.StatusOK)
	body1 := mustBody(t, resp)

	resp = dataPipelineRequest(t, ts, "CreatePipeline", `{"name":"stage-idempotent","uniqueId":"stage-idempotent-unique"}`)
	assertStatus(t, resp, http.StatusOK)
	body2 := mustBody(t, resp)

	var out1, out2 map[string]any
	if err := json.Unmarshal(body1, &out1); err != nil {
		t.Fatalf("decode first CreatePipeline response: %v", err)
	}
	if err := json.Unmarshal(body2, &out2); err != nil {
		t.Fatalf("decode second CreatePipeline response: %v", err)
	}
	if out1["pipelineId"] != out2["pipelineId"] {
		t.Fatalf("expected idempotent CreatePipeline to return same pipelineId, got %v and %v", out1["pipelineId"], out2["pipelineId"])
	}

	resp = dataPipelineRequest(t, ts, "DeletePipeline", `{"pipelineId":"`+out1["pipelineId"].(string)+`"}`)
	assertStatus(t, resp, http.StatusOK)
}
