package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
)

func TestDocDBStage2ClusterParameterGroups(t *testing.T) {
	ts := newDocDBTestServer(t)
	group := map[string]string{"DBClusterParameterGroupName": "tuned", "DBParameterGroupFamily": "docdb5.0", "Description": "tuned"}
	docdbWant(t, ts, http.StatusOK, "CreateDBClusterParameterGroup", group)
	docdbWantError(t, ts, http.StatusBadRequest, "DBParameterGroupAlreadyExists", "CreateDBClusterParameterGroup", group)
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "CreateDBClusterParameterGroup",
		map[string]string{"DBClusterParameterGroupName": "x", "DBParameterGroupFamily": "mysql8.0", "Description": "x"})

	// Unknown parameter, out-of-range value and immediate-on-static are refused.
	for _, p := range []map[string]string{
		{"Parameters.Parameter.1.ParameterName": "nope", "Parameters.Parameter.1.ParameterValue": "1"},
		{"Parameters.Parameter.1.ParameterName": "profiler_threshold_ms", "Parameters.Parameter.1.ParameterValue": "10"},
		{"Parameters.Parameter.1.ParameterName": "tls", "Parameters.Parameter.1.ParameterValue": "disabled", "Parameters.Parameter.1.ApplyMethod": "immediate"},
	} {
		p["DBClusterParameterGroupName"] = "tuned"
		if status, body := docdbCall(t, ts, "ModifyDBClusterParameterGroup", p); status != http.StatusBadRequest {
			t.Fatalf("modify %v: %d %s", p, status, body)
		}
	}
	docdbWant(t, ts, http.StatusOK, "ModifyDBClusterParameterGroup", map[string]string{"DBClusterParameterGroupName": "tuned",
		"Parameters.Parameter.1.ParameterName": "profiler_threshold_ms", "Parameters.Parameter.1.ParameterValue": "250", "Parameters.Parameter.1.ApplyMethod": "immediate"})
	user := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterParameters", map[string]string{"DBClusterParameterGroupName": "tuned", "Source": "user"})
	if !strings.Contains(user, "<ParameterValue>250</ParameterValue>") || strings.Count(user, "<Parameter>") != 1 {
		t.Fatalf("user parameters: %s", user)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBParameterGroupState", "ModifyDBClusterParameterGroup", map[string]string{"DBClusterParameterGroupName": "default.docdb5.0",
		"Parameters.Parameter.1.ParameterName": "profiler", "Parameters.Parameter.1.ParameterValue": "enabled"})

	// Copy keeps the user values; a group in use can't be deleted.
	docdbWant(t, ts, http.StatusOK, "CopyDBClusterParameterGroup", map[string]string{"SourceDBClusterParameterGroupIdentifier": "tuned",
		"TargetDBClusterParameterGroupIdentifier": "tuned-copy", "TargetDBClusterParameterGroupDescription": "copy"})
	if body := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterParameters", map[string]string{"DBClusterParameterGroupName": "tuned-copy", "Source": "user"}); !strings.Contains(body, "250") {
		t.Fatalf("copy lost values: %s", body)
	}
	docdbCreateCluster(t, ts, "uses-tuned", map[string]string{"DBClusterParameterGroupName": "tuned"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBParameterGroupState", "DeleteDBClusterParameterGroup", map[string]string{"DBClusterParameterGroupName": "tuned"})
	docdbWant(t, ts, http.StatusOK, "ResetDBClusterParameterGroup", map[string]string{"DBClusterParameterGroupName": "tuned-copy", "ResetAllParameters": "true"})
	if body := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterParameters", map[string]string{"DBClusterParameterGroupName": "tuned-copy", "Source": "user"}); strings.Contains(body, "<Parameter>") {
		t.Fatalf("reset left user values: %s", body)
	}
	docdbWant(t, ts, http.StatusOK, "DeleteDBClusterParameterGroup", map[string]string{"DBClusterParameterGroupName": "tuned-copy"})
	docdbWantError(t, ts, http.StatusNotFound, "DBParameterGroupNotFound", "DescribeDBClusterParameterGroups", map[string]string{"DBClusterParameterGroupName": "tuned-copy"})
	if body := docdbWant(t, ts, http.StatusOK, "DescribeEngineDefaultClusterParameters", map[string]string{"DBParameterGroupFamily": "docdb5.0"}); !strings.Contains(body, "<ParameterName>tls</ParameterName>") {
		t.Fatalf("engine defaults: %s", body)
	}
}

