package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage7UnknownFieldsRejected(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte(`{"vectorBucketName":"demo-bucket","extra":1}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage7BucketArnMismatch(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte(`{"vectorBucketName":"demo-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create bucket, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucket", []byte(`{"vectorBucketName":"demo-bucket","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/other"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for ARN mismatch, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage7IndexArnMismatch(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte(`{"vectorBucketName":"demo-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create bucket, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":3,"distanceMetric":"COSINE"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create index, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetIndex", []byte(`{"indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/demo-index","indexName":"other"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for index ARN mismatch, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage7InvalidTokensAndPagination(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectorBuckets", []byte(`{"nextToken":"bad"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid nextToken, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte(`{"vectorBucketName":"demo-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create bucket, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":3,"distanceMetric":"COSINE"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create index, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListIndexes", []byte(`{"vectorBucketName":"demo-bucket","nextToken":"bad"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid nextToken on list indexes, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", []byte(`{"indexName":"demo-index","maxResults":-1}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative maxResults, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", []byte(`{"indexName":"demo-index","segmentCount":-1}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative segmentCount, got %d", resp.StatusCode)
	}
}

func TestS3VectorsStage7InvalidVectorDataType(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte(`{"vectorBucketName":"demo-bucket"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create bucket, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":2,"distanceMetric":"COSINE"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 create index, got %d", resp.StatusCode)
	}

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", []byte(`{"indexName":"demo-index","vectors":[{"key":"v1","data":{"int32":[1,2]}}]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid data type, got %d", resp.StatusCode)
	}
}
