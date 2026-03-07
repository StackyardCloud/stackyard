package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPServiceDirectoryRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	namespace := base + "/namespaces/ns-1"
	service := namespace + "/services/svc-1"
	endpoint := service + "/endpoints/ep-1"

	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, base+"/namespaces?pageSize=1", nil, "namespaces")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, namespace, nil, "namespaces/ns-1")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, base+"/namespaces?namespaceId=ns-1", []byte(`{"labels":{"env":"prod"}}`), "namespaces/ns-1")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPatch, namespace+"?updateMask=labels", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1","labels":{"team":"platform"}}`), "platform")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodDelete, namespace, nil, "{}")

	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, namespace+"/services?pageSize=1", nil, "services")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, service, nil, "services/svc-1")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, namespace+"/services?serviceId=svc-1", []byte(`{"annotations":{"owner":"api-team"}}`), "services/svc-1")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPatch, service+"?updateMask=annotations", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","annotations":{"tier":"gold"}}`), "gold")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodDelete, service, nil, "{}")

	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, service+"/endpoints?pageSize=1", nil, "endpoints")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodGet, endpoint, nil, "endpoints/ep-1")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, service+"/endpoints?endpointId=ep-1", []byte(`{"address":"10.10.0.9","port":8088}`), "endpoints/ep-1")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPatch, endpoint+"?updateMask=address,port", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1/endpoints/ep-1","address":"10.10.0.15","port":9090}`), `"port":9090`)
	assertGCPServiceDirectorySuccess(t, ts, http.MethodDelete, endpoint, nil, "{}")

	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, service+":resolve", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","maxEndpoints":1}`), `"service"`)

	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, service+":getIamPolicy", []byte(`{"resource":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1"}`), "bindings")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, service+":setIamPolicy", []byte(`{"resource":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","policy":{"version":1,"bindings":[{"role":"roles/servicedirectory.viewer","members":["user:stackyard@example.com"]}]}}`), "roles/servicedirectory.viewer")
	assertGCPServiceDirectorySuccess(t, ts, http.MethodPost, service+":testIamPermissions", []byte(`{"resource":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","permissions":["servicedirectory.services.get"]}`), "servicedirectory.services.get")
}

func TestGCPServiceDirectoryRouter_ListNamespacesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_CreateNamespaceRequiresNamespaceID(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces", []byte(`{"labels":{"env":"prod"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_UpdateNamespaceRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_UpdateNamespaceNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1?updateMask=labels", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-2","labels":{"team":"platform"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_CreateEndpointRejectsPortOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1/endpoints?endpointId=ep-1", []byte(`{"port":70000}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_ResolveServiceRejectsBadMaxEndpoints(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1:resolve", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","maxEndpoints":-1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_SetIAMPolicyRequiresPolicy(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1:setIamPolicy", []byte(`{"resource":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_TestIAMPermissionsRequiresPermissions(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1:testIamPermissions", []byte(`{"resource":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","permissions":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicedirectory router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceDirectoryRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceDirectoryContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicedirectory",
	}

	namespaceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1", nil, headers)
	if namespaceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory get namespace, got %d body=%s", namespaceResp.StatusCode, string(providerContractBody(t, namespaceResp)))
	}
	namespaceBody := providerContractJSONMap(t, namespaceResp)
	if _, ok := namespaceBody["name"].(string); !ok {
		t.Fatalf("expected namespace.name string, got %#v", namespaceBody["name"])
	}
	if _, ok := namespaceBody["labels"].(map[string]any); !ok {
		t.Fatalf("expected namespace.labels object, got %#v", namespaceBody["labels"])
	}
	if _, ok := namespaceBody["uid"].(string); !ok {
		t.Fatalf("expected namespace.uid string, got %#v", namespaceBody["uid"])
	}

	serviceResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1", nil, headers)
	if serviceResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory get service, got %d body=%s", serviceResp.StatusCode, string(providerContractBody(t, serviceResp)))
	}
	serviceBody := providerContractJSONMap(t, serviceResp)
	if _, ok := serviceBody["annotations"].(map[string]any); !ok {
		t.Fatalf("expected service.annotations object, got %#v", serviceBody["annotations"])
	}

	endpointResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1/endpoints/ep-1", nil, headers)
	if endpointResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory get endpoint, got %d body=%s", endpointResp.StatusCode, string(providerContractBody(t, endpointResp)))
	}
	endpointBody := providerContractJSONMap(t, endpointResp)
	if _, ok := endpointBody["address"].(string); !ok {
		t.Fatalf("expected endpoint.address string, got %#v", endpointBody["address"])
	}
	if _, ok := endpointBody["port"].(float64); !ok {
		t.Fatalf("expected endpoint.port number, got %#v", endpointBody["port"])
	}

	resolveResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1:resolve", []byte(`{"name":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1","maxEndpoints":1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if resolveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory resolve service, got %d body=%s", resolveResp.StatusCode, string(providerContractBody(t, resolveResp)))
	}
	resolveBody := providerContractJSONMap(t, resolveResp)
	serviceObj, ok := resolveBody["service"].(map[string]any)
	if !ok {
		t.Fatalf("expected resolve service object, got %#v", resolveBody["service"])
	}
	endpoints, ok := serviceObj["endpoints"].([]any)
	if !ok || len(endpoints) != 1 {
		t.Fatalf("expected resolve service endpoints array with one item, got %#v", serviceObj["endpoints"])
	}

	policyResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1:getIamPolicy", []byte(`{"resource":"projects/stackyard/locations/us-central1/namespaces/ns-1/services/svc-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicedirectory",
	})
	if policyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory get iam policy, got %d body=%s", policyResp.StatusCode, string(providerContractBody(t, policyResp)))
	}
	policyBody := providerContractJSONMap(t, policyResp)
	if _, ok := policyBody["bindings"].([]any); !ok {
		t.Fatalf("expected policy.bindings array, got %#v", policyBody["bindings"])
	}
}

func TestGCPServiceDirectoryRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/servicedirectory?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "servicedirectory" {
		t.Fatalf("expected service=servicedirectory, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPServiceDirectoryContractServer(t *testing.T) *httptest.Server {
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

func assertGCPServiceDirectorySuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicedirectory",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicedirectory router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
