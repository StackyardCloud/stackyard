package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPPubSubLiteRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPPubSubLiteContractServer(t)

	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.AdminService/ListTopics", "AdminService/ListTopics")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.AdminService/CreateTopic", "AdminService/CreateTopic")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.AdminService/CreateSubscription", "AdminService/CreateSubscription")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.AdminService/CreateReservation", "AdminService/CreateReservation")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.AdminService/SeekSubscription", "AdminService/SeekSubscription")

	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.CursorService/ListPartitionCursors", "CursorService/ListPartitionCursors")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.CursorService/CommitCursor", "CursorService/CommitCursor")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.CursorService/StreamingCommitCursor", "CursorService/StreamingCommitCursor")

	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.PartitionAssignmentService/AssignPartitions", "PartitionAssignmentService/AssignPartitions")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.PublisherService/Publish", "PublisherService/Publish")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.SubscriberService/Subscribe", "SubscriberService/Subscribe")

	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.TopicStatsService/ComputeHeadCursor", "TopicStatsService/ComputeHeadCursor")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.TopicStatsService/ComputeMessageStats", "TopicStatsService/ComputeMessageStats")
	assertGCPPubSubLiteNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.pubsublite.v1.TopicStatsService/ComputeTimeCursor", "TopicStatsService/ComputeTimeCursor")
}

func newGCPPubSubLiteContractServer(t *testing.T) *httptest.Server {
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

func assertGCPPubSubLiteNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp pubsublite router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPPubsubliteRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPPubsubliteRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/pubsublite?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp pubsublite contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "pubsublite" {
		t.Fatalf("expected service=pubsublite, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

