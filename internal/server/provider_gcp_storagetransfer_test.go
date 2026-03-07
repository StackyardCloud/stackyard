package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPStorageTransferRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)

	transferJobName := "transferJobs/job-1"
	operationName := "transferOperations/run.job-1"
	agentPoolName := "projects/stackyard/agentPools/agentpool-1"

	createTransferJobPayload := []byte(`{
		"transferJob":{
			"name":"transferJobs/job-1",
			"projectId":"stackyard",
			"description":"Stackyard transfer job",
			"status":"ENABLED",
			"transferSpec":{
				"gcsDataSource":{"bucketName":"stackyard-source-bucket"},
				"gcsDataSink":{"bucketName":"stackyard-destination-bucket"}
			}
		}
	}`)
	updateTransferJobPayload := []byte(`{
		"projectId":"stackyard",
		"transferJob":{
			"name":"transferJobs/job-1",
			"description":"Updated transfer job",
			"status":"DISABLED"
		}
	}`)
	createAgentPoolPayload := []byte(`{
		"agentPool":{
			"name":"projects/stackyard/agentPools/agentpool-1",
			"displayName":"Stackyard Agent Pool 1",
			"bandwidthLimit":{"limitMbps":250}
		}
	}`)
	updateAgentPoolPayload := []byte(`{
		"agentPool":{
			"name":"projects/stackyard/agentPools/agentpool-1",
			"displayName":"Updated Agent Pool",
			"bandwidthLimit":{"limitMbps":500}
		}
	}`)

	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/googleServiceAccounts/stackyard", nil, "accountEmail")

	assertGCPStorageTransferSuccess(t, ts, http.MethodPost, "/gcp/v1/transferJobs", createTransferJobPayload, transferJobName)
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/"+transferJobName+"?projectId=stackyard", nil, transferJobName)
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/transferJobs?filter=%7B%22projectId%22%3A%22stackyard%22%7D&pageSize=1", nil, "transferJobs")
	assertGCPStorageTransferSuccess(t, ts, http.MethodPatch, "/gcp/v1/"+transferJobName+"?updateTransferJobFieldMask=status&projectId=stackyard", updateTransferJobPayload, "DISABLED")
	assertGCPStorageTransferSuccess(t, ts, http.MethodPost, "/gcp/v1/"+transferJobName+":run", []byte(`{"projectId":"stackyard"}`), operationName)

	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/transferOperations?pageSize=1", nil, "operations")
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/"+operationName, nil, operationName)
	assertGCPStorageTransferSuccess(t, ts, http.MethodPost, "/gcp/v1/"+operationName+":pause", []byte(`{"projectId":"stackyard"}`), "{}")
	assertGCPStorageTransferSuccess(t, ts, http.MethodPost, "/gcp/v1/transferOperations/run.job-1-paused:resume", []byte(`{"projectId":"stackyard"}`), "{}")
	assertGCPStorageTransferSuccess(t, ts, http.MethodPost, "/gcp/v1/"+operationName+":cancel", []byte(`{}`), "{}")
	assertGCPStorageTransferSuccess(t, ts, http.MethodDelete, "/gcp/v1/"+operationName, nil, "{}")

	assertGCPStorageTransferSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/agentPools?agentPoolId=agentpool-1", createAgentPoolPayload, agentPoolName)
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/agentPools?pageSize=1", nil, "agentPools")
	assertGCPStorageTransferSuccess(t, ts, http.MethodGet, "/gcp/v1/"+agentPoolName, nil, agentPoolName)
	assertGCPStorageTransferSuccess(t, ts, http.MethodPatch, "/gcp/v1/"+agentPoolName+"?updateMask=displayName", updateAgentPoolPayload, "Updated Agent Pool")
	assertGCPStorageTransferSuccess(t, ts, http.MethodDelete, "/gcp/v1/"+agentPoolName, nil, "{}")

	assertGCPStorageTransferSuccess(t, ts, http.MethodDelete, "/gcp/v1/"+transferJobName+"?projectId=stackyard", nil, "{}")
}

