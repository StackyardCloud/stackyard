package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	telcoautomationpb "cloud.google.com/go/telcoautomation/apiv1/telcoautomationpb"
)

func TestGCPStage4GRPCParity_TelcoAutomation(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest telcoautomation list orchestration clusters, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restItems, ok := restBody["orchestrationClusters"].([]any)
	if !ok || len(restItems) == 0 {
		t.Fatalf("expected orchestrationClusters list in rest payload, got %#v", restBody["orchestrationClusters"])
	}
	restCluster, _ := restItems[0].(map[string]any)
	restClusterName, _ := restCluster["name"].(string)

	successReq := &telcoautomationpb.ListOrchestrationClustersRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}
	var successResp telcoautomationpb.ListOrchestrationClustersResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTelcoAutomationListOrchestrationClustersMethod, successReq, &successResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(successResp.GetOrchestrationClusters()) != 1 {
		t.Fatalf("expected one grpc orchestration cluster, got %d", len(successResp.GetOrchestrationClusters()))
	}
	if successResp.GetOrchestrationClusters()[0].GetName() != restClusterName {
		t.Fatalf("expected grpc orchestration cluster name %q to match rest %q", successResp.GetOrchestrationClusters()[0].GetName(), restClusterName)
	}

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/orchestrationClusters?orchestrationClusterId=cluster-1", []byte(`{"name":"projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "telcoautomation",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest telcoautomation create orchestration cluster, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restOperationName, _ := restCreateBody["name"].(string)

	var lroResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTelcoAutomationCreateOrchestrationClusterMethod, &telcoautomationpb.CreateOrchestrationClusterRequest{
		Parent:                 "projects/stackyard/locations/us-central1",
		OrchestrationClusterId: "cluster-1",
		OrchestrationCluster: &telcoautomationpb.OrchestrationCluster{
			Name: "projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1",
		},
	}, &lroResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create orchestration cluster, got %q message=%q", grpcStatus, grpcMessage)
	}
	if lroResp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", lroResp.GetName(), restOperationName)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTelcoAutomationCreateOrchestrationClusterMethod, &telcoautomationpb.CreateOrchestrationClusterRequest{
		Parent:                 "projects/stackyard/locations/us-central1",
		OrchestrationClusterId: "cluster-1",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "orchestration_cluster-required") {
		t.Fatalf("expected grpc invalid argument for create orchestration cluster, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTelcoAutomationApproveBlueprintMethod, &telcoautomationpb.ApproveBlueprintRequest{
		Name: "projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/blueprints/blueprint-draft",
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "blueprint-must-be-proposed") {
		t.Fatalf("expected grpc failed precondition for approve blueprint, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTelcoAutomationGetDeploymentMethod, &telcoautomationpb.GetDeploymentRequest{
		Name: "projects/stackyard/locations/us-central1/orchestrationClusters/cluster-1/deployments/missing-deployment",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "deployment-not-found") {
		t.Fatalf("expected grpc not found for get deployment, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
