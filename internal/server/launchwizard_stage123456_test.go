package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLaunchWizardStage12WorkloadPatternAndDeploymentReads(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := launchWizardRequest(t, ts, http.MethodPost, "/listWorkloads", `{}`)
	assertStatus(t, resp, http.StatusOK)
	workloadsBody := string(mustBody(t, resp))
	if !strings.Contains(workloadsBody, "SAP_HANA_SINGLE") {
		t.Fatalf("expected ListWorkloads to include seeded workload, got %q", workloadsBody)
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/getWorkload", `{"workloadName":"SAP_HANA_SINGLE"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = launchWizardRequest(t, ts, http.MethodPost, "/listWorkloadDeploymentPatterns", `{"workloadName":"SAP_HANA_SINGLE"}`)
	assertStatus(t, resp, http.StatusOK)
	patternsBody := string(mustBody(t, resp))
	if !strings.Contains(patternsBody, "single-node-hana") {
		t.Fatalf("expected ListWorkloadDeploymentPatterns to include single-node-hana, got %q", patternsBody)
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/getWorkloadDeploymentPattern", `{"workloadName":"SAP_HANA_SINGLE","deploymentPatternName":"single-node-hana"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = launchWizardRequest(t, ts, http.MethodPost, "/listDeploymentPatternVersions", `{"workloadName":"SAP_HANA_SINGLE","deploymentPatternName":"single-node-hana"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = launchWizardRequest(t, ts, http.MethodPost, "/getDeploymentPatternVersion", `{"workloadName":"SAP_HANA_SINGLE","deploymentPatternName":"single-node-hana","deploymentPatternVersionName":"v1"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestLaunchWizardStage345DeploymentLifecycleEventsAndTagging(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := launchWizardRequest(
		t,
		ts,
		http.MethodPost,
		"/createDeployment",
		`{"name":"stage-launchwizard-deployment","workloadName":"SAP_HANA_SINGLE","workloadDeploymentPatternName":"single-node-hana","workloadVersionName":"v1","clientToken":"stage-launchwizard-create-deployment-token-000001"}`,
	)
	assertStatus(t, resp, http.StatusOK)
	deployment := decodeLaunchWizardPayload(t, resp)
	deploymentInfo := launchWizardMap(deployment, "deployment")
	deploymentID := launchWizardString(deploymentInfo, "deploymentId")
	deploymentARN := launchWizardString(deploymentInfo, "deploymentArn")
	if deploymentID == "" || deploymentARN == "" {
		t.Fatalf("expected CreateDeployment response to include deploymentId and deploymentArn: %#v", deploymentInfo)
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/getDeployment", `{"deploymentId":"`+deploymentID+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = launchWizardRequest(t, ts, http.MethodPost, "/listDeployments", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, deploymentID) {
		t.Fatalf("expected ListDeployments to include deployment id %s, got %q", deploymentID, body)
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/updateDeployment", `{"deploymentId":"`+deploymentID+`","name":"stage-launchwizard-deployment-updated","status":"DEPLOYED"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = launchWizardRequest(t, ts, http.MethodPost, "/listDeploymentEvents", `{"deploymentId":"`+deploymentID+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "deploymentEvents") {
		t.Fatalf("expected ListDeploymentEvents response to include deploymentEvents, got %q", body)
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/tags/", `{"resourceArn":"`+deploymentARN+`","tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = launchWizardRequest(t, ts, http.MethodGet, "/tags/?resourceArn="+deploymentARN, "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = launchWizardRequest(t, ts, http.MethodDelete, "/tags/?resourceArn="+deploymentARN+"&tagKeys=owner", "")
	assertStatus(t, resp, http.StatusOK)
	resp = launchWizardRequest(t, ts, http.MethodGet, "/tags/?resourceArn="+deploymentARN, "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "\"owner\"") {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/deleteDeployment", `{"deploymentId":"`+deploymentID+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestLaunchWizardStage6ValidationIdempotencyAndMalformedJSON(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	token := "stage-launchwizard-idempotent-create-token-000001"
	resp := launchWizardRequest(t, ts, http.MethodPost, "/createDeployment", `{"clientToken":"`+token+`","name":"idempotent-name-1"}`)
	assertStatus(t, resp, http.StatusOK)
	first := decodeLaunchWizardPayload(t, resp)
	firstID := launchWizardString(launchWizardMap(first, "deployment"), "deploymentId")
	if firstID == "" {
		t.Fatalf("expected first CreateDeployment response to include deploymentId")
	}

	resp = launchWizardRequest(t, ts, http.MethodPost, "/createDeployment", `{"clientToken":"`+token+`","name":"idempotent-name-2"}`)
	assertStatus(t, resp, http.StatusOK)
	second := decodeLaunchWizardPayload(t, resp)
	secondID := launchWizardString(launchWizardMap(second, "deployment"), "deploymentId")
	if firstID != secondID {
		t.Fatalf("expected idempotent CreateDeployment to return same deploymentId: %s != %s", firstID, secondID)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/listWorkloads",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"launchwizard",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeLaunchWizardPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func launchWizardMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if raw, ok := payload[key]; ok {
		if m, ok := raw.(map[string]any); ok {
			return m
		}
	}
	return map[string]any{}
}

func launchWizardString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if raw, ok := payload[key]; ok {
		if s, ok := raw.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
