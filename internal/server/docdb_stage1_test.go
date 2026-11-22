package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func docdbCreateCluster(t *testing.T, ts *httptest.Server, id string, extra map[string]string) {
	t.Helper()
	params := map[string]string{"DBClusterIdentifier": id, "Engine": "docdb", "MasterUsername": "stackyard", "MasterUserPassword": "Passw0rd-1"}
	for k, v := range extra {
		params[k] = v
	}
	docdbWant(t, ts, http.StatusOK, "CreateDBCluster", params)
}

func docdbCreateInstance(t *testing.T, ts *httptest.Server, id, cluster string, extra map[string]string) {
	t.Helper()
	params := map[string]string{"DBInstanceIdentifier": id, "DBInstanceClass": "db.r6g.large", "Engine": "docdb", "DBClusterIdentifier": cluster}
	for k, v := range extra {
		params[k] = v
	}
	docdbWant(t, ts, http.StatusOK, "CreateDBInstance", params)
}

type docdbClustersResponse struct {
	Clusters []struct {
		ID      string   `xml:"DBClusterIdentifier"`
		Status  string   `xml:"Status"`
		Engine  string   `xml:"Engine"`
		Version string   `xml:"EngineVersion"`
		Port    int      `xml:"Port"`
		PG      string   `xml:"DBClusterParameterGroup"`
		Protect bool     `xml:"DeletionProtection"`
		Logs    []string `xml:"EnabledCloudwatchLogsExports>member"`
		Members []struct {
			ID     string `xml:"DBInstanceIdentifier"`
			Writer bool   `xml:"IsClusterWriter"`
			Tier   int    `xml:"PromotionTier"`
		} `xml:"DBClusterMembers>DBClusterMember"`
	} `xml:"DescribeDBClustersResult>DBClusters>DBCluster"`
}

func docdbDescribeCluster(t *testing.T, ts *httptest.Server, id string) docdbClustersResponse {
	t.Helper()
	var out docdbClustersResponse
	body := docdbWant(t, ts, http.StatusOK, "DescribeDBClusters", map[string]string{"DBClusterIdentifier": id})
	if err := xml.Unmarshal([]byte(body), &out); err != nil || len(out.Clusters) != 1 {
		t.Fatalf("DescribeDBClusters %s: %v %s", id, err, body)
	}
	return out
}

