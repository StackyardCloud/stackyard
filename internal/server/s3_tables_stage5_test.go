package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage5TableReplication(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	createURL := ts.URL + "/tables/" + url.PathEscape(bucketArn)
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"repl"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	replURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/replication"
	resp = signedRequestWithService(t, http.MethodPut, replURL, []byte(`{"replication":"enabled"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, replURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var replResp s3TablesGetReplicationResponse
	if err := json.Unmarshal(mustBody(t, resp), &replResp); err != nil {
		t.Fatalf("parse replication: %v", err)
	}
	if replResp.Replication != "enabled" {
		t.Fatalf("unexpected replication %q", replResp.Replication)
	}

	statusURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/replication-status"
	resp = signedRequestWithService(t, http.MethodGet, statusURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var statusResp s3TablesGetReplicationStatusResponse
	if err := json.Unmarshal(mustBody(t, resp), &statusResp); err != nil {
		t.Fatalf("parse status: %v", err)
	}
	if statusResp.Status == "" {
		t.Fatalf("expected status value")
	}

	resp = signedRequestWithService(t, http.MethodDelete, replURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
}

func TestS3TablesStage5TableBucketReplication(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	replURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/replication"
	resp := signedRequestWithService(t, http.MethodPut, replURL, []byte(`{"replication":"enabled"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, replURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var replResp s3TablesGetReplicationResponse
	if err := json.Unmarshal(mustBody(t, resp), &replResp); err != nil {
		t.Fatalf("parse replication: %v", err)
	}
	if replResp.Replication != "enabled" {
		t.Fatalf("unexpected replication %q", replResp.Replication)
	}

	resp = signedRequestWithService(t, http.MethodDelete, replURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
}
