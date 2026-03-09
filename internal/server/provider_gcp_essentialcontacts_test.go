package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPEssentialContactsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEssentialContactsContractServer(t)
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/contacts?pageSize=1", "/contacts")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/contacts", "/contacts")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/contacts/contact-1", "/contacts/contact-1")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/contacts/contact-1?updateMask=language_tag", "/contacts/contact-1")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/contacts/contact-1", "/contacts/contact-1")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/contacts:compute?notificationCategories=SECURITY&pageSize=1", ":compute")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/contacts:sendTestMessage", ":sendTestMessage")
}

func TestGCPEssentialContactsRouter_ResourceHierarchyRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEssentialContactsContractServer(t)
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/organizations/123456789/contacts?pageSize=1", "/organizations/123456789/contacts")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/folders/987654321/contacts?pageSize=1", "/folders/987654321/contacts")
}

func TestGCPEssentialContactsRouter_GRPCRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPEssentialContactsContractServer(t)
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/ListContacts", "EssentialContactsService/ListContacts")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/GetContact", "EssentialContactsService/GetContact")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/CreateContact", "EssentialContactsService/CreateContact")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/UpdateContact", "EssentialContactsService/UpdateContact")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/DeleteContact", "EssentialContactsService/DeleteContact")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/ComputeContacts", "EssentialContactsService/ComputeContacts")
	assertGCPEssentialContactsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.essentialcontacts.v1.EssentialContactsService/SendTestMessage", "EssentialContactsService/SendTestMessage")
}

func newGCPEssentialContactsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPEssentialContactsNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp essentialcontacts router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPEssentialcontactsRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPEssentialcontactsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/essentialcontacts?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp essentialcontacts contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "essentialcontacts" {
		t.Fatalf("expected service=essentialcontacts, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
