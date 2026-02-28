package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRedshiftStage4ParameterGroups(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":               []string{"CreateClusterParameterGroup"},
		"ParameterGroupName":   []string{"pg1"},
		"ParameterGroupFamily": []string{"redshift-1.0"},
		"Description":          []string{"test"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ParameterGroupName>pg1</ParameterGroupName>")) {
		t.Fatalf("missing parameter group in response: %s", string(body))
	}

	modify := url.Values{
		"Action":                             []string{"ModifyClusterParameterGroup"},
		"ParameterGroupName":                 []string{"pg1"},
		"Parameters.member.1.ParameterName":  []string{"max_concurrency_scaling_clusters"},
		"Parameters.member.1.ParameterValue": []string{"2"},
		"Parameters.member.2.ParameterName":  []string{"enable_user_activity_logging"},
		"Parameters.member.2.ParameterValue": []string{"true"},
	}
	status, body = redshiftRequest(t, ts, modify)
	if status != http.StatusOK {
		t.Fatalf("expected 200 modify, got %d: %s", status, string(body))
	}

	describe := url.Values{
		"Action":             []string{"DescribeClusterParameters"},
		"ParameterGroupName": []string{"pg1"},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe params, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ParameterName>enable_user_activity_logging</ParameterName>")) {
		t.Fatalf("expected parameter in describe: %s", string(body))
	}

	reset := url.Values{
		"Action":                            []string{"ResetClusterParameterGroup"},
		"ParameterGroupName":                []string{"pg1"},
		"Parameters.member.1.ParameterName": []string{"enable_user_activity_logging"},
	}
	status, body = redshiftRequest(t, ts, reset)
	if status != http.StatusOK {
		t.Fatalf("expected 200 reset, got %d: %s", status, string(body))
	}

	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe params after reset, got %d: %s", status, string(body))
	}
	if bytes.Contains(body, []byte("<ParameterName>enable_user_activity_logging</ParameterName>")) {
		t.Fatalf("expected parameter removed after reset: %s", string(body))
	}

	del := url.Values{
		"Action":             []string{"DeleteClusterParameterGroup"},
		"ParameterGroupName": []string{"pg1"},
	}
	status, body = redshiftRequest(t, ts, del)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete, got %d: %s", status, string(body))
	}
}

func TestRedshiftStage4MaintenanceWindow(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"maint"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}

	bad := url.Values{
		"Action":                     []string{"ModifyClusterMaintenance"},
		"ClusterIdentifier":          []string{"maint"},
		"PreferredMaintenanceWindow": []string{"bad"},
	}
	status, body = redshiftRequest(t, ts, bad)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 bad window, got %d: %s", status, string(body))
	}

	good := url.Values{
		"Action":                     []string{"ModifyClusterMaintenance"},
		"ClusterIdentifier":          []string{"maint"},
		"PreferredMaintenanceWindow": []string{"sun:05:00-sun:06:00"},
	}
	status, body = redshiftRequest(t, ts, good)
	if status != http.StatusOK {
		t.Fatalf("expected 200 maintenance update, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<PreferredMaintenanceWindow>sun:05:00-sun:06:00</PreferredMaintenanceWindow>")) {
		t.Fatalf("missing maintenance window: %s", string(body))
	}
}

func TestRedshiftStage4ResizeFlow(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"resize-demo"},
		"NodeType":           []string{"dc2.large"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret123"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create, got %d: %s", status, string(body))
	}

	resize := url.Values{
		"Action":            []string{"ResizeCluster"},
		"ClusterIdentifier": []string{"resize-demo"},
		"NodeType":          []string{"ra3.large"},
		"NumberOfNodes":     []string{"4"},
	}
	status, body = redshiftRequest(t, ts, resize)
	if status != http.StatusOK {
		t.Fatalf("expected 200 resize, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ResizeInfo>")) {
		t.Fatalf("missing resize info: %s", string(body))
	}

	describe := url.Values{
		"Action":            []string{"DescribeResize"},
		"ClusterIdentifier": []string{"resize-demo"},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe resize, got %d: %s", status, string(body))
	}

	cancel := url.Values{
		"Action":            []string{"CancelResize"},
		"ClusterIdentifier": []string{"resize-demo"},
	}
	status, body = redshiftRequest(t, ts, cancel)
	if status != http.StatusOK {
		t.Fatalf("expected 200 cancel resize, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Status>CANCELED</Status>")) {
		t.Fatalf("missing canceled status: %s", string(body))
	}
}

func TestRedshiftStage4ScheduledActions(t *testing.T) {
	srv := New(Config{
		Addr:      "127.0.0.1:0",
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		LogLevel:  "error",
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	create := url.Values{
		"Action":              []string{"CreateScheduledAction"},
		"ScheduledActionName": []string{"action-1"},
		"TargetAction":        []string{"ResizeCluster"},
		"Schedule":            []string{"cron(0 12 * * ? *)"},
		"IamRole":             []string{"arn:aws:iam::123456789012:role/demo"},
	}
	status, body := redshiftRequest(t, ts, create)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create scheduled action, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<ScheduledActionName>action-1</ScheduledActionName>")) {
		t.Fatalf("missing scheduled action: %s", string(body))
	}

	create2 := url.Values{
		"Action":              []string{"CreateScheduledAction"},
		"ScheduledActionName": []string{"action-2"},
		"TargetAction":        []string{"ResizeCluster"},
		"Schedule":            []string{"cron(0 13 * * ? *)"},
	}
	status, body = redshiftRequest(t, ts, create2)
	if status != http.StatusOK {
		t.Fatalf("expected 200 create scheduled action 2, got %d: %s", status, string(body))
	}

	bad := url.Values{
		"Action":              []string{"CreateScheduledAction"},
		"ScheduledActionName": []string{"bad-action"},
		"TargetAction":        []string{"ResizeCluster"},
		"Schedule":            []string{"invalid"},
	}
	status, body = redshiftRequest(t, ts, bad)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid schedule, got %d: %s", status, string(body))
	}

	describe := url.Values{
		"Action":     []string{"DescribeScheduledActions"},
		"MaxRecords": []string{"1"},
	}
	status, body = redshiftRequest(t, ts, describe)
	if status != http.StatusOK {
		t.Fatalf("expected 200 describe scheduled actions, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<Marker>action-1</Marker>")) {
		t.Fatalf("expected marker for pagination: %s", string(body))
	}

	modify := url.Values{
		"Action":              []string{"ModifyScheduledAction"},
		"ScheduledActionName": []string{"action-1"},
		"State":               []string{"DISABLED"},
		"Description":         []string{"paused"},
	}
	status, body = redshiftRequest(t, ts, modify)
	if status != http.StatusOK {
		t.Fatalf("expected 200 modify scheduled action, got %d: %s", status, string(body))
	}
	if !bytes.Contains(body, []byte("<State>DISABLED</State>")) {
		t.Fatalf("missing updated state: %s", string(body))
	}

	deleteParams := url.Values{
		"Action":              []string{"DeleteScheduledAction"},
		"ScheduledActionName": []string{"action-1"},
	}
	status, body = redshiftRequest(t, ts, deleteParams)
	if status != http.StatusOK {
		t.Fatalf("expected 200 delete scheduled action, got %d: %s", status, string(body))
	}
}
