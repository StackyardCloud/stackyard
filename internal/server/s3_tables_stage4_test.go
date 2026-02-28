package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage4TableBucketEncryptionAndStorageClass(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	encURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/encryption"
	resp := signedRequestWithService(t, http.MethodPut, encURL, []byte(`{"encryption":"SSE-S3"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, encURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var encResp s3TablesGetEncryptionResponse
	if err := json.Unmarshal(mustBody(t, resp), &encResp); err != nil {
		t.Fatalf("parse encryption: %v", err)
	}
	if encResp.Encryption != "SSE-S3" {
		t.Fatalf("unexpected encryption %q", encResp.Encryption)
	}

	resp = signedRequestWithService(t, http.MethodDelete, encURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	scURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/storageclass"
	resp = signedRequestWithService(t, http.MethodPut, scURL, []byte(`{"storageClass":"STANDARD"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequestWithService(t, http.MethodGet, scURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var scResp s3TablesGetStorageClassResponse
	if err := json.Unmarshal(mustBody(t, resp), &scResp); err != nil {
		t.Fatalf("parse storage class: %v", err)
	}
	if scResp.StorageClass != "STANDARD" {
		t.Fatalf("unexpected storage class %q", scResp.StorageClass)
	}
}

func TestS3TablesStage4TableEncryptionAndStorageClass(t *testing.T) {
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
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"enc"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	encURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/encryption"
	resp = signedRequestWithService(t, http.MethodPut, encURL, []byte(`{"encryption":"SSE-KMS"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequestWithService(t, http.MethodGet, encURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var encResp s3TablesGetEncryptionResponse
	if err := json.Unmarshal(mustBody(t, resp), &encResp); err != nil {
		t.Fatalf("parse encryption: %v", err)
	}
	if encResp.Encryption != "SSE-KMS" {
		t.Fatalf("unexpected encryption %q", encResp.Encryption)
	}

	scURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/storageclass"
	resp = signedRequestWithService(t, http.MethodPut, scURL, []byte(`{"storageClass":"STANDARD_IA"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	resp = signedRequestWithService(t, http.MethodGet, scURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var scResp s3TablesGetStorageClassResponse
	if err := json.Unmarshal(mustBody(t, resp), &scResp); err != nil {
		t.Fatalf("parse storage class: %v", err)
	}
	if scResp.StorageClass != "STANDARD_IA" {
		t.Fatalf("unexpected storage class %q", scResp.StorageClass)
	}
}
