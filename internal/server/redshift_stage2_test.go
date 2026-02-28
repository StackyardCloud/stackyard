package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func redshiftRequest(t *testing.T, ts *httptest.Server, params url.Values) (int, []byte) {
	t.Helper()
	if params.Get("Version") == "" {
		params.Set("Version", "2012-12-01")
	}
	body := []byte(params.Encode())
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/redshift", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "redshift")
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp.StatusCode, respBody
}

func TestRedshiftStage2ClusterLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createParams := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
		"DBName":             []string{"dev"},
	}
	status, body := redshiftRequest(t, ts, createParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ClusterIdentifier>demo</ClusterIdentifier>")) {
		t.Fatalf("missing cluster identifier in response: %s", string(body))
	}

	describeParams := url.Values{"Action": []string{"DescribeClusters"}}
	status, body = redshiftRequest(t, ts, describeParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Clusters>")) {
		t.Fatalf("missing clusters element: %s", string(body))
	}
	if !bytes.Contains(body, []byte("<ClusterIdentifier>demo</ClusterIdentifier>")) {
		t.Fatalf("missing cluster in describe: %s", string(body))
	}

	modifyParams := url.Values{
		"Action":            []string{"ModifyCluster"},
		"ClusterIdentifier": []string{"demo"},
		"NodeType":          []string{"ra3.large"},
	}
	status, body = redshiftRequest(t, ts, modifyParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 modify, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<NodeType>ra3.large</NodeType>")) {
		t.Fatalf("missing updated node type: %s", string(body))
	}

	deleteParams := url.Values{
		"Action":            []string{"DeleteCluster"},
		"ClusterIdentifier": []string{"demo"},
	}
	status, body = redshiftRequest(t, ts, deleteParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete, got %d: %s", status, string(body))
	}

	describeParams = url.Values{
		"Action":            []string{"DescribeClusters"},
		"ClusterIdentifier": []string{"demo"},
	}
	status, body = redshiftRequest(t, ts, describeParams)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 describe missing, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterNotFound</Code>")) {
		t.Fatalf("expected ClusterNotFound error: %s", string(body))
	}
}

func TestRedshiftStage2SnapshotLifecycle(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createParams := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, createParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}

	createSnap := url.Values{
		"Action":             []string{"CreateClusterSnapshot"},
		"ClusterIdentifier":  []string{"demo"},
		"SnapshotIdentifier": []string{"snap-1"},
	}
	status, body = redshiftRequest(t, ts, createSnap)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create snapshot, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<SnapshotIdentifier>snap-1</SnapshotIdentifier>")) {
		t.Fatalf("missing snapshot identifier: %s", string(body))
	}

	describeSnaps := url.Values{
		"Action":            []string{"DescribeClusterSnapshots"},
		"ClusterIdentifier": []string{"demo"},
	}
	status, body = redshiftRequest(t, ts, describeSnaps)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe snapshots, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<SnapshotIdentifier>snap-1</SnapshotIdentifier>")) {
		t.Fatalf("missing snapshot in describe: %s", string(body))
	}

	restore := url.Values{
		"Action":             []string{"RestoreFromClusterSnapshot"},
		"SnapshotIdentifier": []string{"snap-1"},
		"ClusterIdentifier":  []string{"restored"},
	}
	status, body = redshiftRequest(t, ts, restore)
	if status != http.StatusOK {
		t.Fatalf("expected 200 restore, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ClusterIdentifier>restored</ClusterIdentifier>")) {
		t.Fatalf("missing restored cluster: %s", string(body))
	}

	deleteSnap := url.Values{
		"Action":             []string{"DeleteClusterSnapshot"},
		"SnapshotIdentifier": []string{"snap-1"},
	}
	status, body = redshiftRequest(t, ts, deleteSnap)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete snapshot, got %d: %s", status, string(body))
	}

	describeSnaps = url.Values{
		"Action":             []string{"DescribeClusterSnapshots"},
		"SnapshotIdentifier": []string{"snap-1"},
	}
	status, body = redshiftRequest(t, ts, describeSnaps)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 describe missing snapshot, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>ClusterSnapshotNotFound</Code>")) {
		t.Fatalf("expected ClusterSnapshotNotFound error: %s", string(body))
	}
}

func TestRedshiftStage2Validation(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	badMulti := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"bad-multi"},
		"NodeType":           []string{"dc2.large"},
		"ClusterType":        []string{"multi-node"},
		"NumberOfNodes":      []string{"1"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, badMulti)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid multi-node, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}

	badSingle := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"bad-single"},
		"NodeType":           []string{"dc2.large"},
		"ClusterType":        []string{"single-node"},
		"NumberOfNodes":      []string{"2"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body = redshiftRequest(t, ts, badSingle)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid single-node, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}

	createParams := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"mod-test"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body = redshiftRequest(t, ts, createParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}

	badModify := url.Values{
		"Action":            []string{"ModifyCluster"},
		"ClusterIdentifier": []string{"mod-test"},
		"ClusterType":       []string{"multi-node"},
		"NumberOfNodes":     []string{"1"},
	}
	status, body = redshiftRequest(t, ts, badModify)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid modify, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Code>InvalidParameterValue</Code>")) {
		t.Fatalf("expected InvalidParameterValue: %s", string(body))
	}
}
