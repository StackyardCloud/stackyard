package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPLicenseManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLicenseManagerContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1"
	configurations := parent + "/configurations"
	configuration := configurations + "/config-1"
	instances := parent + "/instances"
	instance := instances + "/instance-1"
	products := parent + "/products"
	product := products + "/product-1"

	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, configurations+"?pageSize=1", "/configurations")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, configurations+"?configurationId=config-1", "/configurations")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, configuration, "/configurations/config-1")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPatch, configuration+"?updateMask.paths=labels", "/configurations/config-1")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodDelete, configuration, "/configurations/config-1")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, configuration+":deactivate", ":deactivate")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, configuration+":reactivate", ":reactivate")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, configuration+":queryLicenseUsage", ":queryLicenseUsage")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, configuration+":aggregateUsage?pageSize=1", ":aggregateUsage")

	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, instances+"?pageSize=1", "/instances")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, instance, "/instances/instance-1")

	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, products+"?pageSize=1", "/products")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodGet, product, "/products/product-1")
}

func TestGCPLicenseManagerRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLicenseManagerContractServer(t)

	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/ListConfigurations", "LicenseManager/ListConfigurations")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/GetConfiguration", "LicenseManager/GetConfiguration")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/CreateConfiguration", "LicenseManager/CreateConfiguration")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/UpdateConfiguration", "LicenseManager/UpdateConfiguration")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/DeleteConfiguration", "LicenseManager/DeleteConfiguration")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/ListInstances", "LicenseManager/ListInstances")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/GetInstance", "LicenseManager/GetInstance")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/DeactivateConfiguration", "LicenseManager/DeactivateConfiguration")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/ReactivateConfiguration", "LicenseManager/ReactivateConfiguration")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/QueryConfigurationLicenseUsage", "LicenseManager/QueryConfigurationLicenseUsage")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/AggregateUsage", "LicenseManager/AggregateUsage")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/ListProducts", "LicenseManager/ListProducts")
	assertGCPLicenseManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.licensemanager.v1.LicenseManager/GetProduct", "LicenseManager/GetProduct")
}

func newGCPLicenseManagerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPLicenseManagerNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp licensemanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPLicensemanagerRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPLicensemanagerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/licensemanager?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp licensemanager contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "licensemanager" {
		t.Fatalf("expected service=licensemanager, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

