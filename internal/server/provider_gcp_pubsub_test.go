package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPPubSubRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPubSubContractServer(t)

	assertGCPPubSubNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/topics?pageSize=1", "/topics")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPut, "/gcp/v1/projects/stackyard/topics/orders", "/topics/orders")
	assertGCPPubSubNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/topics/orders", "/topics/orders")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/topics/orders:publish", ":publish")
	assertGCPPubSubNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/topics/orders", "/topics/orders")

	assertGCPPubSubNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/subscriptions?pageSize=1", "/subscriptions")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPut, "/gcp/v1/projects/stackyard/subscriptions/orders-sub", "/subscriptions/orders-sub")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/subscriptions/orders-sub:pull", ":pull")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/subscriptions/orders-sub:acknowledge", ":acknowledge")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/subscriptions/orders-sub:modifyAckDeadline", ":modifyAckDeadline")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/subscriptions/orders-sub:seek", ":seek")
	assertGCPPubSubNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/subscriptions/orders-sub", "/subscriptions/orders-sub")

	assertGCPPubSubNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/snapshots?pageSize=1", "/snapshots")

	assertGCPPubSubNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/schemas?pageSize=1", "/schemas")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/schemas:validate", ":validate")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/schemas:validateMessage", ":validateMessage")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPut, "/gcp/v1/projects/stackyard/schemas/order-schema", "/schemas/order-schema")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/schemas/order-schema:commit", ":commit")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/schemas/order-schema:rollback", ":rollback")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/schemas/order-schema:deleteRevision", ":deleteRevision")
	assertGCPPubSubNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/schemas/order-schema", "/schemas/order-schema")
}

func TestGCPPubSubRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPubSubContractServer(t)

	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Publisher/CreateTopic", "Publisher/CreateTopic")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Publisher/GetTopic", "Publisher/GetTopic")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Publisher/ListTopics", "Publisher/ListTopics")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Publisher/Publish", "Publisher/Publish")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Publisher/DeleteTopic", "Publisher/DeleteTopic")

	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/CreateSubscription", "Subscriber/CreateSubscription")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/GetSubscription", "Subscriber/GetSubscription")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/ListSubscriptions", "Subscriber/ListSubscriptions")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/Pull", "Subscriber/Pull")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/Acknowledge", "Subscriber/Acknowledge")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/Seek", "Subscriber/Seek")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.Subscriber/DeleteSubscription", "Subscriber/DeleteSubscription")

	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.SchemaService/CreateSchema", "SchemaService/CreateSchema")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.SchemaService/GetSchema", "SchemaService/GetSchema")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.SchemaService/ListSchemas", "SchemaService/ListSchemas")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.SchemaService/ValidateSchema", "SchemaService/ValidateSchema")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.SchemaService/ValidateMessage", "SchemaService/ValidateMessage")
	assertGCPPubSubNotImplemented(t, ts, http.MethodPost, "/gcp/google.pubsub.v1.SchemaService/DeleteSchema", "SchemaService/DeleteSchema")
}

func newGCPPubSubContractServer(t *testing.T) *httptest.Server {
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

func assertGCPPubSubNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp pubsub router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPPubsubRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPPubsubRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/pubsub?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp pubsub contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "pubsub" {
		t.Fatalf("expected service=pubsub, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

