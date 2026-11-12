package server

import (
	"net/http"
	"testing"
)

func TestAzureMigrateAssessmentLifecycle(t *testing.T) {
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
	resourceGroup := "rg-migrate"
	projectName := "project-a"
	groupName := "group-a"
	assessmentName := "assessment-a"
	base := "/azure/subscriptions/" + subscriptionID
	projectResource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/assessmentProjects/" + projectName + "?api-version=" + azureMigrateAssessmentAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Migrate/operations?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, projectResource, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"projectStatus":"Active"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	project := providerContractJSONMap(t, resp)
	if project["type"] != "Microsoft.Migrate/assessmentProjects" {
		t.Fatalf("unexpected project payload: %#v", project)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, projectResource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Migrate/assessmentProjects?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing projects by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	groupResource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/assessmentProjects/" + projectName + "/groups/" + groupName + "?api-version=" + azureMigrateAssessmentAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, groupResource, []byte(`{
		"properties":{"groupType":"Custom"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/groups/"+groupName+"/updateMachines?api-version="+azureMigrateAssessmentAPIVersion, []byte(`{
		"machineNames":["machine-1","machine-2"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 updating group machines, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/machines?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing machines, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	machines := providerContractJSONMap(t, resp)
	if values, ok := machines["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected machine fixtures, got %#v", machines)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/assessmentOptions/default?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting assessment options, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/projectSummary/default?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting project summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	assessmentResource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/assessmentProjects/" + projectName + "/groups/" + groupName + "/assessments/" + assessmentName + "?api-version=" + azureMigrateAssessmentAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, assessmentResource, []byte(`{
		"properties":{"assessmentType":"Machine","azureLocation":"eastus"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating assessment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/groups/"+groupName+"/assessments/"+assessmentName+"/assessedMachines?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing assessed machines, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	collectorResource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/assessmentProjects/" + projectName + "/vmwarecollectors/collector-a?api-version=" + azureMigrateAssessmentAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, collectorResource, []byte(`{
		"properties":{"discoverySiteId":"site-a"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating collector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/privateEndpointConnections?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private endpoint connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/privateEndpointConnections/default?api-version="+azureMigrateAssessmentAPIVersion, []byte(`{
		"properties":{"privateLinkServiceConnectionState":{"status":"Approved"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 updating private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/privateLinkResources?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/aksOptions/default?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting AKS options, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Migrate/assessmentProjects/"+projectName+"/aksSummary/default?api-version="+azureMigrateAssessmentAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting AKS summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureMigrateAssessmentValidation(t *testing.T) {
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
			name:       "missing project location",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Migrate/assessmentProjects/project-a?api-version=" + azureMigrateAssessmentAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing project on get",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Migrate/assessmentProjects/missing?api-version=" + azureMigrateAssessmentAPIVersion,
			wantStatus: http.StatusNotFound,
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
