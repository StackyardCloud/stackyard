package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCodeBuildStage1ContractShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	projectPayload := `{
		"name":"shape-project",
		"source":{"type":"NO_SOURCE"},
		"environment":{"type":"LINUX_CONTAINER","image":"aws/codebuild/standard:7.0","computeType":"BUILD_GENERAL1_SMALL"}
	}`

	resp := codeBuildRequest(t, ts, "CreateProject", projectPayload)
	assertStatus(t, resp, http.StatusOK)
	var createProject map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &createProject); err != nil {
		t.Fatalf("unmarshal CreateProject: %v", err)
	}
	project, ok := createProject["project"].(map[string]any)
	if !ok {
		t.Fatalf("expected project object, got %#v", createProject["project"])
	}
	if _, exists := project["autoRetryLimit"]; exists {
		t.Fatalf("CreateProject returned autoRetryLimit: %#v", project)
	}

	resp = codeBuildRequest(t, ts, "CreateFleet", `{"name":"shape-fleet"}`)
	assertStatus(t, resp, http.StatusOK)
	var createFleet map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &createFleet); err != nil {
		t.Fatalf("unmarshal CreateFleet: %v", err)
	}
	fleet, ok := createFleet["fleet"].(map[string]any)
	if !ok {
		t.Fatalf("expected fleet object, got %#v", createFleet["fleet"])
	}
	if _, exists := fleet["computeConfiguration"]; exists {
		t.Fatalf("CreateFleet returned computeConfiguration: %#v", fleet)
	}
	if _, exists := fleet["overflowBehavior"]; exists {
		t.Fatalf("CreateFleet returned overflowBehavior: %#v", fleet)
	}

	resp = codeBuildRequest(t, ts, "BatchGetFleets", `{"names":["shape-fleet"]}`)
	assertStatus(t, resp, http.StatusOK)
	var batchGetFleets map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &batchGetFleets); err != nil {
		t.Fatalf("unmarshal BatchGetFleets: %v", err)
	}
	fleets, ok := batchGetFleets["fleets"].([]any)
	if !ok || len(fleets) != 1 {
		t.Fatalf("expected one fleet, got %#v", batchGetFleets["fleets"])
	}
	firstFleet, ok := fleets[0].(map[string]any)
	if !ok {
		t.Fatalf("expected fleet object, got %#v", fleets[0])
	}
	if _, exists := firstFleet["computeConfiguration"]; exists {
		t.Fatalf("BatchGetFleets returned computeConfiguration: %#v", firstFleet)
	}
	if _, exists := firstFleet["overflowBehavior"]; exists {
		t.Fatalf("BatchGetFleets returned overflowBehavior: %#v", firstFleet)
	}

	resp = codeBuildRequest(t, ts, "CreateWebhook", `{"projectName":"shape-project"}`)
	assertStatus(t, resp, http.StatusOK)
	var createWebhook map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &createWebhook); err != nil {
		t.Fatalf("unmarshal CreateWebhook: %v", err)
	}
	webhook, ok := createWebhook["webhook"].(map[string]any)
	if !ok {
		t.Fatalf("expected webhook object, got %#v", createWebhook["webhook"])
	}
	if _, exists := webhook["manualCreation"]; exists {
		t.Fatalf("CreateWebhook returned manualCreation: %#v", webhook)
	}
	if _, exists := webhook["status"]; exists {
		t.Fatalf("CreateWebhook returned status: %#v", webhook)
	}

	resp = codeBuildRequest(t, ts, "GetReportGroupTrend", `{"numOfReports":1,"trendField":"DURATION"}`)
	assertStatus(t, resp, http.StatusOK)
	var trendOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &trendOut); err != nil {
		t.Fatalf("unmarshal GetReportGroupTrend: %v", err)
	}
	stats, ok := trendOut["stats"].(map[string]any)
	if !ok {
		t.Fatalf("expected stats object, got %#v", trendOut["stats"])
	}
	for _, key := range []string{"average", "max", "min"} {
		if _, ok := stats[key].(string); !ok {
			t.Fatalf("expected %s to be string, got %#v", key, stats[key])
		}
	}

	resp = codeBuildRequest(t, ts, "ImportSourceCredentials", `{"serverType":"GITHUB","authType":"PERSONAL_ACCESS_TOKEN","token":"shape-token"}`)
	assertStatus(t, resp, http.StatusOK)
	var importOut struct {
		Arn string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &importOut); err != nil {
		t.Fatalf("unmarshal ImportSourceCredentials: %v", err)
	}
	if importOut.Arn == "" {
		t.Fatalf("expected imported source credential arn")
	}

	resp = codeBuildRequest(t, ts, "ListSourceCredentials", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var listOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal ListSourceCredentials: %v", err)
	}
	infos, ok := listOut["sourceCredentialsInfos"].([]any)
	if !ok || len(infos) == 0 {
		t.Fatalf("expected sourceCredentialsInfos, got %#v", listOut["sourceCredentialsInfos"])
	}
	firstInfo, ok := infos[0].(map[string]any)
	if !ok {
		t.Fatalf("expected sourceCredentialsInfos[0] object, got %#v", infos[0])
	}
	if _, exists := firstInfo["resource"]; exists {
		t.Fatalf("ListSourceCredentials returned resource: %#v", firstInfo)
	}

	resp = codeBuildRequest(t, ts, "DeleteSourceCredentials", fmt.Sprintf(`{"arn":%q}`, importOut.Arn))
	assertStatus(t, resp, http.StatusOK)
	var deleteOut struct {
		Arn string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &deleteOut); err != nil {
		t.Fatalf("unmarshal DeleteSourceCredentials: %v", err)
	}
	if deleteOut.Arn != importOut.Arn {
		t.Fatalf("expected DeleteSourceCredentials to echo arn %q, got %q", importOut.Arn, deleteOut.Arn)
	}
}
