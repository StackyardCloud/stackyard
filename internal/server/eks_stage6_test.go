package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestEKSStage6EncryptionAndTags(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage6-cluster"

	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	var createClusterOut struct {
		Cluster struct {
			Arn string `json:"arn"`
		} `json:"cluster"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createClusterOut); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}
	if createClusterOut.Cluster.Arn == "" {
		t.Fatalf("expected cluster arn")
	}

	resp = eksRequest(t, ts, http.MethodPost, "/clusters/"+clusterName+"/encryption-config/associate", []byte(`{"encryptionConfig":[{"resources":["secrets"],"provider":{"keyArn":"arn:aws:kms:us-east-1:123456789012:key/stackyard"}}]}`))
	assertStatus(t, resp, http.StatusOK)
	var updateOut struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateOut); err != nil {
		t.Fatalf("unmarshal associate encryption config: %v", err)
	}
	if updateOut.Update.ID == "" {
		t.Fatalf("expected update id from associate encryption config")
	}

	encodedArn := url.PathEscape(createClusterOut.Cluster.Arn)
	resp = eksRequest(t, ts, http.MethodPost, "/tags/"+encodedArn, []byte(`{"tags":{"env":"stage6","team":"platform"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/tags/"+encodedArn, nil)
	assertStatus(t, resp, http.StatusOK)
	var listTagsOut struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsOut); err != nil {
		t.Fatalf("unmarshal list tags: %v", err)
	}
	if listTagsOut.Tags["env"] != "stage6" {
		t.Fatalf("expected env tag to be stage6, got %q", listTagsOut.Tags["env"])
	}

	resp = eksRequest(t, ts, http.MethodDelete, "/tags/"+encodedArn+"?tagKeys=env&tagKeys=team", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = eksRequest(t, ts, http.MethodGet, "/tags/"+encodedArn, nil)
	assertStatus(t, resp, http.StatusOK)
	var listTagsAfterDeleteOut struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listTagsAfterDeleteOut); err != nil {
		t.Fatalf("unmarshal list tags after delete: %v", err)
	}
	if len(listTagsAfterDeleteOut.Tags) != 0 {
		t.Fatalf("expected no tags after delete, got %v", listTagsAfterDeleteOut.Tags)
	}
}

func TestEKSStage6ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterName := "eks-stage6-implemented"
	resp := eksRequest(t, ts, http.MethodPost, "/clusters", []byte(`{"name":"`+clusterName+`","roleArn":"arn:aws:iam::123456789012:role/stackyard-eks","resourcesVpcConfig":{"subnetIds":["subnet-12345678"]}}`))
	assertStatus(t, resp, http.StatusOK)

	var createClusterOut struct {
		Cluster struct {
			Arn string `json:"arn"`
		} `json:"cluster"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createClusterOut); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}
	encodedArn := url.PathEscape(createClusterOut.Cluster.Arn)

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodPost, path: "/clusters/" + clusterName + "/encryption-config/associate", body: []byte(`{"encryptionConfig":[{"resources":["secrets"],"provider":{"keyArn":"arn:aws:kms:us-east-1:123456789012:key/stackyard"}}]}`)},
		{method: http.MethodPost, path: "/tags/" + encodedArn, body: []byte(`{"tags":{"env":"stage6"}}`)},
		{method: http.MethodGet, path: "/tags/" + encodedArn},
		{method: http.MethodDelete, path: "/tags/" + encodedArn + "?tagKeys=env"},
	}

	for _, tc := range cases {
		resp := eksRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
