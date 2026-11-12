package server

import (
	"net/http"
	"testing"
)

func TestAzureMigrateProjectsLifecycle(t *testing.T) {
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
	resourceGroup := "rg-migrate-projects"
	projectName := "project-a"
	base := "/azure/subscriptions/" + subscriptionID
	projectPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/migrateProjects/" + projectName + "?api-version=" + azureMigrateProjectsAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Migrate/operations?api-version="+azureMigrateProjectsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, projectPath, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"summary":{"goal":"Servers"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	project := providerContractJSONMap(t, resp)
	if project["type"] != "Microsoft.Migrate/migrateProjects" {
		t.Fatalf("unexpected migrate project payload: %#v", project)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, projectPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on HEAD, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, projectPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, projectPath, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"projectStatus":"Updated"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Migrate/migrateProjects?api-version="+azureMigrateProjectsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing projects by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/migrateProjects?api-version="+azureMigrateProjectsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing projects by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/migrateProjects/"+projectName+"/registerTool?api-version="+azureMigrateProjectsAPIVersion, []byte(`{
		"tool":"ServerDiscovery"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 registering tool, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	registration := providerContractJSONMap(t, resp)
	if ok, _ := registration["isRegistered"].(bool); !ok {
		t.Fatalf("expected isRegistered=true, got %#v", registration)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/migrateProjects/"+projectName+"/refreshSummary?api-version="+azureMigrateProjectsAPIVersion, []byte(`{
		"goal":"Databases"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 refreshing summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	refresh := providerContractJSONMap(t, resp)
	if refresh["status"] != "Succeeded" {
		t.Fatalf("unexpected refresh payload: %#v", refresh)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, projectPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, projectPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureMigrateProjectsValidation(t *testing.T) {
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

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.Migrate/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Migrate/migrateProjects/project-a?api-version=" + azureMigrateProjectsAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid tool",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Migrate/migrateProjects/project-a/registerTool?api-version=" + azureMigrateProjectsAPIVersion,
			body:       []byte(`{"tool":"UnknownTool"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing project on get",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Migrate/migrateProjects/missing?api-version=" + azureMigrateProjectsAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid goal",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Migrate/migrateProjects/project-a/refreshSummary?api-version=" + azureMigrateProjectsAPIVersion,
			body:       []byte(`{"goal":"Unknown"}`),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
