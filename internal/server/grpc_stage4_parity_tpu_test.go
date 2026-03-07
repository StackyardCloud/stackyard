package server

import (
	"net/http"
	"strings"
	"testing"

	tpupb "cloud.google.com/go/tpu/apiv1/tpupb"
)

func TestGCPStage4GRPCParity_TPU(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/nodes?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest tpu list nodes, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restNodes, ok := restListBody["nodes"].([]any)
	if !ok || len(restNodes) == 0 {
		t.Fatalf("expected rest nodes list, got %#v", restListBody["nodes"])
	}
	restNode, _ := restNodes[0].(map[string]any)
	restNodeName, _ := restNode["name"].(string)

	var grpcListNodesResp tpupb.ListNodesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpTPUListNodesMethod, &tpupb.ListNodesRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}, &grpcListNodesResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list nodes, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListNodesResp.GetNodes()) == 0 {
		t.Fatalf("expected grpc nodes list")
	}
	if grpcListNodesResp.GetNodes()[0].GetName() != restNodeName {
		t.Fatalf("expected grpc node name %q to match rest %q", grpcListNodesResp.GetNodes()[0].GetName(), restNodeName)
	}

	restGetNodeResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/nodes/node-1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "tpu",
	})
	if restGetNodeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest tpu get node, got %d body=%s", restGetNodeResp.StatusCode, string(providerContractBody(t, restGetNodeResp)))
	}
	restGetNodeBody := providerContractJSONMap(t, restGetNodeResp)
	restGetNodeState, _ := restGetNodeBody["state"].(string)

	var grpcGetNodeResp tpupb.Node
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTPUGetNodeMethod, &tpupb.GetNodeRequest{
		Name: "projects/stackyard/locations/us-central1/nodes/node-1",
	}, &grpcGetNodeResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for get node, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcGetNodeResp.GetName() != "projects/stackyard/locations/us-central1/nodes/node-1" {
		t.Fatalf("unexpected grpc node name: %q", grpcGetNodeResp.GetName())
	}
	if grpcGetNodeResp.GetState().String() != restGetNodeState {
		t.Fatalf("expected grpc node state %q to match rest %q", grpcGetNodeResp.GetState().String(), restGetNodeState)
	}

	restCreateNodeResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/nodes?nodeId=node-1", []byte(`{
		"node":{
			"name":"projects/stackyard/locations/us-central1/nodes/node-1",
			"acceleratorType":"v3-8",
			"tensorflowVersion":"v2-alpha"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "tpu",
	})
	if restCreateNodeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest tpu create node, got %d body=%s", restCreateNodeResp.StatusCode, string(providerContractBody(t, restCreateNodeResp)))
	}
	restCreateNodeBody := providerContractJSONMap(t, restCreateNodeResp)
	restCreateOperationName, _ := restCreateNodeBody["name"].(string)

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTPUCreateNodeMethod, &tpupb.CreateNodeRequest{
		Parent: "projects/stackyard/locations/us-central1",
		NodeId: "node-1",
		Node: &tpupb.Node{
			Name:              "projects/stackyard/locations/us-central1/nodes/node-1",
			AcceleratorType:   "v3-8",
			TensorflowVersion: "v2-alpha",
		},
	}, nil)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create node, got %q message=%q", grpcStatus, grpcMessage)
	}
	if strings.TrimSpace(restCreateOperationName) == "" {
		t.Fatalf("expected rest create operation name")
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTPUListNodesMethod, &tpupb.ListNodesRequest{
		Parent: "projects/stackyard",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for list nodes parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTPUStartNodeMethod, &tpupb.StartNodeRequest{
		Name: "projects/stackyard/locations/us-central1/nodes/node-1",
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "node-must-be-stopped-to-start") {
		t.Fatalf("expected grpc failed precondition for start node, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpTPUGetNodeMethod, &tpupb.GetNodeRequest{
		Name: "projects/stackyard/locations/us-central1/nodes/missing-node",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "node-not-found") {
		t.Fatalf("expected grpc not found for missing node, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
