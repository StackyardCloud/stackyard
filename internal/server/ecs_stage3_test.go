package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const family = "ecs-stage3-task-family"

	registerBody := []byte(`{
		"family":"` + family + `",
		"networkMode":"awsvpc",
		"cpu":"256",
		"memory":"512",
		"requiresCompatibilities":["FARGATE"],
		"containerDefinitions":[{"name":"app","image":"public.ecr.aws/docker/library/busybox:latest"}],
		"tags":[{"key":"env","value":"test"}]
	}`)

	resp := ecsRequest(t, ts, "RegisterTaskDefinition", registerBody)
	assertStatus(t, resp, http.StatusOK)
	var registerOut1 struct {
		TaskDefinition struct {
			TaskDefinitionArn string `json:"taskDefinitionArn"`
		} `json:"taskDefinition"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerOut1); err != nil {
		t.Fatalf("unmarshal register task definition response: %v", err)
	}

	resp = ecsRequest(t, ts, "RegisterTaskDefinition", registerBody)
	assertStatus(t, resp, http.StatusOK)
	var registerOut2 struct {
		TaskDefinition struct {
			TaskDefinitionArn string `json:"taskDefinitionArn"`
		} `json:"taskDefinition"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerOut2); err != nil {
		t.Fatalf("unmarshal second register task definition response: %v", err)
	}

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeTaskDefinition", body: []byte(`{"taskDefinition":"` + registerOut2.TaskDefinition.TaskDefinitionArn + `"}`)},
		{name: "ListTaskDefinitions", body: []byte(`{"familyPrefix":"` + family + `"}`)},
		{name: "ListTaskDefinitionFamilies", body: []byte(`{"familyPrefix":"` + family + `"}`)},
		{name: "DeregisterTaskDefinition", body: []byte(`{"taskDefinition":"` + registerOut2.TaskDefinition.TaskDefinitionArn + `"}`)},
		{name: "DeleteTaskDefinitions", body: []byte(`{"taskDefinitions":["` + registerOut1.TaskDefinition.TaskDefinitionArn + `","` + registerOut2.TaskDefinition.TaskDefinitionArn + `"]}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
