package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryDBStage1Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "CreateUser", []byte(`{
		"UserName":"stage1-user",
		"AccessString":"on ~* +@all",
		"AuthenticationMode":{"Type":"password","Passwords":["secret"]}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "CreateACL", []byte(`{
		"ACLName":"stage1-acl",
		"UserNames":["stage1-user"]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "CreateSubnetGroup", []byte(`{
		"SubnetGroupName":"stage1-subnet",
		"Description":"stage1",
		"SubnetIds":["subnet-11111111","subnet-22222222"]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "CreateCluster", []byte(`{
		"ClusterName":"stage1-cluster",
		"NodeType":"db.r6g.large",
		"ACLName":"stage1-acl",
		"SubnetGroupName":"stage1-subnet",
		"NumShards":2
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeUsers", []byte(`{"UserName":"stage1-user"}`))
	assertStatus(t, resp, http.StatusOK)
	var usersOut struct {
		Users []map[string]any `json:"Users"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &usersOut); err != nil {
		t.Fatalf("describe users unmarshal: %v", err)
	}
	if len(usersOut.Users) != 1 {
		t.Fatalf("expected one user, got %d", len(usersOut.Users))
	}

	resp = memorydbRequest(t, ts, "DescribeACLs", []byte(`{"ACLName":"stage1-acl"}`))
	assertStatus(t, resp, http.StatusOK)
	var aclOut struct {
		ACLs []map[string]any `json:"ACLs"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &aclOut); err != nil {
		t.Fatalf("describe acls unmarshal: %v", err)
	}
	if len(aclOut.ACLs) != 1 {
		t.Fatalf("expected one acl, got %d", len(aclOut.ACLs))
	}

	resp = memorydbRequest(t, ts, "DescribeSubnetGroups", []byte(`{"SubnetGroupName":"stage1-subnet"}`))
	assertStatus(t, resp, http.StatusOK)
	var subnetOut struct {
		SubnetGroups []map[string]any `json:"SubnetGroups"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &subnetOut); err != nil {
		t.Fatalf("describe subnet groups unmarshal: %v", err)
	}
	if len(subnetOut.SubnetGroups) != 1 {
		t.Fatalf("expected one subnet group, got %d", len(subnetOut.SubnetGroups))
	}

	resp = memorydbRequest(t, ts, "DescribeClusters", []byte(`{"ClusterName":"stage1-cluster"}`))
	assertStatus(t, resp, http.StatusOK)
	var clustersOut struct {
		Clusters []map[string]any `json:"Clusters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &clustersOut); err != nil {
		t.Fatalf("describe clusters unmarshal: %v", err)
	}
	if len(clustersOut.Clusters) != 1 {
		t.Fatalf("expected one cluster, got %d", len(clustersOut.Clusters))
	}

	resp = memorydbRequest(t, ts, "UpdateUser", []byte(`{
		"UserName":"stage1-user",
		"AccessString":"on ~app::* +@read",
		"AuthenticationMode":{"Type":"password","Passwords":["new-secret"]}
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "UpdateACL", []byte(`{
		"ACLName":"stage1-acl",
		"UserNamesToAdd":["stage1-user"],
		"UserNamesToRemove":[]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "UpdateSubnetGroup", []byte(`{
		"SubnetGroupName":"stage1-subnet",
		"Description":"stage1-updated"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "UpdateCluster", []byte(`{
		"ClusterName":"stage1-cluster",
		"Description":"updated",
		"SnapshotRetentionLimit":3,
		"NodeType":"db.r6g.xlarge"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DeleteCluster", []byte(`{"ClusterName":"stage1-cluster"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "DeleteSubnetGroup", []byte(`{"SubnetGroupName":"stage1-subnet"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "DeleteUser", []byte(`{"UserName":"stage1-user"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "DeleteACL", []byte(`{"ACLName":"stage1-acl"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestMemoryDBStage2Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "CreateACL", []byte(`{"ACLName":"stage2-acl"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "CreateCluster", []byte(`{
		"ClusterName":"stage2-cluster",
		"NodeType":"db.r6g.large",
		"ACLName":"stage2-acl"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "CreateParameterGroup", []byte(`{
		"ParameterGroupName":"stage2-params",
		"Family":"memorydb_redis7",
		"Description":"stage2"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "UpdateParameterGroup", []byte(`{
		"ParameterGroupName":"stage2-params",
		"ParameterNameValues":[
			{"ParameterName":"hash-max-ziplist-entries","ParameterValue":"128"},
			{"ParameterName":"activedefrag","ParameterValue":"yes"}
		]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeParameterGroups", []byte(`{"ParameterGroupName":"stage2-params"}`))
	assertStatus(t, resp, http.StatusOK)
	var groupsOut struct {
		ParameterGroups []map[string]any `json:"ParameterGroups"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &groupsOut); err != nil {
		t.Fatalf("describe parameter groups unmarshal: %v", err)
	}
	if len(groupsOut.ParameterGroups) != 1 {
		t.Fatalf("expected one parameter group, got %d", len(groupsOut.ParameterGroups))
	}

	resp = memorydbRequest(t, ts, "DescribeParameters", []byte(`{"ParameterGroupName":"stage2-params"}`))
	assertStatus(t, resp, http.StatusOK)
	var paramsOut struct {
		Parameters []map[string]any `json:"Parameters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &paramsOut); err != nil {
		t.Fatalf("describe parameters unmarshal: %v", err)
	}
	if len(paramsOut.Parameters) == 0 {
		t.Fatalf("expected parameter entries")
	}

	resp = memorydbRequest(t, ts, "ResetParameterGroup", []byte(`{
		"ParameterGroupName":"stage2-params",
		"AllParameters":true
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "CreateSnapshot", []byte(`{
		"ClusterName":"stage2-cluster",
		"SnapshotName":"stage2-snapshot"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "CopySnapshot", []byte(`{
		"SourceSnapshotName":"stage2-snapshot",
		"TargetSnapshotName":"stage2-snapshot-copy"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeSnapshots", []byte(`{"MaxResults":20}`))
	assertStatus(t, resp, http.StatusOK)
	var snapshotsOut struct {
		Snapshots []map[string]any `json:"Snapshots"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &snapshotsOut); err != nil {
		t.Fatalf("describe snapshots unmarshal: %v", err)
	}
	if len(snapshotsOut.Snapshots) < 2 {
		t.Fatalf("expected at least two snapshots, got %d", len(snapshotsOut.Snapshots))
	}

	resp = memorydbRequest(t, ts, "DeleteSnapshot", []byte(`{"SnapshotName":"stage2-snapshot"}`))
	assertStatus(t, resp, http.StatusOK)
	resp = memorydbRequest(t, ts, "DeleteParameterGroup", []byte(`{"ParameterGroupName":"stage2-params"}`))
	assertStatus(t, resp, http.StatusOK)
}

func TestMemoryDBStage3Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := memorydbRequest(t, ts, "CreateMultiRegionCluster", []byte(`{
		"MultiRegionClusterNameSuffix":"stage3-mrc",
		"NodeType":"db.r6g.large",
		"Engine":"redis",
		"EngineVersion":"7.1"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DescribeMultiRegionClusters", []byte(`{"MultiRegionClusterName":"stage3-mrc"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		MultiRegionClusters []map[string]any `json:"MultiRegionClusters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("describe multi-region clusters unmarshal: %v", err)
	}
	if len(describeOut.MultiRegionClusters) != 1 {
		t.Fatalf("expected one multi-region cluster, got %d", len(describeOut.MultiRegionClusters))
	}

	resp = memorydbRequest(t, ts, "UpdateMultiRegionCluster", []byte(`{
		"MultiRegionClusterName":"stage3-mrc",
		"Description":"updated stage3",
		"NodeType":"db.r6g.xlarge"
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "ListAllowedMultiRegionClusterUpdates", []byte(`{"MultiRegionClusterName":"stage3-mrc"}`))
	assertStatus(t, resp, http.StatusOK)
	var allowedOut struct {
		ScaleUpNodeTypes   []string `json:"ScaleUpNodeTypes"`
		ScaleDownNodeTypes []string `json:"ScaleDownNodeTypes"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &allowedOut); err != nil {
		t.Fatalf("list allowed updates unmarshal: %v", err)
	}
	if len(allowedOut.ScaleUpNodeTypes) == 0 || len(allowedOut.ScaleDownNodeTypes) == 0 {
		t.Fatalf("expected scale up/down node type options")
	}

	resp = memorydbRequest(t, ts, "BatchUpdateCluster", []byte(`{"ClusterNames":["stage3-cluster"]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "FailoverShard", []byte(`{"ClusterName":"stage3-cluster","ShardName":"0001"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = memorydbRequest(t, ts, "DeleteMultiRegionCluster", []byte(`{"MultiRegionClusterName":"stage3-mrc"}`))
	assertStatus(t, resp, http.StatusOK)
}
