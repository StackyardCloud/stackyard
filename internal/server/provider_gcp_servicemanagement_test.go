package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPServiceManagementRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	base := "/gcp/v1/services/stackyard.googleapis.com"

	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, "/gcp/v1/services?pageSize=1", nil, "services")
	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, base, nil, "serviceName")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, "/gcp/v1/services", []byte(`{
		"serviceName":"stackyard.googleapis.com",
		"producerProjectId":"stackyard-project"
	}`), "operations/")
	assertGCPServiceManagementSuccess(t, ts, http.MethodDelete, base, nil, "operations/")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+":undelete", []byte(`{}`), "operations/")

	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, base+"/configs?pageSize=1", nil, "serviceConfigs")
	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, base+"/configs/2026-01-01r0?view=FULL", nil, "sourceInfo")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+"/configs", []byte(`{
		"name":"stackyard.googleapis.com",
		"id":"2026-01-03r0",
		"title":"Stackyard Config"
	}`), "2026-01-03r0")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+"/configs:submit", []byte(`{
		"serviceName":"stackyard.googleapis.com",
		"configSource":{
			"files":[{"filePath":"service.yaml","fileContents":"Y29uZmlnVmVyc2lvbjogMw=="}]
		}
	}`), "operations/")

	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, base+"/rollouts?pageSize=1&filter=status=SUCCESS", nil, "rollouts")
	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, base+"/rollouts/2026-01-01r0", nil, "rolloutId")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+"/rollouts", []byte(`{
		"rolloutId":"2026-01-04r0",
		"trafficPercentStrategy":{"percentages":{"2026-01-03r0":100}}
	}`), "operations/")

	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, "/gcp/v1/services:generateConfigReport", []byte(`{
		"newConfig":{"@type":"type.googleapis.com/google.api.Service","name":"stackyard.googleapis.com"}
	}`), "changeReports")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+":getIamPolicy", []byte(`{"resource":"services/stackyard.googleapis.com"}`), "bindings")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+":setIamPolicy", []byte(`{
		"resource":"services/stackyard.googleapis.com",
		"policy":{"version":1,"bindings":[{"role":"roles/viewer","members":["user:tester@example.com"]}]}
	}`), "bindings")
	assertGCPServiceManagementSuccess(t, ts, http.MethodPost, base+":testIamPermissions", []byte(`{
		"resource":"services/stackyard.googleapis.com",
		"permissions":["servicemanagement.services.get"]
	}`), "permissions")

	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", nil, "operations")
	assertGCPServiceManagementSuccess(t, ts, http.MethodGet, "/gcp/v1/operations/servicemanagement-create-service", nil, "operations/")
}

