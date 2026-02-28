package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func neptuneFormRequest(t *testing.T, ts *httptest.Server, values url.Values) (int, string) {
	t.Helper()
	if values.Get("Version") == "" {
		values.Set("Version", "2014-10-31")
	}
	resp := neptuneRequest(t, ts, []byte(values.Encode()))
	body := string(mustBody(t, resp))
	return resp.StatusCode, body
}

func xmlTagValue(payload, tag string) string {
	start := "<" + tag + ">"
	end := "</" + tag + ">"
	i := strings.Index(payload, start)
	if i == -1 {
		return ""
	}
	i += len(start)
	j := strings.Index(payload[i:], end)
	if j == -1 {
		return ""
	}
	return payload[i : i+j]
}

func TestNeptuneStage1ReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, createBody := neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage1-cluster"},
		"Engine":              []string{"neptune"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected create cluster 200, got %d: %s", status, createBody)
	}
	clusterARN := xmlTagValue(createBody, "DBClusterArn")
	if clusterARN == "" {
		t.Fatalf("expected DBClusterArn in create response: %s", createBody)
	}

	status, body := neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"DescribeDBClusters"},
		"DBClusterIdentifier": []string{"neptune-stage1-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBClusters 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<DBClusterIdentifier>neptune-stage1-cluster</DBClusterIdentifier>") {
		t.Fatalf("expected described cluster in response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{"Action": []string{"DescribeDBEngineVersions"}})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBEngineVersions 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<DescribeDBEngineVersionsResult>") {
		t.Fatalf("expected DescribeDBEngineVersions result wrapper: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":       []string{"ListTagsForResource"},
		"ResourceName": []string{clusterARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ListTagsForResource 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<ListTagsForResourceResult>") {
		t.Fatalf("expected ListTagsForResource result wrapper: %s", body)
	}
}

func TestNeptuneStage2ClusterLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	status, body := neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage2-cluster"},
		"Engine":              []string{"neptune"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateDBCluster 200, got %d: %s", status, body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":                []string{"ModifyDBCluster"},
		"DBClusterIdentifier":   []string{"neptune-stage2-cluster"},
		"BackupRetentionPeriod": []string{"7"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected ModifyDBCluster 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<BackupRetentionPeriod>7</BackupRetentionPeriod>") {
		t.Fatalf("expected updated backup retention period in response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"StopDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage2-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected StopDBCluster 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<Status>stopped</Status>") {
		t.Fatalf("expected stopped status in response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"StartDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage2-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected StartDBCluster 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<Status>available</Status>") {
		t.Fatalf("expected available status in response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"FailoverDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage2-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected FailoverDBCluster 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<DBClusterIdentifier>neptune-stage2-cluster</DBClusterIdentifier>") {
		t.Fatalf("expected cluster identifier in failover response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"DeleteDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage2-cluster"},
		"SkipFinalSnapshot":   []string{"true"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeleteDBCluster 200, got %d: %s", status, body)
	}
	if !strings.Contains(body, "<Status>deleting</Status>") {
		t.Fatalf("expected deleting status in response: %s", body)
	}

	status, body = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"DescribeDBClusters"},
		"DBClusterIdentifier": []string{"neptune-stage2-cluster"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeDBClusters after delete 200, got %d: %s", status, body)
	}
	if strings.Contains(body, "<DBClusterIdentifier>neptune-stage2-cluster</DBClusterIdentifier>") {
		t.Fatalf("expected deleted cluster to be absent from describe response: %s", body)
	}
}

func TestNeptuneStage12ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, _ = neptuneFormRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"neptune-stage12-impl"},
		"Engine":              []string{"neptune"},
	})

	cases := []url.Values{
		{"Action": []string{"DescribeDBClusters"}},
		{"Action": []string{"DescribeDBEngineVersions"}},
		{"Action": []string{"DescribeDBClusterParameters"}},
		{"Action": []string{"DescribeDBClusterSnapshots"}},
		{"Action": []string{"DescribeEvents"}},
		{"Action": []string{"ModifyDBCluster"}, "DBClusterIdentifier": []string{"neptune-stage12-impl"}, "BackupRetentionPeriod": []string{"3"}},
		{"Action": []string{"StopDBCluster"}, "DBClusterIdentifier": []string{"neptune-stage12-impl"}},
		{"Action": []string{"StartDBCluster"}, "DBClusterIdentifier": []string{"neptune-stage12-impl"}},
		{"Action": []string{"FailoverDBCluster"}, "DBClusterIdentifier": []string{"neptune-stage12-impl"}},
		{"Action": []string{"DeleteDBCluster"}, "DBClusterIdentifier": []string{"neptune-stage12-impl"}, "SkipFinalSnapshot": []string{"true"}},
	}

	for _, params := range cases {
		status, body := neptuneFormRequest(t, ts, params)
		if status == http.StatusNotImplemented {
			t.Fatalf("action %s returned NotImplemented: %s", params.Get("Action"), body)
		}
	}
}
