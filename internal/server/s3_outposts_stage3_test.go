package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3OutpostsStage3ListOutpostsWithS3Multiple(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"Content-Type":     "application/json",
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"OutpostId":"op-11111111111111111","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","ClientToken":"tok-multi"}`), headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListOutpostsWithS3", nil, map[string]string{
		"x-amz-account-id": "123456789012",
	}, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	var list s3OutpostsListOutpostsWithS3Response
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse list outposts: %v", err)
	}
	if len(list.Outposts) != 2 {
		t.Fatalf("expected 2 outposts, got %d", len(list.Outposts))
	}
	if list.Outposts[0].OutpostId != "op-0123456789abcdef0" || list.Outposts[1].OutpostId != "op-11111111111111111" {
		t.Fatalf("unexpected outpost ordering: %q, %q", list.Outposts[0].OutpostId, list.Outposts[1].OutpostId)
	}
}

func TestS3OutpostsStage3ListOutpostsWithS3Empty(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "123456789012",
	}
	resp := signedRequestWithService(t, http.MethodDelete, ts.URL+"/S3Outposts/DeleteEndpoint?outpostId=op-0123456789abcdef0&endpointId=1234567890123456789", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListOutpostsWithS3", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	var list s3OutpostsListOutpostsWithS3Response
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse list outposts: %v", err)
	}
	if len(list.Outposts) != 0 {
		t.Fatalf("expected empty outposts list, got %d", len(list.Outposts))
	}
	if list.NextToken != "" {
		t.Fatalf("expected empty next token, got %q", list.NextToken)
	}
}
