package server

import (
	"net/http"
	"testing"
)

const azureAdvisorTestAPIVersion = "?api-version=2023-01-01"

func TestAzureAdvisorConfigurations(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-advisor")
	subConfig := sub + "/providers/Microsoft.Advisor/configurations/default"
	groupConfig := sub + "/resourceGroups/rg-advisor/providers/Microsoft.Advisor/configurations/default"

	resp := do(http.MethodPut, subConfig+azureAdvisorTestAPIVersion, `{"properties":{"lowCpuThreshold":"10","exclude":false,"digests":[{"name":"weekly","frequency":7,"state":"Active"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["lowCpuThreshold"] != "10" {
		t.Fatalf("unexpected configuration: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, groupConfig+azureAdvisorTestAPIVersion, `{"properties":{"exclude":true}}`), http.StatusCreated)

	// Each scope lists only its own configuration.
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Advisor/configurations"+azureAdvisorTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 || items[0].(map[string]any)["properties"].(map[string]any)["lowCpuThreshold"] != "10" {
		t.Fatalf("unexpected subscription configurations: %#v", items)
	}

	cases := []struct{ name, path, body, code string }{
		{"bad threshold", subConfig, `{"properties":{"lowCpuThreshold":"7"}}`, "InvalidRequestContent"},
		{"exclude not boolean", groupConfig, `{"properties":{"exclude":"yes"}}`, "InvalidRequestContent"},
		{"named configuration", sub + "/providers/Microsoft.Advisor/configurations/custom", `{"properties":{}}`, "InvalidResourceName"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureAdvisorTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}

func TestAzureAdvisorRecommendations(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-advisor")
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.Advisor/generateRecommendation"+azureAdvisorTestAPIVersion, ""), http.StatusAccepted)
	resp := do(http.MethodGet, sub+"/providers/Microsoft.Advisor/recommendations"+azureAdvisorTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, ok := providerContractJSONMap(t, resp)["value"].([]any); !ok || len(items) != 0 {
		t.Fatalf("expected no recommendations, got %#v", items)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/providers/Microsoft.Advisor/recommendations/abc"+azureAdvisorTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}
