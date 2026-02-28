package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSupplyChainStage12InstanceNamespaceDatasetLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := supplyChainRequest(t, ts, http.MethodPost, "/api/instance", []byte(`{
		"instanceName":"stage-supply-chain",
		"instanceDescription":"stage instance",
		"clientToken":"stage-token-000001"
	}`))
	assertStatus(t, resp, http.StatusOK)
	instance := decodeSupplyChainPayload(t, resp)
	instanceID := supplyChainPayloadString(instance, "instance", "instanceId")
	if instanceID == "" {
		t.Fatalf("expected CreateInstance to include instance.instanceId")
	}

	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/instance/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodPatch, "/api/instance/"+url.PathEscape(instanceID), []byte(`{"instanceDescription":"updated stage description"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/instance?instanceNameFilter=stage&instanceStateFilter=Active&maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "instances") {
		t.Fatalf("expected ListInstances to include instances, got %q", body)
	}

	resp = supplyChainRequest(t, ts, http.MethodPut, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodPatch, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodPut, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace/datasets/stage-dataset", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace/datasets/stage-dataset", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodPatch, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace/datasets/stage-dataset", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace/datasets?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestSupplyChainStage34FlowEventsAndBillOfMaterialsJobs(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "scn-instance-000001"

	resp := supplyChainRequest(t, ts, http.MethodPut, "/api/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodPatch, "/api/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodPost, "/api-data/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-events", []byte(`{"eventType":"DataSetLoad"}`))
	assertStatus(t, resp, http.StatusOK)
	eventPayload := decodeSupplyChainPayload(t, resp)
	eventID := supplyChainPayloadString(eventPayload, "dataIntegrationEvent", "eventId")
	if eventID == "" {
		t.Fatalf("expected SendDataIntegrationEvent to include dataIntegrationEvent.eventId")
	}
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api-data/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-events/"+url.PathEscape(eventID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api-data/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-events?eventType=DataSetLoad&maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodGet, "/api-data/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow/executions?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api-data/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow/executions/exec-000001", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodPost, "/api/configuration/instances/"+url.PathEscape(instanceID)+"/bill-of-materials-import-jobs", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	jobPayload := decodeSupplyChainPayload(t, resp)
	jobID := supplyChainPayloadString(jobPayload, "billOfMaterialsImportJob", "jobId")
	if jobID == "" {
		t.Fatalf("expected CreateBillOfMaterialsImportJob to include billOfMaterialsImportJob.jobId")
	}
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/configuration/instances/"+url.PathEscape(instanceID)+"/bill-of-materials-import-jobs/"+url.PathEscape(jobID), nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestSupplyChainStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	instanceID := "scn-instance-000001"
	resourceARN := url.PathEscape("arn:aws:scn:us-east-1:123456789012:instance/" + instanceID)

	resp := supplyChainRequest(t, ts, http.MethodPost, "/api/tags/"+resourceARN, []byte(`{"tags":{"env":"stage","owner":"qa"}}`))
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodGet, "/api/tags/"+resourceARN, nil)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/tags/"+resourceARN+"?tagKeys=owner", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/data-integration/instance/"+url.PathEscape(instanceID)+"/data-integration-flows/stage-flow", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace/datasets/stage-dataset", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/datalake/instance/"+url.PathEscape(instanceID)+"/namespaces/stage-namespace", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/instance/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)
	resp = supplyChainRequest(t, ts, http.MethodDelete, "/api/instance/"+url.PathEscape(instanceID), nil)
	assertStatus(t, resp, http.StatusOK)

	resp = supplyChainRequest(t, ts, http.MethodPost, "/supplychain/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/api/instance",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"scn",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeSupplyChainPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func supplyChainPayloadString(payload map[string]any, nestedKey, key string) string {
	var source map[string]any
	if nestedKey == "" {
		source = payload
	} else {
		raw, _ := payload[nestedKey]
		source, _ = raw.(map[string]any)
	}
	if source == nil {
		return ""
	}
	raw, ok := source[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
