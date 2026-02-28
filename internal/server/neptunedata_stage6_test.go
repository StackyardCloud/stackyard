package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNeptuneDataStage6CompatibilityHardening(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodGet, "/loader?limit=bad", nil)
	if status != http.StatusBadRequest || !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected ListLoaderJobs validation error, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/propertygraph/statistics", []byte(`{"mode":"unknown"}`))
	if status != http.StatusBadRequest || !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected ManagePropertygraphStatistics validation error, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/sparql/stream?iteratorType=NOT_A_MODE", nil)
	if status != http.StatusBadRequest || !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected GetSparqlStream validation error, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/system", []byte(`{}`))
	if status != http.StatusBadRequest || !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected ExecuteFastReset action validation error, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/system", []byte(`{"action":"initiateDatabaseReset"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteFastReset initiate 200, got %d: %s", status, body)
	}
	resetToken := jsonTagValue(body, "token")
	if strings.TrimSpace(resetToken) == "" {
		t.Fatalf("expected reset token in initiate response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/system", []byte(`{
		"action":"performDatabaseReset",
		"token":"`+resetToken+`"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected ExecuteFastReset perform 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/dataprocessing", []byte(`{
		"id":"stage6-dp",
		"inputDataS3Location":"s3://stackyard-neptunedata/stage6/raw",
		"processedDataS3Location":"s3://stackyard-neptunedata/stage6/processed"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected StartMLDataProcessingJob 200, got %d: %s", status, body)
	}
	firstDPID := jsonTagValue(body, "id")

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/dataprocessing", []byte(`{
		"id":"stage6-dp",
		"inputDataS3Location":"s3://stackyard-neptunedata/stage6/raw",
		"processedDataS3Location":"s3://stackyard-neptunedata/stage6/processed"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected idempotent StartMLDataProcessingJob 200, got %d: %s", status, body)
	}
	if secondDPID := jsonTagValue(body, "id"); secondDPID != firstDPID {
		t.Fatalf("expected idempotent ML data processing id reuse, got first=%s second=%s", firstDPID, secondDPID)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/endpoints", []byte(`{
		"id":"stage6-endpoint",
		"modelName":"stage6-model"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected CreateMLEndpoint 200, got %d: %s", status, body)
	}
	firstEndpointARN := jsonTagValue(body, "arn")
	if firstEndpointARN == "" {
		t.Fatalf("expected endpoint arn in response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/endpoints", []byte(`{
		"id":"stage6-endpoint",
		"modelName":"stage6-model"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected idempotent CreateMLEndpoint 200, got %d: %s", status, body)
	}
	if secondEndpointARN := jsonTagValue(body, "arn"); secondEndpointARN != firstEndpointARN {
		t.Fatalf("expected idempotent endpoint arn reuse, got first=%s second=%s", firstEndpointARN, secondEndpointARN)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/endpoints?maxItems=0", nil)
	if status != http.StatusBadRequest || !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected ListMLEndpoints maxItems validation error, got %d: %s", status, body)
	}
}

func TestNeptuneDataStage3456ImplementedRoutesDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{
			method: http.MethodPost,
			path:   "/loader",
			body: []byte(`{
				"source":"s3://stackyard-neptunedata/stage6/input.csv",
				"format":"csv",
				"s3BucketRegion":"us-east-1",
				"iamRoleArn":"arn:aws:iam::123456789012:role/stackyard-neptunedata"
			}`),
		},
		{method: http.MethodGet, path: "/loader"},
		{method: http.MethodGet, path: "/loader/stackyard-loader"},
		{method: http.MethodDelete, path: "/loader/stackyard-loader"},
		{method: http.MethodGet, path: "/propertygraph/statistics"},
		{method: http.MethodPost, path: "/propertygraph/statistics", body: []byte(`{"mode":"refresh"}`)},
		{method: http.MethodDelete, path: "/propertygraph/statistics"},
		{method: http.MethodGet, path: "/propertygraph/stream?iteratorType=LATEST"},
		{method: http.MethodGet, path: "/sparql/statistics"},
		{method: http.MethodPost, path: "/sparql/statistics", body: []byte(`{"mode":"refresh"}`)},
		{method: http.MethodDelete, path: "/sparql/statistics"},
		{method: http.MethodGet, path: "/sparql/stream?iteratorType=LATEST"},
		{
			method: http.MethodPost,
			path:   "/ml/dataprocessing",
			body: []byte(`{
				"inputDataS3Location":"s3://stackyard-neptunedata/stage6/raw",
				"processedDataS3Location":"s3://stackyard-neptunedata/stage6/processed"
			}`),
		},
		{method: http.MethodGet, path: "/ml/dataprocessing"},
		{method: http.MethodGet, path: "/ml/dataprocessing/stackyard-dp"},
		{method: http.MethodDelete, path: "/ml/dataprocessing/stackyard-dp"},
		{
			method: http.MethodPost,
			path:   "/ml/modeltraining",
			body: []byte(`{
				"dataProcessingJobId":"stackyard-dp",
				"trainModelS3Location":"s3://stackyard-neptunedata/stage6/model"
			}`),
		},
		{method: http.MethodGet, path: "/ml/modeltraining"},
		{method: http.MethodGet, path: "/ml/modeltraining/stackyard-train"},
		{method: http.MethodDelete, path: "/ml/modeltraining/stackyard-train"},
		{
			method: http.MethodPost,
			path:   "/ml/modeltransform",
			body:   []byte(`{"modelTransformOutputS3Location":"s3://stackyard-neptunedata/stage6/transform"}`),
		},
		{method: http.MethodGet, path: "/ml/modeltransform"},
		{method: http.MethodGet, path: "/ml/modeltransform/stackyard-transform"},
		{method: http.MethodDelete, path: "/ml/modeltransform/stackyard-transform"},
		{method: http.MethodPost, path: "/ml/endpoints", body: []byte(`{"id":"stackyard-endpoint"}`)},
		{method: http.MethodGet, path: "/ml/endpoints"},
		{method: http.MethodGet, path: "/ml/endpoints/stackyard-endpoint"},
		{method: http.MethodDelete, path: "/ml/endpoints/stackyard-endpoint"},
		{method: http.MethodPost, path: "/system", body: []byte(`{"action":"initiateDatabaseReset"}`)},
	}

	for _, tc := range cases {
		status, body := neptuneDataCall(t, ts, tc.method, tc.path, tc.body)
		if status == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s %s unexpectedly returned not implemented: status=%d body=%s", tc.method, tc.path, status, body)
		}
	}
}
