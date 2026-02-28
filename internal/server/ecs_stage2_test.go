package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECSStage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const clusterName = "ecs-stage2-cluster"
	const providerName = "ecs-stage2-capacity-provider"

	resp := ecsRequest(t, ts, "CreateCluster", []byte(`{"clusterName":"`+clusterName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecsRequest(t, ts, "CreateCapacityProvider", []byte(`{
		"name":"`+providerName+`",
		"autoScalingGroupProvider":{
			"autoScalingGroupArn":"arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:uuid:autoScalingGroupName/demo",
			"managedScaling":{"status":"ENABLED"},
			"managedTerminationProtection":"ENABLED"
		},
		"tags":[{"key":"env","value":"test"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "DescribeCapacityProviders", body: []byte(`{"capacityProviders":["` + providerName + `"]}`)},
		{name: "UpdateCapacityProvider", body: []byte(`{
			"name":"` + providerName + `",
			"autoScalingGroupProvider":{
				"managedScaling":{"status":"DISABLED"},
				"managedTerminationProtection":"DISABLED"
			}
		}`)},
		{name: "PutClusterCapacityProviders", body: []byte(`{
			"cluster":"` + clusterName + `",
			"capacityProviders":["` + providerName + `"],
			"defaultCapacityProviderStrategy":[{"capacityProvider":"` + providerName + `","base":1,"weight":1}]
		}`)},
		{name: "DeleteCapacityProvider", body: []byte(`{"capacityProvider":"` + providerName + `"}`)},
	}

	for _, action := range actions {
		resp = ecsRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}

	resp = ecsRequest(t, ts, "DescribeCapacityProviders", []byte(`{"capacityProviders":["`+providerName+`"]}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Failures []struct {
			Reason string `json:"reason"`
		} `json:"failures"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe capacity providers response: %v", err)
	}
	if len(describeOut.Failures) != 1 {
		t.Fatalf("expected one failure for deleted capacity provider")
	}
}
