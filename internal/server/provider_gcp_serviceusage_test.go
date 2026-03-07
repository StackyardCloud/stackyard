package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPServiceUsageRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	serviceName := "projects/stackyard/services/serviceusage.googleapis.com"

	assertGCPServiceUsageSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services?pageSize=1&filter=state:ENABLED", nil, "services")
	assertGCPServiceUsageSuccess(t, ts, http.MethodGet, "/gcp/v1/"+serviceName, nil, serviceName)
	assertGCPServiceUsageSuccess(t, ts, http.MethodPost, "/gcp/v1/"+serviceName+":enable", []byte(`{
		"name":"projects/stackyard/services/serviceusage.googleapis.com"
	}`), `"type.googleapis.com/google.api.serviceusage.v1.EnableServiceResponse"`)
	assertGCPServiceUsageSuccess(t, ts, http.MethodPost, "/gcp/v1/"+serviceName+":disable", []byte(`{
		"name":"projects/stackyard/services/serviceusage.googleapis.com",
		"checkIfServiceHasUsage":"SKIP"
	}`), `"type.googleapis.com/google.api.serviceusage.v1.DisableServiceResponse"`)
	assertGCPServiceUsageSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/services:batchEnable", []byte(`{
		"parent":"projects/stackyard",
		"serviceIds":["serviceusage.googleapis.com","stackyard.googleapis.com"]
	}`), `"type.googleapis.com/google.api.serviceusage.v1.BatchEnableServicesResponse"`)
	assertGCPServiceUsageSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services:batchGet?names=projects/stackyard/services/serviceusage.googleapis.com", nil, "serviceusage.googleapis.com")
	assertGCPServiceUsageSuccess(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", nil, "operations")
	assertGCPServiceUsageSuccess(t, ts, http.MethodGet, "/gcp/v1/operations/serviceusage-enable-serviceusage.googleapis.com", nil, "operations/serviceusage-enable-serviceusage.googleapis.com")
}

func TestGCPServiceUsageRouter_ListServicesRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services?pageSize=0", nil, map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage list services, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"OutOfRange"`) {
		t.Fatalf("expected OutOfRange error in response")
	}
}

func TestGCPServiceUsageRouter_ListServicesRejectsInvalidFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services?filter=state==ENABLED", nil, map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage list services, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_EnableServiceRequiresName(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/services/serviceusage.googleapis.com:enable", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage enable service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_EnableServiceRejectsMismatchedName(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/services/serviceusage.googleapis.com:enable", []byte(`{
		"name":"projects/other/services/serviceusage.googleapis.com"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage enable service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_DisableServiceRejectsInvalidUsageCheck(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/services/serviceusage.googleapis.com:disable", []byte(`{
		"name":"projects/stackyard/services/serviceusage.googleapis.com",
		"checkIfServiceHasUsage":99
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage disable service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_BatchEnableRejectsTooManyServices(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	payload := `{"parent":"projects/stackyard","serviceIds":[` +
		`"serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com",` +
		`"serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com",` +
		`"serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com",` +
		`"serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com","serviceusage.googleapis.com",` +
		`"serviceusage.googleapis.com"]}`
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/services:batchEnable", []byte(payload), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage batch enable, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"OutOfRange"`) {
		t.Fatalf("expected OutOfRange error in response")
	}
}

func TestGCPServiceUsageRouter_BatchGetRejectsMissingNames(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services:batchGet", nil, map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage batch get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_BatchGetRejectsMismatchedParent(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services:batchGet?names=projects/other/services/serviceusage.googleapis.com", nil, map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage batch get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_ListOperationsRejectsInvalidName(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations?name=projects/stackyard", nil, map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp serviceusage list operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPServiceUsageRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceUsageContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	}
	serviceName := "projects/stackyard/services/serviceusage.googleapis.com"

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+serviceName, nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp serviceusage get service, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["name"].(string); !ok {
		t.Fatalf("expected service name string, got %#v", getBody["name"])
	}
	if _, ok := getBody["parent"].(string); !ok {
		t.Fatalf("expected service parent string, got %#v", getBody["parent"])
	}
	if _, ok := getBody["state"].(string); !ok {
		t.Fatalf("expected service state string, got %#v", getBody["state"])
	}
	if _, ok := getBody["config"].(map[string]any); !ok {
		t.Fatalf("expected service config object, got %#v", getBody["config"])
	}

	enableResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+serviceName+":enable", []byte(`{
		"name":"projects/stackyard/services/serviceusage.googleapis.com"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "serviceusage",
	})
	if enableResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp serviceusage enable service, got %d body=%s", enableResp.StatusCode, string(providerContractBody(t, enableResp)))
	}
	enableBody := providerContractJSONMap(t, enableResp)
	if _, ok := enableBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", enableBody["name"])
	}
	if _, ok := enableBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", enableBody["done"])
	}
	metadata, ok := enableBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation metadata object, got %#v", enableBody["metadata"])
	}
	if _, ok := metadata["resourceNames"].([]any); !ok {
		t.Fatalf("expected operation metadata.resourceNames array, got %#v", metadata["resourceNames"])
	}
	response, ok := enableBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation response object, got %#v", enableBody["response"])
	}
	if _, ok := response["@type"].(string); !ok {
		t.Fatalf("expected operation response @type string, got %#v", response["@type"])
	}

	batchGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services:batchGet?names="+serviceName, nil, headers)
	if batchGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp serviceusage batch get, got %d body=%s", batchGetResp.StatusCode, string(providerContractBody(t, batchGetResp)))
	}
	batchGetBody := providerContractJSONMap(t, batchGetResp)
	services, ok := batchGetBody["services"].([]any)
	if !ok || len(services) == 0 {
		t.Fatalf("expected services array, got %#v", batchGetBody["services"])
	}
	firstService, _ := services[0].(map[string]any)
	if _, ok := firstService["name"].(string); !ok {
		t.Fatalf("expected services[0].name string, got %#v", firstService["name"])
	}

	opsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", nil, headers)
	if opsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp serviceusage list operations, got %d body=%s", opsResp.StatusCode, string(providerContractBody(t, opsResp)))
	}
	opsBody := providerContractJSONMap(t, opsResp)
	operations, ok := opsBody["operations"].([]any)
	if !ok || len(operations) == 0 {
		t.Fatalf("expected operations array, got %#v", opsBody["operations"])
	}
	firstOperation, _ := operations[0].(map[string]any)
	if _, ok := firstOperation["name"].(string); !ok {
		t.Fatalf("expected operations[0].name string, got %#v", firstOperation["name"])
	}
	if _, ok := opsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", opsBody["nextPageToken"])
	}
}

func TestGCPServiceUsageRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/services/serviceusage.googleapis.com?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp serviceusage contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "serviceusage" {
		t.Fatalf("expected service=serviceusage, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPServiceUsageContractServer(t *testing.T) *httptest.Server {
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

func assertGCPServiceUsageSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "serviceusage",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp serviceusage router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
