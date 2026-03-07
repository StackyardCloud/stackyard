package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecurityCenterManagementRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1"
	serviceName := parent + "/securityCenterServices/security-health-analytics"
	shaName := parent + "/securityHealthAnalyticsCustomModules/sha-module-1"
	etdName := parent + "/eventThreatDetectionCustomModules/etd-module-1"

	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/securityCenterServices?pageSize=1", nil, "securityCenterServices")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, serviceName+"?showEligibleModulesOnly=true", nil, "security-health-analytics")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPatch, serviceName+"?updateMask=intendedEnablementState,modules", []byte(`{"name":"projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics","intendedEnablementState":"ENABLED","modules":{"default-module":{"intendedEnablementState":"ENABLED"}}}`), "securityCenterServices/security-health-analytics")

	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/securityHealthAnalyticsCustomModules?pageSize=1", nil, "securityHealthAnalyticsCustomModules")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/securityHealthAnalyticsCustomModules:listDescendant?pageSize=1", nil, "descendant-sha-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, shaName, nil, "sha-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPost, parent+"/securityHealthAnalyticsCustomModules", []byte(`{"displayName":"stackyard_sha_custom","customConfig":{"predicate":{"expression":"true"},"resourceSelector":{"resourceTypes":["compute.googleapis.com/Instance"]}}}`), "securityHealthAnalyticsCustomModules")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPatch, shaName+"?updateMask=customConfig,enablementState", []byte(`{"name":"projects/stackyard/locations/us-central1/securityHealthAnalyticsCustomModules/sha-module-1","enablementState":"DISABLED","customConfig":{"predicate":{"expression":"false"},"resourceSelector":{"resourceTypes":["compute.googleapis.com/Instance"]}}}`), "securityHealthAnalyticsCustomModules/sha-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodDelete, shaName, nil, "{}")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/effectiveSecurityHealthAnalyticsCustomModules?pageSize=1", nil, "effectiveSecurityHealthAnalyticsCustomModules")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/effectiveSecurityHealthAnalyticsCustomModules/effective-sha-module-1", nil, "effective-sha-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPost, parent+"/securityHealthAnalyticsCustomModules:simulate", []byte(`{"parent":"projects/stackyard/locations/us-central1","customConfig":{"predicate":{"expression":"true"},"resourceSelector":{"resourceTypes":["compute.googleapis.com/Instance"]}},"resource":{"resourceType":"compute.googleapis.com/Instance","resourceData":{"name":"instances/i-1"}}}`), "simulated-finding-1")

	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/eventThreatDetectionCustomModules?pageSize=1", nil, "eventThreatDetectionCustomModules")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/eventThreatDetectionCustomModules:listDescendant?pageSize=1", nil, "descendant-etd-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, etdName, nil, "eventThreatDetectionCustomModules/etd-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPost, parent+"/eventThreatDetectionCustomModules", []byte(`{"type":"CONFIGURABLE_BAD_IP","config":{"allowedIp":"10.0.0.1"}}`), "eventThreatDetectionCustomModules")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPatch, etdName+"?updateMask=config,enablementState", []byte(`{"name":"projects/stackyard/locations/us-central1/eventThreatDetectionCustomModules/etd-module-1","enablementState":"DISABLED","type":"CONFIGURABLE_BAD_IP","config":{"allowedIp":"127.0.0.1"}}`), "eventThreatDetectionCustomModules/etd-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodDelete, etdName, nil, "{}")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/effectiveEventThreatDetectionCustomModules?pageSize=1", nil, "effectiveEventThreatDetectionCustomModules")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodGet, parent+"/effectiveEventThreatDetectionCustomModules/effective-etd-module-1", nil, "effective-etd-module-1")
	assertGCPSecurityCenterManagementSuccess(t, ts, http.MethodPost, parent+"/eventThreatDetectionCustomModules:validate", []byte(`{"parent":"projects/stackyard/locations/us-central1","rawText":"rule allow { when true }","type":"CONFIGURABLE_BAD_IP"}`), "\"errors\":[]")
}

func TestGCPSecurityCenterManagementRouter_ListInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/securityCenterServices?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_CreateSHARequiresCustomConfig(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/securityHealthAnalyticsCustomModules", []byte(`{"displayName":"module"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_UpdateSHARequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/securityHealthAnalyticsCustomModules/sha-module-1", []byte(`{"name":"projects/stackyard/locations/us-central1/securityHealthAnalyticsCustomModules/sha-module-1","customConfig":{"predicate":{"expression":"true"}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_UpdateSHANameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/securityHealthAnalyticsCustomModules/sha-module-1?updateMask=customConfig", []byte(`{"name":"projects/stackyard/locations/us-central1/securityHealthAnalyticsCustomModules/sha-module-2","customConfig":{"predicate":{"expression":"true"}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_ValidateETDRequiresRawText(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/eventThreatDetectionCustomModules:validate", []byte(`{"type":"CONFIGURABLE_BAD_IP"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_UpdateServiceRejectsInvalidUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics?updateMask=badField", []byte(`{"name":"projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics","intendedEnablementState":"ENABLED"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_UpdateServiceRejectsReadOnlyIngestOnlyState(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityCenterManagementContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics?updateMask=intendedEnablementState", []byte(`{"name":"projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics","intendedEnablementState":"INGEST_ONLY"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securitycentermanagement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityCenterManagementRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/securitycentermanagement?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securitycentermanagement contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "securitycentermanagement" {
		t.Fatalf("expected service=securitycentermanagement, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSecurityCenterManagementContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecurityCenterManagementSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "securitycentermanagement",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securitycentermanagement router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
