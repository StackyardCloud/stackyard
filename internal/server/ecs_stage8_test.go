package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage8ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage8-cluster"
	const family = "ecs-stage8-task-family"

	resp := ecsRequest(t, ts, "CreateCluster", []byte(`{"clusterName":"`+clusterName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "RegisterTaskDefinition", []byte(`{
		"family":"`+family+`",
		"networkMode":"awsvpc",
		"cpu":"256",
		"memory":"512",
		"requiresCompatibilities":["FARGATE"],
		"containerDefinitions":[{"name":"app","image":"public.ecr.aws/docker/library/busybox:latest"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	var registerOut struct {
		TaskDefinition struct {
			TaskDefinitionArn string `json:"taskDefinitionArn"`
		} `json:"taskDefinition"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerOut); err != nil {
		t.Fatalf("unmarshal register task definition response: %v", err)
	}

	resp = ecsRequest(t, ts, "RunTask", []byte(`{"cluster":"`+clusterName+`","taskDefinition":"`+registerOut.TaskDefinition.TaskDefinitionArn+`","count":1}`))
	assertStatus(t, resp, http.StatusOK)
	var runOut struct {
		Tasks []struct {
			TaskArn string `json:"taskArn"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &runOut); err != nil {
		t.Fatalf("unmarshal run task response: %v", err)
	}
	if len(runOut.Tasks) != 1 {
		t.Fatalf("expected one task from run task, got %d", len(runOut.Tasks))
	}
	taskArn := runOut.Tasks[0].TaskArn

	actions := []struct {
		name string
		body []byte
	}{
		{name: "ExecuteCommand", body: []byte(`{"cluster":"` + clusterName + `","task":"` + taskArn + `","container":"app","command":"echo hi","interactive":true}`)},
		{name: "GetTaskProtection", body: []byte(`{"cluster":"` + clusterName + `","tasks":["` + taskArn + `"]}`)},
		{name: "UpdateTaskProtection", body: []byte(`{"cluster":"` + clusterName + `","tasks":["` + taskArn + `"],"protectionEnabled":true,"expiresInMinutes":30}`)},
		{name: "DiscoverPollEndpoint", body: []byte(`{"cluster":"` + clusterName + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
