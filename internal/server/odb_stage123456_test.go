package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestODBStage12NetworkAndInfrastructureLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := odbRequest(t, ts, http.MethodPost, "/InitializeService", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/GetOciOnboardingStatus", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/CreateOdbNetwork", `{"name":"stage-odb-network-001"}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodeODBPayload(t, resp)
	networkID := odbPayloadStringValue(payload, "odbNetworkId")
	if networkID == "" {
		t.Fatalf("expected CreateOdbNetwork to return odbNetworkId")
	}

	resp = odbRequest(t, ts, http.MethodPost, "/GetOdbNetwork", `{"odbNetworkId":"`+networkID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListOdbNetworks", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, networkID) {
		t.Fatalf("expected ListOdbNetworks to include %s, got %q", networkID, body)
	}
	resp = odbRequest(t, ts, http.MethodPost, "/UpdateOdbNetwork", `{"odbNetworkId":"`+networkID+`","displayName":"stage-odb-network-001-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/CreateOdbPeeringConnection", `{"name":"stage-odb-peering-001"}`)
	assertStatus(t, resp, http.StatusOK)
	payload = decodeODBPayload(t, resp)
	peeringID := odbPayloadStringValue(payload, "odbPeeringConnectionId")
	if peeringID == "" {
		t.Fatalf("expected CreateOdbPeeringConnection to return odbPeeringConnectionId")
	}
	resp = odbRequest(t, ts, http.MethodPost, "/GetOdbPeeringConnection", `{"odbPeeringConnectionId":"`+peeringID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListOdbPeeringConnections", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/UpdateOdbPeeringConnection", `{"odbPeeringConnectionId":"`+peeringID+`","status":"ACTIVE"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/CreateCloudExadataInfrastructure", `{"name":"stage-exadata-001"}`)
	assertStatus(t, resp, http.StatusOK)
	payload = decodeODBPayload(t, resp)
	exadataID := odbPayloadStringValue(payload, "cloudExadataInfrastructureId")
	if exadataID == "" {
		t.Fatalf("expected CreateCloudExadataInfrastructure to return cloudExadataInfrastructureId")
	}
	resp = odbRequest(t, ts, http.MethodPost, "/GetCloudExadataInfrastructure", `{"cloudExadataInfrastructureId":"`+exadataID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListCloudExadataInfrastructures", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/GetCloudExadataInfrastructureUnallocatedResources", `{"cloudExadataInfrastructureId":"`+exadataID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/UpdateCloudExadataInfrastructure", `{"cloudExadataInfrastructureId":"`+exadataID+`","displayName":"stage-exadata-001-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/CreateCloudVmCluster", `{"name":"stage-vm-cluster-001"}`)
	assertStatus(t, resp, http.StatusOK)
	payload = decodeODBPayload(t, resp)
	vmClusterID := odbPayloadStringValue(payload, "cloudVmClusterId")
	if vmClusterID == "" {
		t.Fatalf("expected CreateCloudVmCluster to return cloudVmClusterId")
	}
	resp = odbRequest(t, ts, http.MethodPost, "/GetCloudVmCluster", `{"cloudVmClusterId":"`+vmClusterID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListCloudVmClusters", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/DeleteCloudVmCluster", `{"cloudVmClusterId":"`+vmClusterID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/DeleteCloudExadataInfrastructure", `{"cloudExadataInfrastructureId":"`+exadataID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/DeleteOdbPeeringConnection", `{"odbPeeringConnectionId":"`+peeringID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/DeleteOdbNetwork", `{"odbNetworkId":"`+networkID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestODBStage34AutonomousAndDatabaseNodeSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := odbRequest(t, ts, http.MethodPost, "/CreateCloudAutonomousVmCluster", `{"name":"stage-autonomous-cluster-001"}`)
	assertStatus(t, resp, http.StatusOK)
	payload := decodeODBPayload(t, resp)
	autonomousClusterID := odbPayloadStringValue(payload, "cloudAutonomousVmClusterId")
	if autonomousClusterID == "" {
		t.Fatalf("expected CreateCloudAutonomousVmCluster to return cloudAutonomousVmClusterId")
	}
	resp = odbRequest(t, ts, http.MethodPost, "/GetCloudAutonomousVmCluster", `{"cloudAutonomousVmClusterId":"`+autonomousClusterID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListCloudAutonomousVmClusters", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListAutonomousVirtualMachines", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/ListDbNodes", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/GetDbNode", `{"dbNodeId":"db-node-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/RebootDbNode", `{"dbNodeId":"db-node-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/StopDbNode", `{"dbNodeId":"db-node-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/StartDbNode", `{"dbNodeId":"db-node-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/ListDbServers", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/GetDbServer", `{"dbServerId":"db-server-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListDbSystemShapes", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListGiVersions", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListSystemVersions", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/DeleteCloudAutonomousVmCluster", `{"cloudAutonomousVmClusterId":"`+autonomousClusterID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestODBStage56TaggingIAMAndValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:odb:us-east-1:123456789012:odb-network/odb-network-000001"

	resp := odbRequest(
		t,
		ts,
		http.MethodPost,
		"/AssociateIamRoleToResource",
		`{"resourceArn":"`+resourceARN+`","iamRoleArn":"arn:aws:iam::123456789012:role/stage-odb-role"}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(
		t,
		ts,
		http.MethodPost,
		"/DisassociateIamRoleFromResource",
		`{"resourceArn":"`+resourceARN+`","iamRoleArn":"arn:aws:iam::123456789012:role/stage-odb-role"}`,
	)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/TagResource", `{"resourceArn":"`+resourceARN+`","tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = odbRequest(t, ts, http.MethodPost, "/UntagResource", `{"resourceArn":"`+resourceARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = odbRequest(t, ts, http.MethodPost, "/ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"owner"`) {
		t.Fatalf("expected owner tag removed, got %q", body)
	}

	resp = odbRequest(t, ts, http.MethodPost, "/AcceptMarketplaceRegistration", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = odbRequest(t, ts, http.MethodPost, "/odb/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/ListOdbNetworks",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"odb",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeODBPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func odbPayloadStringValue(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	asString, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString)
}
