package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestOpenSearchShard1ResponseShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := opensearchRequest(
		t,
		ts,
		http.MethodPut,
		"/2021-01-01/opensearch/domain/stackyard-opensearch-domain/scheduledAction/update",
		[]byte(`{"ActionID":"scheduled-action-1","ActionType":"SERVICE_SOFTWARE_UPDATE","ScheduleAt":"2026-03-13T12:00:00Z"}`),
	)
	assertStatus(t, resp, http.StatusOK)
	var updateScheduledAction map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &updateScheduledAction); err != nil {
		t.Fatalf("unmarshal update scheduled action response: %v", err)
	}
	scheduledAction, ok := updateScheduledAction["ScheduledAction"].(map[string]any)
	if !ok {
		t.Fatalf("expected ScheduledAction object, got %#v", updateScheduledAction["ScheduledAction"])
	}
	for _, key := range []string{"Id", "Type", "Severity", "ScheduledTime"} {
		if _, ok := scheduledAction[key]; !ok {
			t.Fatalf("expected ScheduledAction.%s in response: %#v", key, scheduledAction)
		}
	}

	resp = opensearchRequest(t, ts, http.MethodGet, "/2021-01-01/opensearch/domain/stackyard-opensearch-domain/health", nil)
	assertStatus(t, resp, http.StatusOK)
	var describeDomainHealth map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeDomainHealth); err != nil {
		t.Fatalf("unmarshal describe domain health response: %v", err)
	}
	for _, key := range []string{"AvailabilityZoneCount", "ActiveAvailabilityZoneCount", "DataNodeCount"} {
		if _, ok := describeDomainHealth[key].(string); !ok {
			t.Fatalf("expected %s to be a string, got %#v", key, describeDomainHealth[key])
		}
	}

	resp = opensearchRequest(t, ts, http.MethodGet, "/2021-01-01/opensearch/domain/stackyard-opensearch-domain/dryRun", nil)
	assertStatus(t, resp, http.StatusOK)
	var dryRunProgress map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &dryRunProgress); err != nil {
		t.Fatalf("unmarshal describe dry run progress response: %v", err)
	}
	if _, ok := dryRunProgress["DryRunResults"].(map[string]any); !ok {
		t.Fatalf("expected DryRunResults object, got %#v", dryRunProgress["DryRunResults"])
	}

	resp = opensearchRequest(
		t,
		ts,
		http.MethodGet,
		"/2021-01-01/opensearch/instanceTypeLimits/OpenSearch_2.13/m5.large.search",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	var instanceTypeLimits map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &instanceTypeLimits); err != nil {
		t.Fatalf("unmarshal describe instance type limits response: %v", err)
	}
	if _, ok := instanceTypeLimits["LimitsByRole"].(map[string]any); !ok {
		t.Fatalf("expected LimitsByRole object, got %#v", instanceTypeLimits["LimitsByRole"])
	}

	resp = opensearchRequest(
		t,
		ts,
		http.MethodGet,
		"/2021-01-01/opensearch/domain/stackyard-opensearch-domain/dataSource/stackyard-ds",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	var getDataSource map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &getDataSource); err != nil {
		t.Fatalf("unmarshal get data source response: %v", err)
	}
	dataSourceType, ok := getDataSource["DataSourceType"].(map[string]any)
	if !ok {
		t.Fatalf("expected DataSourceType object, got %#v", getDataSource["DataSourceType"])
	}
	if _, ok := dataSourceType["S3GlueDataCatalog"].(map[string]any); !ok {
		t.Fatalf("expected S3GlueDataCatalog object, got %#v", dataSourceType["S3GlueDataCatalog"])
	}
}

func TestRedshiftShard1ResponseShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createClusterStatus, createClusterBody := redshiftRequest(t, ts, url.Values{
		"Action":             []string{"CreateCluster"},
		"ClusterIdentifier":  []string{"shard1-cluster"},
		"NodeType":           []string{"ra3.xlplus"},
		"ClusterType":        []string{"single-node"},
		"MasterUsername":     []string{"admin"},
		"MasterUserPassword": []string{"Secret1234"},
		"DBName":             []string{"dev"},
	})
	if createClusterStatus != http.StatusOK {
		t.Fatalf("create cluster failed: %d: %s", createClusterStatus, string(createClusterBody))
	}

	resizeStatus, resizeBody := redshiftRequest(t, ts, url.Values{
		"Action":            []string{"ResizeCluster"},
		"ClusterIdentifier": []string{"shard1-cluster"},
		"NodeType":          []string{"ra3.large"},
		"NumberOfNodes":     []string{"2"},
	})
	if resizeStatus != http.StatusOK {
		t.Fatalf("resize cluster failed: %d: %s", resizeStatus, string(resizeBody))
	}
	if !bytes.Contains(resizeBody, []byte("<Cluster>")) {
		t.Fatalf("expected ResizeCluster response to return a Cluster, got: %s", string(resizeBody))
	}
	if !bytes.Contains(resizeBody, []byte("<ResizeInfo>")) {
		t.Fatalf("expected ResizeCluster response to include ResizeInfo, got: %s", string(resizeBody))
	}

	createScheduleStatus, createScheduleBody := redshiftRequest(t, ts, url.Values{
		"Action":                       []string{"CreateSnapshotSchedule"},
		"SnapshotScheduleIdentifier":   []string{"shard1-schedule"},
		"ScheduleDefinitions.member.1": []string{"cron(0 1 * * ? *)"},
	})
	if createScheduleStatus != http.StatusOK {
		t.Fatalf("create snapshot schedule failed: %d: %s", createScheduleStatus, string(createScheduleBody))
	}

	modifyScheduleStatus, modifyScheduleBody := redshiftRequest(t, ts, url.Values{
		"Action":                     []string{"ModifySnapshotSchedule"},
		"SnapshotScheduleIdentifier": []string{"shard1-schedule"},
		"ScheduleDefinitions.member.1": []string{
			"cron(0 2 * * ? *)",
		},
	})
	if modifyScheduleStatus != http.StatusOK {
		t.Fatalf("modify snapshot schedule failed: %d: %s", modifyScheduleStatus, string(modifyScheduleBody))
	}
	if !bytes.Contains(modifyScheduleBody, []byte("<SnapshotScheduleIdentifier>shard1-schedule</SnapshotScheduleIdentifier>")) {
		t.Fatalf("expected flattened snapshot schedule identifier in response, got: %s", string(modifyScheduleBody))
	}
	if bytes.Contains(modifyScheduleBody, []byte("<SnapshotSchedule>")) {
		t.Fatalf("expected ModifySnapshotSchedule response to omit nested SnapshotSchedule wrapper: %s", string(modifyScheduleBody))
	}

	authorizeDataShareStatus, authorizeDataShareBody := redshiftRequest(t, ts, url.Values{
		"Action":             []string{"AuthorizeDataShare"},
		"DataShareArn":       []string{"arn:aws:redshift:us-east-1:123456789012:datashare:shard1-share"},
		"ConsumerIdentifier": []string{"111122223333"},
	})
	if authorizeDataShareStatus != http.StatusOK {
		t.Fatalf("authorize data share failed: %d: %s", authorizeDataShareStatus, string(authorizeDataShareBody))
	}
	if bytes.Contains(authorizeDataShareBody, []byte("<DataShareType>")) {
		t.Fatalf("expected data share response to omit DataShareType, got: %s", string(authorizeDataShareBody))
	}
}
