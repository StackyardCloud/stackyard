package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage8ReadAfterWriteConsistency(t *testing.T) {
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

	putReq := []byte(`{"indexName":"demo-index","vectors":[{"key":"v1","data":{"float32":[0.1,0.2]}},{"key":"v2","data":{"float32":[0.3,0.4]}}]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", putReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 put vectors, got %d", resp.StatusCode)
	}

	getReq := []byte(`{"indexName":"demo-index","keys":["v1","v2"]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectors", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 get vectors after put, got %d", resp.StatusCode)
	}

	listReq := []byte(`{"indexName":"demo-index","maxResults":10}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 list vectors after put, got %d", resp.StatusCode)
	}
}