func TestDocDBStage2SnapshotsAndRestore(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbCreateCluster(t, ts, "ledger", map[string]string{"StorageEncrypted": "true"})
	docdbWant(t, ts, http.StatusOK, "CreateDBClusterSnapshot", map[string]string{"DBClusterSnapshotIdentifier": "ledger-1", "DBClusterIdentifier": "ledger"})
	docdbWantError(t, ts, http.StatusBadRequest, "DBClusterSnapshotAlreadyExistsFault", "CreateDBClusterSnapshot", map[string]string{"DBClusterSnapshotIdentifier": "ledger-1", "DBClusterIdentifier": "ledger"})
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterNotFoundFault", "CreateDBClusterSnapshot", map[string]string{"DBClusterSnapshotIdentifier": "x", "DBClusterIdentifier": "missing"})

	var snaps struct {
		Items []struct {
			ID   string `xml:"DBClusterSnapshotIdentifier"`
			Type string `xml:"SnapshotType"`
		} `xml:"DescribeDBClusterSnapshotsResult>DBClusterSnapshots>DBClusterSnapshot"`
	}
	body := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterSnapshots", map[string]string{"DBClusterIdentifier": "ledger"})
	if err := xml.Unmarshal([]byte(body), &snaps); err != nil || len(snaps.Items) != 2 {
		t.Fatalf("want one automated and one manual snapshot: %v %s", err, body)
	}
	automated := ""
	for _, s := range snaps.Items {
		if s.Type == "automated" {
			automated = s.ID
		}
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterSnapshotStateFault", "DeleteDBClusterSnapshot", map[string]string{"DBClusterSnapshotIdentifier": automated})

	// Sharing: account ids only, never public for an encrypted snapshot.
	docdbWant(t, ts, http.StatusOK, "ModifyDBClusterSnapshotAttribute", map[string]string{"DBClusterSnapshotIdentifier": "ledger-1", "AttributeName": "restore",
		"ValuesToAdd.AttributeValue.1": "210987654321"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "ModifyDBClusterSnapshotAttribute", map[string]string{"DBClusterSnapshotIdentifier": "ledger-1",
		"AttributeName": "restore", "ValuesToAdd.AttributeValue.1": "all"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "ModifyDBClusterSnapshotAttribute", map[string]string{"DBClusterSnapshotIdentifier": "ledger-1",
		"AttributeName": "copy", "ValuesToAdd.AttributeValue.1": "210987654321"})
	attrs := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterSnapshotAttributes", map[string]string{"DBClusterSnapshotIdentifier": "ledger-1"})
	if !strings.Contains(attrs, "<AttributeValue>210987654321</AttributeValue>") {
		t.Fatalf("attributes: %s", attrs)
	}

	docdbWant(t, ts, http.StatusOK, "CopyDBClusterSnapshot", map[string]string{"SourceDBClusterSnapshotIdentifier": "arn:aws:rds:us-east-1:123456789012:cluster-snapshot:ledger-1",
		"TargetDBClusterSnapshotIdentifier": "ledger-copy"})
	if body := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterSnapshotAttributes", map[string]string{"DBClusterSnapshotIdentifier": "ledger-copy"}); strings.Contains(body, "210987654321") {
		t.Fatalf("a copy must not inherit sharing: %s", body)
	}

	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterCombination", "RestoreDBClusterFromSnapshot", map[string]string{
		"DBClusterIdentifier": "ledger-restored", "SnapshotIdentifier": "ledger-1", "Engine": "docdb", "EngineVersion": "1.0.0"})
	docdbWant(t, ts, http.StatusOK, "RestoreDBClusterFromSnapshot", map[string]string{"DBClusterIdentifier": "ledger-restored", "SnapshotIdentifier": "ledger-1", "Engine": "docdb"})
	if c := docdbWant(t, ts, http.StatusOK, "DescribeDBClusters", map[string]string{"DBClusterIdentifier": "ledger-restored"}); !strings.Contains(c, "<StorageEncrypted>true</StorageEncrypted>") ||
		!strings.Contains(c, "<MasterUsername>stackyard</MasterUsername>") {
		t.Fatalf("restored cluster should keep the snapshot's encryption and user: %s", c)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterCombination", "RestoreDBClusterToPointInTime", map[string]string{
		"DBClusterIdentifier": "pitr", "SourceDBClusterIdentifier": "ledger"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidRestoreFault", "RestoreDBClusterToPointInTime", map[string]string{
		"DBClusterIdentifier": "pitr", "SourceDBClusterIdentifier": "ledger", "RestoreToTime": "2001-01-01T00:00:00Z"})
	docdbWant(t, ts, http.StatusOK, "RestoreDBClusterToPointInTime", map[string]string{"DBClusterIdentifier": "pitr", "SourceDBClusterIdentifier": "ledger", "UseLatestRestorableTime": "true"})
}

