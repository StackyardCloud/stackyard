package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage11ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage11-cluster"
	const family = "ecs-stage11-task-family"
	const serviceName = "ecs-stage11-service"
	const namespace = "ns-stage11"

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
	var registerTaskOut struct {
		TaskDefinition struct {
			TaskDefinitionArn string `json:"taskDefinitionArn"`
		} `json:"taskDefinition"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerTaskOut); err != nil {
		t.Fatalf("unmarshal register task definition response: %v", err)
	}

	resp = ecsRequest(t, ts, "CreateService", []byte(`{
		"cluster":"`+clusterName+`",
		"serviceName":"`+serviceName+`",
		"taskDefinition":"`+registerTaskOut.TaskDefinition.TaskDefinitionArn+`",
		"launchType":"FARGATE",
		"desiredCount":1,
		"tags":[{"key":"namespace","value":"`+namespace+`"}]
	}`))
	assertStatus(t, resp, http.StatusOK)
	var createServiceOut struct {
		Service struct {
			ServiceArn string `json:"serviceArn"`
		} `json:"service"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createServiceOut); err != nil {
		t.Fatalf("unmarshal create service response: %v", err)
	}
	if createServiceOut.Service.ServiceArn == "" {
		t.Fatalf("expected serviceArn from CreateService")
	}
	serviceArn := createServiceOut.Service.ServiceArn

	actions := []struct {
		name string
		body []byte
	}{
		{name: "TagResource", body: []byte(`{"resourceArn":"` + serviceArn + `","tags":[{"key":"env","value":"dev"},{"key":"team","value":"core"}]}`)},
		{name: "ListTagsForResource", body: []byte(`{"resourceArn":"` + serviceArn + `"}`)},
		{name: "UntagResource", body: []byte(`{"resourceArn":"` + serviceArn + `","tagKeys":["team"]}`)},
		{name: "ListServicesByLaunchType", body: []byte(`{"cluster":"` + clusterName + `","launchType":"FARGATE"}`)},
		{name: "ListServicesByNamespace", body: []byte(`{"cluster":"` + clusterName + `","namespace":"` + namespace + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
