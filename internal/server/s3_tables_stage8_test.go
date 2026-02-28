package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage8SigV4ServiceName(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/buckets", nil, nil, "s3")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong service signing, got %d", resp.StatusCode)
	}
}

func TestS3TablesStage8InvalidJSON(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	metricsURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/metrics"
	resp := signedRequestWithService(t, http.MethodPut, metricsURL, []byte("{"), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

func TestS3TablesStage8InvalidEnums(t *testing.T) {
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
	resp := signedRequestWithService(t, http.MethodPost, createURL, []byte(`{"namespace":["analytics"],"name":"stage8"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var createResp s3TablesCreateTableResponse
	if err := json.Unmarshal(mustBody(t, resp), &createResp); err != nil {
		t.Fatalf("parse create response: %v", err)
	}

	tableArn := createResp.TableArn
	encURL := ts.URL + "/tables/" + url.PathEscape(tableArn) + "/encryption"
	resp = signedRequestWithService(t, http.MethodPut, encURL, []byte(`{"encryption":"BAD"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid encryption, got %d", resp.StatusCode)
	}

	scURL := ts.URL + "/tables/" + url.PathEscape(tableArn) + "/storageclass"
	resp = signedRequestWithService(t, http.MethodPut, scURL, []byte(`{"storageClass":"COLD"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid storage class, got %d", resp.StatusCode)
	}

	replURL := ts.URL + "/tables/" + url.PathEscape(tableArn) + "/replication"
	resp = signedRequestWithService(t, http.MethodPut, replURL, []byte(`{"replication":"maybe"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid replication, got %d", resp.StatusCode)
	}

	maintURL := ts.URL + "/tables/" + url.PathEscape(tableArn) + "/maintenance"
	resp = signedRequestWithService(t, http.MethodPut, maintURL, []byte(`{"maintenance":"weekly"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid maintenance, got %d", resp.StatusCode)
	}

	expURL := ts.URL + "/tables/" + url.PathEscape(tableArn) + "/record-expiration"
	resp = signedRequestWithService(t, http.MethodPut, expURL, []byte(`{"recordExpiration":"abc"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid record expiration, got %d", resp.StatusCode)
	}

	bucketEncURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/encryption"
	resp = signedRequestWithService(t, http.MethodPut, bucketEncURL, []byte(`{"encryption":"BAD"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid bucket encryption, got %d", resp.StatusCode)
	}

	bucketSCURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/storageclass"
	resp = signedRequestWithService(t, http.MethodPut, bucketSCURL, []byte(`{"storageClass":"COLD"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid bucket storage class, got %d", resp.StatusCode)
	}

	bucketReplURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/replication"
	resp = signedRequestWithService(t, http.MethodPut, bucketReplURL, []byte(`{"replication":"maybe"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid bucket replication, got %d", resp.StatusCode)
	}

	bucketMaintURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/maintenance"
	resp = signedRequestWithService(t, http.MethodPut, bucketMaintURL, []byte(`{"maintenance":"weekly"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid bucket maintenance, got %d", resp.StatusCode)
	}

	metricsURL := ts.URL + "/buckets/" + url.PathEscape(bucketArn) + "/metrics"
	resp = signedRequestWithService(t, http.MethodPut, metricsURL, []byte(`{"metricsConfiguration":"maybe"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid metrics configuration, got %d", resp.StatusCode)
	}
}

func TestS3TablesStage8InvalidPagination(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/buckets?continuationToken=bad", nil, nil, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid continuation token, got %d", resp.StatusCode)
	}

	bucketArn := "arn:aws:s3tables:us-east-1:123456789012:bucket/demo-bucket"
	listURL := ts.URL + "/tables/" + url.PathEscape(bucketArn) + "?namespace=analytics&maxTables=0"
	resp = signedRequestWithService(t, http.MethodGet, listURL, nil, nil, "s3tables")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid maxTables, got %d", resp.StatusCode)
	}
}
