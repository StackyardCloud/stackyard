package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDRSShard6WrappedMutationResponses(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := drsRequest(t, ts, http.MethodPost, "/CreateLaunchConfigurationTemplate", `{
		"copyPrivateIp":true,
		"copyTags":true,
		"launchDisposition":"STOPPED",
		"targetInstanceTypeRightSizingMethod":"NONE"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var createLaunchTemplateOut struct {
		LaunchConfigurationTemplate map[string]any `json:"launchConfigurationTemplate"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createLaunchTemplateOut); err != nil {
		t.Fatalf("unmarshal create launch configuration template response: %v", err)
	}
	launchTemplateID, _ := createLaunchTemplateOut.LaunchConfigurationTemplate["launchConfigurationTemplateID"].(string)
	if launchTemplateID == "" {
		t.Fatalf("expected launchConfigurationTemplate.launchConfigurationTemplateID")
	}

	resp = drsRequest(t, ts, http.MethodPost, "/CreateExtendedSourceServer", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var createSourceServerOut struct {
		SourceServer map[string]any `json:"sourceServer"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createSourceServerOut); err != nil {
		t.Fatalf("unmarshal create extended source server response: %v", err)
	}
	if id, _ := createSourceServerOut.SourceServer["sourceServerID"].(string); id == "" {
		t.Fatalf("expected sourceServer.sourceServerID")
	}

	resp = drsRequest(t, ts, http.MethodPost, "/AssociateSourceNetworkStack", `{
		"sourceNetworkID":"sn-00000001",
		"cfnStackName":"stackyard-source-network-stack"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var associateSourceNetworkStackOut struct {
		Job map[string]any `json:"job"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &associateSourceNetworkStackOut); err != nil {
		t.Fatalf("unmarshal associate source network stack response: %v", err)
	}
	if jobID, _ := associateSourceNetworkStackOut.Job["jobID"].(string); jobID == "" {
		t.Fatalf("expected job.jobID")
	}

	resp = drsRequest(t, ts, http.MethodPost, "/StartReplication", `{"sourceServerID":"s-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var startReplicationOut struct {
		SourceServer map[string]any `json:"sourceServer"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startReplicationOut); err != nil {
		t.Fatalf("unmarshal start replication response: %v", err)
	}
	if id, _ := startReplicationOut.SourceServer["sourceServerID"].(string); id != "s-00000001" {
		t.Fatalf("expected sourceServer.sourceServerID to be preserved, got %q", id)
	}

	resp = drsRequest(t, ts, http.MethodPost, "/StartSourceNetworkReplication", `{"sourceNetworkID":"sn-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var startSourceNetworkReplicationOut struct {
		SourceNetwork map[string]any `json:"sourceNetwork"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startSourceNetworkReplicationOut); err != nil {
		t.Fatalf("unmarshal start source network replication response: %v", err)
	}
	if id, _ := startSourceNetworkReplicationOut.SourceNetwork["sourceNetworkID"].(string); id != "sn-00000001" {
		t.Fatalf("expected sourceNetwork.sourceNetworkID to be preserved, got %q", id)
	}

	resp = drsRequest(t, ts, http.MethodPost, "/StopReplication", `{"sourceServerID":"s-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var stopReplicationOut struct {
		SourceServer map[string]any `json:"sourceServer"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &stopReplicationOut); err != nil {
		t.Fatalf("unmarshal stop replication response: %v", err)
	}
	if id, _ := stopReplicationOut.SourceServer["sourceServerID"].(string); id != "s-00000001" {
		t.Fatalf("expected stop replication to preserve sourceServer.sourceServerID, got %q", id)
	}

	resp = drsRequest(t, ts, http.MethodPost, "/StopSourceNetworkReplication", `{"sourceNetworkID":"sn-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var stopSourceNetworkReplicationOut struct {
		SourceNetwork map[string]any `json:"sourceNetwork"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &stopSourceNetworkReplicationOut); err != nil {
		t.Fatalf("unmarshal stop source network replication response: %v", err)
	}
	if id, _ := stopSourceNetworkReplicationOut.SourceNetwork["sourceNetworkID"].(string); id != "sn-00000001" {
		t.Fatalf("expected stop source network replication to preserve sourceNetwork.sourceNetworkID, got %q", id)
	}

	resp = drsRequest(t, ts, http.MethodPost, "/UpdateLaunchConfigurationTemplate", `{
		"launchConfigurationTemplateID":"`+launchTemplateID+`",
		"copyTags":true
	}`)
	assertStatus(t, resp, http.StatusOK)
	var updateLaunchTemplateOut struct {
		LaunchConfigurationTemplate map[string]any `json:"launchConfigurationTemplate"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateLaunchTemplateOut); err != nil {
		t.Fatalf("unmarshal update launch configuration template response: %v", err)
	}
	if id, _ := updateLaunchTemplateOut.LaunchConfigurationTemplate["launchConfigurationTemplateID"].(string); id != launchTemplateID {
		t.Fatalf("expected launchConfigurationTemplate.launchConfigurationTemplateID to be preserved, got %q", id)
	}

	resp = drsRequest(t, ts, http.MethodPost, "/RetryDataReplication", `{"sourceServerID":"s-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var retryDataReplicationOut struct {
		SourceServer map[string]any `json:"sourceServer"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &retryDataReplicationOut); err != nil {
		t.Fatalf("unmarshal retry data replication response: %v", err)
	}
	if id, _ := retryDataReplicationOut.SourceServer["sourceServerID"].(string); id != "s-00000001" {
		t.Fatalf("expected sourceServer.sourceServerID to be preserved, got %q", id)
	}

	resp = drsRequest(t, ts, http.MethodPost, "/ReverseReplication", `{"sourceServerID":"s-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var reverseReplicationOut struct {
		ReversedDirectionSourceServerArn string `json:"reversedDirectionSourceServerArn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &reverseReplicationOut); err != nil {
		t.Fatalf("unmarshal reverse replication response: %v", err)
	}
	if reverseReplicationOut.ReversedDirectionSourceServerArn == "" {
		t.Fatalf("expected reversedDirectionSourceServerArn")
	}

	resp = drsRequest(t, ts, http.MethodPost, "/DisconnectSourceServer", `{"sourceServerID":"s-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var disconnectSourceServerOut struct {
		SourceServer map[string]any `json:"sourceServer"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &disconnectSourceServerOut); err != nil {
		t.Fatalf("unmarshal disconnect source server response: %v", err)
	}
	if id, _ := disconnectSourceServerOut.SourceServer["sourceServerID"].(string); id != "s-00000001" {
		t.Fatalf("expected sourceServer.sourceServerID to be preserved, got %q", id)
	}
}

func TestDRSShard6RecoverySnapshotAndFailbackShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := drsRequest(t, ts, http.MethodPost, "/DescribeRecoverySnapshots", `{"sourceServerID":"s-00000001"}`)
	assertStatus(t, resp, http.StatusOK)
	var describeRecoverySnapshotsOut struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeRecoverySnapshotsOut); err != nil {
		t.Fatalf("unmarshal describe recovery snapshots response: %v", err)
	}
	if len(describeRecoverySnapshotsOut.Items) == 0 {
		t.Fatalf("expected at least one recovery snapshot")
	}
	if snapshotID, _ := describeRecoverySnapshotsOut.Items[0]["snapshotID"].(string); snapshotID == "" {
		t.Fatalf("expected snapshotID on recovery snapshot item")
	}

	resp = drsRequest(t, ts, http.MethodPost, "/GetFailbackReplicationConfiguration", `{
		"sourceServerID":"s-00000001",
		"recoveryInstanceID":"i-00000001"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var getFailbackOut struct {
		RecoveryInstanceID string `json:"recoveryInstanceID"`
		UsePrivateIP       bool   `json:"usePrivateIP"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getFailbackOut); err != nil {
		t.Fatalf("unmarshal get failback replication configuration response: %v", err)
	}
	if getFailbackOut.RecoveryInstanceID != "i-00000001" {
		t.Fatalf("expected recoveryInstanceID to be preserved, got %q", getFailbackOut.RecoveryInstanceID)
	}
	if getFailbackOut.UsePrivateIP {
		t.Fatalf("expected usePrivateIP to default to false")
	}
}
