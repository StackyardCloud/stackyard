package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudhsmRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "BaldrApiService." + action,
		},
		"cloudhsm",
	)
}

func TestCloudHSMOperationCoverage(t *testing.T) {
	if len(cloudhsmOperations) != 18 {
		t.Fatalf("expected 18 CloudHSM operations from docs, got %d", len(cloudhsmOperations))
	}
	if len(cloudhsmOperationByName) != len(cloudhsmOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CopyBackupToRegion",
		"CreateCluster",
		"CreateHsm",
		"DeleteBackup",
		"DeleteCluster",
		"DeleteHsm",
		"DeleteResourcePolicy",
		"DescribeBackups",
		"DescribeClusters",
		"GetResourcePolicy",
		"InitializeCluster",
		"ListTags",
		"ModifyBackupAttributes",
		"ModifyCluster",
		"PutResourcePolicy",
		"RestoreBackup",
		"TagResource",
		"UntagResource",
	}
	for _, name := range required {
		if _, ok := cloudhsmOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestCloudHSMStage1234Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cloudhsmRequest(t, ts, "CreateCluster", []byte(`{
		"HsmType":"hsm1.medium",
		"SubnetIds":["subnet-12345678"],
		"TagList":[{"Key":"env","Value":"dev"}]
	}`))
	assertStatus(t, createResp, http.StatusOK)
	var createOut struct {
		Cluster struct {
			ClusterID string `json:"ClusterId"`
		} `json:"Cluster"`
	}
	if err := json.Unmarshal(mustBody(t, createResp), &createOut); err != nil {
		t.Fatalf("unmarshal create cluster response: %v", err)
	}
	if createOut.Cluster.ClusterID == "" {
		t.Fatalf("expected cluster id")
	}
	clusterID := createOut.Cluster.ClusterID
	clusterARN := fmt.Sprintf("arn:aws:cloudhsm:us-east-1:123456789012:cluster/%s", clusterID)

	describeClustersResp := cloudhsmRequest(t, ts, "DescribeClusters", []byte(`{"MaxResults":10}`))
	assertStatus(t, describeClustersResp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, describeClustersResp)), clusterID) {
		t.Fatalf("expected describe clusters to include cluster id")
	}

	initializeResp := cloudhsmRequest(t, ts, "InitializeCluster", []byte(`{
		"ClusterId":"`+clusterID+`",
		"SignedCert":"signed-cert",
		"TrustAnchor":"trust-anchor"
	}`))
	assertStatus(t, initializeResp, http.StatusOK)

	createHsmResp := cloudhsmRequest(t, ts, "CreateHsm", []byte(`{
		"ClusterId":"`+clusterID+`",
		"AvailabilityZone":"us-east-1a"
	}`))
	assertStatus(t, createHsmResp, http.StatusOK)
	var createHsmOut struct {
		Hsm struct {
			HsmID string `json:"HsmId"`
		} `json:"Hsm"`
	}
	if err := json.Unmarshal(mustBody(t, createHsmResp), &createHsmOut); err != nil {
		t.Fatalf("unmarshal create hsm response: %v", err)
	}
	if createHsmOut.Hsm.HsmID == "" {
		t.Fatalf("expected hsm id")
	}

	modifyClusterResp := cloudhsmRequest(t, ts, "ModifyCluster", []byte(`{
		"ClusterId":"`+clusterID+`",
		"BackupRetentionPolicy":{"Type":"DAYS","Value":"45"}
	}`))
	assertStatus(t, modifyClusterResp, http.StatusOK)

	describeBackupsResp := cloudhsmRequest(t, ts, "DescribeBackups", []byte(`{
		"Filters":{"clusterIds":["`+clusterID+`"]},
		"MaxResults":10
	}`))
	assertStatus(t, describeBackupsResp, http.StatusOK)
	var describeBackupsOut struct {
		Backups []struct {
			BackupID string `json:"BackupId"`
		} `json:"Backups"`
	}
	if err := json.Unmarshal(mustBody(t, describeBackupsResp), &describeBackupsOut); err != nil {
		t.Fatalf("unmarshal describe backups response: %v", err)
	}
	if len(describeBackupsOut.Backups) == 0 || describeBackupsOut.Backups[0].BackupID == "" {
		t.Fatalf("expected backup id")
	}
	backupID := describeBackupsOut.Backups[0].BackupID

	copyBackupResp := cloudhsmRequest(t, ts, "CopyBackupToRegion", []byte(`{
		"DestinationRegion":"us-west-2",
		"BackupId":"`+backupID+`",
		"TagList":[{"Key":"copied","Value":"true"}]
	}`))
	assertStatus(t, copyBackupResp, http.StatusOK)

	modifyBackupResp := cloudhsmRequest(t, ts, "ModifyBackupAttributes", []byte(`{
		"BackupId":"`+backupID+`",
		"NeverExpires":true
	}`))
	assertStatus(t, modifyBackupResp, http.StatusOK)

	restoreBackupResp := cloudhsmRequest(t, ts, "RestoreBackup", []byte(`{"BackupId":"`+backupID+`"}`))
	assertStatus(t, restoreBackupResp, http.StatusOK)

	tagResourceResp := cloudhsmRequest(t, ts, "TagResource", []byte(`{
		"ResourceId":"`+clusterID+`",
		"TagList":[{"Key":"team","Value":"platform"}]
	}`))
	assertStatus(t, tagResourceResp, http.StatusOK)

	listTagsResp := cloudhsmRequest(t, ts, "ListTags", []byte(`{"ResourceId":"`+clusterID+`","MaxResults":10}`))
	assertStatus(t, listTagsResp, http.StatusOK)
	if !strings.Contains(string(mustBody(t, listTagsResp)), "\"Key\":\"env\"") {
		t.Fatalf("expected env tag in list tags")
	}

	untagResp := cloudhsmRequest(t, ts, "UntagResource", []byte(`{
		"ResourceId":"`+clusterID+`",
		"TagKeyList":["team"]
	}`))
	assertStatus(t, untagResp, http.StatusOK)

	putPolicyResp := cloudhsmRequest(t, ts, "PutResourcePolicy", []byte(`{
		"ResourceArn":"`+clusterARN+`",
		"Policy":"{}"
	}`))
	assertStatus(t, putPolicyResp, http.StatusOK)

	getPolicyResp := cloudhsmRequest(t, ts, "GetResourcePolicy", []byte(`{"ResourceArn":"`+clusterARN+`"}`))
	assertStatus(t, getPolicyResp, http.StatusOK)

	deletePolicyResp := cloudhsmRequest(t, ts, "DeleteResourcePolicy", []byte(`{"ResourceArn":"`+clusterARN+`"}`))
	assertStatus(t, deletePolicyResp, http.StatusOK)

	deleteHsmResp := cloudhsmRequest(t, ts, "DeleteHsm", []byte(`{
		"ClusterId":"`+clusterID+`",
		"HsmId":"`+createHsmOut.Hsm.HsmID+`"
	}`))
	assertStatus(t, deleteHsmResp, http.StatusOK)

	deleteBackupResp := cloudhsmRequest(t, ts, "DeleteBackup", []byte(`{"BackupId":"`+backupID+`"}`))
	assertStatus(t, deleteBackupResp, http.StatusOK)

	deleteClusterResp := cloudhsmRequest(t, ts, "DeleteCluster", []byte(`{"ClusterId":"`+clusterID+`"}`))
	assertStatus(t, deleteClusterResp, http.StatusOK)
}
