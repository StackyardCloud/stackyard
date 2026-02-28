package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEKSStage8PaginationAndFilters(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterA := "eks-stage8-a"
	clusterB := "eks-stage8-b"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterA+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterB+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters?maxResults=1", nil)
	assertStatus(t, resp, http.StatusOK)
	var listClustersPage1 struct {
		Clusters  []string `json:"clusters"`
		NextToken string   `json:"nextToken"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listClustersPage1); err != nil {
		t.Fatalf("unmarshal list clusters page1: %v", err)
	}
	if len(listClustersPage1.Clusters) != 1 || listClustersPage1.NextToken == "" {
		t.Fatalf("expected one cluster and next token, got %+v", listClustersPage1)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters?maxResults=1&nextToken="+listClustersPage1.NextToken, nil)
	assertStatus(t, resp, http.StatusOK)
	var listClustersPage2 struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listClustersPage2); err != nil {
		t.Fatalf("unmarshal list clusters page2: %v", err)
	}
	if len(listClustersPage2.Clusters) != 1 {
		t.Fatalf("expected one cluster on second page, got %+v", listClustersPage2)
	}

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterA+"/addons", []byte(`{"addonName":"vpc-cni","addonVersion":"latest"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterA+"/addons", []byte(`{"addonName":"coredns","addonVersion":"latest"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterA+"/addons?maxResults=1", nil)
	assertStatus(t, resp, http.StatusOK)
	var listAddonsOut struct {
		Addons    []string `json:"addons"`
		NextToken string   `json:"nextToken"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listAddonsOut); err != nil {
		t.Fatalf("unmarshal list addons: %v", err)
	}
	if len(listAddonsOut.Addons) != 1 || listAddonsOut.NextToken == "" {
		t.Fatalf("expected one addon and next token, got %+v", listAddonsOut)
	}

	resp = eksRequest(t, ts, http.MethodPost, "/eks-anywhere-subscriptions", []byte(`{"name":"stage8-sub-1","term":{"duration":12,"unit":"MONTHS"},"licenseQuantity":1,"licenseType":"Cluster","autoRenew":true}`))
	assertStatus(t, resp, http.StatusOK)
	resp = eksRequest(t, ts, http.MethodPost, "/eks-anywhere-subscriptions", []byte(`{"name":"stage8-sub-2","term":{"duration":12,"unit":"MONTHS"},"licenseQuantity":1,"licenseType":"Cluster","autoRenew":false}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/eks-anywhere-subscriptions?includeStatus=ACTIVE&maxResults=1", nil)
	assertStatus(t, resp, http.StatusOK)
	var listSubscriptionsOut struct {
		Subscriptions []map[string]any `json:"subscriptions"`
		NextToken     string           `json:"nextToken"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listSubscriptionsOut); err != nil {
		t.Fatalf("unmarshal list subscriptions: %v", err)
	}
	if len(listSubscriptionsOut.Subscriptions) != 1 || listSubscriptionsOut.NextToken == "" {
		t.Fatalf("expected one subscription and next token, got %+v", listSubscriptionsOut)
	}
}
