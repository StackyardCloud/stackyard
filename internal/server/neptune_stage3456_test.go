package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func requireNeptuneOK(t *testing.T, action string, status int, body string) {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("expected %s 200, got %d: %s", action, status, body)
	}
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("expected %s implemented, got: %s", action, body)
	}
}

func TestNeptuneStage3Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneFormRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"neptune-stage3-instance"},
		"Engine":               []string{"neptune"},
	})
	requireNeptuneOK(t, "CreateDBInstance", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":               []string{"ModifyDBInstance"},
		"DBInstanceIdentifier": []string{"neptune-stage3-instance"},
		"DBInstanceClass":      []string{"db.r5.xlarge"},
	})
	requireNeptuneOK(t, "ModifyDBInstance", status, body)
	if !strings.Contains(body, "<DBInstanceIdentifier>neptune-stage3-instance</DBInstanceIdentifier>") {
		t.Fatalf("expected modified DB instance identifier in response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBParameterGroup"},
		"DBParameterGroupName": []string{"neptune-stage3-pg"},
		"DBParameterGroupFamily": []string{
			"neptune1",
		},
		"Description": []string{"stage3 group"},
	})
	requireNeptuneOK(t, "CreateDBParameterGroup", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                             []string{"ModifyDBParameterGroup"},
		"DBParameterGroupName":               []string{"neptune-stage3-pg"},
		"Parameters.member.1.ParameterName":  []string{"neptune_query_timeout"},
		"Parameters.member.1.ParameterValue": []string{"180000"},
		"Parameters.member.1.ApplyMethod":    []string{"immediate"},
	})
	requireNeptuneOK(t, "ModifyDBParameterGroup", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action": []string{"DescribeDBParameters"},
		"DBParameterGroupName": []string{
			"neptune-stage3-pg",
		},
	})
	requireNeptuneOK(t, "DescribeDBParameters", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action": []string{"CreateDBSubnetGroup"},
		"DBSubnetGroupName": []string{
			"neptune-stage3-subnets",
		},
		"SubnetIds.member.1": []string{"subnet-11111111"},
		"SubnetIds.member.2": []string{"subnet-22222222"},
	})
	requireNeptuneOK(t, "CreateDBSubnetGroup", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action": []string{"DeleteDBInstance"},
		"DBInstanceIdentifier": []string{
			"neptune-stage3-instance",
		},
		"SkipFinalSnapshot": []string{"true"},
	})
	requireNeptuneOK(t, "DeleteDBInstance", status, body)
}

func TestNeptuneStage4SnapshotAndRestore(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneFormRequest(t, ts, url.Values{
		"Action":                      []string{"CreateDBClusterSnapshot"},
		"DBClusterSnapshotIdentifier": []string{"neptune-stage4-snapshot"},
		"DBClusterIdentifier":         []string{"neptune-stage4-cluster"},
	})
	requireNeptuneOK(t, "CreateDBClusterSnapshot", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                      []string{"DescribeDBClusterSnapshots"},
		"DBClusterSnapshotIdentifier": []string{"neptune-stage4-snapshot"},
	})
	requireNeptuneOK(t, "DescribeDBClusterSnapshots", status, body)
	if !strings.Contains(body, "<DBClusterSnapshotIdentifier>neptune-stage4-snapshot</DBClusterSnapshotIdentifier>") {
		t.Fatalf("expected snapshot identifier in describe response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                            []string{"ModifyDBClusterSnapshotAttribute"},
		"DBClusterSnapshotIdentifier":       []string{"neptune-stage4-snapshot"},
		"AttributeName":                     []string{"restore"},
		"ValuesToAdd.member.1":              []string{"all"},
		"ValuesToRemove.member.1":           []string{"none"},
		"SharedSnapshotQuotas.member.1.Tag": []string{"unused"},
	})
	requireNeptuneOK(t, "ModifyDBClusterSnapshotAttribute", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                      []string{"RestoreDBClusterFromSnapshot"},
		"DBClusterIdentifier":         []string{"neptune-stage4-restored"},
		"DBClusterSnapshotIdentifier": []string{"neptune-stage4-snapshot"},
		"Engine":                      []string{"neptune"},
	})
	requireNeptuneOK(t, "RestoreDBClusterFromSnapshot", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                    []string{"RestoreDBClusterToPointInTime"},
		"TargetDBClusterIdentifier": []string{"neptune-stage4-pitr"},
		"SourceDBClusterIdentifier": []string{"neptune-stage4-restored"},
	})
	requireNeptuneOK(t, "RestoreDBClusterToPointInTime", status, body)
}

