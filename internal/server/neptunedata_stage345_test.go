package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNeptuneDataStage3LoaderLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodPost, "/loader", []byte(`{
		"source":"s3://stackyard-neptunedata/stage3/input.csv",
		"format":"csv",
		"s3BucketRegion":"us-east-1",
		"iamRoleArn":"arn:aws:iam::123456789012:role/stackyard-neptunedata"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected StartLoaderJob 200, got %d: %s", status, body)
	}
	loadID := jsonTagValue(body, "loadId")
	if strings.TrimSpace(loadID) == "" {
		t.Fatalf("expected loadId in StartLoaderJob response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/loader?limit=10&includeQueuedLoads=false", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListLoaderJobs 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, loadID) {
		t.Fatalf("expected loader id in ListLoaderJobs response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/loader/"+loadID+"?details=true&errors=true&page=1&errorsPerPage=10", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetLoaderJobStatus 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, loadID) {
		t.Fatalf("expected loader id in GetLoaderJobStatus response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/loader/"+loadID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelLoaderJob 200, got %d: %s", status, body)
	}
	if !strings.Contains(strings.ToLower(body), "cancel") {
		t.Fatalf("expected cancellation marker in CancelLoaderJob response: %s", body)
	}
}

func TestNeptuneDataStage4StatisticsAndStreamSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodPost, "/propertygraph/statistics", []byte(`{"mode":"refresh"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ManagePropertygraphStatistics 200, got %d: %s", status, body)
	}
	if jsonTagValue(body, "statisticsId") == "" {
		t.Fatalf("expected statisticsId in ManagePropertygraphStatistics response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/propertygraph/statistics", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetPropertygraphStatistics 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"payload\"") {
		t.Fatalf("expected payload in GetPropertygraphStatistics response: %s", body)
	}

	status, body = neptuneDataCall(
		t,
		ts,
		http.MethodGet,
		"/propertygraph/stream?limit=2&iteratorType=TRIM_HORIZON&commitNum=1&opNum=1",
		nil,
	)
	if status != http.StatusOK {
		t.Fatalf("expected GetPropertygraphStream 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"records\"") {
		t.Fatalf("expected records in GetPropertygraphStream response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/propertygraph/statistics", nil)
	if status != http.StatusOK {
		t.Fatalf("expected DeletePropertygraphStatistics 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/sparql/statistics", []byte(`{"mode":"enableAutoCompute"}`))
	if status != http.StatusOK {
		t.Fatalf("expected ManageSparqlStatistics 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/sparql/statistics", nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetSparqlStatistics 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(
		t,
		ts,
		http.MethodGet,
		"/sparql/stream?limit=2&iteratorType=LATEST&commitNum=1&opNum=1",
		nil,
	)
	if status != http.StatusOK {
		t.Fatalf("expected GetSparqlStream 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "\"records\"") {
		t.Fatalf("expected records in GetSparqlStream response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/sparql/statistics", nil)
	if status != http.StatusOK {
		t.Fatalf("expected DeleteSparqlStatistics 200, got %d: %s", status, body)
	}
}

func TestNeptuneDataStage5MLLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneDataCall(t, ts, http.MethodPost, "/ml/dataprocessing", []byte(`{
		"id":"stage5-dp-job",
		"inputDataS3Location":"s3://stackyard-neptunedata/stage5/raw",
		"processedDataS3Location":"s3://stackyard-neptunedata/stage5/processed"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected StartMLDataProcessingJob 200, got %d: %s", status, body)
	}
	dpID := jsonTagValue(body, "id")
	if dpID == "" {
		t.Fatalf("expected data processing id in response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/dataprocessing?maxItems=5", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListMLDataProcessingJobs 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, dpID) {
		t.Fatalf("expected data processing id in list response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/dataprocessing/"+dpID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetMLDataProcessingJob 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, dpID) {
		t.Fatalf("expected data processing id in get response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/ml/dataprocessing/"+dpID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelMLDataProcessingJob 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/modeltraining", []byte(`{
		"id":"stage5-train-job",
		"dataProcessingJobId":"`+dpID+`",
		"trainModelS3Location":"s3://stackyard-neptunedata/stage5/model"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected StartMLModelTrainingJob 200, got %d: %s", status, body)
	}
	trainID := jsonTagValue(body, "id")
	if trainID == "" {
		t.Fatalf("expected model training id in response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/modeltraining?maxItems=5", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListMLModelTrainingJobs 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, trainID) {
		t.Fatalf("expected model training id in list response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/modeltraining/"+trainID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetMLModelTrainingJob 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, trainID) {
		t.Fatalf("expected model training id in get response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/ml/modeltraining/"+trainID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelMLModelTrainingJob 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/modeltransform", []byte(`{
		"id":"stage5-transform-job",
		"modelTransformOutputS3Location":"s3://stackyard-neptunedata/stage5/transform"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected StartMLModelTransformJob 200, got %d: %s", status, body)
	}
	transformID := jsonTagValue(body, "id")
	if transformID == "" {
		t.Fatalf("expected model transform id in response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/modeltransform?maxItems=5", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListMLModelTransformJobs 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, transformID) {
		t.Fatalf("expected model transform id in list response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/modeltransform/"+transformID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetMLModelTransformJob 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, transformID) {
		t.Fatalf("expected model transform id in get response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/ml/modeltransform/"+transformID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected CancelMLModelTransformJob 200, got %d: %s", status, body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodPost, "/ml/endpoints", []byte(`{
		"id":"stage5-endpoint",
		"mlModelTrainingJobId":"`+trainID+`",
		"modelName":"stage5-model"
	}`))
	if status != http.StatusOK {
		t.Fatalf("expected CreateMLEndpoint 200, got %d: %s", status, body)
	}
	endpointID := jsonTagValue(body, "id")
	if endpointID == "" {
		t.Fatalf("expected endpoint id in response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/endpoints?maxItems=5", nil)
	if status != http.StatusOK {
		t.Fatalf("expected ListMLEndpoints 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, endpointID) {
		t.Fatalf("expected endpoint id in list response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodGet, "/ml/endpoints/"+endpointID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected GetMLEndpoint 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, endpointID) {
		t.Fatalf("expected endpoint id in get response: %s", body)
	}

	status, body = neptuneDataCall(t, ts, http.MethodDelete, "/ml/endpoints/"+endpointID, nil)
	if status != http.StatusOK {
		t.Fatalf("expected DeleteMLEndpoint 200, got %d: %s", status, body)
	}
}
