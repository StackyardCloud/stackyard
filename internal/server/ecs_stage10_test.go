package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage10ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage10-cluster"
	const family = "ecs-stage10-task-family"

	resp := ecsRequest(t, ts, "CreateCluster", []byte(`{"clusterName":"`+clusterName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "RegisterContainerInstance", []byte(`{"cluster":"`+clusterName+`","ec2InstanceId":"i-stage100001"}`))
	assertStatus(t, resp, http.StatusOK)
	var registerContainerOut struct {
		ContainerInstance struct {
			ContainerInstanceArn string `json:"containerInstanceArn"`
		} `json:"containerInstance"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerContainerOut); err != nil {
		t.Fatalf("unmarshal register container instance response: %v", err)
	}

	resp = ecsRequest(t, ts, "RegisterTaskDefinition", []byte(`{
		"family":"`+family+`",
		"networkMode":"awsvpc",
		"cpu":"256",
		"memory":"512",
		"requiresCompatibilities":["EC2"],
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

	resp = ecsRequest(t, ts, "StartTask", []byte(`{
		"cluster":"`+clusterName+`",
		"taskDefinition":"`+registerTaskOut.TaskDefinition.TaskDefinitionArn+`",
		"containerInstances":["`+registerContainerOut.ContainerInstance.ContainerInstanceArn+`"]
	}`))
	assertStatus(t, resp, http.StatusOK)
	var startTaskOut struct {
		Tasks []struct {
			TaskArn string `json:"taskArn"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startTaskOut); err != nil {
		t.Fatalf("unmarshal start task response: %v", err)
	}
	if len(startTaskOut.Tasks) != 1 {
		t.Fatalf("expected one started task, got %d", len(startTaskOut.Tasks))
	}
	taskArn := startTaskOut.Tasks[0].TaskArn

	actions := []struct {
		name string
		body []byte
	}{
		{name: "SubmitAttachmentStateChanges", body: []byte(`{"cluster":"` + clusterName + `","containerInstance":"` + registerContainerOut.ContainerInstance.ContainerInstanceArn + `","attachments":[{"attachmentArn":"arn:aws:ecs:us-east-1:123456789012:attachment/demo","status":"ATTACHED"}]}`)},
		{name: "SubmitContainerStateChange", body: []byte(`{"cluster":"` + clusterName + `","containerInstance":"` + registerContainerOut.ContainerInstance.ContainerInstanceArn + `","task":"` + taskArn + `","containerName":"app","status":"RUNNING"}`)},
		{name: "SubmitTaskStateChange", body: []byte(`{"cluster":"` + clusterName + `","task":"` + taskArn + `","status":"RUNNING"}`)},
		{name: "SubmitTaskStateChangeByAgent", body: []byte(`{"cluster":"` + clusterName + `","task":"` + taskArn + `","status":"RUNNING"}`)},
		{name: "SubmitTaskStateChangeByManagedAgents", body: []byte(`{"cluster":"` + clusterName + `","task":"` + taskArn + `","status":"RUNNING"}`)},
		{name: "StartTelemetrySession", body: []byte(`{"cluster":"` + clusterName + `","containerInstance":"` + registerContainerOut.ContainerInstance.ContainerInstanceArn + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
