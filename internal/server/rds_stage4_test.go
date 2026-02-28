package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRDSStage4ClusterTopology(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage4-cluster"},
		"Engine":              []string{"aurora-mysql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
		"DatabaseName":        []string{"app"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create DB cluster 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBClusterIdentifier>rds-stage4-cluster</DBClusterIdentifier>")) {
		t.Fatalf("missing cluster identifier: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                      []string{"CreateDBClusterEndpoint"},
		"DBClusterIdentifier":         []string{"rds-stage4-cluster"},
		"DBClusterEndpointIdentifier": []string{"rds-stage4-endpoint"},
		"EndpointType":                []string{"READER"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create cluster endpoint 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DescribeDBClusterEndpoints"},
		"DBClusterIdentifier": []string{"rds-stage4-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected describe cluster endpoints 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<DBClusterEndpointIdentifier>rds-stage4-endpoint</DBClusterEndpointIdentifier>")) {
		t.Fatalf("expected cluster endpoint in response: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                      []string{"ModifyDBClusterEndpoint"},
		"DBClusterEndpointIdentifier": []string{"rds-stage4-endpoint"},
		"EndpointType":                []string{"ANY"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected modify cluster endpoint 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<EndpointType>ANY</EndpointType>")) {
		t.Fatalf("expected modified endpoint type: %s", string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                  []string{"CreateGlobalCluster"},
		"GlobalClusterIdentifier": []string{"rds-stage4-global"},
		"SourceDBClusterArn":      []string{"arn:aws:rds:us-east-1:123456789012:cluster:rds-stage4-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create global cluster 200, got %d: %s", status, string(body))
	}

	for _, action := range []string{"FailoverGlobalCluster", "SwitchoverGlobalCluster"} {
		status, body = rdsRequest(t, ts, url.Values{
			"Action":                    []string{action},
			"GlobalClusterIdentifier":   []string{"rds-stage4-global"},
			"TargetDBClusterIdentifier": []string{"arn:aws:rds:us-east-1:123456789012:cluster:rds-stage4-cluster"},
		})
		if status != http.StatusOK {
			t.Fatalf("expected %s 200, got %d: %s", action, status, string(body))
		}
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"StopDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage4-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected stop DB cluster 200, got %d: %s", status, string(body))
	}
	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"StartDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage4-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected start DB cluster 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"FailoverDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage4-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected failover DB cluster 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                      []string{"DeleteDBClusterEndpoint"},
		"DBClusterEndpointIdentifier": []string{"rds-stage4-endpoint"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected delete cluster endpoint 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                  []string{"DeleteGlobalCluster"},
		"GlobalClusterIdentifier": []string{"rds-stage4-global"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected delete global cluster 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"DeleteDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage4-cluster"},
		"SkipFinalSnapshot":   []string{"true"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected delete DB cluster 200, got %d: %s", status, string(body))
	}
}

func TestRDSStage4ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"rds-stage4-impl-cluster"},
		"Engine":              []string{"aurora-mysql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                      []string{"CreateDBClusterEndpoint"},
		"DBClusterIdentifier":         []string{"rds-stage4-impl-cluster"},
		"DBClusterEndpointIdentifier": []string{"rds-stage4-impl-endpoint"},
	})
	_, _ = rdsRequest(t, ts, url.Values{
		"Action":                  []string{"CreateGlobalCluster"},
		"GlobalClusterIdentifier": []string{"rds-stage4-impl-global"},
	})

	cases := []url.Values{
		{"Action": []string{"DescribeDBClusters"}},
		{"Action": []string{"ModifyDBCluster"}, "DBClusterIdentifier": []string{"rds-stage4-impl-cluster"}, "BackupRetentionPeriod": []string{"3"}},
		{"Action": []string{"StopDBCluster"}, "DBClusterIdentifier": []string{"rds-stage4-impl-cluster"}},
		{"Action": []string{"StartDBCluster"}, "DBClusterIdentifier": []string{"rds-stage4-impl-cluster"}},
		{"Action": []string{"RebootDBCluster"}, "DBClusterIdentifier": []string{"rds-stage4-impl-cluster"}},
		{"Action": []string{"FailoverDBCluster"}, "DBClusterIdentifier": []string{"rds-stage4-impl-cluster"}},
		{"Action": []string{"DescribeDBClusterEndpoints"}, "DBClusterIdentifier": []string{"rds-stage4-impl-cluster"}},
		{"Action": []string{"ModifyDBClusterEndpoint"}, "DBClusterEndpointIdentifier": []string{"rds-stage4-impl-endpoint"}, "EndpointType": []string{"ANY"}},
		{"Action": []string{"DescribeGlobalClusters"}},
		{"Action": []string{"ModifyGlobalCluster"}, "GlobalClusterIdentifier": []string{"rds-stage4-impl-global"}, "DeletionProtection": []string{"true"}},
		{"Action": []string{"FailoverGlobalCluster"}, "GlobalClusterIdentifier": []string{"rds-stage4-impl-global"}},
		{"Action": []string{"SwitchoverGlobalCluster"}, "GlobalClusterIdentifier": []string{"rds-stage4-impl-global"}},
	}

	for _, params := range cases {
		status, body := rdsRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), string(body))
		}
	}
}
