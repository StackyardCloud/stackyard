package server

import (
	"net/http"
	"strings"
	"testing"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	workstationspb "cloud.google.com/go/workstations/apiv1/workstationspb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestGCPStage4GRPCParity_Workstations(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "workstations",
	}

	locationParent := "projects/stackyard/locations/us-central1"
	clusterName := locationParent + "/workstationClusters/cluster-1"
	configName := clusterName + "/workstationConfigs/config-1"
	workstationRunning := configName + "/workstations/workstation-running"

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+locationParent+"/workstationClusters?pageSize=1", nil, headers)
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workstations list clusters, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restClusters, ok := restListBody["workstationClusters"].([]any)
	if !ok || len(restClusters) == 0 {
		t.Fatalf("expected rest workstationClusters list, got %#v", restListBody["workstationClusters"])
	}
	restFirstCluster, _ := restClusters[0].(map[string]any)
	restFirstClusterName, _ := restFirstCluster["name"].(string)

	var grpcListResp workstationspb.ListWorkstationClustersResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsListWorkstationClustersMethod, &workstationspb.ListWorkstationClustersRequest{
		Parent:   locationParent,
		PageSize: 1,
	}, &grpcListResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list workstation clusters, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListResp.GetWorkstationClusters()) != 1 {
		t.Fatalf("expected one grpc workstation cluster, got %d", len(grpcListResp.GetWorkstationClusters()))
	}
	if grpcListResp.GetWorkstationClusters()[0].GetName() != restFirstClusterName {
		t.Fatalf("expected grpc cluster name %q to match rest %q", grpcListResp.GetWorkstationClusters()[0].GetName(), restFirstClusterName)
	}

	restGetResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/"+workstationRunning, nil, headers)
	if restGetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workstations get, got %d body=%s", restGetResp.StatusCode, string(providerContractBody(t, restGetResp)))
	}
	restGetBody := providerContractJSONMap(t, restGetResp)
	restState, _ := restGetBody["state"].(string)

	var grpcGetResp workstationspb.Workstation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsGetWorkstationMethod, &workstationspb.GetWorkstationRequest{
		Name: workstationRunning,
	}, &grpcGetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for get workstation, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcGetResp.GetName() != workstationRunning {
		t.Fatalf("expected grpc workstation name %q, got %q", workstationRunning, grpcGetResp.GetName())
	}
	if grpcGetResp.GetState().String() != restState {
		t.Fatalf("expected grpc workstation state %q to match rest %q", grpcGetResp.GetState().String(), restState)
	}

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+locationParent+"/workstationClusters?workstationClusterId=cluster-new", []byte(`{
		"workstationCluster": {
			"name": "projects/stackyard/locations/us-central1/workstationClusters/cluster-new",
			"network": "projects/stackyard/global/networks/default"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workstations",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest workstations create cluster, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restCreateOperationName, _ := restCreateBody["name"].(string)

	var grpcCreateResp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsCreateWorkstationClusterMethod, &workstationspb.CreateWorkstationClusterRequest{
		Parent:               locationParent,
		WorkstationClusterId: "cluster-new",
		WorkstationCluster: &workstationspb.WorkstationCluster{
			Name:    locationParent + "/workstationClusters/cluster-new",
			Network: "projects/stackyard/global/networks/default",
		},
	}, &grpcCreateResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create workstation cluster, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCreateResp.GetName() != restCreateOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", grpcCreateResp.GetName(), restCreateOperationName)
	}

	restTokenResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/"+workstationRunning+":generateAccessToken", []byte(`{
		"workstation": "projects/stackyard/locations/us-central1/workstationClusters/cluster-1/workstationConfigs/config-1/workstations/workstation-running"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "workstations",
	})
	if restTokenResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest generate access token, got %d body=%s", restTokenResp.StatusCode, string(providerContractBody(t, restTokenResp)))
	}
	restTokenBody := providerContractJSONMap(t, restTokenResp)
	restToken, _ := restTokenBody["accessToken"].(string)

	var grpcTokenResp workstationspb.GenerateAccessTokenResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsGenerateAccessTokenMethod, &workstationspb.GenerateAccessTokenRequest{
		Workstation: workstationRunning,
	}, &grpcTokenResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for generate access token, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcTokenResp.GetAccessToken() != restToken {
		t.Fatalf("expected grpc access token %q to match rest %q", grpcTokenResp.GetAccessToken(), restToken)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsListWorkstationClustersMethod, &workstationspb.ListWorkstationClustersRequest{
		PageSize: 1,
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for list workstation clusters missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsStartWorkstationMethod, &workstationspb.StartWorkstationRequest{
		Name: workstationRunning,
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "workstation-must-be-stopped-before-start") {
		t.Fatalf("expected grpc failed precondition for start workstation, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsGetWorkstationMethod, &workstationspb.GetWorkstationRequest{
		Name: configName + "/workstations/missing-workstation",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "workstation-not-found") {
		t.Fatalf("expected grpc not found for missing workstation, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpWorkstationsGenerateAccessTokenMethod, &workstationspb.GenerateAccessTokenRequest{
		Workstation: workstationRunning,
		Expiration: &workstationspb.GenerateAccessTokenRequest_Ttl{
			Ttl: durationpb.New(25 * time.Hour),
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "access_token_expiration-invalid") {
		t.Fatalf("expected grpc invalid argument for generate access token invalid ttl, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