func TestGCPServiceManagementRouter_CreateServiceRequiresServiceName(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services", []byte(`{"producerProjectId":"stackyard-project"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement create service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_ListServicesRejectsOutOfRangePageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services?pageSize=999", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement list services, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_ListServiceRolloutsRequiresFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/stackyard.googleapis.com/rollouts", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement list rollouts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_GetServiceConfigRejectsInvalidView(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/stackyard.googleapis.com/configs/2026-01-01r0?view=UNKNOWN", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement get service config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_SubmitConfigSourceRequiresConfigSource(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com/configs:submit", []byte(`{"serviceName":"stackyard.googleapis.com"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement submit config source, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_GenerateConfigReportRequiresNewConfig(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services:generateConfigReport", []byte(`{"oldConfig":{"name":"stackyard.googleapis.com"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement generate config report, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_SetIAMPolicyRequiresPolicy(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:setIamPolicy", []byte(`{"resource":"services/stackyard.googleapis.com"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement setIamPolicy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_TestIAMPermissionsRequiresPermissions(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:testIamPermissions", []byte(`{"resource":"services/stackyard.googleapis.com","permissions":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement testIamPermissions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_ListOperationsRejectsMalformedFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations?filter=status==SUCCESS", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicemanagement list operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceManagementRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceManagementContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services", []byte(`{
		"serviceName":"stackyard.googleapis.com",
		"producerProjectId":"stackyard-project"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement create service, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}
	metadata, ok := createBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation metadata object, got %#v", createBody["metadata"])
	}
	if _, ok := metadata["resourceNames"].([]any); !ok {
		t.Fatalf("expected metadata.resourceNames array, got %#v", metadata["resourceNames"])
	}

	configResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/stackyard.googleapis.com/configs/2026-01-01r0?view=FULL", nil, headers)
	if configResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement get service config, got %d body=%s", configResp.StatusCode, string(providerContractBody(t, configResp)))
	}
	configBody := providerContractJSONMap(t, configResp)
	if _, ok := configBody["id"].(string); !ok {
		t.Fatalf("expected service config id string, got %#v", configBody["id"])
	}
	if _, ok := configBody["sourceInfo"].(map[string]any); !ok {
		t.Fatalf("expected sourceInfo object, got %#v", configBody["sourceInfo"])
	}

	rolloutResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/stackyard.googleapis.com/rollouts/2026-01-01r0", nil, headers)
	if rolloutResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement get rollout, got %d body=%s", rolloutResp.StatusCode, string(providerContractBody(t, rolloutResp)))
	}
	rolloutBody := providerContractJSONMap(t, rolloutResp)
	if _, ok := rolloutBody["status"].(string); !ok {
		t.Fatalf("expected rollout status string, got %#v", rolloutBody["status"])
	}
	if _, ok := rolloutBody["trafficPercentStrategy"].(map[string]any); !ok {
		t.Fatalf("expected rollout trafficPercentStrategy object, got %#v", rolloutBody["trafficPercentStrategy"])
	}

	reportResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services:generateConfigReport", []byte(`{
		"newConfig":{"@type":"type.googleapis.com/google.api.Service","name":"stackyard.googleapis.com"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if reportResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement generate report, got %d body=%s", reportResp.StatusCode, string(providerContractBody(t, reportResp)))
	}
	reportBody := providerContractJSONMap(t, reportResp)
	if _, ok := reportBody["changeReports"].([]any); !ok {
		t.Fatalf("expected changeReports array, got %#v", reportBody["changeReports"])
	}
	if _, ok := reportBody["diagnostics"].([]any); !ok {
		t.Fatalf("expected diagnostics array, got %#v", reportBody["diagnostics"])
	}

	iamResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:getIamPolicy", []byte(`{"resource":"services/stackyard.googleapis.com"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicemanagement",
	})
	if iamResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement getIamPolicy, got %d body=%s", iamResp.StatusCode, string(providerContractBody(t, iamResp)))
	}
	iamBody := providerContractJSONMap(t, iamResp)
	if _, ok := iamBody["bindings"].([]any); !ok {
		t.Fatalf("expected bindings array, got %#v", iamBody["bindings"])
	}

	opsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", nil, headers)
	if opsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement list operations, got %d body=%s", opsResp.StatusCode, string(providerContractBody(t, opsResp)))
	}
	opsBody := providerContractJSONMap(t, opsResp)
	operations, ok := opsBody["operations"].([]any)
	if !ok || len(operations) == 0 {
		t.Fatalf("expected operations array, got %#v", opsBody["operations"])
	}
	firstOp, _ := operations[0].(map[string]any)
	if _, ok := firstOp["name"].(string); !ok {
		t.Fatalf("expected operations[0].name string, got %#v", firstOp["name"])
	}
}

func TestGCPServiceManagementRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/services/stackyard.googleapis.com?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "servicemanagement" {
		t.Fatalf("expected service=servicemanagement, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["serviceName"].(string); !ok {
		t.Fatalf("expected typed serviceName in probe response, got %#v", body["serviceName"])
	}
}

func newGCPServiceManagementContractServer(t *testing.T) *httptest.Server {
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

func assertGCPServiceManagementSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicemanagement",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicemanagement router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
