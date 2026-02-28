package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEKSStage9ListAndDescribeUpdateAddonFilters(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage9-cluster"
	addonName := "vpc-cni"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/addons", []byte(`{"addonName":"`+addonName+`","addonVersion":"latest"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/addons/"+addonName+"/update", []byte(`{"addonVersion":"v1.19.0"}`))
	assertStatus(t, resp, http.StatusOK)
	var updateOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateOut); err != nil {
		t.Fatalf("unmarshal update addon: %v", err)
	}
	if updateOut.Update.ID == "" {
		t.Fatalf("expected update id from addon update")
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/updates?addonName="+addonName, nil)
	assertStatus(t, resp, http.StatusOK)
	var listUpdatesOut struct {
		UpdateIDs []string `json:"updateIds"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listUpdatesOut); err != nil {
		t.Fatalf("unmarshal list updates: %v", err)
	}
	found := false
	for _, id := range listUpdatesOut.UpdateIDs {
		if id == updateOut.Update.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected update id %q in filtered list, got %v", updateOut.Update.ID, listUpdatesOut.UpdateIDs)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/updates/"+updateOut.Update.ID+"?addonName="+addonName, nil)
	assertStatus(t, resp, http.StatusOK)
}
