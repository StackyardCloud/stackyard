package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVPCAccessRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)

	parent := "/gcp/v1/projects/stackyard/locations/us-central1"
	connectorName := parent + "/connectors/connector-1"

	assertGCPVPCAccessSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPVPCAccessSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPVPCAccessSuccess(t, ts, http.MethodGet, parent+"/connectors?pageSize=1", nil, "connectors")
	assertGCPVPCAccessSuccess(t, ts, http.MethodGet, connectorName, nil, "/connectors/connector-1")
	assertGCPVPCAccessSuccess(t, ts, http.MethodPost, parent+"/connectors?connectorId=connector-1", []byte(`{"network":"default","ipCidrRange":"10.8.0.0/28"}`), "operations/createConnector.connector-1")
	assertGCPVPCAccessSuccess(t, ts, http.MethodDelete, connectorName, nil, "operations/deleteConnector.connector-1")

	assertGCPVPCAccessSuccess(t, ts, http.MethodGet, parent+"/operations?pageSize=1", nil, "operations")
	assertGCPVPCAccessSuccess(t, ts, http.MethodGet, parent+"/operations/vpcaccess-op-1", nil, "operations/vpcaccess-op-1")
	assertGCPVPCAccessSuccess(t, ts, http.MethodPost, parent+"/operations/vpcaccess-op-1:cancel", []byte(`{}`), "{}")
	assertGCPVPCAccessSuccess(t, ts, http.MethodDelete, parent+"/operations/vpcaccess-op-1", nil, "{}")

	assertGCPVPCAccessSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vpcaccess.v1.VpcAccessService/ListConnectors", []byte(`{"parent":"projects/stackyard/locations/us-central1","pageSize":1}`), "connectors")
	assertGCPVPCAccessSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vpcaccess.v1.VpcAccessService/GetConnector", []byte(`{"name":"projects/stackyard/locations/us-central1/connectors/connector-1"}`), "/connectors/connector-1")
	assertGCPVPCAccessSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vpcaccess.v1.VpcAccessService/CreateConnector", []byte(`{"parent":"projects/stackyard/locations/us-central1","connectorId":"connector-1","connector":{"network":"default"}}`), "operations/createConnector.connector-1")
	assertGCPVPCAccessSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.vpcaccess.v1.VpcAccessService/DeleteConnector", []byte(`{"name":"projects/stackyard/locations/us-central1/connectors/connector-1"}`), "operations/deleteConnector.connector-1")
}

func TestGCPVPCAccessRouter_ListConnectorsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vpcaccess router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_CreateConnectorRequiresConnectorID(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectors", []byte(`{"network":"default"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vpcaccess create connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_CreateConnectorRequiresConnectorBody(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?connectorId=connector-1", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vpcaccess create connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_CreateConnectorRequiresNetworkOrSubnet(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?connectorId=connector-1", []byte(`{"ipCidrRange":"10.8.0.0/28"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vpcaccess create connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_CreateConnectorAlreadyExists(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?connectorId=connector-existing", []byte(`{"network":"default"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp vpcaccess create connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"AlreadyExists"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_DeleteConnectorNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/connectors/missing-connector", nil, map[string]string{
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp vpcaccess delete connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_GRPCBridgeCreateConnectorRequiresConnector(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vpcaccess.v1.VpcAccessService/CreateConnector", []byte(`{"parent":"projects/stackyard/locations/us-central1","connectorId":"connector-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp vpcaccess grpc bridge create connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVPCAccessRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVPCAccessContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vpcaccess",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vpcaccess list connectors, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	connectors, ok := listBody["connectors"].([]any)
	if !ok || len(connectors) == 0 {
		t.Fatalf("expected connectors array, got %#v", listBody["connectors"])
	}
	connector, _ := connectors[0].(map[string]any)
	if _, ok := connector["name"].(string); !ok {
		t.Fatalf("expected connectors[0].name string, got %#v", connector["name"])
	}
	if _, ok := connector["state"].(string); !ok {
		t.Fatalf("expected connectors[0].state string, got %#v", connector["state"])
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectors/connector-1", nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vpcaccess get connector, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["network"].(string); !ok {
		t.Fatalf("expected connector network string, got %#v", getBody["network"])
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectors?connectorId=connector-1", []byte(`{"network":"default","ipCidrRange":"10.8.0.0/28"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vpcaccess",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vpcaccess create connector, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", createBody["done"])
	}
	if _, ok := createBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected operation metadata object, got %#v", createBody["metadata"])
	}
}

func TestGCPVPCAccessRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vpcaccess?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vpcaccess contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "vpcaccess" {
		t.Fatalf("expected service=vpcaccess, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name in probe response, got %#v", body["name"])
	}
}

func newGCPVPCAccessContractServer(t *testing.T) *httptest.Server {
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

func assertGCPVPCAccessSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "vpcaccess",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp vpcaccess router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