func TestDocDBStage1ClusterLifecycle(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbCreateCluster(t, ts, "orders", map[string]string{"EnableCloudwatchLogsExports.member.1": "audit"})
	c := docdbDescribeCluster(t, ts, "orders").Clusters[0]
	if c.Engine != "docdb" || c.Version != "5.0.0" || c.Port != 27017 || c.PG != "default.docdb5.0" || c.Status != "available" || len(c.Logs) != 1 {
		t.Fatalf("cluster defaults: %+v", c)
	}

	docdbWantError(t, ts, http.StatusBadRequest, "DBClusterAlreadyExistsFault", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "orders", "Engine": "docdb", "MasterUsername": "stackyard", "MasterUserPassword": "Passw0rd-1"})
	for name, params := range map[string]map[string]string{
		"bad id":         {"DBClusterIdentifier": "1orders", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1"},
		"wrong engine":   {"DBClusterIdentifier": "x", "Engine": "mysql", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1"},
		"short password": {"DBClusterIdentifier": "x", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "short"},
		"no user":        {"DBClusterIdentifier": "x", "Engine": "docdb", "MasterUserPassword": "Passw0rd-1"},
		"bad version":    {"DBClusterIdentifier": "x", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1", "EngineVersion": "9.9.9"},
		"bad retention":  {"DBClusterIdentifier": "x", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1", "BackupRetentionPeriod": "40"},
	} {
		if status, body := docdbCall(t, ts, "CreateDBCluster", params); status != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", name, status, body)
		}
	}
	docdbWantError(t, ts, http.StatusNotFound, "DBSubnetGroupNotFoundFault", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "y", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1", "DBSubnetGroupName": "missing"})
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterParameterGroupNotFound", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "y", "Engine": "docdb", "MasterUsername": "u", "MasterUserPassword": "Passw0rd-1", "DBClusterParameterGroupName": "missing"})
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterNotFoundFault", "DescribeDBClusters", map[string]string{"DBClusterIdentifier": "missing"})

	// Modify validates before applying: a bad window leaves the port alone.
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "ModifyDBCluster", map[string]string{
		"DBClusterIdentifier": "orders", "Port": "28000", "PreferredBackupWindow": "25:00-26:00"})
	if got := docdbDescribeCluster(t, ts, "orders").Clusters[0].Port; got != 27017 {
		t.Fatalf("rejected modify changed the port to %d", got)
	}
	docdbWant(t, ts, http.StatusOK, "ModifyDBCluster", map[string]string{"DBClusterIdentifier": "orders", "DeletionProtection": "true",
		"CloudwatchLogsExportConfiguration.DisableLogTypes.member.1": "audit"})
	if c := docdbDescribeCluster(t, ts, "orders").Clusters[0]; !c.Protect || len(c.Logs) != 0 {
		t.Fatalf("modify: %+v", c)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterCombination", "ModifyDBCluster", map[string]string{"DBClusterIdentifier": "orders", "EngineVersion": "8.0.0"})
	docdbWant(t, ts, http.StatusOK, "ModifyDBCluster", map[string]string{"DBClusterIdentifier": "orders", "EngineVersion": "8.0.0", "AllowMajorVersionUpgrade": "true"})
	if c := docdbDescribeCluster(t, ts, "orders").Clusters[0]; c.Version != "8.0.0" || c.PG != "default.docdb8.0" {
		t.Fatalf("major upgrade should move to the new default parameter group: %+v", c)
	}

	// Deletion protection on its own (no instances yet), then the instance
	// rule, then the final snapshot rule.
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "orders", "SkipFinalSnapshot": "true"})
	docdbCreateInstance(t, ts, "orders-1", "orders", nil)
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "orders", "SkipFinalSnapshot": "true"})
	docdbWant(t, ts, http.StatusOK, "ModifyDBCluster", map[string]string{"DBClusterIdentifier": "orders", "DeletionProtection": "false"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "orders", "SkipFinalSnapshot": "true"})
	docdbWant(t, ts, http.StatusOK, "DeleteDBInstance", map[string]string{"DBInstanceIdentifier": "orders-1"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterCombination", "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "orders"})
	docdbWant(t, ts, http.StatusOK, "DeleteDBCluster", map[string]string{"DBClusterIdentifier": "orders", "FinalDBSnapshotIdentifier": "orders-final"})
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterNotFoundFault", "DescribeDBClusters", map[string]string{"DBClusterIdentifier": "orders"})
	if body := docdbWant(t, ts, http.StatusOK, "DescribeDBClusterSnapshots", map[string]string{"DBClusterSnapshotIdentifier": "orders-final"}); !strings.Contains(body, "<SnapshotType>manual</SnapshotType>") {
		t.Fatalf("final snapshot: %s", body)
	}
}

