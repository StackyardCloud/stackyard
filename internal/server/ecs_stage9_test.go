package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage9ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage9-cluster"

	resp := ecsRequest(t, ts, "CreateCluster", []byte(`{"clusterName":"`+clusterName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "RegisterContainerInstance", []byte(`{"cluster":"`+clusterName+`","ec2InstanceId":"i-stage90001"}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("action %s returned not implemented", "RegisterContainerInstance")
	}
	assertStatus(t, resp, http.StatusOK)
	var registerOut struct {
		ContainerInstance struct {
			ContainerInstanceArn string `json:"containerInstanceArn"`
		} `json:"containerInstance"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &registerOut); err != nil {
		t.Fatalf("unmarshal register container instance response: %v", err)
	}
	if registerOut.ContainerInstance.ContainerInstanceArn == "" {
		t.Fatalf("expected containerInstanceArn from RegisterContainerInstance")
	}
	containerInstanceArn := registerOut.ContainerInstance.ContainerInstanceArn

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeContainerInstances", body: []byte(`{"cluster":"` + clusterName + `","containerInstances":["` + containerInstanceArn + `"]}`)},
		{name: "ListContainerInstances", body: []byte(`{"cluster":"` + clusterName + `"}`)},
		{name: "UpdateContainerInstancesState", body: []byte(`{"cluster":"` + clusterName + `","containerInstances":["` + containerInstanceArn + `"],"status":"DRAINING"}`)},
		{name: "PutAttributes", body: []byte(`{"cluster":"` + clusterName + `","attributes":[{"name":"rack","value":"a1","targetType":"container-instance","targetId":"` + containerInstanceArn + `"}]}`)},
		{name: "ListAttributes", body: []byte(`{"cluster":"` + clusterName + `","targetType":"container-instance","targetId":"` + containerInstanceArn + `"}`)},
		{name: "DeleteAttributes", body: []byte(`{"cluster":"` + clusterName + `","attributes":[{"name":"rack","targetType":"container-instance","targetId":"` + containerInstanceArn + `"}]}`)},
		{name: "DeregisterContainerInstance", body: []byte(`{"cluster":"` + clusterName + `","containerInstance":"` + containerInstanceArn + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}
