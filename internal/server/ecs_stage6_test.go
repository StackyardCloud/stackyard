package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage6-cluster"
	const family = "ecs-stage6-task-family"
	const serviceName = "ecs-stage6-service"

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

	resp = ecsRequest(t, ts, "CreateService", []byte(`{"cluster":"`+clusterName+`","serviceName":"`+serviceName+`","taskDefinition":"`+registerOut.TaskDefinition.TaskDefinitionArn+`","desiredCount":1}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "CreateTaskSet", []byte(`{"cluster":"`+clusterName+`","service":"`+serviceName+`","taskDefinition":"`+registerOut.TaskDefinition.TaskDefinitionArn+`","scale":{"value":80}}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("action %s returned not implemented", "CreateTaskSet")
	}
	assertStatus(t, resp, http.StatusOK)
	var createTaskSetOut struct {
		TaskSet struct {
			TaskSetArn string `json:"taskSetArn"`
		} `json:"taskSet"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createTaskSetOut); err != nil {
		t.Fatalf("unmarshal create task set response: %v", err)
	}

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeTaskSets", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","taskSets":["` + createTaskSetOut.TaskSet.TaskSetArn + `"]}`)},
		{name: "UpdateTaskSet", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","taskSet":"` + createTaskSetOut.TaskSet.TaskSetArn + `","scale":{"value":60}}`)},
		{name: "UpdateServicePrimaryTaskSet", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","primaryTaskSet":"` + createTaskSetOut.TaskSet.TaskSetArn + `"}`)},
		{name: "DeleteTaskSet", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","taskSet":"` + createTaskSetOut.TaskSet.TaskSetArn + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}

	resp = ecsRequest(t, ts, "DescribeTaskSets", []byte(`{"cluster":"`+clusterName+`","service":"`+serviceName+`","taskSets":["`+createTaskSetOut.TaskSet.TaskSetArn+`"]}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Failures []struct {
			Reason string `json:"reason"`
		} `json:"failures"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe task sets response: %v", err)
	}
	if len(describeOut.Failures) != 1 {
		t.Fatalf("expected one failure after delete, got %d", len(describeOut.Failures))
	}
}
