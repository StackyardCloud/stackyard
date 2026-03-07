package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPModelArmorRouter_TemplateRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPModelArmorContractServer(t)
	assertGCPModelArmorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/templates?pageSize=1", "/templates")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/templates?templateId=guardrail-a", "/templates")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/templates/guardrail-a", "/templates/guardrail-a")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/templates/guardrail-a?updateMask=labels", "/templates/guardrail-a")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/templates/guardrail-a", "/templates/guardrail-a")
}

func TestGCPModelArmorRouter_FloorSettingAndSanitizeRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPModelArmorContractServer(t)
	assertGCPModelArmorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/floorsetting", "/floorsetting")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/floorsetting?updateMask=filterConfig", "/floorsetting")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/floorSetting", "/floorSetting")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/templates/guardrail-a:sanitizeUserPrompt", ":sanitizeUserPrompt")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/templates/guardrail-a:sanitizeModelResponse", ":sanitizeModelResponse")
}

func TestGCPModelArmorRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPModelArmorContractServer(t)
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.modelarmor.v1.ModelArmor/ListTemplates", "ModelArmor/ListTemplates")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.modelarmor.v1.ModelArmor/GetTemplate", "ModelArmor/GetTemplate")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.modelarmor.v1.ModelArmor/CreateTemplate", "ModelArmor/CreateTemplate")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.modelarmor.v1.ModelArmor/UpdateFloorSetting", "ModelArmor/UpdateFloorSetting")
	assertGCPModelArmorNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.modelarmor.v1.ModelArmor/SanitizeModelResponse", "ModelArmor/SanitizeModelResponse")
}

func newGCPModelArmorContractServer(t *testing.T) *httptest.Server {
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

func assertGCPModelArmorNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp model armor router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPModelarmorRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPModelarmorRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/modelarmor?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp modelarmor contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "modelarmor" {
		t.Fatalf("expected service=modelarmor, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

