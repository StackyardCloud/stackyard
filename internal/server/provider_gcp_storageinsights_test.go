package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPStorageInsightsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)

	locationParent := "/gcp/v1/projects/stackyard/locations/us-central1"
	reportConfigName := locationParent + "/reportConfigs/reportconfig1"
	reportDetailName := reportConfigName + "/reportDetails/reportdetail1"
	datasetConfigName := locationParent + "/datasetConfigs/datasetconfig1"
	operationName := locationParent + "/operations/createDatasetConfig.datasetconfig1"

	reportConfigPayload := []byte(`{
		"reportConfig":{
			"displayName":"Stackyard Report Config",
			"frequencyOptions":{"frequency":"DAILY"},
			"csvOptions":{"recordSeparator":"\n","delimiter":",","headerRequired":true},
			"objectMetadataReportOptions":{"metadataFields":["name","size","updated"]}
		}
	}`)
	updateReportConfigPayload := []byte(`{
		"reportConfig":{
			"name":"projects/stackyard/locations/us-central1/reportConfigs/reportconfig1",
			"displayName":"Updated Report Config",
			"frequencyOptions":{"frequency":"DAILY"},
			"csvOptions":{"recordSeparator":"\n","delimiter":",","headerRequired":true},
			"objectMetadataReportOptions":{"metadataFields":["name","size","updated"]}
		}
	}`)
	createDatasetConfigPayload := []byte(`{
		"datasetConfig":{
			"description":"Stackyard dataset config",
			"sourceProjects":{"projectNumbers":[123456789]}
		}
	}`)
	updateDatasetConfigPayload := []byte(`{
		"datasetConfig":{
			"name":"projects/stackyard/locations/us-central1/datasetConfigs/datasetconfig1",
			"description":"Updated dataset config",
			"sourceProjects":{"projectNumbers":[123456789]}
		}
	}`)

	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, locationParent, nil, "us-central1")

	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, locationParent+"/reportConfigs?pageSize=1", nil, "reportConfigs")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, reportConfigName, nil, "reportConfigs/reportconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPost, locationParent+"/reportConfigs?requestId=11111111-1111-4111-8111-111111111111", reportConfigPayload, "reportConfigs/reportconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPatch, reportConfigName+"?updateMask=displayName&requestId=22222222-2222-4222-8222-222222222222", updateReportConfigPayload, "reportConfigs/reportconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, reportConfigName+"/reportDetails?pageSize=1", nil, "reportDetails")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, reportDetailName, nil, "reportDetails/reportdetail1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodDelete, reportConfigName+"?force=true", nil, "{}")

	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, locationParent+"/datasetConfigs?pageSize=1", nil, "datasetConfigs")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, datasetConfigName, nil, "datasetConfigs/datasetconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPost, locationParent+"/datasetConfigs?datasetConfigId=datasetconfig1&requestId=33333333-3333-4333-8333-333333333333", createDatasetConfigPayload, "operations/createDatasetConfig.datasetconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPatch, datasetConfigName+"?updateMask=description&requestId=44444444-4444-4444-8444-444444444444", updateDatasetConfigPayload, "operations/updateDatasetConfig.datasetconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPost, datasetConfigName+":linkDataset", []byte(`{}`), "operations/linkDataset.datasetconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPost, datasetConfigName+":unlinkDataset", []byte(`{}`), "operations/unlinkDataset.datasetconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodDelete, datasetConfigName+"?requestId=55555555-5555-4555-8555-555555555555", nil, "operations/deleteDatasetConfig.datasetconfig1")

	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, locationParent+"/operations?pageSize=1", nil, "operations")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodGet, operationName, nil, "createDatasetConfig.datasetconfig1")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodPost, operationName+":cancel", []byte(`{}`), "{}")
	assertGCPStorageInsightsSuccess(t, ts, http.MethodDelete, operationName, nil, "{}")
}

