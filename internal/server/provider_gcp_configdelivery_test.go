package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPConfigDeliveryRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPConfigDeliveryContractServer(t)

	baseLocation := "/gcp/v1/projects/stackyard/locations/us-central1"
	resourceBundleBase := baseLocation + "/resourceBundles/platform-bundle"
	releaseBase := resourceBundleBase + "/releases/r-1"
	fleetPackageBase := baseLocation + "/fleetPackages/platform-package"
	rolloutBase := fleetPackageBase + "/rollouts/rollout-1"

	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, baseLocation+"/resourceBundles?pageSize=1", nil, "resourceBundles")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, resourceBundleBase, nil, "platform-bundle")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, baseLocation+"/resourceBundles?resourceBundleId=platform-bundle", []byte(`{"resourceBundle":{"description":"Team bundle"}}`), "operations/createResourceBundle")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodDelete, resourceBundleBase, nil, "operations/deleteResourceBundle")

	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, baseLocation+"/fleetPackages?pageSize=1", nil, "fleetPackages")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, fleetPackageBase, nil, "platform-package")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, baseLocation+"/fleetPackages?fleetPackageId=platform-package", []byte(`{"fleetPackage":{"variantSelector":{"variantNameTemplate":"default"}}}`), "operations/createFleetPackage")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodDelete, fleetPackageBase, nil, "operations/deleteFleetPackage")

	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, resourceBundleBase+"/releases?pageSize=1", nil, "releases")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, releaseBase, nil, "r-1")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, resourceBundleBase+"/releases?releaseId=r-1", []byte(`{"release":{"version":"v1.0.0"}}`), "operations/createRelease")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodDelete, releaseBase, nil, "operations/deleteRelease")

	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, releaseBase+"/variants?pageSize=1", nil, "variants")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, releaseBase+"/variants/default", nil, "default")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, releaseBase+"/variants?variantId=default", []byte(`{"variant":{"resources":["apiVersion: v1"]}}`), "operations/createVariant")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodDelete, releaseBase+"/variants/default", nil, "operations/deleteVariant")

	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, fleetPackageBase+"/rollouts?pageSize=1", nil, "rollouts")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodGet, rolloutBase, nil, "rollout-1")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, rolloutBase+":suspend", []byte(`{"reason":"pause rollout for verification"}`), "suspendRollout")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, rolloutBase+":resume", []byte(`{"reason":"resume rollout after verification"}`), "resumeRollout")
	assertGCPConfigDeliverySuccess(t, ts, http.MethodPost, rolloutBase+":abort", []byte(`{"reason":"abort rollout for staged test"}`), "abortRollout")
}

func TestGCPConfigDeliveryRouter_ListResourceBundlesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPConfigDeliveryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/resourceBundles?pageSize=bad", nil, map[string]string{"X-Stackyard-GCP-Service": "configdelivery"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp configdelivery router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPConfigDeliveryRouter_CreateResourceBundleRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPConfigDeliveryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/resourceBundles", []byte(`{"resourceBundle":{"description":"bundle"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "configdelivery",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp configdelivery router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPConfigDeliveryRouter_RolloutActionRequiresReason(t *testing.T) {
	t.Parallel()

	ts := newGCPConfigDeliveryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/fleetPackages/platform-package/rollouts/rollout-1:suspend", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "configdelivery",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp configdelivery router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPConfigDeliveryContractServer(t *testing.T) *httptest.Server {
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

func assertGCPConfigDeliverySuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "configdelivery",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp configdelivery router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