func TestGCPStorageTransferRouter_ListTransferJobsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferJobs?filter=%7B%22projectId%22%3A%22stackyard%22%7D&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer list transfer jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_ListTransferJobsOversizedPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferJobs?filter=%7B%22projectId%22%3A%22stackyard%22%7D&pageSize=5001", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer list transfer jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_ListTransferJobsInvalidFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferJobs?filter=invalid-json", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer list transfer jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_CreateTransferJobRequiresTransferSpec(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/transferJobs", []byte(`{
		"transferJob":{
			"name":"transferJobs/job-1",
			"projectId":"stackyard"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer create transfer job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_UpdateTransferJobRequiresMask(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/transferJobs/job-1", []byte(`{
		"projectId":"stackyard",
		"transferJob":{"name":"transferJobs/job-1"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer update transfer job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_RunTransferJobRequiresProjectID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/transferJobs/job-1:run", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer run transfer job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_PauseTransferOperationAlreadyPaused(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/transferOperations/run.job-1-paused:pause", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer pause transfer operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_CreateAgentPoolRequiresAgentPoolID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/agentPools", []byte(`{
		"agentPool":{"displayName":"Pool"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer create agent pool, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_UpdateAgentPoolInvalidMask(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/agentPools/agentpool-1?updateMask=state", []byte(`{
		"agentPool":{"name":"projects/stackyard/agentPools/agentpool-1","displayName":"Pool"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storagetransfer update agent pool, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_GetOperationNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferOperations/missing-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp storagetransfer get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageTransferRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageTransferContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	}

	serviceAccountResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/googleServiceAccounts/stackyard", nil, headers)
	if serviceAccountResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer get google service account, got %d body=%s", serviceAccountResp.StatusCode, string(providerContractBody(t, serviceAccountResp)))
	}
	serviceAccountBody := providerContractJSONMap(t, serviceAccountResp)
	if _, ok := serviceAccountBody["accountEmail"].(string); !ok {
		t.Fatalf("expected accountEmail string, got %#v", serviceAccountBody["accountEmail"])
	}
	if _, ok := serviceAccountBody["subjectId"].(string); !ok {
		t.Fatalf("expected subjectId string, got %#v", serviceAccountBody["subjectId"])
	}

	transferJobResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferJobs/job-1?projectId=stackyard", nil, headers)
	if transferJobResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer get transfer job, got %d body=%s", transferJobResp.StatusCode, string(providerContractBody(t, transferJobResp)))
	}
	transferJobBody := providerContractJSONMap(t, transferJobResp)
	if _, ok := transferJobBody["name"].(string); !ok {
		t.Fatalf("expected transfer job name string, got %#v", transferJobBody["name"])
	}
	if _, ok := transferJobBody["projectId"].(string); !ok {
		t.Fatalf("expected transfer job projectId string, got %#v", transferJobBody["projectId"])
	}
	if _, ok := transferJobBody["status"].(string); !ok {
		t.Fatalf("expected transfer job status string, got %#v", transferJobBody["status"])
	}
	if _, ok := transferJobBody["transferSpec"].(map[string]any); !ok {
		t.Fatalf("expected transfer job transferSpec object, got %#v", transferJobBody["transferSpec"])
	}

	listJobsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferJobs?filter=%7B%22projectId%22%3A%22stackyard%22%7D&pageSize=1", nil, headers)
	if listJobsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer list transfer jobs, got %d body=%s", listJobsResp.StatusCode, string(providerContractBody(t, listJobsResp)))
	}
	listJobsBody := providerContractJSONMap(t, listJobsResp)
	if _, ok := listJobsBody["transferJobs"].([]any); !ok {
		t.Fatalf("expected transferJobs array, got %#v", listJobsBody["transferJobs"])
	}
	if _, ok := listJobsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listJobsBody["nextPageToken"])
	}

	runResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/transferJobs/job-1:run", []byte(`{"projectId":"stackyard"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storagetransfer",
	})
	if runResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer run transfer job, got %d body=%s", runResp.StatusCode, string(providerContractBody(t, runResp)))
	}
	runBody := providerContractJSONMap(t, runResp)
	if _, ok := runBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", runBody["name"])
	}
	if _, ok := runBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", runBody["done"])
	}
	if _, ok := runBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata object, got %#v", runBody["metadata"])
	}

	operationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/transferOperations/run.job-1", nil, headers)
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer get operation, got %d body=%s", operationResp.StatusCode, string(providerContractBody(t, operationResp)))
	}
	operationBody := providerContractJSONMap(t, operationResp)
	metadata, ok := operationBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation metadata object, got %#v", operationBody["metadata"])
	}
	if _, ok := metadata["status"].(string); !ok {
		t.Fatalf("expected operation metadata status string, got %#v", metadata["status"])
	}
	if _, ok := metadata["transferJobName"].(string); !ok {
		t.Fatalf("expected operation metadata transferJobName string, got %#v", metadata["transferJobName"])
	}

	agentPoolResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/agentPools/agentpool-1", nil, headers)
	if agentPoolResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer get agent pool, got %d body=%s", agentPoolResp.StatusCode, string(providerContractBody(t, agentPoolResp)))
	}
	agentPoolBody := providerContractJSONMap(t, agentPoolResp)
	if _, ok := agentPoolBody["name"].(string); !ok {
		t.Fatalf("expected agent pool name string, got %#v", agentPoolBody["name"])
	}
	if _, ok := agentPoolBody["displayName"].(string); !ok {
		t.Fatalf("expected agent pool displayName string, got %#v", agentPoolBody["displayName"])
	}
	if _, ok := agentPoolBody["state"].(string); !ok {
		t.Fatalf("expected agent pool state string, got %#v", agentPoolBody["state"])
	}
	if _, ok := agentPoolBody["bandwidthLimit"].(map[string]any); !ok {
		t.Fatalf("expected agent pool bandwidthLimit object, got %#v", agentPoolBody["bandwidthLimit"])
	}

	listAgentPoolsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/agentPools?pageSize=1", nil, headers)
	if listAgentPoolsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer list agent pools, got %d body=%s", listAgentPoolsResp.StatusCode, string(providerContractBody(t, listAgentPoolsResp)))
	}
	listAgentPoolsBody := providerContractJSONMap(t, listAgentPoolsResp)
	if _, ok := listAgentPoolsBody["agentPools"].([]any); !ok {
		t.Fatalf("expected agentPools array, got %#v", listAgentPoolsBody["agentPools"])
	}
	if _, ok := listAgentPoolsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listAgentPoolsBody["nextPageToken"])
	}
}

func TestGCPStorageTransferRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/storagetransfer?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "storagetransfer" {
		t.Fatalf("expected service=storagetransfer, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPStorageTransferContractServer(t *testing.T) *httptest.Server {
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

func assertGCPStorageTransferSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storagetransfer",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storagetransfer router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