func TestGCPStorageInsightsRouter_ListReportConfigsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storageinsights",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storageinsights list report configs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageInsightsRouter_CreateReportConfigRejectsInvalidRequestID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs?requestId=invalid", []byte(`{
		"reportConfig":{
			"displayName":"Stackyard Report Config",
			"frequencyOptions":{"frequency":"DAILY"},
			"csvOptions":{"recordSeparator":"\n","delimiter":",","headerRequired":true},
			"objectMetadataReportOptions":{"metadataFields":["name","size","updated"]}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storageinsights",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storageinsights create report config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageInsightsRouter_CreateDatasetConfigRequiresDatasetConfigID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/datasetConfigs", []byte(`{
		"datasetConfig":{"sourceProjects":{"projectNumbers":[123456789]}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storageinsights",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storageinsights create dataset config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageInsightsRouter_CreateDatasetConfigRejectsHyphenatedID(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/datasetConfigs?datasetConfigId=dataset-config-1", []byte(`{
		"datasetConfig":{"sourceProjects":{"projectNumbers":[123456789]}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storageinsights",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storageinsights create dataset config with invalid id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageInsightsRouter_UpdateDatasetConfigRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/datasetConfigs/datasetconfig1", []byte(`{
		"datasetConfig":{
			"name":"projects/stackyard/locations/us-central1/datasetConfigs/datasetconfig1",
			"sourceProjects":{"projectNumbers":[123456789]}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "storageinsights",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp storageinsights update dataset config without updateMask, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageInsightsRouter_GetOperationNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/missing-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "storageinsights",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp storageinsights get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPStorageInsightsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPStorageInsightsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storageinsights",
	}

	reportConfigResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs/reportconfig1", nil, headers)
	if reportConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights get report config, got %d body=%s", reportConfigResp.StatusCode, string(providerContractBody(t, reportConfigResp)))
	}
	reportConfigBody := providerContractJSONMap(t, reportConfigResp)
	if _, ok := reportConfigBody["name"].(string); !ok {
		t.Fatalf("expected report config name string, got %#v", reportConfigBody["name"])
	}
	if _, ok := reportConfigBody["createTime"].(string); !ok {
		t.Fatalf("expected report config createTime string, got %#v", reportConfigBody["createTime"])
	}
	if _, ok := reportConfigBody["frequencyOptions"].(map[string]any); !ok {
		t.Fatalf("expected report config frequencyOptions object, got %#v", reportConfigBody["frequencyOptions"])
	}
	if _, ok := reportConfigBody["objectMetadataReportOptions"].(map[string]any); !ok {
		t.Fatalf("expected report config objectMetadataReportOptions object, got %#v", reportConfigBody["objectMetadataReportOptions"])
	}

	listReportConfigsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs?pageSize=1", nil, headers)
	if listReportConfigsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights list report configs, got %d body=%s", listReportConfigsResp.StatusCode, string(providerContractBody(t, listReportConfigsResp)))
	}
	listReportConfigsBody := providerContractJSONMap(t, listReportConfigsResp)
	if _, ok := listReportConfigsBody["reportConfigs"].([]any); !ok {
		t.Fatalf("expected reportConfigs array, got %#v", listReportConfigsBody["reportConfigs"])
	}
	if _, ok := listReportConfigsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listReportConfigsBody["nextPageToken"])
	}
	if _, ok := listReportConfigsBody["unreachable"].([]any); !ok {
		t.Fatalf("expected unreachable array, got %#v", listReportConfigsBody["unreachable"])
	}

	reportDetailResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/reportConfigs/reportconfig1/reportDetails/reportdetail1", nil, headers)
	if reportDetailResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights get report detail, got %d body=%s", reportDetailResp.StatusCode, string(providerContractBody(t, reportDetailResp)))
	}
	reportDetailBody := providerContractJSONMap(t, reportDetailResp)
	if _, ok := reportDetailBody["name"].(string); !ok {
		t.Fatalf("expected report detail name string, got %#v", reportDetailBody["name"])
	}
	if _, ok := reportDetailBody["snapshotTime"].(string); !ok {
		t.Fatalf("expected report detail snapshotTime string, got %#v", reportDetailBody["snapshotTime"])
	}
	if _, ok := reportDetailBody["status"].(map[string]any); !ok {
		t.Fatalf("expected report detail status object, got %#v", reportDetailBody["status"])
	}
	if _, ok := reportDetailBody["reportMetrics"].(map[string]any); !ok {
		t.Fatalf("expected report detail reportMetrics object, got %#v", reportDetailBody["reportMetrics"])
	}

	datasetConfigResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datasetConfigs/datasetconfig1", nil, headers)
	if datasetConfigResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights get dataset config, got %d body=%s", datasetConfigResp.StatusCode, string(providerContractBody(t, datasetConfigResp)))
	}
	datasetConfigBody := providerContractJSONMap(t, datasetConfigResp)
	if _, ok := datasetConfigBody["name"].(string); !ok {
		t.Fatalf("expected dataset config name string, got %#v", datasetConfigBody["name"])
	}
	if _, ok := datasetConfigBody["uid"].(string); !ok {
		t.Fatalf("expected dataset config uid string, got %#v", datasetConfigBody["uid"])
	}
	if _, ok := datasetConfigBody["identity"].(map[string]any); !ok {
		t.Fatalf("expected dataset config identity object, got %#v", datasetConfigBody["identity"])
	}
	if _, ok := datasetConfigBody["link"].(map[string]any); !ok {
		t.Fatalf("expected dataset config link object, got %#v", datasetConfigBody["link"])
	}

	operationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/createDatasetConfig.datasetconfig1", nil, headers)
	if operationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights get operation, got %d body=%s", operationResp.StatusCode, string(providerContractBody(t, operationResp)))
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
	if _, ok := metadata["target"].(string); !ok {
		t.Fatalf("expected operation metadata target string, got %#v", metadata["target"])
	}

	locationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, headers)
	if locationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights get location, got %d body=%s", locationResp.StatusCode, string(providerContractBody(t, locationResp)))
	}
	locationBody := providerContractJSONMap(t, locationResp)
	if _, ok := locationBody["name"].(string); !ok {
		t.Fatalf("expected location name string, got %#v", locationBody["name"])
	}
	locationMetadata, ok := locationBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected location metadata object, got %#v", locationBody["metadata"])
	}
	if _, ok := locationMetadata["reportConfigAvailable"].(bool); !ok {
		t.Fatalf("expected location metadata reportConfigAvailable bool, got %#v", locationMetadata["reportConfigAvailable"])
	}
	if _, ok := locationMetadata["datasetConfigAvailable"].(bool); !ok {
		t.Fatalf("expected location metadata datasetConfigAvailable bool, got %#v", locationMetadata["datasetConfigAvailable"])
	}
}

func TestGCPStorageInsightsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/storageinsights?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "storageinsights" {
		t.Fatalf("expected service=storageinsights, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPStorageInsightsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPStorageInsightsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "storageinsights",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp storageinsights router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
