package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage6RejectsUnsignedAccountHeader(t *testing.T) {
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
	body := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/CreateVectorBucket", body, headers, "s3vectors", testRegion, "host;x-amz-content-sha256;x-amz-date")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for unsigned account header, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage6AcceptsSignedAccountHeader(t *testing.T) {
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
	body := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/CreateVectorBucket", body, headers, "s3vectors", testRegion, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for signed account header, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage6RejectsInvalidAccountHeader(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	headers := map[string]string{
		"x-amz-account-id": "bad",
	}
	body := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/CreateVectorBucket", body, headers, "s3vectors", testRegion, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid account header, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage6RejectsRegionMismatch(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/CreateVectorBucket", body, nil, "s3vectors", "us-west-2", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for region mismatch, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage6AcceptsRegionMatch(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signRequestWithHeaders(t, http.MethodPost, ts.URL+"/CreateVectorBucket", body, nil, "s3vectors", testRegion, "")
	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		t.Fatalf("expected 200 for region match, got %d (%s)", resp.StatusCode, buf.String())
	}
}
