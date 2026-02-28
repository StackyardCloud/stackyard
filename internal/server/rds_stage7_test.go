package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage7IdentityNetworkingAndIntegrations(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "rds-stage7-db"
	clusterID := "rds-stage7-cluster"
	clusterARN := "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
	instanceARN := "arn:aws:rds:us-east-1:123456789012:db:" + instanceID

	status, body := rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create DB instance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"Engine":              []string{"aurora-mysql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create DB cluster 200, got %d: %s", status, string(body))
	}

	roleArn := "arn:aws:iam::123456789012:role/rds-stage7-role"
	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"AddRoleToDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"RoleArn":              []string{roleArn},
		"FeatureName":          []string{"S3_INTEGRATION"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected AddRoleToDBInstance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"RemoveRoleFromDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"RoleArn":              []string{roleArn},
		"FeatureName":          []string{"S3_INTEGRATION"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected RemoveRoleFromDBInstance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"AddRoleToDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"RoleArn":             []string{roleArn},
		"FeatureName":         []string{"Lambda"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected AddRoleToDBCluster 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"RemoveRoleFromDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"RoleArn":             []string{roleArn},
		"FeatureName":         []string{"Lambda"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected RemoveRoleFromDBCluster 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":      []string{"StartActivityStream"},
		"ResourceArn": []string{clusterARN},
		"Mode":        []string{"async"},
		"KmsKeyId":    []string{"alias/aws/rds"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected StartActivityStream 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":      []string{"StopActivityStream"},
		"ResourceArn": []string{clusterARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected StopActivityStream 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateDBProxy"},
		"DBProxyName":           []string{"rds-stage7-proxy"},
		"EngineFamily":          []string{"MYSQL"},
		"RoleArn":               []string{roleArn},
		"VpcSubnetIds.member.1": []string{"subnet-12345678"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateDBProxy 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":      []string{"DescribeDBProxies"},
		"DBProxyName": []string{"rds-stage7-proxy"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBProxies 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBProxyName>rds-stage7-proxy</DBProxyName>")) {
		t.Fatalf("expected DB proxy in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":       []string{"ModifyDBProxy"},
		"DBProxyName":  []string{"rds-stage7-proxy"},
		"DebugLogging": []string{"true"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ModifyDBProxy 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateDBProxyEndpoint"},
		"DBProxyName":           []string{"rds-stage7-proxy"},
		"DBProxyEndpointName":   []string{"rds-stage7-endpoint"},
		"VpcSubnetIds.member.1": []string{"subnet-12345678"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateDBProxyEndpoint 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DescribeDBProxyEndpoints"},
		"DBProxyName":         []string{"rds-stage7-proxy"},
		"DBProxyEndpointName": []string{"rds-stage7-endpoint"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBProxyEndpoints 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBProxyEndpointName>rds-stage7-endpoint</DBProxyEndpointName>")) {
		t.Fatalf("expected DB proxy endpoint in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"ModifyDBProxyEndpoint"},
		"DBProxyEndpointName": []string{"rds-stage7-endpoint"},
		"TargetRole":          []string{"READ_ONLY"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ModifyDBProxyEndpoint 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                         []string{"RegisterDBProxyTargets"},
		"DBProxyName":                    []string{"rds-stage7-proxy"},
		"TargetGroupName":                []string{"default"},
		"DBInstanceIdentifiers.member.1": []string{instanceID},
	})
	if status != http.StatusOK {
		t.Fatalf("expected RegisterDBProxyTargets 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":          []string{"DescribeDBProxyTargets"},
		"DBProxyName":     []string{"rds-stage7-proxy"},
		"TargetGroupName": []string{"default"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBProxyTargets 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<RdsResourceId>"+instanceID+"</RdsResourceId>")) {
		t.Fatalf("expected proxy target in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                         []string{"DeregisterDBProxyTargets"},
		"DBProxyName":                    []string{"rds-stage7-proxy"},
		"TargetGroupName":                []string{"default"},
		"DBInstanceIdentifiers.member.1": []string{instanceID},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeregisterDBProxyTargets 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateIntegration"},
		"IntegrationIdentifier": []string{"rds-stage7-int"},
		"IntegrationName":       []string{"stage7"},
		"SourceArn":             []string{instanceARN},
		"TargetArn":             []string{clusterARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateIntegration 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"DescribeIntegrations"},
		"IntegrationIdentifier": []string{"rds-stage7-int"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeIntegrations 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<IntegrationIdentifier>rds-stage7-int</IntegrationIdentifier>")) {
		t.Fatalf("expected integration in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"ModifyIntegration"},
		"IntegrationIdentifier": []string{"rds-stage7-int"},
		"IntegrationName":       []string{"stage7b"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ModifyIntegration 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"DeleteIntegration"},
		"IntegrationIdentifier": []string{"rds-stage7-int"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeleteIntegration 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DeleteDBProxyEndpoint"},
		"DBProxyEndpointName": []string{"rds-stage7-endpoint"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeleteDBProxyEndpoint 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":      []string{"DeleteDBProxy"},
		"DBProxyName": []string{"rds-stage7-proxy"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeleteDBProxy 200, got %d: %s", status, string(body))
	}
}

func TestRDSStage7ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "rds-stage7-impl-db"
	clusterID := "rds-stage7-impl-cluster"
	clusterARN := "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
	instanceARN := "arn:aws:rds:us-east-1:123456789012:db:" + instanceID
	roleArn := "arn:aws:iam::123456789012:role/rds-stage7-impl"

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"Engine":              []string{"aurora-mysql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateDBProxy"},
		"DBProxyName":           []string{"rds-stage7-impl-proxy"},
		"EngineFamily":          []string{"MYSQL"},
		"RoleArn":               []string{roleArn},
		"VpcSubnetIds.member.1": []string{"subnet-12345678"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateDBProxyEndpoint"},
		"DBProxyName":           []string{"rds-stage7-impl-proxy"},
		"DBProxyEndpointName":   []string{"rds-stage7-impl-endpoint"},
		"VpcSubnetIds.member.1": []string{"subnet-12345678"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateIntegration"},
		"IntegrationIdentifier": []string{"rds-stage7-impl-int"},
		"IntegrationName":       []string{"impl"},
		"SourceArn":             []string{instanceARN},
		"TargetArn":             []string{clusterARN},
	})

	cases := []url.Values{
		{"Action": []string{"AddRoleToDBInstance"}, "DBInstanceIdentifier": []string{instanceID}, "RoleArn": []string{roleArn}, "FeatureName": []string{"S3_INTEGRATION"}},
		{"Action": []string{"RemoveRoleFromDBInstance"}, "DBInstanceIdentifier": []string{instanceID}, "RoleArn": []string{roleArn}, "FeatureName": []string{"S3_INTEGRATION"}},
		{"Action": []string{"AddRoleToDBCluster"}, "DBClusterIdentifier": []string{clusterID}, "RoleArn": []string{roleArn}, "FeatureName": []string{"Lambda"}},
		{"Action": []string{"RemoveRoleFromDBCluster"}, "DBClusterIdentifier": []string{clusterID}, "RoleArn": []string{roleArn}, "FeatureName": []string{"Lambda"}},
		{"Action": []string{"StartActivityStream"}, "ResourceArn": []string{clusterARN}, "KmsKeyId": []string{"alias/aws/rds"}},
		{"Action": []string{"StopActivityStream"}, "ResourceArn": []string{clusterARN}},
		{"Action": []string{"DescribeDBProxies"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}},
		{"Action": []string{"ModifyDBProxy"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}, "DebugLogging": []string{"true"}},
		{"Action": []string{"DescribeDBProxyEndpoints"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}},
		{"Action": []string{"ModifyDBProxyEndpoint"}, "DBProxyEndpointName": []string{"rds-stage7-impl-endpoint"}, "TargetRole": []string{"READ_ONLY"}},
		{"Action": []string{"RegisterDBProxyTargets"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}, "DBInstanceIdentifiers.member.1": []string{instanceID}},
		{"Action": []string{"DescribeDBProxyTargets"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}},
		{"Action": []string{"DeregisterDBProxyTargets"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}, "DBInstanceIdentifiers.member.1": []string{instanceID}},
		{"Action": []string{"DescribeIntegrations"}, "IntegrationIdentifier": []string{"rds-stage7-impl-int"}},
		{"Action": []string{"ModifyIntegration"}, "IntegrationIdentifier": []string{"rds-stage7-impl-int"}, "IntegrationName": []string{"impl2"}},
		{"Action": []string{"DeleteIntegration"}, "IntegrationIdentifier": []string{"rds-stage7-impl-int"}},
		{"Action": []string{"DeleteDBProxyEndpoint"}, "DBProxyEndpointName": []string{"rds-stage7-impl-endpoint"}},
		{"Action": []string{"DeleteDBProxy"}, "DBProxyName": []string{"rds-stage7-impl-proxy"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
