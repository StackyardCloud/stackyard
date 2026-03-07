package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPStorageBatchOperationsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)

	locationParent := "/gcp/v1/projects/stackyard/locations/global"
	jobName := locationParent + "/jobs/job-1"
	bucketOperationName := jobName + "/bucketOperations/bucket-op-1"
	operationName := locationParent + "/operations/createJob.job-1"

	createPayload := []byte(`{
		"description":"Stackyard create job",
		"bucketList":{
			"buckets":[
				{
					"bucket":"stackyard-source-bucket",
					"prefixList":{"includedObjectPrefixes":["incoming/"]}
				}
			]
		},
		"deleteObject":{"permanentObjectDeletionEnabled":true}
	}`)

	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global", nil, "global")

	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, locationParent+"/jobs?pageSize=1", nil, "jobs")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, jobName, nil, "jobs/job-1")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodPost, locationParent+"/jobs?jobId=job-1&requestId=11111111-1111-4111-8111-111111111111", createPayload, "operations/createJob.job-1")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodPost, jobName+":cancel", []byte(`{"name":"projects/stackyard/locations/global/jobs/job-1"}`), "{}")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodDelete, jobName+"?force=true", nil, "{}")

	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, jobName+"/bucketOperations?pageSize=1", nil, "bucketOperations")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, bucketOperationName, nil, "bucket-op-1")

	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, locationParent+"/operations?pageSize=1", nil, "operations")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodGet, operationName, nil, "createJob.job-1")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodPost, operationName+":cancel", []byte(`{"name":"projects/stackyard/locations/global/operations/createJob.job-1"}`), "{}")
	assertGCPStorageBatchOperationsSuccess(t, ts, http.MethodDelete, operationName, nil, "{}")
}

func TestGCPStorageBatchOperationsRouter_ListJobsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/jobs?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations list jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_ListJobsOversizedPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/jobs?pageSize=5001", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations list jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_ListJobsInvalidOrderBy(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/jobs?orderBy=update_time", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations list jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_CreateJobRequiresJobID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/jobs", []byte(`{
		"bucketList":{"buckets":[{"bucket":"stackyard-source-bucket"}]},
		"deleteObject":{"permanentObjectDeletionEnabled":true}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations create job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_CreateJobRejectsInvalidRequestID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/jobs?jobId=job-1&requestId=invalid", []byte(`{
		"bucketList":{"buckets":[{"bucket":"stackyard-source-bucket"}]},
		"deleteObject":{"permanentObjectDeletionEnabled":true}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations create job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_CreateJobRequiresBucketList(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/jobs?jobId=job-1", []byte(`{
		"deleteObject":{"permanentObjectDeletionEnabled":true}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations create job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_CreateJobRejectsUnsupportedTransformation(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/jobs?jobId=job-1", []byte(`{
		"bucketList":{"buckets":[{"bucket":"stackyard-source-bucket"}]},
		"updateObjectCustomContext":{"clearAll":true}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations create job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_CancelJobTerminalStateFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/jobs/job-succeeded:cancel", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagebatchoperations cancel job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_GetOperationNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/operations/missing-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp storagebatchoperations get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageBatchOperationsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageBatchOperationsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	}

	jobResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/jobs/job-1", nil, headers)
	if jobResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations get job, got %d body=%s", jobResp.StatusCode, string(providerContractBody(t, jobResp)))
	}
	jobBody := providerContractJSONMap(t, jobResp)
	if _, ok := jobBody["name"].(string); !ok {
		t.Fatalf("expected job name string, got %#v", jobBody["name"])
	}
	if _, ok := jobBody["state"].(string); !ok {
		t.Fatalf("expected job state string, got %#v", jobBody["state"])
	}
	if _, ok := jobBody["createTime"].(string); !ok {
		t.Fatalf("expected job createTime string, got %#v", jobBody["createTime"])
	}
	if _, ok := jobBody["scheduleTime"].(string); !ok {
		t.Fatalf("expected job scheduleTime string, got %#v", jobBody["scheduleTime"])
	}
	if _, ok := jobBody["counters"].(map[string]any); !ok {
		t.Fatalf("expected job counters object, got %#v", jobBody["counters"])
	}
	if _, ok := jobBody["errorSummaries"].([]any); !ok {
		t.Fatalf("expected job errorSummaries array, got %#v", jobBody["errorSummaries"])
	}

	listJobsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/jobs?pageSize=1", nil, headers)
	if listJobsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations list jobs, got %d body=%s", listJobsResp.StatusCode, string(providerContractBody(t, listJobsResp)))
	}
	listJobsBody := providerContractJSONMap(t, listJobsResp)
	if _, ok := listJobsBody["jobs"].([]any); !ok {
		t.Fatalf("expected jobs array, got %#v", listJobsBody["jobs"])
	}
	if _, ok := listJobsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listJobsBody["nextPageToken"])
	}
	if _, ok := listJobsBody["unreachable"].([]any); !ok {
		t.Fatalf("expected unreachable array, got %#v", listJobsBody["unreachable"])
	}

	cancelJobResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/jobs/job-1:cancel", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	})
	if cancelJobResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations cancel job, got %d body=%s", cancelJobResp.StatusCode, string(providerContractBody(t, cancelJobResp)))
	}
	cancelJobBody := providerContractJSONMap(t, cancelJobResp)
	if len(cancelJobBody) != 0 {
		t.Fatalf("expected empty cancel response object, got %#v", cancelJobBody)
	}

	operationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/operations/createJob.job-succeeded", nil, headers)
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations get operation, got %d body=%s", operationResp.StatusCode, string(providerContractBody(t, operationResp)))
	}
	operationBody := providerContractJSONMap(t, operationResp)
	if _, ok := operationBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", operationBody["name"])
	}
	if _, ok := operationBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", operationBody["done"])
	}
	metadata, ok := operationBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation metadata object, got %#v", operationBody["metadata"])
	}
	if _, ok := metadata["@type"].(string); !ok {
		t.Fatalf("expected operation metadata @type string, got %#v", metadata["@type"])
	}
	if _, ok := metadata["operation"].(string); !ok {
		t.Fatalf("expected operation metadata.operation string, got %#v", metadata["operation"])
	}
	if _, ok := metadata["job"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata.job object, got %#v", metadata["job"])
	}

	locationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global", nil, headers)
	if locationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations get location, got %d body=%s", locationResp.StatusCode, string(providerContractBody(t, locationResp)))
	}
	locationBody := providerContractJSONMap(t, locationResp)
	if _, ok := locationBody["name"].(string); !ok {
		t.Fatalf("expected location name string, got %#v", locationBody["name"])
	}
	if _, ok := locationBody["locationId"].(string); !ok {
		t.Fatalf("expected locationId string, got %#v", locationBody["locationId"])
	}
}

func TestGCPStorageBatchOperationsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/storagebatchoperations?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "storagebatchoperations" {
		t.Fatalf("expected service=storagebatchoperations, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPStorageBatchOperationsContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPStorageBatchOperationsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storagebatchoperations",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagebatchoperations router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
