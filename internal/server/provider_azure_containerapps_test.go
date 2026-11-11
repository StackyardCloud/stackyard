package server

import (
	"net/http"
	"testing"
)

func TestAzureContainerAppsLifecycle(t *testing.T) {
	t.Parallel()

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
	resourceGroup := "rg-containerapps"
	region := "eastus"
	base := "/azure/subscriptions/" + subscriptionID
	envPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/managedEnvironments/env-a?api-version=2026-01-01"
	appPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/containerApps/app-a?api-version=2026-01-01"
	jobPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/jobs/job-a?api-version=2026-01-01"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.App/operations?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/locations/"+region+"/providers/Microsoft.App/availableManagedEnvironmentsWorkloadProfileTypes?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing workload profile types, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.App/getCustomDomainVerificationId?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting custom domain verification id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, envPath, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"workloadProfiles":[{"name":"Consumption","workloadProfileType":"Consumption"}]}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating managed environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	environment := providerContractJSONMap(t, resp)
	if environment["type"] != "Microsoft.App/managedEnvironments" {
		t.Fatalf("unexpected managed environment payload: %#v", environment)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, envPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting managed environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	envGet := providerContractJSONMap(t, resp)
	envProperties, ok := envGet["properties"].(map[string]any)
	if !ok || envProperties["defaultDomain"] == "" {
		t.Fatalf("expected environment default domain, got %#v", envGet)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, envPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading managed environment, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.App/managedEnvironments?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing managed environments by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, appPath, []byte(`{
		"location":"eastus",
		"tags":{"component":"web"},
		"properties":{
			"environmentId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-containerapps/providers/Microsoft.App/managedEnvironments/env-a",
			"configuration":{"ingress":{"external":true,"targetPort":8080}},
			"template":{"containers":[{"name":"web","image":"nginx:latest"}]}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	app := providerContractJSONMap(t, resp)
	appProperties, ok := app["properties"].(map[string]any)
	if !ok || appProperties["latestRevisionName"] == "" {
		t.Fatalf("expected container app latest revision, got %#v", app)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, appPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.App/containerApps?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing container apps by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.App/containerApps/app-a/revisions?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing revisions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	revisions := providerContractJSONMap(t, resp)
	values, ok := revisions["value"].([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("expected one revision fixture, got %#v", revisions)
	}
	revision := values[0].(map[string]any)
	revisionName, _ := revision["name"].(string)
	if revisionName == "" {
		t.Fatalf("expected revision name, got %#v", revision)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.App/containerApps/app-a/revisions/"+revisionName+"/replicas?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing replicas, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	authConfigPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/containerApps/app-a/authConfigs/auth-a?api-version=2026-01-01"
	resp = providerContractRequest(t, ts, http.MethodPut, authConfigPath, []byte(`{
		"properties":{"platform":{"enabled":true}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating auth config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.App/containerApps/app-a/authConfigs?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing auth configs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, appPath, []byte(`{
		"tags":{"component":"web","owner":"stackyard"},
		"properties":{"configuration":{"maxInactiveRevisions":5}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patchedApp := providerContractJSONMap(t, resp)
	tags, ok := patchedApp["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched app tags, got %#v", patchedApp)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, jobPath, []byte(`{
		"location":"eastus",
		"properties":{
			"environmentId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-containerapps/providers/Microsoft.App/managedEnvironments/env-a",
			"configuration":{"triggerType":"Manual"},
			"template":{"containers":[{"name":"job","image":"busybox"}]}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.App/jobs/job-a/executions?api-version=2026-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing job executions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	executions := providerContractJSONMap(t, resp)
	executionValues, ok := executions["value"].([]any)
	if !ok || len(executionValues) != 1 {
		t.Fatalf("expected one job execution fixture, got %#v", executions)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, authConfigPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting auth config, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, jobPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, appPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, envPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting managed environment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureContainerAppsValidationAndNotFound(t *testing.T) {
	t.Parallel()

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
	resourceGroup := "rg-containerapps"
	appPath := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/containerApps/app-a?api-version=2026-01-01"

	resp := providerContractRequest(t, ts, http.MethodPut, appPath, []byte(`{"location":"eastus","properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing environment id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, appPath, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, appPath, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, appPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/subscriptions/"+subscriptionID+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.App/containerApps/app-a/authConfigs/auth-a?api-version=2026-01-01", []byte(`{"properties":{}}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 creating child under missing app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, appPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing container app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.App/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
