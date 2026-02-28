package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage4-cluster"
	const family = "ecs-stage4-task-family"
	const serviceName = "ecs-stage4-service"

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

	actions := []struct {
		name string
		body []byte
	}{
		{name: "CreateService", body: []byte(`{"cluster":"` + clusterName + `","serviceName":"` + serviceName + `","taskDefinition":"` + registerOut.TaskDefinition.TaskDefinitionArn + `","desiredCount":1}`)},
		{name: "DescribeServices", body: []byte(`{"cluster":"` + clusterName + `","services":["` + serviceName + `"]}`)},
		{name: "ListServices", body: []byte(`{"cluster":"` + clusterName + `"}`)},
		{name: "UpdateService", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","desiredCount":2}`)},
		{name: "DeleteService", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}

	resp = ecsRequest(t, ts, "DescribeServices", []byte(`{"cluster":"`+clusterName+`","services":["`+serviceName+`"]}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Failures []struct {
			Reason string `json:"reason"`
		} `json:"failures"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe services response: %v", err)
	}
	if len(describeOut.Failures) != 1 {
		t.Fatalf("expected one failure after delete, got %d", len(describeOut.Failures))
	}
}
