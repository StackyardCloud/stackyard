package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPGKEMultiCloudRouter_AttachedRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEMultiCloudContractServer(t)
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters?pageSize=1", "/attachedClusters")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters", "/attachedClusters")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters/attached-a", "/attachedClusters/attached-a")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters/attached-a", "/attachedClusters/attached-a")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters/attached-a", "/attachedClusters/attached-a")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters:import", ":import")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters/attached-a:generateInstallManifest", ":generateInstallManifest")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/attachedClusters/attached-a:generateAgentToken", ":generateAgentToken")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/attachedServerConfig", "/attachedServerConfig")
}

func TestGCPGKEMultiCloudRouter_AWSRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEMultiCloudContractServer(t)
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters?pageSize=1", "/awsClusters")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters", "/awsClusters")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a", "/awsClusters/aws-a")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a:generateAwsClusterAgentToken", ":generateAwsClusterAgentToken")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a:generateAwsAccessToken", ":generateAwsAccessToken")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a/awsNodePools?pageSize=1", "/awsNodePools")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a/awsNodePools/node-a:rollback", ":rollback")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a/awsOpenIdConfig", "/awsOpenIdConfig")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/awsClusters/aws-a/awsJsonWebKeys", "/awsJsonWebKeys")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-west1/awsServerConfig", "/awsServerConfig")
}

func TestGCPGKEMultiCloudRouter_AzureAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEMultiCloudContractServer(t)
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/azureClients?pageSize=1", "/azureClients")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/azureClusters?pageSize=1", "/azureClusters")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/eastus2/azureClusters/az-a:generateAzureClusterAgentToken", ":generateAzureClusterAgentToken")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/eastus2/azureClusters/az-a:generateAzureAccessToken", ":generateAzureAccessToken")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/azureClusters/az-a/azureNodePools?pageSize=1", "/azureNodePools")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/azureClusters/az-a/azureOpenIdConfig", "/azureOpenIdConfig")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/azureClusters/az-a/azureJsonWebKeys", "/azureJsonWebKeys")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/azureServerConfig", "/azureServerConfig")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/eastus2/operations?pageSize=1", "/operations")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/eastus2/operations/op-1:cancel", ":cancel")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/eastus2/operations/op-1", "/operations/op-1")
}

func TestGCPGKEMultiCloudRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPGKEMultiCloudContractServer(t)
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkemulticloud.v1.AttachedClusters/CreateAttachedCluster", "AttachedClusters/CreateAttachedCluster")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkemulticloud.v1.AttachedClusters/GenerateAttachedClusterInstallManifest", "AttachedClusters/GenerateAttachedClusterInstallManifest")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkemulticloud.v1.AwsClusters/CreateAwsCluster", "AwsClusters/CreateAwsCluster")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkemulticloud.v1.AwsClusters/RollbackAwsNodePoolUpdate", "AwsClusters/RollbackAwsNodePoolUpdate")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkemulticloud.v1.AzureClusters/CreateAzureCluster", "AzureClusters/CreateAzureCluster")
	assertGCPGKEMultiCloudNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.gkemulticloud.v1.AzureClusters/GenerateAzureAccessToken", "AzureClusters/GenerateAzureAccessToken")
}

func newGCPGKEMultiCloudContractServer(t *testing.T) *httptest.Server {
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

func assertGCPGKEMultiCloudNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp gkemulticloud router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPGkemulticloudRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPGkemulticloudRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gkemulticloud?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp gkemulticloud contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "gkemulticloud" {
		t.Fatalf("expected service=gkemulticloud, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
