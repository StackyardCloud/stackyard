package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPWebRiskRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	project := "123456789"
	parent := "/gcp/v1/projects/" + project

	assertGCPWebRiskSuccess(t, ts, http.MethodGet, "/gcp/v1/threatLists:computeDiff?threatType=MALWARE&constraints.maxDiffEntries=1024&constraints.maxDatabaseEntries=2048", nil, "newVersionToken")
	assertGCPWebRiskSuccess(t, ts, http.MethodGet, "/gcp/v1/uris:search?uri=http://phish.stackyard.test/path&threatTypes=SOCIAL_ENGINEERING", nil, `"threat"`)
	assertGCPWebRiskSuccess(t, ts, http.MethodGet, "/gcp/v1/hashes:search?hashPrefix=%5B1+2+3+4%5D&threatTypes=MALWARE", nil, "negativeExpireTime")
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, parent+"/submissions", []byte(`{
		"uri": "http://phish.stackyard.test/report"
	}`), "threatTypes")
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, parent+"/uris:submit", []byte(`{
		"parent": "projects/123456789",
		"submission": {
			"uri": "http://phish.stackyard.test/report"
		}
	}`), "operations/submitUri.op-1")

	assertGCPWebRiskSuccess(t, ts, http.MethodGet, parent+"/operations?pageSize=1", nil, "operations")
	assertGCPWebRiskSuccess(t, ts, http.MethodGet, parent+"/operations/submitUri.done-2", nil, "submitUri.done-2")
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, parent+"/operations/submitUri.op-1:cancel", []byte(`{}`), "submitUri.op-1")
	assertGCPWebRiskSuccess(t, ts, http.MethodDelete, parent+"/operations/submitUri.op-1", nil, "{}")

	assertGCPWebRiskSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.webrisk.v1.WebRiskService/ComputeThreatListDiff", []byte(`{
		"threatType": 1,
		"constraints": {
			"maxDiffEntries": 1024
		}
	}`), "newVersionToken")
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.webrisk.v1.WebRiskService/SearchUris", []byte(`{
		"uri": "http://phish.stackyard.test/path",
		"threatTypes": [2]
	}`), `"threat"`)
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.webrisk.v1.WebRiskService/SearchHashes", []byte(`{
		"hashPrefix": "AQIDBA==",
		"threatTypes": [1]
	}`), "negativeExpireTime")
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.webrisk.v1.WebRiskService/CreateSubmission", []byte(`{
		"parent": "projects/123456789",
		"submission": {
			"uri": "http://phish.stackyard.test/report"
		}
	}`), "threatTypes")
	assertGCPWebRiskSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.webrisk.v1.WebRiskService/SubmitUri", []byte(`{
		"parent": "projects/123456789",
		"submission": {
			"uri": "http://phish.stackyard.test/report"
		}
	}`), "operations/submitUri.grpc-op-1")
}

func TestGCPWebRiskRouter_ComputeThreatListDiffRequiresConstraints(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/threatLists:computeDiff", []byte(`{"threatType":1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp webrisk compute diff, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_SearchHashesRejectsShortHashPrefix(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/hashes:search", []byte(`{
		"hashPrefix": "AQI=",
		"threatTypes": [1]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp webrisk search hashes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_CreateSubmissionRequiresURI(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/submissions", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp webrisk create submission, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_SubmitURIParentMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/uris:submit", []byte(`{
		"parent": "projects/111111111",
		"submission": {
			"uri": "http://phish.stackyard.test/report"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp webrisk submit uri, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_ListOperationsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/123456789/operations?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp webrisk list operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_GetOperationNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/123456789/operations/missing-op", nil, map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp webrisk get operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_GRPCBridgeSubmitURIRequiresSubmission(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.webrisk.v1.WebRiskService/SubmitUri", []byte(`{
		"parent": "projects/123456789"
	}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp webrisk grpc bridge submit uri, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebRiskRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPWebRiskContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "webrisk",
	}

	diffResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/threatLists:computeDiff?threatType=MALWARE&constraints.maxDiffEntries=1024", nil, map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if diffResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk compute diff, got %d body=%s", diffResp.StatusCode, string(providerContractBody(t, diffResp)))
	}
	diffBody := providerContractJSONMap(t, diffResp)
	if _, ok := diffBody["responseType"].(string); !ok {
		t.Fatalf("expected responseType string, got %#v", diffBody["responseType"])
	}
	if _, ok := diffBody["newVersionToken"].(string); !ok {
		t.Fatalf("expected newVersionToken string, got %#v", diffBody["newVersionToken"])
	}
	checksum, ok := diffBody["checksum"].(map[string]any)
	if !ok {
		t.Fatalf("expected checksum object, got %#v", diffBody["checksum"])
	}
	if _, ok := checksum["sha256"].(string); !ok {
		t.Fatalf("expected checksum.sha256 string, got %#v", checksum["sha256"])
	}

	searchURIResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/uris:search?uri=http://phish.stackyard.test/path&threatTypes=SOCIAL_ENGINEERING", nil, map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if searchURIResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk search uris, got %d body=%s", searchURIResp.StatusCode, string(providerContractBody(t, searchURIResp)))
	}
	searchURIBody := providerContractJSONMap(t, searchURIResp)
	threat, ok := searchURIBody["threat"].(map[string]any)
	if !ok {
		t.Fatalf("expected threat object, got %#v", searchURIBody["threat"])
	}
	if _, ok := threat["threatTypes"].([]any); !ok {
		t.Fatalf("expected threat.threatTypes array, got %#v", threat["threatTypes"])
	}
	if _, ok := threat["expireTime"].(string); !ok {
		t.Fatalf("expected threat.expireTime string, got %#v", threat["expireTime"])
	}

	searchHashResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/hashes:search?hashPrefix=%5B1+2+3+4%5D&threatTypes=MALWARE", nil, map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if searchHashResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk search hashes, got %d body=%s", searchHashResp.StatusCode, string(providerContractBody(t, searchHashResp)))
	}
	searchHashBody := providerContractJSONMap(t, searchHashResp)
	hashThreats, ok := searchHashBody["threats"].([]any)
	if !ok || len(hashThreats) == 0 {
		t.Fatalf("expected threats array, got %#v", searchHashBody["threats"])
	}
	firstThreat, _ := hashThreats[0].(map[string]any)
	if _, ok := firstThreat["hash"].(string); !ok {
		t.Fatalf("expected threats[0].hash string, got %#v", firstThreat["hash"])
	}
	if _, ok := searchHashBody["negativeExpireTime"].(string); !ok {
		t.Fatalf("expected negativeExpireTime string, got %#v", searchHashBody["negativeExpireTime"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/submissions", []byte(`{
		"uri": "http://phish.stackyard.test/report"
	}`), headers)
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk create submission, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["uri"].(string); !ok {
		t.Fatalf("expected submission uri string, got %#v", createBody["uri"])
	}
	if _, ok := createBody["threatTypes"].([]any); !ok {
		t.Fatalf("expected submission threatTypes array, got %#v", createBody["threatTypes"])
	}

	submitResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/123456789/uris:submit", []byte(`{
		"parent": "projects/123456789",
		"submission": {
			"uri": "http://phish.stackyard.test/report"
		}
	}`), headers)
	if submitResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk submit uri, got %d body=%s", submitResp.StatusCode, string(providerContractBody(t, submitResp)))
	}
	submitBody := providerContractJSONMap(t, submitResp)
	if _, ok := submitBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", submitBody["name"])
	}
	if _, ok := submitBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", submitBody["done"])
	}
	metadata, ok := submitBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected operation metadata object, got %#v", submitBody["metadata"])
	}
	if _, ok := metadata["state"].(string); !ok {
		t.Fatalf("expected metadata.state string, got %#v", metadata["state"])
	}

	listOpsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/123456789/operations?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	})
	if listOpsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk list operations, got %d body=%s", listOpsResp.StatusCode, string(providerContractBody(t, listOpsResp)))
	}
	listOpsBody := providerContractJSONMap(t, listOpsResp)
	operations, ok := listOpsBody["operations"].([]any)
	if !ok || len(operations) == 0 {
		t.Fatalf("expected operations array, got %#v", listOpsBody["operations"])
	}
	firstOp, _ := operations[0].(map[string]any)
	if _, ok := firstOp["name"].(string); !ok {
		t.Fatalf("expected operations[0].name string, got %#v", firstOp["name"])
	}
	if _, ok := firstOp["done"].(bool); !ok {
		t.Fatalf("expected operations[0].done bool, got %#v", firstOp["done"])
	}
}

func TestGCPWebRiskRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/webrisk?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "webrisk" {
		t.Fatalf("expected service=webrisk, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in probe response, got %#v", body["name"])
	}
}

func newGCPWebRiskContractServer(t *testing.T) *httptest.Server {
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

func assertGCPWebRiskSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "webrisk",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp webrisk router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
