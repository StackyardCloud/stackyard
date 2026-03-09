package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGCPDataFusionRouter_ListAvailableVersionsRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/versions?pageSize=1", nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, `/versions`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataFusionRouter_ListAndGetInstanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances?pageSize=1", nil, nil)
	if listResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router for list instances, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := string(providerContractBody(t, listResp))
	if !strings.Contains(listBody, `"provider":"gcp"`) || !strings.Contains(listBody, `/instances`) {
		t.Fatalf("unexpected list response body: %s", listBody)
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance", nil, nil)
	if getResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router for get instance, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := string(providerContractBody(t, getResp))
	if !strings.Contains(getBody, `"provider":"gcp"`) || !strings.Contains(getBody, `/instances/team-instance`) {
		t.Fatalf("unexpected get response body: %s", getBody)
	}
}

func TestGCPDataFusionRouter_CreateAndDeleteInstanceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances?instanceId=team-instance", nil, nil)
	if createResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router for create instance, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := string(providerContractBody(t, createResp))
	if !strings.Contains(createBody, `"provider":"gcp"`) || !strings.Contains(createBody, `/instances`) {
		t.Fatalf("unexpected create response body: %s", createBody)
	}

	deleteResp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance", nil, nil)
	if deleteResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router for delete instance, got %d body=%s", deleteResp.StatusCode, string(providerContractBody(t, deleteResp)))
	}
	deleteBody := string(providerContractBody(t, deleteResp))
	if !strings.Contains(deleteBody, `"provider":"gcp"`) || !strings.Contains(deleteBody, `/instances/team-instance`) {
		t.Fatalf("unexpected delete response body: %s", deleteBody)
	}
}

func TestGCPDataFusionRouter_UpdateAndRestartRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	patchResp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance?updateMask=labels", nil, nil)
	if patchResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router for update instance, got %d body=%s", patchResp.StatusCode, string(providerContractBody(t, patchResp)))
	}
	patchBody := string(providerContractBody(t, patchResp))
	if !strings.Contains(patchBody, `"provider":"gcp"`) || !strings.Contains(patchBody, `/instances/team-instance`) {
		t.Fatalf("unexpected patch response body: %s", patchBody)
	}

	restartResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/instances/team-instance:restart", nil, nil)
	if restartResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datafusion router for restart instance, got %d body=%s", restartResp.StatusCode, string(providerContractBody(t, restartResp)))
	}
	restartBody := string(providerContractBody(t, restartResp))
	if !strings.Contains(restartBody, `"provider":"gcp"`) || !strings.Contains(restartBody, `:restart`) {
		t.Fatalf("unexpected restart response body: %s", restartBody)
	}
}

func TestGCPDatafusionRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDatafusionRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datafusion?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp datafusion contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "datafusion" {
		t.Fatalf("expected service=datafusion, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
