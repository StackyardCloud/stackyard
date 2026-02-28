package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func eksRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "eks")
}

func TestEKSStage1ClusterLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage1-cluster"
	createBody := []byte(`{"name":"` + clusterName + `","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"],"endpointPublicAccess":true}}`)

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", createBody)
	assertStatus(t, resp, http.StatusOK)

	var createOut struct {
		Cluster struct {
			Name   string `json:"name"`
			Arn    string `json:"arn"`
			Status string `json:"status"`
		} `json:"cluster"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createOut); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}
	if createOut.Cluster.Name != clusterName {
		t.Fatalf("expected cluster name %q, got %q", clusterName, createOut.Cluster.Name)
	}
	if createOut.Cluster.Arn == "" {
		t.Fatalf("expected cluster arn")
	}
	if createOut.Cluster.Status == "" {
		t.Fatalf("expected cluster status")
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters", nil)
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list clusters: %v", err)
	}
	if !slices.Contains(listOut.Clusters, clusterName) {
		t.Fatalf("expected cluster list to contain %q, got %v", clusterName, listOut.Clusters)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/update-config", []byte(`{"resourcesVpcConfig":{"endpointPublicAccess":false}}`))
	assertStatus(t, resp, http.StatusOK)
	var updateConfigOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateConfigOut); err != nil {
		t.Fatalf("unmarshal update cluster config: %v", err)
	}
	if updateConfigOut.Update.ID == "" {
		t.Fatalf("expected update id from update-config")
	}

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/updates", []byte(`{"version":"1.30"}`))
	assertStatus(t, resp, http.StatusOK)
	var updateVersionOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateVersionOut); err != nil {
		t.Fatalf("unmarshal update cluster version: %v", err)
	}
	if updateVersionOut.Update.ID == "" {
		t.Fatalf("expected update id from update-version")
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/updates", nil)
	assertStatus(t, resp, http.StatusOK)
	var listUpdatesOut struct {
		UpdateIDs []string `json:"updateIds"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listUpdatesOut); err != nil {
		t.Fatalf("unmarshal list updates: %v", err)
	}
	if !slices.Contains(listUpdatesOut.UpdateIDs, updateConfigOut.Update.ID) {
		t.Fatalf("expected update list to include config update %q, got %v", updateConfigOut.Update.ID, listUpdatesOut.UpdateIDs)
	}
	if !slices.Contains(listUpdatesOut.UpdateIDs, updateVersionOut.Update.ID) {
		t.Fatalf("expected update list to include version update %q, got %v", updateVersionOut.Update.ID, listUpdatesOut.UpdateIDs)
	}

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName+"/updates/"+updateVersionOut.Update.ID, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodDelete, "/clusters/"+clusterName, nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/clusters/"+clusterName, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException, got %s", body)
	}
}

func TestEKSStage1ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage1-implemented"
	createBody := []byte(`{"name":"` + clusterName + `","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`)
	resp := eksRequest(t, ts, http.MethodPost, "/clusters", createBody)
	assertStatus(t, resp, http.StatusOK)

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/cluster-versions"},
		{method: http.MethodGet, path: "/clusters"},
		{method: http.MethodGet, path: "/clusters/" + clusterName},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/update-config", body: []byte(`{}`)},
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/updates", body: []byte(`{"version":"1.31"}`)},
		{method: http.MethodGet, path: "/clusters/" + clusterName + "/updates"},
	}

	for _, tc := range cases {
		resp := eksRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
