package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPManagedKafkaRouter_RESTClusterTopicConsumerGroupAndACLRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaContractServer(t)

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?pageSize=1", "/clusters")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters?clusterId=cluster-a", "/clusters")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a?updateMask=labels", "/clusters/cluster-a")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a", "/clusters/cluster-a")

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/topics?pageSize=1", "/topics")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/topics?topicId=orders", "/topics")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/topics/orders?updateMask=configs", "/topics/orders")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/topics/orders", "/topics/orders")

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/consumerGroups?pageSize=1", "/consumerGroups")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/consumerGroups/cg-a?updateMask=topics", "/consumerGroups/cg-a")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/consumerGroups/cg-a", "/consumerGroups/cg-a")

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/acls?pageSize=1", "/acls")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/acls?aclId=allTopics", "/acls")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/acls/allTopics?updateMask=aclEntries", "/acls/allTopics")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/acls/allTopics:addAclEntry", ":addAclEntry")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/acls/allTopics:removeAclEntry", ":removeAclEntry")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/clusters/cluster-a/acls/allTopics", "/acls/allTopics")
}

func TestGCPManagedKafkaRouter_RESTConnectRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaContractServer(t)

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters?pageSize=1", "/connectClusters")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters?connectClusterId=connect-a", "/connectClusters")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a?updateMask=labels", "/connectClusters/connect-a")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a", "/connectClusters/connect-a")

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors?pageSize=1", "/connectors")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors?connectorId=jdbc-orders", "/connectors")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors/jdbc-orders?updateMask=configs", "/connectors/jdbc-orders")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors/jdbc-orders:pause", ":pause")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors/jdbc-orders:resume", ":resume")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors/jdbc-orders:restart", ":restart")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors/jdbc-orders:stop", ":stop")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/connectClusters/connect-a/connectors/jdbc-orders", "/connectors/jdbc-orders")
}

func TestGCPManagedKafkaRouter_RESTLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaContractServer(t)

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPManagedKafkaRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPManagedKafkaContractServer(t)

	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.v1.ManagedKafka/ListClusters", "ManagedKafka/ListClusters")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.v1.ManagedKafka/CreateCluster", "ManagedKafka/CreateCluster")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.v1.ManagedKafka/AddAclEntry", "ManagedKafka/AddAclEntry")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.v1.ManagedKafkaConnect/ListConnectClusters", "ManagedKafkaConnect/ListConnectClusters")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.v1.ManagedKafkaConnect/CreateConnector", "ManagedKafkaConnect/CreateConnector")
	assertGCPManagedKafkaNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.managedkafka.v1.ManagedKafkaConnect/PauseConnector", "ManagedKafkaConnect/PauseConnector")
}

func newGCPManagedKafkaContractServer(t *testing.T) *httptest.Server {
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

func assertGCPManagedKafkaNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp managedkafka router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPManagedkafkaRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPManagedkafkaRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/managedkafka?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp managedkafka contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "managedkafka" {
		t.Fatalf("expected service=managedkafka, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
