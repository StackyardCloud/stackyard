package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage6TableMaintenance(t *testing.T) {
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
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"maint"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	maintURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/maintenance"
	resp = signedRequestWithService(t, http.MethodPut, maintURL, []byte(`{"maintenance":"enabled"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, maintURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var maintResp s3TablesGetMaintenanceResponse
	if err := json.Unmarshal(mustBody(t, resp), &maintResp); err != nil {
		t.Fatalf("parse maintenance: %v", err)
	}
	if maintResp.Maintenance != "enabled" {
		t.Fatalf("unexpected maintenance %q", maintResp.Maintenance)
	}

	statusURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/maintenance-status"
	resp = signedRequestWithService(t, http.MethodGet, statusURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var statusResp s3TablesGetMaintenanceStatusResponse
	if err := json.Unmarshal(mustBody(t, resp), &statusResp); err != nil {
		t.Fatalf("parse maintenance status: %v", err)
	}
	if statusResp.Status == "" {
		t.Fatalf("expected maintenance status")
	}
}

func TestS3TablesStage6BucketMaintenance(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	maintURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/maintenance"
	resp := signedRequestWithService(t, http.MethodPut, maintURL, []byte(`{"maintenance":"daily"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, maintURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var maintResp s3TablesGetMaintenanceResponse
	if err := json.Unmarshal(mustBody(t, resp), &maintResp); err != nil {
		t.Fatalf("parse maintenance: %v", err)
	}
	if maintResp.Maintenance != "daily" {
		t.Fatalf("unexpected maintenance %q", maintResp.Maintenance)
	}
}

func TestS3TablesStage6RecordExpiration(t *testing.T) {
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
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"expire"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	expURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/record-expiration"
	resp = signedRequestWithService(t, http.MethodPut, expURL, []byte(`{"recordExpiration":"30d"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, expURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var expResp s3TablesGetRecordExpirationResponse
	if err := json.Unmarshal(mustBody(t, resp), &expResp); err != nil {
		t.Fatalf("parse record expiration: %v", err)
	}
	if expResp.RecordExpiration != "30d" {
		t.Fatalf("unexpected record expiration %q", expResp.RecordExpiration)
	}

	statusURL := ts.URL + "/tables/" + url.PathEscape(createResp.TableArn) + "/record-expiration-status"
	resp = signedRequestWithService(t, http.MethodGet, statusURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var statusResp s3TablesGetRecordExpirationStatusResponse
	if err := json.Unmarshal(mustBody(t, resp), &statusResp); err != nil {
		t.Fatalf("parse record expiration status: %v", err)
	}
	if statusResp.Status == "" {
		t.Fatalf("expected record expiration status")
	}
}
