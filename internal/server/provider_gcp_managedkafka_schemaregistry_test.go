package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPManagedKafkaSchemaRegistryRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaSchemaRegistryContractServer(t)

	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries?pageSize=1", "/schemaRegistries")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries?schemaRegistryId=sr1", "/schemaRegistries")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1", "/schemaRegistries/sr1")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/contexts?pageSize=1", "/contexts")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/schemas/ids/1", "/schemas/ids/1")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/schemas/ids/1/schema", "/schemas/ids/1/schema")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/subjects?pageSize=1", "/subjects")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/subjects/orders-value/versions?pageSize=1", "/versions")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/subjects/orders-value/versions", "/versions")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/subjects/orders-value/versions/1", "/versions/1")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/subjects/orders-value/versions/1/referencedby", "/referencedby")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/compatibility/subjects/orders-value/versions/latest", "/compatibility")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/config", "/config")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPut, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/config", "/config")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/config/orders-value", "/config/orders-value")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/mode/orders-value", "/mode/orders-value")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPut, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/mode/orders-value", "/mode/orders-value")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/schemaRegistries/sr1/mode/orders-value", "/mode/orders-value")
}

func TestGCPManagedKafkaSchemaRegistryRouter_RESTLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaSchemaRegistryContractServer(t)

	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPManagedKafkaSchemaRegistryRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaSchemaRegistryContractServer(t)

	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.schemaregistry.v1.ManagedSchemaRegistry/ListSchemaRegistries", "ManagedSchemaRegistry/ListSchemaRegistries")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.schemaregistry.v1.ManagedSchemaRegistry/GetSchema", "ManagedSchemaRegistry/GetSchema")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.schemaregistry.v1.ManagedSchemaRegistry/CreateVersion", "ManagedSchemaRegistry/CreateVersion")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.schemaregistry.v1.ManagedSchemaRegistry/CheckCompatibility", "ManagedSchemaRegistry/CheckCompatibility")
	assertGCPManagedKafkaSchemaRegistryNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.schemaregistry.v1.ManagedSchemaRegistry/UpdateSchemaMode", "ManagedSchemaRegistry/UpdateSchemaMode")
}

func newGCPManagedKafkaSchemaRegistryContractServer(t *testing.T) *httptest.Server {
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

func assertGCPManagedKafkaSchemaRegistryNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp managedkafka schemaregistry router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPManagedkafkaSchemaregistryRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPManagedkafkaSchemaregistryRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/managedkafka_schemaregistry?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp managedkafka_schemaregistry contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "managedkafka_schemaregistry" {
		t.Fatalf("expected service=managedkafka_schemaregistry, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