func TestDocDBStage2GlobalClusters(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbCreateCluster(t, ts, "primary", nil)
	docdbWant(t, ts, http.StatusOK, "CreateGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world", "SourceDBClusterIdentifier": "primary"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "CreateGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world2", "SourceDBClusterIdentifier": "primary"})
	docdbWantError(t, ts, http.StatusNotFound, "GlobalClusterNotFoundFault", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "secondary", "Engine": "docdb", "GlobalClusterIdentifier": "nope"})
	// Secondaries need no credentials.
	docdbWant(t, ts, http.StatusOK, "CreateDBCluster", map[string]string{"DBClusterIdentifier": "secondary", "Engine": "docdb", "GlobalClusterIdentifier": "world"})

	writerOf := func() string {
		var out struct {
			Members []struct {
				ARN    string `xml:"DBClusterArn"`
				Writer bool   `xml:"IsWriter"`
			} `xml:"DescribeGlobalClustersResult>GlobalClusters>GlobalClusterMember>GlobalClusterMembers>GlobalClusterMember"`
		}
		body := docdbWant(t, ts, http.StatusOK, "DescribeGlobalClusters", map[string]string{"GlobalClusterIdentifier": "world"})
		if err := xml.Unmarshal([]byte(body), &out); err != nil || len(out.Members) != 2 {
			t.Fatalf("global members: %v %s", err, body)
		}
		for _, m := range out.Members {
			if m.Writer {
				return m.ARN
			}
		}
		return ""
	}
	if w := writerOf(); !strings.HasSuffix(w, ":cluster:primary") {
		t.Fatalf("writer: %s", w)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "secondary", "SkipFinalSnapshot": "true"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidGlobalClusterStateFault", "DeleteGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world"})
	docdbWant(t, ts, http.StatusOK, "SwitchoverGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world", "TargetDbClusterIdentifier": "secondary"})
	if w := writerOf(); !strings.HasSuffix(w, ":cluster:secondary") {
		t.Fatalf("after switchover: %s", w)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidGlobalClusterStateFault", "FailoverGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world", "TargetDbClusterIdentifier": "secondary"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidGlobalClusterStateFault", "RemoveFromGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world", "DbClusterIdentifier": "secondary"})
	docdbWant(t, ts, http.StatusOK, "RemoveFromGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world", "DbClusterIdentifier": "primary"})
	docdbWant(t, ts, http.StatusOK, "RemoveFromGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world", "DbClusterIdentifier": "secondary"})
	docdbWant(t, ts, http.StatusOK, "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "secondary", "SkipFinalSnapshot": "true"})
	docdbWant(t, ts, http.StatusOK, "DeleteGlobalCluster", map[string]string{"GlobalClusterIdentifier": "world"})
}

