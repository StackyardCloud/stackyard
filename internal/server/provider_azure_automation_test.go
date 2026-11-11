package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAzureAutomationLifecycle(t *testing.T) {
	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-automation"
	accountName := "aa-stage"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Automation/automationAccounts/" + accountName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Automation/operations?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"?api-version=2024-10-23", []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"sku":{"name":"Basic"},
		"properties":{"publicNetworkAccess":"Enabled","disableLocalAuth":false}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating automation account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	account := providerContractJSONMap(t, resp)
	if account["type"] != "Microsoft.Automation/automationAccounts" || account["name"] != accountName {
		t.Fatalf("unexpected automation account payload: %#v", account)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting automation account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Automation/automationAccounts?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing accounts by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/providers/Microsoft.Automation/automationAccounts?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing accounts by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, base+"?api-version=2024-10-23", []byte(`{"tags":{"env":"stage","owner":"stackyard"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching automation account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	variablePath := base + "/variables/var-a"
	resp = providerContractRequest(t, ts, http.MethodPut, variablePath+"?api-version=2024-10-23", []byte(`{"properties":{"value":"stackyard","isEncrypted":false}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating variable, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, variablePath+"?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting variable, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/variables?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing variables, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	credentialPath := base + "/credentials/cred-a"
	resp = providerContractRequest(t, ts, http.MethodPut, credentialPath+"?api-version=2024-10-23", []byte(`{"properties":{"userName":"stackyard-user","password":"secret"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating credential, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	certificatePath := base + "/certificates/cert-a"
	resp = providerContractRequest(t, ts, http.MethodPut, certificatePath+"?api-version=2024-10-23", []byte(`{"properties":{"base64Value":"c3RhY2t5YXJk"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating certificate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	runbookPath := base + "/runbooks/rb-a"
	resp = providerContractRequest(t, ts, http.MethodPut, runbookPath+"?api-version=2024-10-23", []byte(`{
		"properties":{"runbookType":"PowerShellWorkflow","content":"workflow rb-a { Write-Output \"stackyard\" }","logVerbose":true}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating runbook, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, runbookPath+"/content?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting runbook content, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, runbookPath+"/draft?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting runbook draft, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, runbookPath+"/draft/content?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting runbook draft content, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, runbookPath+"/draft/undoEdit?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 undoing draft edit, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, runbookPath+"/publish?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 publishing runbook, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	schedulePath := base + "/schedules/sched-a"
	resp = providerContractRequest(t, ts, http.MethodPut, schedulePath+"?api-version=2024-10-23", []byte(`{"properties":{"startTime":"2026-01-02T03:04:05Z","frequency":"Hour"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating schedule, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	jobSchedulePath := base + "/jobSchedules/js-a"
	resp = providerContractRequest(t, ts, http.MethodPut, jobSchedulePath+"?api-version=2024-10-23", []byte(`{
		"properties":{"schedule":{"name":"sched-a"},"runbook":{"name":"rb-a"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating job schedule, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/webhooks/generateUri?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 generating webhook uri, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	webhookPath := base + "/webhooks/wh-a"
	resp = providerContractRequest(t, ts, http.MethodPut, webhookPath+"?api-version=2024-10-23", []byte(`{
		"properties":{"runbook":{"name":"rb-a"},"isEnabled":true}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating webhook, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/webhooks?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing webhooks, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	jobPath := base + "/jobs/job-a"
	resp = providerContractRequest(t, ts, http.MethodPut, jobPath+"?api-version=2024-10-23", []byte(`{
		"properties":{"runbook":{"name":"rb-a"},"parameters":{"message":"hello"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, jobPath+"?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, jobPath+"/output?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting job output, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, jobPath+"/runbookContent?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting job runbook content, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, jobPath+"/streams?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing job streams, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	var streamList struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(providerContractBody(t, resp), &streamList); err != nil {
		t.Fatalf("unmarshal job stream list: %v", err)
	}
	if len(streamList.Value) == 0 {
		t.Fatalf("expected at least one job stream, got %#v", streamList.Value)
	}
	streamID, _ := streamList.Value[0]["name"].(string)
	resp = providerContractRequest(t, ts, http.MethodGet, jobPath+"/streams/"+streamID+"?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting job stream, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	for _, action := range []string{"suspend", "resume", "stop"} {
		resp = providerContractRequest(t, ts, http.MethodPost, jobPath+"/"+action+"?api-version=2024-10-23", nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 posting job action %s, got %d body=%s", action, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	sourceControlPath := base + "/sourceControls/sc-a"
	resp = providerContractRequest(t, ts, http.MethodPut, sourceControlPath+"?api-version=2024-10-23", []byte(`{
		"properties":{"repoUrl":"https://github.com/stackyard/example","branch":"main","folderPath":"/","sourceType":"GitHub"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating source control, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sourceControls?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing source controls, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	syncJobPath := sourceControlPath + "/sourceControlSyncJobs/sync-a"
	resp = providerContractRequest(t, ts, http.MethodPut, syncJobPath+"?api-version=2024-10-23", []byte(`{"properties":{"syncType":"FullSync"}}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating source control sync job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, sourceControlPath+"/sourceControlSyncJobs?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing source control sync jobs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, syncJobPath+"/streams?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing sync job streams, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	runtimePath := base + "/runtimeEnvironments/re-a"
	resp = providerContractRequest(t, ts, http.MethodPut, runtimePath+"?api-version=2024-10-23", []byte(`{"properties":{"runtime":{"language":"PowerShell","version":"7.4"}}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating runtime environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	packagePath := runtimePath + "/packages/pkg-a"
	resp = providerContractRequest(t, ts, http.MethodPut, packagePath+"?api-version=2024-10-23", []byte(`{"properties":{"contentLink":{"uri":"https://example.invalid/pkg.whl"}}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating package, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	privateEndpointConnectionPath := base + "/privateEndpointConnections/pec-a"
	resp = providerContractRequest(t, ts, http.MethodPut, privateEndpointConnectionPath+"?api-version=2024-10-23", []byte(`{
		"properties":{"privateEndpoint":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-automation/providers/Microsoft.Network/privateEndpoints/pe-a"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, listPath := range []string{
		base + "/statistics?api-version=2024-10-23",
		base + "/usages?api-version=2024-10-23",
		base + "/privateLinkResources?api-version=2024-10-23",
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, listPath, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", listPath, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	for _, deletePath := range []string{
		privateEndpointConnectionPath,
		packagePath,
		runtimePath,
		sourceControlPath,
		webhookPath,
		jobSchedulePath,
		schedulePath,
		runbookPath,
		certificatePath,
		credentialPath,
		variablePath,
		base,
	} {
		resp = providerContractRequest(t, ts, http.MethodDelete, deletePath+"?api-version=2024-10-23", nil, headers)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 deleting %s, got %d body=%s", deletePath, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
}

func TestAzureAutomationValidationAndNotFound(t *testing.T) {
	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	base := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-automation/providers/Microsoft.Automation/automationAccounts/aa-stage"

	resp := providerContractRequest(t, ts, http.MethodPut, base+"?api-version=2024-10-23", []byte(`{"tags":{"env":"dev"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"?api-version=2024-10-23", []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/variables/var-a?api-version=2024-10-23", []byte(`{"properties":{"value":"x"}}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for child resource under missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"?api-version=2024-10-23", []byte(`{"location":"eastus"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/credentials/cred-a?api-version=2024-10-23", []byte(`{"properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing credential username, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/runbooks/rb-a?api-version=2024-10-23", []byte(`{"properties":{"runbookType":"PowerShellWorkflow"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating runbook, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/jobs/job-a?api-version=2024-10-23", []byte(`{"properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing job runbook name, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/jobs/job-missing?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/sourceControls/sc-a/sourceControlSyncJobs/sync-a/streams/stream-1?api-version=2024-10-23", nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing sync job stream, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureAutomationInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Automation/operations?api-version=", nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
