package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPWebSecurityScannerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)

	parent := "/gcp/v1/projects/stackyard"
	scanConfigName := parent + "/scanConfigs/scan-config-1"
	scanRunName := scanConfigName + "/scanRuns/scan-run-1"
	findingName := scanRunName + "/findings/finding-1"

	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, parent+"/scanConfigs?pageSize=1", nil, "scanConfigs")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, parent+"/scanConfigs", []byte(`{
		"displayName": "Stackyard Scan Config",
		"startingUrls": ["https://scan-config-1.stackyard.test"],
		"maxQps": 10
	}`), "scanConfigs/scan-config-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, scanConfigName, nil, "scanConfigs/scan-config-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPatch, scanConfigName+"?updateMask=display_name", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1",
		"displayName": "Stackyard Scan Config Updated",
		"startingUrls": ["https://scan-config-1.stackyard.test"],
		"maxQps": 15
	}`), "Scan Config Updated")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, scanConfigName+":start", []byte(`{}`), "scanRuns/scan-run-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, scanConfigName+"/scanRuns?pageSize=1", nil, "scanRuns")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, scanRunName, nil, "scan-run-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, scanRunName+":stop", []byte(`{}`), "\"FINISHED\"")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, scanRunName+"/crawledUrls?pageSize=1", nil, "crawledUrls")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, scanRunName+"/findings?pageSize=1&filter=finding_type=MIXED_CONTENT", nil, "findings")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, findingName, nil, "finding-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodGet, scanRunName+"/findingTypeStats", nil, "findingTypeStats")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodDelete, scanConfigName, nil, "{}")
}

func TestGCPWebSecurityScannerRouter_GRPCBridgeRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	base := "/gcp/google.cloud.websecurityscanner.v1.WebSecurityScanner"

	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/CreateScanConfig", []byte(`{
		"parent": "projects/stackyard",
		"scanConfig": {
			"displayName": "Stackyard Scan Config",
			"startingUrls": ["https://scan-config-1.stackyard.test"],
			"maxQps": 15
		}
	}`), "scanConfigs/scan-config-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/ListScanConfigs", []byte(`{
		"parent": "projects/stackyard",
		"pageSize": 1
	}`), "scanConfigs")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/GetScanConfig", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1"
	}`), "scanConfigs/scan-config-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/UpdateScanConfig", []byte(`{
		"scanConfig": {
			"name": "projects/stackyard/scanConfigs/scan-config-1",
			"displayName": "Stackyard Scan Config Updated",
			"startingUrls": ["https://scan-config-1.stackyard.test"],
			"maxQps": 15
		},
		"updateMask": {
			"paths": ["display_name"]
		}
	}`), "Scan Config Updated")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/StartScanRun", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1"
	}`), "scanRuns/scan-run-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/GetScanRun", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1"
	}`), "scan-run-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/ListScanRuns", []byte(`{
		"parent": "projects/stackyard/scanConfigs/scan-config-1",
		"pageSize": 1
	}`), "scanRuns")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/StopScanRun", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1"
	}`), "\"FINISHED\"")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/ListCrawledUrls", []byte(`{
		"parent": "projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1",
		"pageSize": 1
	}`), "crawledUrls")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/GetFinding", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/findings/finding-1"
	}`), "finding-1")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/ListFindings", []byte(`{
		"parent": "projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1",
		"filter": "finding_type=MIXED_CONTENT",
		"pageSize": 1
	}`), "findings")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/ListFindingTypeStats", []byte(`{
		"parent": "projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1"
	}`), "findingTypeStats")
	assertGCPWebSecurityScannerSuccess(t, ts, http.MethodPost, base+"/DeleteScanConfig", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1"
	}`), "{}")
}