func TestDocDBStage1StartStopAndFailover(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbCreateCluster(t, ts, "events", nil)
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "StartDBCluster", map[string]string{"DBClusterIdentifier": "events"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "FailoverDBCluster", map[string]string{"DBClusterIdentifier": "events"})

	docdbCreateInstance(t, ts, "events-a", "events", map[string]string{"PromotionTier": "5"})
	docdbCreateInstance(t, ts, "events-b", "events", map[string]string{"PromotionTier": "2"})
	docdbCreateInstance(t, ts, "events-c", "events", map[string]string{"PromotionTier": "0"})
	writer := func() string {
		for _, m := range docdbDescribeCluster(t, ts, "events").Clusters[0].Members {
			if m.Writer {
				return m.ID
			}
		}
		return ""
	}
	if w := writer(); w != "events-a" {
		t.Fatalf("first instance should be the writer, got %q", w)
	}
	// Failover without a target picks the lowest promotion tier.
	docdbWant(t, ts, http.StatusOK, "FailoverDBCluster", map[string]string{"DBClusterIdentifier": "events"})
	if w := writer(); w != "events-c" {
		t.Fatalf("failover should promote tier 0, got %q", w)
	}
	docdbWant(t, ts, http.StatusOK, "FailoverDBCluster", map[string]string{"DBClusterIdentifier": "events", "TargetDBInstanceIdentifier": "events-a"})
	if w := writer(); w != "events-a" {
		t.Fatalf("targeted failover: %q", w)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBInstanceState", "FailoverDBCluster", map[string]string{"DBClusterIdentifier": "events", "TargetDBInstanceIdentifier": "nope"})
	// Deleting the writer hands the role to the best reader.
	docdbWant(t, ts, http.StatusOK, "DeleteDBInstance", map[string]string{"DBInstanceIdentifier": "events-a"})
	if w := writer(); w != "events-c" {
		t.Fatalf("after deleting the writer: %q", w)
	}

	docdbWant(t, ts, http.StatusOK, "StopDBCluster", map[string]string{"DBClusterIdentifier": "events"})
	body := docdbWant(t, ts, http.StatusOK, "DescribeDBInstances", map[string]string{"DBInstanceIdentifier": "events-b"})
	if !strings.Contains(body, "<DBInstanceStatus>stopped</DBInstanceStatus>") {
		t.Fatalf("instances should stop with their cluster: %s", body)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "StopDBCluster", map[string]string{"DBClusterIdentifier": "events"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidDBClusterStateFault", "CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "events-d", "DBInstanceClass": "db.r6g.large", "Engine": "docdb", "DBClusterIdentifier": "events"})
	docdbWant(t, ts, http.StatusOK, "StartDBCluster", map[string]string{"DBClusterIdentifier": "events"})
}

func TestDocDBStage1Instances(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbCreateCluster(t, ts, "catalog", nil)
	docdbWantError(t, ts, http.StatusNotFound, "DBClusterNotFoundFault", "CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "i1", "DBInstanceClass": "db.r6g.large", "Engine": "docdb", "DBClusterIdentifier": "missing"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "i1", "DBInstanceClass": "db.m1.small", "Engine": "docdb", "DBClusterIdentifier": "catalog"})
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "i1", "DBInstanceClass": "db.r6g.large", "Engine": "docdb", "DBClusterIdentifier": "catalog", "PromotionTier": "16"})
	docdbCreateInstance(t, ts, "i1", "catalog", map[string]string{"AvailabilityZone": "us-east-1b"})
	docdbWantError(t, ts, http.StatusBadRequest, "DBInstanceAlreadyExists", "CreateDBInstance", map[string]string{
		"DBInstanceIdentifier": "i1", "DBInstanceClass": "db.r6g.large", "Engine": "docdb", "DBClusterIdentifier": "catalog"})

	var out struct {
		Instances []struct {
			ID      string `xml:"DBInstanceIdentifier"`
			Cluster string `xml:"DBClusterIdentifier"`
			AZ      string `xml:"AvailabilityZone"`
			Port    int    `xml:"Endpoint>Port"`
			Class   string `xml:"DBInstanceClass"`
		} `xml:"DescribeDBInstancesResult>DBInstances>DBInstance"`
	}
	body := docdbWant(t, ts, http.StatusOK, "DescribeDBInstances", map[string]string{"Filters.Filter.1.Name": "db-cluster-id", "Filters.Filter.1.Values.Value.1": "catalog"})
	if err := xml.Unmarshal([]byte(body), &out); err != nil || len(out.Instances) != 1 || out.Instances[0].AZ != "us-east-1b" || out.Instances[0].Port != 27017 {
		t.Fatalf("describe instances: %v %+v", err, out)
	}
	docdbWant(t, ts, http.StatusOK, "ModifyDBInstance", map[string]string{"DBInstanceIdentifier": "i1", "DBInstanceClass": "db.r5.large", "NewDBInstanceIdentifier": "i1-renamed"})
	docdbWantError(t, ts, http.StatusNotFound, "DBInstanceNotFound", "DescribeDBInstances", map[string]string{"DBInstanceIdentifier": "i1"})
	if c := docdbDescribeCluster(t, ts, "catalog").Clusters[0]; len(c.Members) != 1 || c.Members[0].ID != "i1-renamed" {
		t.Fatalf("rename should update cluster membership: %+v", c.Members)
	}
	docdbWantError(t, ts, http.StatusNotFound, "CertificateNotFound", "ModifyDBInstance", map[string]string{"DBInstanceIdentifier": "i1-renamed", "CACertificateIdentifier": "nope"})
	docdbWant(t, ts, http.StatusOK, "RebootDBInstance", map[string]string{"DBInstanceIdentifier": "i1-renamed"})
	docdbWantError(t, ts, http.StatusNotFound, "DBInstanceNotFound", "RebootDBInstance", map[string]string{"DBInstanceIdentifier": "gone"})
}
