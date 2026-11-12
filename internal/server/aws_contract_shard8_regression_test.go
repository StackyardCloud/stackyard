package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloudhsmsvc "github.com/stackyard/stackyard/internal/services/cloudhsm"
	memorydbsvc "github.com/stackyard/stackyard/internal/services/memorydb"
)

func TestAthenaShard8ScalarAndSessionEndpointShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createWG := []byte(`{"Name":"shard8-wg","Description":"demo","State":"ENABLED"}`)
	resp := athenaRequest(t, ts, "AmazonAthena.CreateWorkGroup", createWG)
	assertStatus(t, resp, http.StatusOK)

	startQuery := []byte(`{"QueryString":"SELECT 1","QueryExecutionContext":{"Database":"db1","Catalog":"AwsDataCatalog"},"WorkGroup":"shard8-wg","ResultConfiguration":{"OutputLocation":"s3://demo/output/"}}`)
	resp = athenaRequest(t, ts, "AmazonAthena.StartQueryExecution", startQuery)
	assertStatus(t, resp, http.StatusOK)
	var startQueryOut struct {
		QueryExecutionId string `json:"QueryExecutionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startQueryOut); err != nil {
		t.Fatalf("unmarshal start query response: %v", err)
	}
	if startQueryOut.QueryExecutionId == "" {
		t.Fatalf("expected query execution id")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.GetQueryResults", []byte(`{"QueryExecutionId":"`+startQueryOut.QueryExecutionId+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var queryResultsOut struct {
		UpdateCount int `json:"UpdateCount"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &queryResultsOut); err != nil {
		t.Fatalf("unmarshal get query results response: %v", err)
	}
	if queryResultsOut.UpdateCount != 0 {
		t.Fatalf("expected UpdateCount 0, got %d", queryResultsOut.UpdateCount)
	}

	resp = athenaRequest(t, ts, "AmazonAthena.CreatePresignedNotebookUrl", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	var presignedOut struct {
		AuthTokenExpirationTime int64 `json:"AuthTokenExpirationTime"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &presignedOut); err != nil {
		t.Fatalf("unmarshal create presigned notebook url response: %v", err)
	}
	if presignedOut.AuthTokenExpirationTime <= time.Now().UTC().Unix() {
		t.Fatalf("expected future AuthTokenExpirationTime, got %d", presignedOut.AuthTokenExpirationTime)
	}

	resp = athenaRequest(t, ts, "AmazonAthena.StartSession", []byte(`{"WorkGroup":"shard8-wg","Description":"coverage session"}`))
	assertStatus(t, resp, http.StatusOK)
	var startSessionOut struct {
		SessionId string `json:"SessionId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startSessionOut); err != nil {
		t.Fatalf("unmarshal start session response: %v", err)
	}
	if startSessionOut.SessionId == "" {
		t.Fatalf("expected session id")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.GetSession", []byte(`{"SessionId":"`+startSessionOut.SessionId+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var getSessionOut struct {
		EngineVersion string `json:"EngineVersion"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getSessionOut); err != nil {
		t.Fatalf("unmarshal get session response: %v", err)
	}
	if getSessionOut.EngineVersion == "" {
		t.Fatalf("expected string EngineVersion in GetSession response")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.GetSessionEndpoint", []byte(`{"SessionId":"`+startSessionOut.SessionId+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var endpointOut struct {
		EndpointUrl             string `json:"EndpointUrl"`
		AuthTokenExpirationTime int64  `json:"AuthTokenExpirationTime"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &endpointOut); err != nil {
		t.Fatalf("unmarshal get session endpoint response: %v", err)
	}
	if endpointOut.EndpointUrl == "" {
		t.Fatalf("expected EndpointUrl in GetSessionEndpoint response")
	}
	if endpointOut.AuthTokenExpirationTime <= time.Now().UTC().Unix() {
		t.Fatalf("expected future session endpoint expiration, got %d", endpointOut.AuthTokenExpirationTime)
	}

	resp = athenaRequest(t, ts, "AmazonAthena.GetResourceDashboard", []byte(`{"ResourceARN":"arn:aws:athena:us-east-1:123456789012:workgroup/shard8-wg"}`))
	assertStatus(t, resp, http.StatusOK)
	var dashboardOut struct {
		URL string `json:"Url"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &dashboardOut); err != nil {
		t.Fatalf("unmarshal get resource dashboard response: %v", err)
	}
	if dashboardOut.URL == "" {
		t.Fatalf("expected Url in GetResourceDashboard response")
	}

	resp = athenaRequest(t, ts, "AmazonAthena.ListExecutors", []byte(`{"SessionId":"`+startSessionOut.SessionId+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var listExecutorsOut struct {
		ExecutorsSummary []struct {
			StartDateTime int64 `json:"StartDateTime"`
		} `json:"ExecutorsSummary"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listExecutorsOut); err != nil {
		t.Fatalf("unmarshal list executors response: %v", err)
	}
	if len(listExecutorsOut.ExecutorsSummary) == 0 {
		t.Fatalf("expected at least one executor")
	}
	if listExecutorsOut.ExecutorsSummary[0].StartDateTime <= 0 {
		t.Fatalf("expected integer StartDateTime in ListExecutors response")
	}
}

func TestCloudHSMShard8PayloadsOmitLegacyFields(t *testing.T) {
	cluster := cloudhsmsvc.Cluster{
		ClusterID:       "cluster-1",
		CreateTimestamp: time.Unix(1700000000, 0).UTC(),
		Hsms: []cloudhsmsvc.Hsm{{
			HsmID:     "hsm-1",
			EniID:     "eni-1",
			EniIP:     "10.0.0.10",
			EniIPv6:   "2001:db8::1",
			ClusterID: "cluster-1",
		}},
		HsmType:       "hsm1.medium",
		SubnetMapping: map[string]string{"us-east-1a": "subnet-1"},
		VpcID:         "vpc-1",
		NetworkType:   "IPV4",
		Mode:          "FIPS",
	}
	clusterOut := cloudhsmClusterPayload(cluster)
	if _, ok := clusterOut["NetworkType"]; ok {
		t.Fatalf("expected cloudhsm cluster payload to omit NetworkType")
	}
	if _, ok := clusterOut["Mode"]; ok {
		t.Fatalf("expected cloudhsm cluster payload to omit Mode")
	}
	hsms, _ := clusterOut["Hsms"].([]map[string]any)
	if len(hsms) == 0 {
		t.Fatalf("expected cloudhsm cluster payload to include Hsms")
	}
	if _, ok := hsms[0]["EniIpV6"]; ok {
		t.Fatalf("expected nested cloudhsm HSM payload to omit EniIpV6")
	}

	backupOut := cloudhsmBackupPayload(cloudhsmsvc.Backup{
		BackupID:        "backup-1",
		BackupARN:       "arn:aws:cloudhsm:us-east-1:123456789012:backup/backup-1",
		ClusterID:       "cluster-1",
		CreateTimestamp: time.Unix(1700000000, 0).UTC(),
		HsmType:         "hsm1.medium",
		Mode:            "FIPS",
	})
	for _, field := range []string{"BackupArn", "HsmType", "Mode"} {
		if _, ok := backupOut[field]; ok {
			t.Fatalf("expected cloudhsm backup payload to omit %s", field)
		}
	}
}

func TestMemoryDBShard8PayloadsOmitLegacyFields(t *testing.T) {
	clusterOut := memorydbClusterToAPI(memorydbsvc.Cluster{
		Name:                   "cluster-1",
		Status:                 "available",
		MultiRegionClusterName: "global-cluster",
		NodeType:               "db.t4g.small",
		Engine:                 "redis",
		EngineVersion:          "7.1",
		Port:                   6379,
	})
	for _, field := range []string{"MultiRegionClusterName", "Engine"} {
		if _, ok := clusterOut[field]; ok {
			t.Fatalf("expected memorydb cluster payload to omit %s", field)
		}
	}

	engineVersionOut := memorydbEngineVersionToAPI(memorydbsvc.EngineVersion{
		Engine:               "redis",
		EngineVersion:        "7.1",
		EnginePatchVersion:   "7.1.0",
		ParameterGroupFamily: "memorydb_redis7",
	})
	if _, ok := engineVersionOut["Engine"]; ok {
		t.Fatalf("expected memorydb engine version payload to omit Engine")
	}

	serviceUpdateOut := memorydbServiceUpdateToAPI(memorydbsvc.ServiceUpdate{
		ClusterName:       "cluster-1",
		ServiceUpdateName: "update-1",
		ReleaseDate:       time.Unix(1700000000, 0).UTC(),
		Description:       "demo",
		Status:            "available",
		Type:              "security-update",
		Engine:            "redis",
	})
	if _, ok := serviceUpdateOut["Engine"]; ok {
		t.Fatalf("expected memorydb service update payload to omit Engine")
	}
}
