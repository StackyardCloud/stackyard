package server

import (
	"net/http"
	"testing"
)

const azureStreamAnalyticsTestAPIVersion = "?api-version=2020-03-01"

func TestAzureStreamAnalyticsLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-asa")
	job := sub + "/resourceGroups/rg-asa/providers/Microsoft.StreamAnalytics/streamingjobs/clicks"

	resp := do(http.MethodPut, job+azureStreamAnalyticsTestAPIVersion, `{"location":"eastus","properties":{"sku":{"name":"StandardV2"},"outputErrorPolicy":"Drop","jobState":"Running","jobStorageAccount":{"accountName":"jobs","accountKey":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["jobState"] != "Created" || props["jobStorageAccount"].(map[string]any)["accountKey"] != nil || props["outputErrorPolicy"] != "Drop" || props["eventsOutOfOrderPolicy"] != "Adjust" || len(props["jobId"].(string)) != 36 {
		t.Fatalf("unexpected job properties: %#v", props)
	}

	resp = do(http.MethodPut, job+"/inputs/events"+azureStreamAnalyticsTestAPIVersion, `{"properties":{"type":"Stream","datasource":{"type":"Microsoft.ServiceBus/EventHub","properties":{"serviceBusNamespace":"ns","eventHubName":"clicks","sharedAccessPolicyName":"listen","sharedAccessPolicyKey":"secret"}},"serialization":{"type":"Json","properties":{"encoding":"UTF8"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	source := providerContractJSONMap(t, resp)["properties"].(map[string]any)["datasource"].(map[string]any)["properties"].(map[string]any)
	if source["sharedAccessPolicyKey"] != nil || source["sharedAccessPolicyName"] != "listen" {
		t.Fatalf("expected the key stripped, got %#v", source)
	}
	resp = do(http.MethodPut, job+"/outputs/lake"+azureStreamAnalyticsTestAPIVersion, `{"properties":{"datasource":{"type":"Microsoft.Storage/Blob","properties":{"container":"out","pathPattern":"{date}","storageAccounts":[{"accountName":"lake","accountKey":"secret"}]}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	accounts := providerContractJSONMap(t, resp)["properties"].(map[string]any)["datasource"].(map[string]any)["properties"].(map[string]any)["storageAccounts"].([]any)
	if account := accounts[0].(map[string]any); account["accountKey"] != nil || account["accountName"] != "lake" {
		t.Fatalf("expected the storage key stripped, got %#v", account)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, job+"/transformations/main"+azureStreamAnalyticsTestAPIVersion, `{"properties":{"query":"SELECT * INTO lake FROM events","streamingUnits":6}}`), http.StatusCreated)

	for _, step := range []struct{ action, state string }{{"start", "Running"}, {"stop", "Stopped"}} {
		azureResourcesExpectStatus(t, do(http.MethodPost, job+"/"+step.action+azureStreamAnalyticsTestAPIVersion, `{"outputStartMode":"JobStartTime"}`), http.StatusOK)
		resp = do(http.MethodPatch, job+azureStreamAnalyticsTestAPIVersion, `{"tags":{"step":"`+step.action+`"}}`)
		if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["jobState"]; got != step.state {
			t.Fatalf("after %s: expected %s, got %v", step.action, step.state, got)
		}
	}
	// Only start and stop change the state: not a PATCH, and not a PUT that replaces the job.
	resp = do(http.MethodPatch, job+azureStreamAnalyticsTestAPIVersion, `{"properties":{"jobState":"Running"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["jobState"]; got != "Stopped" {
		t.Fatalf("a PATCH must not set jobState, got %v", got)
	}
	resp = do(http.MethodPut, job+azureStreamAnalyticsTestAPIVersion, `{"location":"eastus","properties":{}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["jobState"]; got != "Stopped" {
		t.Fatalf("a PUT must not reset jobState, got %v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, job+azureStreamAnalyticsTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, job+"/inputs/events"+azureStreamAnalyticsTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureStreamAnalyticsValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-asa")
	job := sub + "/resourceGroups/rg-asa/providers/Microsoft.StreamAnalytics/streamingjobs/clicks"
	cases := []struct{ name, path, body string }{
		{"bad sku", job, `{"location":"eastus","properties":{"sku":{"name":"Premium"}}}`},
		{"bad out-of-order policy", job, `{"location":"eastus","properties":{"eventsOutOfOrderPolicy":"Keep"}}`},
		{"bad compatibility level", job, `{"location":"eastus","properties":{"compatibilityLevel":"2.0"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureStreamAnalyticsTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, job+azureStreamAnalyticsTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	children := []struct{ name, path, body string }{
		{"input type", job + "/inputs/in", `{"properties":{"type":"Batch","datasource":{"type":"Microsoft.Storage/Blob"}}}`},
		{"input datasource", job + "/inputs/in", `{"properties":{"type":"Stream"}}`},
		{"output datasource", job + "/outputs/out", `{"properties":{}}`},
		{"no query", job + "/transformations/main", `{"properties":{"streamingUnits":1}}`},
		{"bad streaming units", job + "/transformations/main", `{"properties":{"query":"SELECT 1","streamingUnits":4}}`},
	}
	for _, tc := range children {
		resp := do(http.MethodPut, tc.path+azureStreamAnalyticsTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
