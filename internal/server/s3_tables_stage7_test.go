package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestS3TablesStage7TableBucketMetricsConfiguration(t *testing.T) {
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

	resp := signedRequestWithService(t, http.MethodPut, metricsURL, []byte(`{"metricsConfiguration":"enabled"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, metricsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)
	var getResp s3TablesGetMetricsConfigurationResponse
	if err := json.Unmarshal(mustBody(t, resp), &getResp); err != nil {
		t.Fatalf("parse metrics response: %v", err)
	}
	if getResp.MetricsConfiguration != "enabled" {
		t.Fatalf("unexpected metrics configuration %q", getResp.MetricsConfiguration)
	}

	resp = signedRequestWithService(t, http.MethodDelete, metricsURL, nil, nil, "s3tables")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodGet, metricsURL, nil, nil, "s3tables")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}
