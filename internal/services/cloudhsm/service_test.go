package cloudhsm

import "testing"

func TestClusterLifecycleStages12(t *testing.T) {
	svc := NewService()

	cluster, err := svc.CreateCluster(
		BackupRetentionPolicy{Type: "DAYS", Value: "30"},
		"hsm1.medium",
		"",
		[]string{"subnet-12345678"},
		"IPV4",
		[]Tag{{Key: "env", Value: "dev"}},
		"FIPS",
	)
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	if cluster.ClusterID == "" {
		t.Fatalf("expected cluster id")
	}

	clusters, _, err := svc.DescribeClusters(nil, "", 10)
	if err != nil {
		t.Fatalf("describe clusters: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected one cluster, got %d", len(clusters))
	}

	state, _, err := svc.InitializeCluster(cluster.ClusterID, "signed-cert", "trust-anchor")
	if err != nil {
		t.Fatalf("initialize cluster: %v", err)
	}
	if state != "INITIALIZED" {
		t.Fatalf("expected INITIALIZED, got %s", state)
	}

	hsm, err := svc.CreateHsm(cluster.ClusterID, "us-east-1a", "")
	if err != nil {
		t.Fatalf("create hsm: %v", err)
	}
	if hsm.HsmID == "" {
		t.Fatalf("expected hsm id")
	}

	if _, err := svc.ModifyCluster(BackupRetentionPolicy{Type: "DAYS", Value: "45"}, cluster.ClusterID); err != nil {
		t.Fatalf("modify cluster: %v", err)
	}

	removedHsmID, err := svc.DeleteHsm(cluster.ClusterID, hsm.HsmID, "", "")
	if err != nil {
		t.Fatalf("delete hsm: %v", err)
	}
	if removedHsmID != hsm.HsmID {
		t.Fatalf("expected removed hsm id %s, got %s", hsm.HsmID, removedHsmID)
	}

	if _, err := svc.DeleteCluster(cluster.ClusterID); err != nil {
		t.Fatalf("delete cluster: %v", err)
	}
}

func TestBackupLifecycleStage3(t *testing.T) {
	svc := NewService()
	cluster, err := svc.CreateCluster(
		BackupRetentionPolicy{Type: "DAYS", Value: "30"},
		"hsm1.medium",
		"",
		[]string{"subnet-12345678"},
		"",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	backups, _, err := svc.DescribeBackups("", 10, map[string][]string{"clusterIds": {cluster.ClusterID}}, nil, nil)
	if err != nil {
		t.Fatalf("describe backups: %v", err)
	}
	if len(backups) == 0 {
		t.Fatalf("expected at least one backup")
	}

	backup := backups[0]
	if _, err := svc.CopyBackupToRegion("us-west-2", backup.BackupID, []Tag{{Key: "copied", Value: "true"}}); err != nil {
		t.Fatalf("copy backup: %v", err)
	}
	if _, err := svc.ModifyBackupAttributes(backup.BackupID, true); err != nil {
		t.Fatalf("modify backup attributes: %v", err)
	}
	if _, err := svc.RestoreBackup(backup.BackupID); err != nil {
		t.Fatalf("restore backup: %v", err)
	}
	if _, err := svc.DeleteBackup(backup.BackupID); err != nil {
		t.Fatalf("delete backup: %v", err)
	}
}

func TestTagsAndPoliciesStage4(t *testing.T) {
	svc := NewService()
	cluster, err := svc.CreateCluster(
		BackupRetentionPolicy{Type: "DAYS", Value: "30"},
		"hsm1.medium",
		"",
		[]string{"subnet-12345678"},
		"",
		[]Tag{{Key: "env", Value: "dev"}},
		"",
	)
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	if err := svc.TagResource(cluster.ClusterID, []Tag{{Key: "team", Value: "platform"}}); err != nil {
		t.Fatalf("tag resource: %v", err)
	}
	tags, _, err := svc.ListTags(cluster.ClusterID, "", 10)
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) < 2 {
		t.Fatalf("expected at least two tags, got %d", len(tags))
	}
	if err := svc.UntagResource(cluster.ClusterID, []string{"team"}); err != nil {
		t.Fatalf("untag resource: %v", err)
	}

	clusterARN := clusterARN(cluster.ClusterID)
	if _, _, err := svc.PutResourcePolicy(clusterARN, "{\"Version\":\"2012-10-17\"}"); err != nil {
		t.Fatalf("put resource policy: %v", err)
	}
	policy, err := svc.GetResourcePolicy(clusterARN)
	if err != nil {
		t.Fatalf("get resource policy: %v", err)
	}
	if policy == "" {
		t.Fatalf("expected policy")
	}
	if _, _, err := svc.DeleteResourcePolicy(clusterARN); err != nil {
		t.Fatalf("delete resource policy: %v", err)
	}
}
