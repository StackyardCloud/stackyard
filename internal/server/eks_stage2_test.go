package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestEKSStage2NodegroupsAndFargate(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage2-cluster"
	nodegroupName := "stage2-ng"
	fargateProfileName := "stage2-fp"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/node-groups", []byte(`{"nodegroupName":"`+nodegroupName+`","nodeRole":"arn:aws:iam::123456789012:role/stackyard-eks","subnets":["subnet-12345678"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/node-groups", nil)
	assertStatus(t, resp, http.StatusOK)
	var listNodegroupsOut struct {
		Nodegroups []string `json:"nodegroups"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listNodegroupsOut); err != nil {
		t.Fatalf("unmarshal list nodegroups: %v", err)
	}
	if !slices.Contains(listNodegroupsOut.Nodegroups, nodegroupName) {
		t.Fatalf("expected nodegroup %q in list %v", nodegroupName, listNodegroupsOut.Nodegroups)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/node-groups/"+nodegroupName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/node-groups/"+nodegroupName+"/update-config", []byte(`{"labels":{"env":"test"}}`))
	assertStatus(t, resp, http.StatusOK)
	var updateConfigOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateConfigOut); err != nil {
		t.Fatalf("unmarshal nodegroup update-config: %v", err)
	}
	if updateConfigOut.Update.ID == "" {
		t.Fatalf("expected update id from nodegroup update-config")
	}

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/node-groups/"+nodegroupName+"/update-version", []byte(`{"version":"1.30"}`))
	assertStatus(t, resp, http.StatusOK)
	var updateVersionOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateVersionOut); err != nil {
		t.Fatalf("unmarshal nodegroup update-version: %v", err)
	}
	if updateVersionOut.Update.ID == "" {
		t.Fatalf("expected update id from nodegroup update-version")
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/updates?nodegroupName="+nodegroupName, nil)
	assertStatus(t, resp, http.StatusOK)
	var listUpdatesOut struct {
		UpdateIDs []string `json:"updateIds"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listUpdatesOut); err != nil {
		t.Fatalf("unmarshal list updates: %v", err)
	}
	if !slices.Contains(listUpdatesOut.UpdateIDs, updateConfigOut.Update.ID) {
		t.Fatalf("expected update list to include %q, got %v", updateConfigOut.Update.ID, listUpdatesOut.UpdateIDs)
	}
	if !slices.Contains(listUpdatesOut.UpdateIDs, updateVersionOut.Update.ID) {
		t.Fatalf("expected update list to include %q, got %v", updateVersionOut.Update.ID, listUpdatesOut.UpdateIDs)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/updates/"+updateVersionOut.Update.ID+"?nodegroupName="+nodegroupName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/node-groups/"+nodegroupName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/fargate-profiles", []byte(`{"fargateProfileName":"`+fargateProfileName+`","podExecutionRoleArn":"arn:aws:iam::123456789012:role/stackyard-eks","subnets":["subnet-12345678"],"selectors":[{"namespace":"default"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/fargate-profiles", nil)
	assertStatus(t, resp, http.StatusOK)
	var listFargateOut struct {
		FargateProfileNames []string `json:"fargateProfileNames"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listFargateOut); err != nil {
		t.Fatalf("unmarshal list fargate profiles: %v", err)
	}
	if !slices.Contains(listFargateOut.FargateProfileNames, fargateProfileName) {
		t.Fatalf("expected fargate profile %q in list %v", fargateProfileName, listFargateOut.FargateProfileNames)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/fargate-profiles/"+fargateProfileName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName+"/fargate-profiles/"+fargateProfileName, nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestEKSStage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage2-implemented"
	nodegroupName := "stage2-implemented-ng"
	fargateProfileName := "stage2-implemented-fp"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/node-groups", []byte(`{"nodegroupName":"`+nodegroupName+`","nodeRole":"arn:aws:iam::123456789012:role/stackyard-eks","subnets":["subnet-12345678"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/fargate-profiles", []byte(`{"fargateProfileName":"`+fargateProfileName+`","podExecutionRoleArn":"arn:aws:iam::123456789012:role/stackyard-eks","subnets":["subnet-12345678"]}`))
	assertStatus(t, resp, http.StatusOK)

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/node-groups"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/node-groups/" + nodegroupName},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/node-groups/" + nodegroupName + "/update-config", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/node-groups/" + nodegroupName + "/update-version", body: []byte(`{}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/fargate-profiles"},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/fargate-profiles/" + fargateProfileName},
	}

	for _, tc := range cases {
		resp := eksRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
