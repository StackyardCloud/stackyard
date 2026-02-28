package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage1GetAndDeleteNamespace(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	getURL := ts.URL + "/namespaces/" + url.PathEscape(arn) + "?namespace=analytics"
	resp := signedRequestWithService(t, http.MethodGet, getURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	delURL := ts.URL + "/namespaces/" + url.PathEscape(arn) + "?namespace=analytics"
	resp = signedRequestWithService(t, http.MethodDelete, delURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, getURL, nil, nil, "s3tables")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestS3TablesStage1DeleteTableBucket(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	arn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	urlStr := ts.URL + "/buckets/" + url.PathEscape(arn)
	resp := signedRequestWithService(t, http.MethodDelete, urlStr, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, urlStr, nil, nil, "s3tables")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}
