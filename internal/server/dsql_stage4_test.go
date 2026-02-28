package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestDSQLStage4TagLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	var created struct {
		Identifier string `json:"identifier"`
		ARN        string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &created); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}
	if created.ARN == "" {
		t.Fatalf("expected cluster arn")
	}

	resourcePath := "/tags/" + url.PathEscape(created.ARN)
	resp = dsqlRequest(t, ts, http.MethodPost, resourcePath, []byte(`{"tags":{"env":"dev","team":"platform"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dsqlRequest(t, ts, http.MethodGet, resourcePath, nil)
	assertStatus(t, resp, http.StatusOK)
	var listed struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listed); err != nil {
		t.Fatalf("unmarshal list tags: %v", err)
	}
	if listed.Tags["env"] != "dev" || listed.Tags["team"] != "platform" {
		t.Fatalf("unexpected tags: %#v", listed.Tags)
	}

	resp = dsqlRequest(t, ts, http.MethodPost, resourcePath, []byte(`{"tags":{"env":"prod","owner":"stackyard"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dsqlRequest(t, ts, http.MethodDelete, resourcePath+"?tagKeys=team&tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = dsqlRequest(t, ts, http.MethodGet, resourcePath, nil)
	assertStatus(t, resp, http.StatusOK)
	var listedAfter struct {
		Tags map[string]string `json:"tags"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listedAfter); err != nil {
		t.Fatalf("unmarshal list tags after untag: %v", err)
	}
	if listedAfter.Tags["env"] != "prod" {
		t.Fatalf("expected env=prod after overwrite, got %#v", listedAfter.Tags)
	}
	if _, ok := listedAfter.Tags["team"]; ok {
		t.Fatalf("expected team tag removed")
	}
	if _, ok := listedAfter.Tags["owner"]; ok {
		t.Fatalf("expected owner tag removed")
	}
}

func TestDSQLStage4TagValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	invalidResource := url.PathEscape("arn:aws:s3:::not-dsql")
	resp := dsqlRequest(t, ts, http.MethodPost, "/tags/"+invalidResource, []byte(`{"tags":{"k":"v"}}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid resource arn, got %d", resp.StatusCode)
	}

	resp = dsqlRequest(t, ts, http.MethodDelete, "/tags/"+invalidResource+"?tagKeys=env", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid resource arn, got %d", resp.StatusCode)
	}

	resp = dsqlRequest(t, ts, http.MethodGet, "/tags/"+invalidResource, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid resource arn, got %d", resp.StatusCode)
	}
}

func TestDSQLStage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	var created struct {
		ARN string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &created); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}

	resourcePath := fmt.Sprintf("/tags/%s", url.PathEscape(created.ARN))
	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodPost, path: resourcePath, body: []byte(`{"tags":{"env":"dev"}}`)},
		{method: http.MethodGet, path: resourcePath},
		{method: http.MethodDelete, path: resourcePath + "?tagKeys=env"},
	}
	for _, tc := range cases {
		resp := dsqlRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
