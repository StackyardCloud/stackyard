package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3OutpostsStage2ListEndpointsEmpty(t *testing.T) {
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

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListEndpoints", nil, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	var list s3OutpostsListEndpointsResponse
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse list endpoints: %v", err)
	}
	if len(list.Endpoints) != 0 {
		t.Fatalf("expected empty endpoints list, got %d", len(list.Endpoints))
	}
	if list.NextToken != "" {
		t.Fatalf("expected empty next token, got %q", list.NextToken)
	}
}

func TestS3OutpostsStage2ListEndpointsFilter(t *testing.T) {
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
	body := []byte(`{"OutpostId":"op-11111111111111111","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", body, headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListEndpoints?outpostId=op-0123456789abcdef0", nil, map[string]string{
		"x-amz-account-id": "123456789012",
	}, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	var list s3OutpostsListEndpointsResponse
	if err := json.Unmarshal(mustBody(t, resp), &list); err != nil {
		t.Fatalf("parse filtered list: %v", err)
	}
	if len(list.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint for filter, got %d", len(list.Endpoints))
	}
	if list.Endpoints[0].OutpostsId != "op-0123456789abcdef0" {
		t.Fatalf("unexpected outpost id %q", list.Endpoints[0].OutpostsId)
	}
}

func TestS3OutpostsStage2ListEndpointsPagination(t *testing.T) {
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
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","ClientToken":"tok-1"}`), headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/S3Outposts/CreateEndpoint", []byte(`{"OutpostId":"op-0123456789abcdef0","SecurityGroupId":"sg-0123456789abcdef0","SubnetId":"subnet-0123456789abcdef0","ClientToken":"tok-2"}`), headers, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListEndpoints?maxResults=1", nil, map[string]string{
		"x-amz-account-id": "123456789012",
	}, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	var page1 s3OutpostsListEndpointsResponse
	if err := json.Unmarshal(mustBody(t, resp), &page1); err != nil {
		t.Fatalf("parse page1: %v", err)
	}
	if len(page1.Endpoints) != 1 || page1.NextToken == "" {
		t.Fatalf("expected 1 endpoint and next token, got %d token=%q", len(page1.Endpoints), page1.NextToken)
	}

	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/S3Outposts/ListEndpoints?maxResults=1&nextToken="+page1.NextToken, nil, map[string]string{
		"x-amz-account-id": "123456789012",
	}, "s3-outposts")
	assertStatus(t, resp, http.StatusOK)
	var page2 s3OutpostsListEndpointsResponse
	if err := json.Unmarshal(mustBody(t, resp), &page2); err != nil {
		t.Fatalf("parse page2: %v", err)
	}
	if len(page2.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint on page2, got %d", len(page2.Endpoints))
	}
}
