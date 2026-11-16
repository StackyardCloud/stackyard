package server

import (
	"net/http"
	"testing"
)

const azurePortalTestAPIVersion = "?api-version=2020-09-01-preview"

func TestAzurePortalDashboardLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-portal")
	dashboard := sub + "/resourceGroups/rg-portal/providers/Microsoft.Portal/dashboards/ops-overview"
	body := `{"location":"eastus","tags":{"hidden-title":"Ops overview"},"properties":{"lenses":[{"order":0,"parts":[{"position":{"x":0,"y":0,"rowSpan":2,"colSpan":3},"metadata":{"type":"Extension/HubsExtension/PartType/MarkdownPart","inputs":[],"settings":{"content":{"settings":{"content":"# Ops","title":"Notes"}}}}}]}],"metadata":{"model":{"timeRange":{"value":{"relative":{"duration":24,"timeUnit":1}},"type":"MsPortalFx.Composition.Configuration.ValueTypes.TimeRange"}}}}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, dashboard+azurePortalTestAPIVersion, body), http.StatusCreated)
	// Older clients key lenses and parts by index.
	legacy := `{"location":"eastus","properties":{"lenses":{"0":{"order":0,"parts":{"0":{"position":{"x":0,"y":0,"rowSpan":4,"colSpan":6}}}}}}}`
	azureResourcesExpectStatus(t, do(http.MethodPut, dashboard+azurePortalTestAPIVersion, legacy), http.StatusOK)
	resp := do(http.MethodPatch, dashboard+azurePortalTestAPIVersion, `{"tags":{"team":"ops"}}`)
	if got := providerContractJSONMap(t, resp)["tags"].(map[string]any)["team"]; got != "ops" {
		t.Fatalf("expected the tag patched, got %v", got)
	}

	config := "/azure/providers/Microsoft.Portal/tenantConfigurations/default"
	azureResourcesExpectStatus(t, do(http.MethodPut, config+azurePortalTestAPIVersion, `{"properties":{"enforcePrivateMarkdownStorage":true}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, config+azurePortalTestAPIVersion, `{"properties":{}}`), http.StatusMethodNotAllowed, "MethodNotAllowed")
	resp = do(http.MethodPost, "/azure/providers/Microsoft.Portal/listTenantConfigurationViolations"+azurePortalTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 0 {
		t.Fatalf("expected no violations, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, config+azurePortalTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, dashboard+azurePortalTestAPIVersion, ""), http.StatusOK)
}

func TestAzurePortalValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-portal")
	dashboard := sub + "/resourceGroups/rg-portal/providers/Microsoft.Portal/dashboards/ops" + azurePortalTestAPIVersion
	for name, body := range map[string]string{
		"lenses type": `{"location":"eastus","properties":{"lenses":"all"}}`,
		"lens order":  `{"location":"eastus","properties":{"lenses":[{"parts":[]}]}}`,
		"lens parts":  `{"location":"eastus","properties":{"lenses":[{"order":0}]}}`,
		"position":    `{"location":"eastus","properties":{"lenses":[{"order":0,"parts":[{"position":{"x":0,"y":0,"rowSpan":2}}]}]}}`,
	} {
		resp := do(http.MethodPut, dashboard, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-portal/providers/Microsoft.Portal/dashboards/ops_board"+azurePortalTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure/providers/Microsoft.Portal/tenantConfigurations/default"+azurePortalTestAPIVersion, `{"properties":{"enforcePrivateMarkdownStorage":"yes"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, "/azure/providers/Microsoft.Portal/tenantConfigurations/other"+azurePortalTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidResourceName")
}
