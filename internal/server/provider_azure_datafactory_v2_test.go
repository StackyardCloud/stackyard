package server

import (
	"net/http"
	"testing"
	"time"
)

const azureADFTestVersion = "?api-version=" + azureDataFactoryV2APIVersion

func TestAzureDataFactoryV2Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-adf")
	factory := sub + "/resourceGroups/rg-adf/providers/Microsoft.DataFactory/factories/adf1"

	resp := do(http.MethodPut, factory+azureADFTestVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	body := providerContractJSONMap(t, resp)
	if body["etag"] == nil || body["properties"].(map[string]any)["version"] != "2018-06-01" {
		t.Fatalf("unexpected factory: %#v", body)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, factory+"/linkedservices/blob"+azureADFTestVersion, `{"properties":{"type":"AzureBlobStorage","typeProperties":{"connectionString":"x"}}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, factory+"/datasets/input"+azureADFTestVersion, `{"properties":{"type":"Binary","linkedServiceName":{"referenceName":"blob","type":"LinkedServiceReference"}}}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, factory+"/pipelines/copy"+azureADFTestVersion, `{"properties":{"activities":[{"name":"Copy1","type":"Copy"},{"name":"Wait1","type":"Wait"}],"parameters":{"p":{"type":"String"}}}}`), http.StatusOK)

	resp = do(http.MethodPost, factory+"/pipelines/copy/createRun"+azureADFTestVersion, `{"p":"value","n":3}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	runID, _ := providerContractJSONMap(t, resp)["runId"].(string)
	if runID == "" {
		t.Fatal("createRun returned no runId")
	}
	resp = do(http.MethodGet, factory+"/pipelineruns/"+runID+azureADFTestVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	run := providerContractJSONMap(t, resp)
	if run["status"] != "Succeeded" || run["pipelineName"] != "copy" || run["parameters"].(map[string]any)["n"] != "3" {
		t.Fatalf("unexpected run: %#v", run)
	}
	if _, leaked := run[azureADFActivityKey]; leaked {
		t.Fatal("run body leaks server-only fields")
	}
	window := `{"lastUpdatedAfter":"` + time.Now().Add(-time.Hour).UTC().Format(time.RFC3339) + `","lastUpdatedBefore":"` + time.Now().Add(time.Hour).UTC().Format(time.RFC3339) + `"`
	resp = do(http.MethodPost, factory+"/pipelineruns/"+runID+"/queryActivityruns"+azureADFTestVersion, window+`}`)
	if acts := providerContractJSONMap(t, resp)["value"].([]any); len(acts) != 2 {
		t.Fatalf("want 2 activity runs, got %d", len(acts))
	}
	resp = do(http.MethodPost, factory+"/queryPipelineRuns"+azureADFTestVersion, window+`,"filters":[{"operand":"PipelineName","operator":"Equals","values":["copy"]}]}`)
	if runs := providerContractJSONMap(t, resp)["value"].([]any); len(runs) != 1 {
		t.Fatalf("want 1 queried run, got %d", len(runs))
	}
	resp = do(http.MethodPost, factory+"/queryPipelineRuns"+azureADFTestVersion, window+`,"filters":[{"operand":"PipelineName","operator":"Equals","values":["other"]}]}`)
	if runs := providerContractJSONMap(t, resp)["value"].([]any); len(runs) != 0 {
		t.Fatalf("filter matched %d runs", len(runs))
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, factory+"/pipelineruns/"+runID+"/cancel"+azureADFTestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, factory+"/queryTriggerRuns"+azureADFTestVersion, window+`}`), http.StatusOK)

	trigger := factory + "/triggers/nightly"
	azureResourcesExpectStatus(t, do(http.MethodPut, trigger+azureADFTestVersion, `{"properties":{"type":"ScheduleTrigger","runtimeState":"Started"}}`), http.StatusOK)
	resp = do(http.MethodPost, trigger+"/getEventSubscriptionStatus"+azureADFTestVersion, "")
	if got := providerContractJSONMap(t, resp)["status"]; got != "Disabled" {
		t.Fatalf("subscription status of a stopped trigger = %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, trigger+"/start"+azureADFTestVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, trigger+azureADFTestVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["runtimeState"]; got != "Started" {
		t.Fatalf("runtimeState after start = %v", got)
	}
	resp = do(http.MethodPost, factory+"/querytriggers"+azureADFTestVersion, `{}`)
	if triggers := providerContractJSONMap(t, resp)["value"].([]any); len(triggers) != 1 {
		t.Fatalf("want 1 trigger, got %d", len(triggers))
	}

	ir := factory + "/integrationRuntimes/shir"
	azureResourcesExpectStatus(t, do(http.MethodPut, ir+azureADFTestVersion, `{"properties":{"type":"SelfHosted"}}`), http.StatusOK)
	resp = do(http.MethodPost, ir+"/listAuthKeys"+azureADFTestVersion, "")
	if keys := providerContractJSONMap(t, resp); keys["authKey1"] == nil || keys["authKey2"] == nil {
		t.Fatalf("unexpected auth keys: %#v", keys)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, ir+"/regenerateAuthKey"+azureADFTestVersion, `{"keyName":"authKey2"}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, ir+"/start"+azureADFTestVersion, ""), http.StatusOK)
	resp = do(http.MethodPost, ir+"/getStatus"+azureADFTestVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["state"]; got != "Started" {
		t.Fatalf("integration runtime state = %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, ir+"/outboundNetworkDependenciesEndpoints"+azureADFTestVersion, ""), http.StatusOK)

	resp = do(http.MethodPost, sub+"/providers/Microsoft.DataFactory/locations/eastus/configureFactoryRepo"+azureADFTestVersion,
		`{"factoryResourceId":"`+factory[len("/azure"):]+`","repoConfiguration":{"type":"FactoryGitHubConfiguration","accountName":"a","repositoryName":"r","collaborationBranch":"main","rootFolder":"/"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	resp = do(http.MethodGet, "/azure/providers/Microsoft.DataFactory/operations"+azureADFTestVersion, "")
	if ops := providerContractJSONMap(t, resp)["value"].([]any); len(ops) == 0 {
		t.Fatal("operations list is empty")
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, factory+azureADFTestVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, factory+"/pipelineruns/"+runID+azureADFTestVersion, ""), http.StatusNotFound)
}

func TestAzureDataFactoryV2Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-adf")
	factory := sub + "/resourceGroups/rg-adf/providers/Microsoft.DataFactory/factories/adf2"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, factory+azureADFTestVersion, `{"location":"eastus","identity":{"type":"Magic"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, factory+azureADFTestVersion, `{"location":"eastus"}`), http.StatusOK)
	cases := []struct{ name, path, body string }{
		{"linked service without type", "/linkedservices/l", `{"properties":{}}`},
		{"dataset without linked service", "/datasets/d", `{"properties":{"type":"Binary"}}`},
		{"duplicate activity", "/pipelines/p", `{"properties":{"activities":[{"name":"a","type":"Wait"},{"name":"a","type":"Wait"}]}}`},
		{"bad runtime type", "/integrationRuntimes/ir", `{"properties":{"type":"Cloud"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, factory+tc.path+azureADFTestVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, factory+"/queryPipelineRuns"+azureADFTestVersion, `{"lastUpdatedAfter":"yesterday"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPost, factory+"/pipelines/missing/createRun"+azureADFTestVersion, `{}`), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodGet, factory+"/pipelineruns/00000000-0000-0000-0000-000000000000"+azureADFTestVersion, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPut, factory+"/integrationRuntimes/managed"+azureADFTestVersion, `{"properties":{"type":"Managed"}}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, factory+"/integrationRuntimes/managed/listAuthKeys"+azureADFTestVersion, ""), http.StatusBadRequest, "InvalidRequestContent")
}
