package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage5ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage5-cluster"
	const family = "ecs-stage5-task-family"
	const serviceName = "ecs-stage5-service"

	resp := ecsRequest(t, ts, "CreateCluster", []byte(`{"clusterName":"`+clusterName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	registerTaskDefinition := func(image string) string {
		t.Helper()
		resp := ecsRequest(t, ts, "RegisterTaskDefinition", []byte(`{
			"family":"`+family+`",
			"networkMode":"awsvpc",
			"cpu":"256",
			"memory":"512",
			"requiresCompatibilities":["FARGATE"],
			"containerDefinitions":[{"name":"app","image":"`+image+`"}]
		}`))
		assertStatus(t, resp, http.StatusOK)
		var out struct {
			TaskDefinition struct {
				TaskDefinitionArn string `json:"taskDefinitionArn"`
			} `json:"taskDefinition"`
		}
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
			t.Fatalf("unmarshal register task definition response: %v", err)
		}
		return out.TaskDefinition.TaskDefinitionArn
	}

	td1 := registerTaskDefinition("public.ecr.aws/docker/library/busybox:latest")
	td2 := registerTaskDefinition("public.ecr.aws/docker/library/alpine:latest")

	resp = ecsRequest(t, ts, "CreateService", []byte(`{"cluster":"`+clusterName+`","serviceName":"`+serviceName+`","taskDefinition":"`+td1+`","desiredCount":1}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "UpdateService", []byte(`{"cluster":"`+clusterName+`","service":"`+serviceName+`","taskDefinition":"`+td2+`","desiredCount":2}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "UpdateService", []byte(`{"cluster":"`+clusterName+`","service":"`+serviceName+`","desiredCount":3}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "DescribeServiceRevisions", []byte(`{"cluster":"`+clusterName+`","service":"`+serviceName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var revisionOut struct {
		ServiceRevisions []struct {
			ServiceRevisionArn string `json:"serviceRevisionArn"`
		} `json:"serviceRevisions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &revisionOut); err != nil {
		t.Fatalf("unmarshal describe service revisions response: %v", err)
	}
	if len(revisionOut.ServiceRevisions) == 0 {
		t.Fatalf("expected service revisions to be returned")
	}

	resp = ecsRequest(t, ts, "ListServiceDeployments", []byte(`{"cluster":"`+clusterName+`","service":"`+serviceName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var listDeploymentsOut struct {
		ServiceDeployments []struct {
			ServiceDeploymentArn string `json:"serviceDeploymentArn"`
		} `json:"serviceDeployments"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listDeploymentsOut); err != nil {
		t.Fatalf("unmarshal list service deployments response: %v", err)
	}
	if len(listDeploymentsOut.ServiceDeployments) == 0 {
		t.Fatalf("expected at least one service deployment")
	}
	deploymentArn := listDeploymentsOut.ServiceDeployments[0].ServiceDeploymentArn

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeServiceDeployments", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","serviceDeployments":["` + deploymentArn + `","missing-deployment"]}`)},
		{name: "StopServiceDeployment", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","serviceDeployment":"` + deploymentArn + `","stopReason":"test"}`)},
		{name: "ListServiceDeploymentsByCreatedAt", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","sortOrder":"ASC"}`)},
		{name: "ListServiceDeploymentsByServiceRevision", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","serviceRevision":"` + revisionOut.ServiceRevisions[0].ServiceRevisionArn + `"}`)},
		{name: "DescribeServiceRevisions", body: []byte(`{"cluster":"` + clusterName + `","service":"` + serviceName + `","serviceRevisions":["` + revisionOut.ServiceRevisions[0].ServiceRevisionArn + `","missing-revision"]}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
