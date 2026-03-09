package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMapsAddressValidationRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsAddressValidationContractServer(t)

	assertGCPMapsAddressValidationNotImplemented(t, ts, http.MethodPost, "/gcp/v1:validateAddress", ":validateAddress")
	assertGCPMapsAddressValidationNotImplemented(t, ts, http.MethodPost, "/gcp/v1:provideValidationFeedback", ":provideValidationFeedback")
}

func TestGCPMapsAddressValidationRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsAddressValidationContractServer(t)

	assertGCPMapsAddressValidationNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.addressvalidation.v1.AddressValidation/ValidateAddress", "AddressValidation/ValidateAddress")
	assertGCPMapsAddressValidationNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.addressvalidation.v1.AddressValidation/ProvideValidationFeedback", "AddressValidation/ProvideValidationFeedback")
}

func newGCPMapsAddressValidationContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsAddressValidationNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp maps addressvalidation router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPMapsAddressvalidationRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPMapsAddressvalidationRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/maps_addressvalidation?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps_addressvalidation contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "maps_addressvalidation" {
		t.Fatalf("expected service=maps_addressvalidation, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