func TestNeptuneStage5Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage5-cluster"},
		"Engine":              []string{"neptune"},
	})
	requireNeptuneOK(t, "CreateDBCluster", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                      []string{"CreateDBClusterEndpoint"},
		"DBClusterEndpointIdentifier": []string{"neptune-stage5-endpoint"},
		"DBClusterIdentifier":         []string{"neptune-stage5-cluster"},
		"EndpointType":                []string{"READER"},
	})
	requireNeptuneOK(t, "CreateDBClusterEndpoint", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                      []string{"ModifyDBClusterEndpoint"},
		"DBClusterEndpointIdentifier": []string{"neptune-stage5-endpoint"},
		"EndpointType":                []string{"ANY"},
	})
	requireNeptuneOK(t, "ModifyDBClusterEndpoint", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                      []string{"DeleteDBClusterEndpoint"},
		"DBClusterEndpointIdentifier": []string{"neptune-stage5-endpoint"},
	})
	requireNeptuneOK(t, "DeleteDBClusterEndpoint", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                    []string{"CreateGlobalCluster"},
		"GlobalClusterIdentifier":   []string{"neptune-stage5-global"},
		"SourceDBClusterIdentifier": []string{"neptune-stage5-cluster"},
	})
	requireNeptuneOK(t, "CreateGlobalCluster", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                   []string{"CreateEventSubscription"},
		"SubscriptionName":         []string{"neptune-stage5-sub"},
		"SnsTopicArn":              []string{"arn:aws:sns:us-east-1:123456789012:neptune-stage5-topic"},
		"SourceType":               []string{"db-cluster"},
		"Enabled":                  []string{"true"},
		"EventCategories.member.1": []string{"availability"},
	})
	requireNeptuneOK(t, "CreateEventSubscription", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":           []string{"DescribeEventSubscriptions"},
		"SubscriptionName": []string{"neptune-stage5-sub"},
	})
	requireNeptuneOK(t, "DescribeEventSubscriptions", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":           []string{"DeleteEventSubscription"},
		"SubscriptionName": []string{"neptune-stage5-sub"},
	})
	requireNeptuneOK(t, "DeleteEventSubscription", status, body)
}

func TestNeptuneStage6Lifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage6-cluster"},
		"Engine":              []string{"neptune"},
	})
	requireNeptuneOK(t, "CreateDBCluster", status, body)
	clusterARN := xmlTagValue(body, "DBClusterArn")
	if clusterARN == "" {
		t.Fatalf("expected DBClusterArn in create response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":           []string{"AddTagsToResource"},
		"ResourceName":     []string{clusterARN},
		"Tags.Tag.1.Key":   []string{"env"},
		"Tags.Tag.1.Value": []string{"stage6"},
		"Tags.Tag.2.Key":   []string{"team"},
		"Tags.Tag.2.Value": []string{"platform"},
	})
	requireNeptuneOK(t, "AddTagsToResource", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":       []string{"ListTagsForResource"},
		"ResourceName": []string{clusterARN},
	})
	requireNeptuneOK(t, "ListTagsForResource", status, body)
	if !strings.Contains(body, "<Key>env</Key>") {
		t.Fatalf("expected env tag after AddTagsToResource: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":       []string{"RemoveTagsFromResource"},
		"ResourceName": []string{clusterARN},
		"TagKeys.member.1": []string{
			"team",
		},
	})
	requireNeptuneOK(t, "RemoveTagsFromResource", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":             []string{"ApplyPendingMaintenanceAction"},
		"ResourceIdentifier": []string{"stackyard-neptune-resource"},
		"ApplyAction":        []string{"system-update"},
		"OptInType":          []string{"immediate"},
	})
	requireNeptuneOK(t, "ApplyPendingMaintenanceAction", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"AddRoleToDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage6-cluster"},
		"RoleArn":             []string{"arn:aws:iam::123456789012:role/neptune-stage6-role"},
	})
	requireNeptuneOK(t, "AddRoleToDBCluster", status, body)

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"RemoveRoleFromDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage6-cluster"},
		"RoleArn":             []string{"arn:aws:iam::123456789012:role/neptune-stage6-role"},
	})
	requireNeptuneOK(t, "RemoveRoleFromDBCluster", status, body)
}

