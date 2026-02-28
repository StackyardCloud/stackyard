package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoveryStage12ApplicationAndTaggingLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := discoveryRequest(t, ts, "CreateApplication", `{"name":"stage-app","description":"stage description"}`)
	assertStatus(t, resp, http.StatusOK)

	body := mustBody(t, resp)
	var createOut map[string]any
	if err := json.Unmarshal(body, &createOut); err != nil {
		t.Fatalf("decode CreateApplication response: %v", err)
	}
	applicationID, _ := createOut["configurationId"].(string)
	if strings.TrimSpace(applicationID) == "" {
		t.Fatalf("expected configurationId in CreateApplication response, got %q", string(body))
	}

	resp = discoveryRequest(t, ts, "AssociateConfigurationItemsToApplication", `{"applicationConfigurationId":"`+applicationID+`","configurationIds":["srv-000001","srv-000002"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "UpdateApplication", `{"configurationId":"`+applicationID+`","name":"stage-app-updated","description":"updated stage description"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "CreateTags", `{"configurationIds":["`+applicationID+`"],"tags":[{"key":"env","value":"stage"},{"key":"owner","value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "DescribeTags", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected DescribeTags to include owner tag, got %q", body)
	}

	resp = discoveryRequest(t, ts, "DeleteTags", `{"configurationIds":["`+applicationID+`"],"tags":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "DisassociateConfigurationItemsFromApplication", `{"applicationConfigurationId":"`+applicationID+`","configurationIds":["srv-000002"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "DeleteApplications", `{"configurationIds":["`+applicationID+`"]}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDiscoveryStage34AgentCollectionAndExportSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := discoveryRequest(t, ts, "DescribeAgents", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "agent-000001") {
		t.Fatalf("expected DescribeAgents to include seeded agent, got %q", body)
	}

	resp = discoveryRequest(t, ts, "StartDataCollectionByAgentIds", `{"agentIds":["agent-000001"]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "operationSucceeded") {
		t.Fatalf("expected StartDataCollectionByAgentIds response to include operationSucceeded, got %q", body)
	}

	resp = discoveryRequest(t, ts, "StopDataCollectionByAgentIds", `{"agentIds":["agent-000001"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "BatchDeleteAgents", `{"deleteAgents":[{"agentId":"agent-999999"}]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "errors") {
		t.Fatalf("expected BatchDeleteAgents response to include errors, got %q", body)
	}

	resp = discoveryRequest(t, ts, "StartContinuousExport", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "continuousExportId") {
		t.Fatalf("expected StartContinuousExport to include continuousExportId, got %q", body)
	}

	resp = discoveryRequest(t, ts, "DescribeContinuousExports", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "descriptions") {
		t.Fatalf("expected DescribeContinuousExports response to include descriptions, got %q", body)
	}

	resp = discoveryRequest(t, ts, "StopContinuousExport", `{}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDiscoveryStage56ImportExportBatchDeleteReadSurfaceAndValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := discoveryRequest(t, ts, "StartExportTask", `{"exportDataFormat":"CSV"}`)
	assertStatus(t, resp, http.StatusOK)
	exportBody := mustBody(t, resp)
	if !strings.Contains(string(exportBody), "exportId") {
		t.Fatalf("expected StartExportTask to include exportId, got %q", string(exportBody))
	}

	resp = discoveryRequest(t, ts, "DescribeExportTasks", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "exportsInfo") {
		t.Fatalf("expected DescribeExportTasks response to include exportsInfo, got %q", body)
	}

	resp = discoveryRequest(t, ts, "ExportConfigurations", `{"exportDataFormat":"CSV"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "DescribeExportConfigurations", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "StartImportTask", `{"name":"stage-import","importUrl":"s3://stackyard/stage-import.csv","clientRequestToken":"stage-import-token"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "importTaskId") {
		t.Fatalf("expected StartImportTask response to include importTaskId, got %q", body)
	}

	resp = discoveryRequest(t, ts, "DescribeImportTasks", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "BatchDeleteImportData", `{"importTaskIds":["stage-import-token"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "StartBatchDeleteConfigurationTask", `{"configurationType":"SERVER","configurationIds":["srv-000001"]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "taskId") {
		t.Fatalf("expected StartBatchDeleteConfigurationTask response to include taskId, got %q", body)
	}

	resp = discoveryRequest(t, ts, "DescribeBatchDeleteConfigurationTask", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "task") {
		t.Fatalf("expected DescribeBatchDeleteConfigurationTask response to include task, got %q", body)
	}

	resp = discoveryRequest(t, ts, "DescribeConfigurations", `{"configurationIds":["srv-000001"]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "configurations") {
		t.Fatalf("expected DescribeConfigurations response to include configurations, got %q", body)
	}

	resp = discoveryRequest(t, ts, "ListConfigurations", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = discoveryRequest(t, ts, "ListServerNeighbors", `{"configurationId":"srv-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "knownDependencyCount") {
		t.Fatalf("expected ListServerNeighbors response to include knownDependencyCount, got %q", body)
	}

	resp = discoveryRequest(t, ts, "GetDiscoverySummary", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "servers") {
		t.Fatalf("expected GetDiscoverySummary response to include servers, got %q", body)
	}

	resp = discoveryRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSPoseidonService_V2015_11_01.DescribeAgents",
		},
		"discovery",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