func TestDocDBStage2TagsSubscriptionsAndSubnetGroups(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbCreateCluster(t, ts, "tagged", map[string]string{"Tags.Tag.1.Key": "team", "Tags.Tag.1.Value": "data"})
	arn := "arn:aws:rds:us-east-1:123456789012:cluster:tagged"
	if body := docdbWant(t, ts, http.StatusOK, "ListTagsForResource", map[string]string{"ResourceName": arn}); !strings.Contains(body, "<Key>team</Key>") {
		t.Fatalf("create-time tags: %s", body)
	}
	docdbWant(t, ts, http.StatusOK, "AddTagsToResource", map[string]string{"ResourceName": arn, "Tags.Tag.1.Key": "env", "Tags.Tag.1.Value": "dev"})
	docdbWant(t, ts, http.StatusOK, "RemoveTagsFromResource", map[string]string{"ResourceName": arn, "TagKeys.member.1": "team"})
	if body := docdbWant(t, ts, http.StatusOK, "ListTagsForResource", map[string]string{"ResourceName": arn}); strings.Contains(body, "team") || !strings.Contains(body, "env") {
		t.Fatalf("tags after edit: %s", body)
	}
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterNotFoundFault", "ListTagsForResource", map[string]string{"ResourceName": "arn:aws:rds:us-east-1:123456789012:cluster:missing"})
	docdbWantError(t, ts, http.StatusNotFound, "DBInstanceNotFound", "AddTagsToResource", map[string]string{"ResourceName": "arn:aws:rds:us-east-1:123456789012:db:missing", "Tags.Tag.1.Key": "k"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "AddTagsToResource", map[string]string{"ResourceName": arn, "Tags.Tag.1.Key": "aws:reserved"})

	docdbWant(t, ts, http.StatusOK, "CreateEventSubscription", map[string]string{"SubscriptionName": "alerts", "SnsTopicArn": "arn:aws:sns:us-east-1:123456789012:alerts", "SourceType": "db-cluster"})
	docdbWantError(t, ts, http.StatusNotFound, "SourceNotFound", "AddSourceIdentifierToSubscription", map[string]string{"SubscriptionName": "alerts", "SourceIdentifier": "ghost"})
	body := docdbWant(t, ts, http.StatusOK, "AddSourceIdentifierToSubscription", map[string]string{"SubscriptionName": "alerts", "SourceIdentifier": "tagged"})
	if !strings.Contains(body, "<SourceId>tagged</SourceId>") {
		t.Fatalf("add source: %s", body)
	}
	docdbWant(t, ts, http.StatusOK, "RemoveSourceIdentifierFromSubscription", map[string]string{"SubscriptionName": "alerts", "SourceIdentifier": "tagged"})
	docdbWantError(t, ts, http.StatusNotFound, "SourceNotFound", "RemoveSourceIdentifierFromSubscription", map[string]string{"SubscriptionName": "alerts", "SourceIdentifier": "tagged"})
	docdbWantError(t, ts, http.StatusNotFound, "SubscriptionNotFound", "AddSourceIdentifierToSubscription", map[string]string{"SubscriptionName": "nope", "SourceIdentifier": "tagged"})

	docdbWant(t, ts, http.StatusOK, "CreateDBSubnetGroup", map[string]string{"DBSubnetGroupName": "private", "DBSubnetGroupDescription": "private",
		"SubnetIds.SubnetIdentifier.1": "subnet-0a1b2c3d", "SubnetIds.SubnetIdentifier.2": "subnet-1a2b3c4d"})
	docdbCreateCluster(t, ts, "in-private", map[string]string{"DBSubnetGroupName": "private"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBSubnetGroupStateFault", "DeleteDBSubnetGroup", map[string]string{"DBSubnetGroupName": "private"})
	docdbWant(t, ts, http.StatusOK, "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "in-private", "SkipFinalSnapshot": "true"})
	docdbWant(t, ts, http.StatusOK, "DeleteDBSubnetGroup", map[string]string{"DBSubnetGroupName": "private"})
}

func TestDocDBStage2ReviewFixes(t *testing.T) {
	ts := newDocDBTestServer(t)
	// A bad create-time tag rejects the call before anything exists.
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "half-made", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1", "Tags.Tag.1.Key": "aws:nope"})
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterNotFoundFault", "DescribeDBClusters", map[string]string{"DBClusterIdentifier": "half-made"})

	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "DescribeDBClusters", map[string]string{
		"Filters.Filter.1.Name": "colour", "Filters.Filter.1.Values.Value.1": "blue"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "c", "Engine": "docdb", "MasterUsername": "has-hyphen", "MasterUserPassword": "Passw0rd-1"})

	// A docdb-engine cluster made through the rds API has no zone list; adding
	// a docdb instance to it must not divide by zero.
	if status, body := queryCall(t, ts, "aws-cli/2.41.0 md/command#rds.create-db-cluster", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "via-rds", "Engine": "docdb", "MasterUsername": "admin", "MasterUserPassword": "Passw0rd!123"}); status != http.StatusOK {
		t.Fatalf("rds create: %d %s", status, body)
	}
	docdbCreateInstance(t, ts, "via-rds-1", "via-rds", nil)

	big := strings.Repeat("x", docdbMaxBodyBytes)
	if status, body := docdbCall(t, ts, "DescribeDBClusters", map[string]string{"Marker": big}); status != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body: %d %s", status, body[:min(len(body), 200)])
	}
}