func TestNeptuneStage3456ActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	cases := []url.Values{
		{"Action": []string{"CreateDBInstance"}},
		{"Action": []string{"ModifyDBInstance"}},
		{"Action": []string{"RebootDBInstance"}},
		{"Action": []string{"DeleteDBInstance"}},
		{"Action": []string{"CreateDBParameterGroup"}},
		{"Action": []string{"DescribeDBParameterGroups"}},
		{"Action": []string{"DescribeDBParameters"}},
		{"Action": []string{"ModifyDBParameterGroup"}},
		{"Action": []string{"ResetDBParameterGroup"}},
		{"Action": []string{"DeleteDBParameterGroup"}},
		{"Action": []string{"CreateDBClusterParameterGroup"}},
		{"Action": []string{"DescribeDBClusterParameterGroups"}},
		{"Action": []string{"DescribeDBClusterParameters"}},
		{"Action": []string{"ModifyDBClusterParameterGroup"}},
		{"Action": []string{"ResetDBClusterParameterGroup"}},
		{"Action": []string{"DeleteDBClusterParameterGroup"}},
		{"Action": []string{"CopyDBParameterGroup"}},
		{"Action": []string{"CopyDBClusterParameterGroup"}},
		{"Action": []string{"CreateDBSubnetGroup"}},
		{"Action": []string{"DescribeDBSubnetGroups"}},
		{"Action": []string{"ModifyDBSubnetGroup"}},
		{"Action": []string{"DeleteDBSubnetGroup"}},
		{"Action": []string{"CreateDBClusterSnapshot"}},
		{"Action": []string{"DeleteDBClusterSnapshot"}},
		{"Action": []string{"CopyDBClusterSnapshot"}},
		{"Action": []string{"DescribeDBClusterSnapshots"}},
		{"Action": []string{"DescribeDBClusterSnapshotAttributes"}},
		{"Action": []string{"ModifyDBClusterSnapshotAttribute"}},
		{"Action": []string{"RestoreDBClusterFromSnapshot"}},
		{"Action": []string{"RestoreDBClusterToPointInTime"}},
		{"Action": []string{"CreateDBClusterEndpoint"}},
		{"Action": []string{"ModifyDBClusterEndpoint"}},
		{"Action": []string{"DeleteDBClusterEndpoint"}},
		{"Action": []string{"CreateGlobalCluster"}},
		{"Action": []string{"ModifyGlobalCluster"}},
		{"Action": []string{"DeleteGlobalCluster"}},
		{"Action": []string{"FailoverGlobalCluster"}},
		{"Action": []string{"SwitchoverGlobalCluster"}},
		{"Action": []string{"CreateEventSubscription"}},
		{"Action": []string{"ModifyEventSubscription"}},
		{"Action": []string{"DeleteEventSubscription"}},
		{"Action": []string{"DescribeEventSubscriptions"}},
		{"Action": []string{"AddSourceIdentifierToSubscription"}},
		{"Action": []string{"RemoveSourceIdentifierFromSubscription"}},
		{"Action": []string{"PromoteReadReplicaDBCluster"}},
		{"Action": []string{"RemoveFromGlobalCluster"}},
		{"Action": []string{"AddTagsToResource"}},
		{"Action": []string{"RemoveTagsFromResource"}},
		{"Action": []string{"ApplyPendingMaintenanceAction"}},
		{"Action": []string{"AddRoleToDBCluster"}},
		{"Action": []string{"RemoveRoleFromDBCluster"}},
	}

	for _, params := range cases {
		status, body := neptuneFormRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), body)
		}
	}
}
