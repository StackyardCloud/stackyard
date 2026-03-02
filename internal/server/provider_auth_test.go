package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGCPAuthValidatorModes(t *testing.T) {
	t.Parallel()

	emulator := gcpAuthValidator{mode: "emulator"}
	ok, _, _, _ := emulator.Validate(httptest.NewRequest(http.MethodGet, "/gcp/storage/v1/b", nil))
	if !ok {
		t.Fatalf("expected emulator mode to allow unsigned requests")
	}

	required := gcpAuthValidator{mode: "bearer_required"}
	req := httptest.NewRequest(http.MethodGet, "/gcp/storage/v1/b", nil)
	ok, status, _, _ := required.Validate(req)
	if ok || status != http.StatusUnauthorized {
		t.Fatalf("expected bearer_required mode to reject missing auth")
	}
	req.Header.Set("Authorization", "Bearer token")
	ok, _, _, _ = required.Validate(req)
	if !ok {
		t.Fatalf("expected bearer_required mode to accept Bearer token")
	}
}

func TestAzureAuthValidatorModes(t *testing.T) {
	t.Parallel()

	shared := azureAuthValidator{mode: "shared_key"}
	req := httptest.NewRequest(http.MethodGet, "/azure/storage", nil)
	req.Header.Set("Authorization", "SharedKey account:signature")
	ok, _, _, _ := shared.Validate(req)
	if !ok {
		t.Fatalf("expected shared_key mode to accept SharedKey auth")
	}

	sas := azureAuthValidator{mode: "sas"}
	req = httptest.NewRequest(http.MethodGet, "/azure/storage?sig=testsig", nil)
	ok, _, _, _ = sas.Validate(req)
	if !ok {
		t.Fatalf("expected sas mode to accept sig query parameter")
	}
}

func TestOCIAuthValidatorSignature(t *testing.T) {
	t.Parallel()

	validator := ociAuthValidator{mode: "signature"}
	req := httptest.NewRequest(http.MethodGet, "/oci/objectstorage", nil)
	req.Header.Set("Date", "Mon, 02 Mar 2026 16:00:00 GMT")
	req.Header.Set("Authorization", `Signature keyId="ocid1.user.oc1..aaaa",algorithm="rsa-sha256",headers="date (request-target) host",signature="abc123"`)
	ok, _, _, _ := validator.Validate(req)
	if !ok {
		t.Fatalf("expected valid OCI signature header to pass")
	}
}

func TestProviderRouterAuthEnforcement(t *testing.T) {
	t.Parallel()

	srv := New(Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerGCP},
		GCPAuthMode:   "bearer_required",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
		OCIAuthMode:   "disabled",
		AzureAuthMode: "disabled",
	})

	req := httptest.NewRequest(http.MethodGet, "/gcp/storage/v1/b", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing gcp bearer auth, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/gcp/not-implemented", nil)
	req.Header.Set("Authorization", "Bearer local-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 after auth pass-through to stub, got %d", rr.Code)
	}
}
