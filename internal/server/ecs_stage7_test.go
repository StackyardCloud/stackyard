package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage7ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage7-cluster"
	const family = "ecs-stage7-task-family"

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

	resp = ecsRequest(t, ts, "RunTask", []byte(`{"cluster":"`+clusterName+`","taskDefinition":"`+registerOut.TaskDefinition.TaskDefinitionArn+`","count":2,"startedBy":"stage7"}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("action %s returned not implemented", "RunTask")
	}
	assertStatus(t, resp, http.StatusOK)
	var runTaskOut struct {
		Tasks []struct {
			TaskArn string `json:"taskArn"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &runTaskOut); err != nil {
		t.Fatalf("unmarshal run task response: %v", err)
	}
	if len(runTaskOut.Tasks) == 0 {
		t.Fatalf("expected run task to create tasks")
	}

	actions := []struct {
		name string
		body []byte
	}{
		{name: "StartTask", body: []byte(`{"cluster":"` + clusterName + `","taskDefinition":"` + registerOut.TaskDefinition.TaskDefinitionArn + `","containerInstances":["arn:aws:ecs:us-east-1:123456789012:container-instance/demo"]}`)},
		{name: "ListTasks", body: []byte(`{"cluster":"` + clusterName + `","desiredStatus":"RUNNING"}`)},
		{name: "DescribeTasks", body: []byte(`{"cluster":"` + clusterName + `","tasks":["` + runTaskOut.Tasks[0].TaskArn + `","missing-task"]}`)},
		{name: "StopTask", body: []byte(`{"cluster":"` + clusterName + `","task":"` + runTaskOut.Tasks[0].TaskArn + `","reason":"test stop"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
