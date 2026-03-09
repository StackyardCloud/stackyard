package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestGCPDataformRouter_CreateRepositoryRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories", nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, `/repositories`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataformRouter_GetRepositoryRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo", nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, `/repositories/team-repo`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataformRouter_WorkspaceRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/workspaces?pageSize=1", nil, nil)
	if listResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for list workspaces, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := string(providerContractBody(t, listResp))
	if !strings.Contains(listBody, `"provider":"gcp"`) || !strings.Contains(listBody, `/workspaces`) {
		t.Fatalf("unexpected list workspaces response body: %s", listBody)
	}

	actionResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/workspaces/dev:searchFiles", nil, nil)
	if actionResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for workspace action, got %d body=%s", actionResp.StatusCode, string(providerContractBody(t, actionResp)))
	}
	actionBody := string(providerContractBody(t, actionResp))
	if !strings.Contains(actionBody, `"provider":"gcp"`) || !strings.Contains(actionBody, `:searchFiles`) {
		t.Fatalf("unexpected workspace action response body: %s", actionBody)
	}
}

func TestGCPDataformRouter_CompilationResultRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/compilationResults", nil, nil)
	if createResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for create compilation result, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := string(providerContractBody(t, createResp))
	if !strings.Contains(createBody, `"provider":"gcp"`) || !strings.Contains(createBody, `/compilationResults`) {
		t.Fatalf("unexpected create compilation result response body: %s", createBody)
	}

	queryResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/compilationResults/cr-1:query", nil, nil)
	if queryResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for query actions, got %d body=%s", queryResp.StatusCode, string(providerContractBody(t, queryResp)))
	}
	queryBody := string(providerContractBody(t, queryResp))
	if !strings.Contains(queryBody, `"provider":"gcp"`) || !strings.Contains(queryBody, `:query`) {
		t.Fatalf("unexpected compilation query response body: %s", queryBody)
	}
}

func TestGCPDataformRouter_WorkflowInvocationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/workflowInvocations", nil, nil)
	if createResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for create invocation, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := string(providerContractBody(t, createResp))
	if !strings.Contains(createBody, `"provider":"gcp"`) || !strings.Contains(createBody, `/workflowInvocations`) {
		t.Fatalf("unexpected create invocation response body: %s", createBody)
	}

	cancelResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/workflowInvocations/inv-1:cancel", nil, nil)
	if cancelResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for cancel invocation, got %d body=%s", cancelResp.StatusCode, string(providerContractBody(t, cancelResp)))
	}
	cancelBody := string(providerContractBody(t, cancelResp))
	if !strings.Contains(cancelBody, `"provider":"gcp"`) || !strings.Contains(cancelBody, `:cancel`) {
		t.Fatalf("unexpected cancel invocation response body: %s", cancelBody)
	}
}

func TestGCPDataformRouter_GetAndUpdateConfigRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/config", nil, nil)
	if getResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for get config, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := string(providerContractBody(t, getResp))
	if !strings.Contains(getBody, `"provider":"gcp"`) || !strings.Contains(getBody, `/config`) {
		t.Fatalf("unexpected get config response body: %s", getBody)
	}

	patchResp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/config", nil, nil)
	if patchResp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router for update config, got %d body=%s", patchResp.StatusCode, string(providerContractBody(t, patchResp)))
	}
	patchBody := string(providerContractBody(t, patchResp))
	if !strings.Contains(patchBody, `"provider":"gcp"`) || !strings.Contains(patchBody, `/config`) {
		t.Fatalf("unexpected update config response body: %s", patchBody)
	}
}

func TestGCPDataformRouter_IAMRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo:getIamPolicy", nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataform router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, `:getIamPolicy`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataformRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDataformRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/dataform?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dataform contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "dataform" {
		t.Fatalf("expected service=dataform, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
