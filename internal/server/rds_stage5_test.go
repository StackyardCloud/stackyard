package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage5ReplicationBlueGreenAndTenantDB(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage5-source"},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create source DB instance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"CreateDBInstanceReadReplica"},
		"DBInstanceIdentifier":       []string{"rds-stage5-replica"},
		"SourceDBInstanceIdentifier": []string{"rds-stage5-source"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create read replica 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"PromoteReadReplica"},
		"DBInstanceIdentifier": []string{"rds-stage5-replica"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected PromoteReadReplica 200, got %d: %s", status, string(body))
	}
	status, body = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"CreateDBInstanceReadReplica"},
		"DBInstanceIdentifier":       []string{"rds-stage5-replica2"},
		"SourceDBInstanceIdentifier": []string{"rds-stage5-source"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create second read replica 200, got %d: %s", status, string(body))
	}
	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"SwitchoverReadReplica"},
		"DBInstanceIdentifier": []string{"rds-stage5-replica2"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected SwitchoverReadReplica 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                  []string{"CreateBlueGreenDeployment"},
		"BlueGreenDeploymentName": []string{"rds-stage5-bgd"},
		"Source":                  []string{"arn:aws:rds:us-east-1:123456789012:db:rds-stage5-source"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create blue/green deployment 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<BlueGreenDeploymentIdentifier>")) {
		t.Fatalf("expected blue/green identifier in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action": []string{"DescribeBlueGreenDeployments"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe blue/green deployments 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                        []string{"SwitchoverBlueGreenDeployment"},
		"BlueGreenDeploymentIdentifier": []string{"bgd-rds-stage5-bgd"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected switchover blue/green deployment 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                        []string{"DeleteBlueGreenDeployment"},
		"BlueGreenDeploymentIdentifier": []string{"bgd-rds-stage5-bgd"},
		"DeleteTarget":                  []string{"true"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected delete blue/green deployment 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage5-cluster"},
		"Engine":              []string{"aurora-postgresql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create cluster for tenant DB 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateTenantDatabase"},
		"DBClusterIdentifier": []string{"rds-stage5-cluster"},
		"TenantDBName":        []string{"tenant_a"},
		"MasterUsername":      []string{"tenant_admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create tenant database 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DescribeTenantDatabases"},
		"DBClusterIdentifier": []string{"rds-stage5-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe tenant databases 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<TenantDatabaseName>tenant_a</TenantDatabaseName>")) {
		t.Fatalf("expected tenant database in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"ModifyTenantDatabase"},
		"DBClusterIdentifier": []string{"rds-stage5-cluster"},
		"TenantDBName":        []string{"tenant_a"},
		"NewTenantDBName":     []string{"tenant_b"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected modify tenant database 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DeleteTenantDatabase"},
		"DBClusterIdentifier": []string{"rds-stage5-cluster"},
		"TenantDBName":        []string{"tenant_b"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected delete tenant database 200, got %d: %s", status, string(body))
	}
}

func TestRDSStage5ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{"rds-stage5-impl-source"},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage5-impl-cluster"},
		"Engine":              []string{"aurora-mysql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                     []string{"CreateDBInstanceReadReplica"},
		"DBInstanceIdentifier":       []string{"rds-stage5-impl-replica"},
		"SourceDBInstanceIdentifier": []string{"rds-stage5-impl-source"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                  []string{"CreateBlueGreenDeployment"},
		"BlueGreenDeploymentName": []string{"rds-stage5-impl-bgd"},
		"Source":                  []string{"arn:aws:rds:us-east-1:123456789012:db:rds-stage5-impl-source"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateTenantDatabase"},
		"DBClusterIdentifier": []string{"rds-stage5-impl-cluster"},
		"TenantDBName":        []string{"tenant_impl"},
		"MasterUsername":      []string{"tenant_admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})

	cases := []url.Values{
		{"Action": []string{"PromoteReadReplica"}, "DBInstanceIdentifier": []string{"rds-stage5-impl-replica"}},
		{"Action": []string{"SwitchoverReadReplica"}, "DBInstanceIdentifier": []string{"rds-stage5-impl-replica"}},
		{"Action": []string{"DescribeBlueGreenDeployments"}},
		{"Action": []string{"SwitchoverBlueGreenDeployment"}, "BlueGreenDeploymentIdentifier": []string{"bgd-rds-stage5-impl-bgd"}},
		{"Action": []string{"DeleteBlueGreenDeployment"}, "BlueGreenDeploymentIdentifier": []string{"bgd-rds-stage5-impl-bgd"}},
		{"Action": []string{"DescribeTenantDatabases"}, "DBClusterIdentifier": []string{"rds-stage5-impl-cluster"}},
		{"Action": []string{"ModifyTenantDatabase"}, "DBClusterIdentifier": []string{"rds-stage5-impl-cluster"}, "TenantDBName": []string{"tenant_impl"}, "NewTenantDBName": []string{"tenant_impl2"}},
		{"Action": []string{"DeleteTenantDatabase"}, "DBClusterIdentifier": []string{"rds-stage5-impl-cluster"}, "TenantDBName": []string{"tenant_impl2"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