func TestGCPWebSecurityScannerRouter_CreateScanConfigRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/scanConfigs", []byte(`{
		"startingUrls": ["https://scan-config-1.stackyard.test"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner create scan config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_CreateScanConfigRejectsInvalidStartingURL(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/scanConfigs", []byte(`{
		"displayName": "Stackyard Scan Config",
		"startingUrls": ["ftp://scan-config-1.stackyard.test"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner create scan config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_UpdateScanConfigRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1", []byte(`{
		"name": "projects/stackyard/scanConfigs/scan-config-1",
		"displayName": "Stackyard Scan Config Updated",
		"startingUrls": ["https://scan-config-1.stackyard.test"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner update scan config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_ListScanConfigsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner list scan configs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_ListScanConfigsPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs?pageToken=99", nil, map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner list scan configs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_ListFindingsRejectsInvalidFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/findings?filter=severity=HIGH", nil, map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner list findings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_GetFindingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/findings/missing-finding", nil, map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp websecurityscanner get finding, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_GRPCBridgeListScanConfigsRequiresParent(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.websecurityscanner.v1.WebSecurityScanner/ListScanConfigs", []byte(`{}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner grpc bridge list scan configs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPWebSecurityScannerRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPWebSecurityScannerContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	}

	listScanConfigsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs?pageSize=1", nil, headers)
	if listScanConfigsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner list scan configs, got %d body=%s", listScanConfigsResp.StatusCode, string(providerContractBody(t, listScanConfigsResp)))
	}
	listScanConfigsBody := providerContractJSONMap(t, listScanConfigsResp)
	scanConfigs, ok := listScanConfigsBody["scanConfigs"].([]any)
	if !ok || len(scanConfigs) == 0 {
		t.Fatalf("expected scanConfigs array, got %#v", listScanConfigsBody["scanConfigs"])
	}
	scanConfig, _ := scanConfigs[0].(map[string]any)
	if _, ok := scanConfig["name"].(string); !ok {
		t.Fatalf("expected scanConfigs[0].name string, got %#v", scanConfig["name"])
	}
	if _, ok := scanConfig["displayName"].(string); !ok {
		t.Fatalf("expected scanConfigs[0].displayName string, got %#v", scanConfig["displayName"])
	}
	if _, ok := scanConfig["startingUrls"].([]any); !ok {
		t.Fatalf("expected scanConfigs[0].startingUrls array, got %#v", scanConfig["startingUrls"])
	}

	getScanRunResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1", nil, headers)
	if getScanRunResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner get scan run, got %d body=%s", getScanRunResp.StatusCode, string(providerContractBody(t, getScanRunResp)))
	}
	getScanRunBody := providerContractJSONMap(t, getScanRunResp)
	if _, ok := getScanRunBody["executionState"].(string); !ok {
		t.Fatalf("expected executionState string, got %#v", getScanRunBody["executionState"])
	}
	if _, ok := getScanRunBody["resultState"].(string); !ok {
		t.Fatalf("expected resultState string, got %#v", getScanRunBody["resultState"])
	}
	if _, ok := getScanRunBody["progressPercent"].(float64); !ok {
		t.Fatalf("expected progressPercent number, got %#v", getScanRunBody["progressPercent"])
	}

	listCrawledURLsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/crawledUrls?pageSize=1", nil, headers)
	if listCrawledURLsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner list crawled urls, got %d body=%s", listCrawledURLsResp.StatusCode, string(providerContractBody(t, listCrawledURLsResp)))
	}
	listCrawledURLsBody := providerContractJSONMap(t, listCrawledURLsResp)
	crawledURLs, ok := listCrawledURLsBody["crawledUrls"].([]any)
	if !ok || len(crawledURLs) == 0 {
		t.Fatalf("expected crawledUrls array, got %#v", listCrawledURLsBody["crawledUrls"])
	}
	crawledURL, _ := crawledURLs[0].(map[string]any)
	if _, ok := crawledURL["httpMethod"].(string); !ok {
		t.Fatalf("expected crawledUrls[0].httpMethod string, got %#v", crawledURL["httpMethod"])
	}
	if _, ok := crawledURL["url"].(string); !ok {
		t.Fatalf("expected crawledUrls[0].url string, got %#v", crawledURL["url"])
	}

	getFindingResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/findings/finding-1", nil, headers)
	if getFindingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner get finding, got %d body=%s", getFindingResp.StatusCode, string(providerContractBody(t, getFindingResp)))
	}
	getFindingBody := providerContractJSONMap(t, getFindingResp)
	if _, ok := getFindingBody["findingType"].(string); !ok {
		t.Fatalf("expected findingType string, got %#v", getFindingBody["findingType"])
	}
	if _, ok := getFindingBody["severity"].(string); !ok {
		t.Fatalf("expected severity string, got %#v", getFindingBody["severity"])
	}

	listFindingsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/findings?pageSize=1", nil, headers)
	if listFindingsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner list findings, got %d body=%s", listFindingsResp.StatusCode, string(providerContractBody(t, listFindingsResp)))
	}
	listFindingsBody := providerContractJSONMap(t, listFindingsResp)
	findings, ok := listFindingsBody["findings"].([]any)
	if !ok || len(findings) == 0 {
		t.Fatalf("expected findings array, got %#v", listFindingsBody["findings"])
	}
	if _, ok := listFindingsBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listFindingsBody["nextPageToken"])
	}

	listStatsResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/scanConfigs/scan-config-1/scanRuns/scan-run-1/findingTypeStats", nil, headers)
	if listStatsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner list finding type stats, got %d body=%s", listStatsResp.StatusCode, string(providerContractBody(t, listStatsResp)))
	}
	listStatsBody := providerContractJSONMap(t, listStatsResp)
	stats, ok := listStatsBody["findingTypeStats"].([]any)
	if !ok || len(stats) == 0 {
		t.Fatalf("expected findingTypeStats array, got %#v", listStatsBody["findingTypeStats"])
	}
	firstStat, _ := stats[0].(map[string]any)
	if _, ok := firstStat["findingType"].(string); !ok {
		t.Fatalf("expected findingTypeStats[0].findingType string, got %#v", firstStat["findingType"])
	}
	if _, ok := firstStat["findingCount"].(float64); !ok {
		t.Fatalf("expected findingTypeStats[0].findingCount number, got %#v", firstStat["findingCount"])
	}
}

func TestGCPWebSecurityScannerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/websecurityscanner?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "websecurityscanner" {
		t.Fatalf("expected service=websecurityscanner, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in probe response, got %#v", body["name"])
	}
}

func TestGCPWebSecurityScannerRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/websecurityscanner?stackyard_contract_probe=1&pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp websecurityscanner contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPWebSecurityScannerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPWebSecurityScannerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "websecurityscanner",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp websecurityscanner router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
