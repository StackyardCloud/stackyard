package server

import (
	"net/http"
	"strings"
	"testing"
)

const azurePolicyInsightsTestAPIVersion = "?api-version=2021-10-01"

func TestAzurePolicyInsightsRemediationLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-policy")
	assignment := sub + "/providers/Microsoft.Authorization/policyAssignments/require-tags"
	remediation := sub + "/resourceGroups/rg-policy/providers/Microsoft.PolicyInsights/remediations/fix-tags"

	resp := do(http.MethodPut, remediation+azurePolicyInsightsTestAPIVersion, `{"properties":{"policyAssignmentId":"`+assignment+`","failureThreshold":{"percentage":0.1},"parallelDeployments":10}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["resourceDiscoveryMode"] != "ExistingNonCompliant" || props["deploymentStatus"].(map[string]any)["totalDeployments"] != float64(0) {
		t.Fatalf("unexpected remediation properties: %#v", props)
	}

	// Actions on extension resources.
	resp = do(http.MethodPost, remediation+"/listDeployments"+azurePolicyInsightsTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 0 {
		t.Fatalf("expected no deployments, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, remediation+"/cancel"+azurePolicyInsightsTestAPIVersion, ""), http.StatusOK)
	resp = do(http.MethodGet, remediation+azurePolicyInsightsTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["provisioningState"]; got != "Canceled" {
		t.Fatalf("expected Canceled, got %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/rg-policy/providers/Microsoft.PolicyInsights/remediations/missing/cancel"+azurePolicyInsightsTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")

	resp = do(http.MethodGet, sub+"/resourceGroups/rg-policy/providers/Microsoft.PolicyInsights/remediations"+azurePolicyInsightsTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one remediation, got %d", len(got))
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, remediation+azurePolicyInsightsTestAPIVersion, ""), http.StatusOK)
}

func TestAzurePolicyInsightsAttestationsAndStates(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-policy")
	assignment := sub + "/providers/Microsoft.Authorization/policyAssignments/manual-review"
	attestation := sub + "/providers/Microsoft.PolicyInsights/attestations/reviewed?api-version=2022-09-01"

	resp := do(http.MethodPut, attestation, `{"properties":{"policyAssignmentId":"`+assignment+`","complianceState":"Compliant","expiresOn":"2027-01-01T00:00:00Z","owner":"ops"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if props := providerContractJSONMap(t, resp)["properties"].(map[string]any); props["complianceState"] != "Compliant" || props["lastComplianceStateChangeAt"] == nil {
		t.Fatalf("unexpected attestation properties: %#v", props)
	}
	resp = do(http.MethodPut, attestation, `{"properties":{"policyAssignmentId":"`+assignment+`"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["complianceState"]; got != "Unknown" {
		t.Fatalf("expected the Unknown default, got %v", got)
	}

	// Scope actions at subscription, resource group and resource scope.
	for _, scope := range []string{sub, sub + "/resourceGroups/rg-policy", sub + "/resourceGroups/rg-policy/providers/Microsoft.Storage/storageAccounts/logs"} {
		resp = do(http.MethodPost, scope+"/providers/Microsoft.PolicyInsights/policyStates/latest/queryResults?api-version=2019-10-01", "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		body := providerContractJSONMap(t, resp)
		if body["@odata.count"] != float64(0) || !strings.HasSuffix(body["@odata.context"].(string), "/policyStates/$metadata#latest") {
			t.Fatalf("unexpected query results at %s: %#v", scope, body)
		}
	}
	resp = do(http.MethodPost, sub+"/providers/Microsoft.PolicyInsights/policyStates/latest/summarize?api-version=2019-10-01", "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	summary := providerContractJSONMap(t, resp)["value"].([]any)[0].(map[string]any)["results"].(map[string]any)
	if summary["nonCompliantResources"] != float64(0) {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/missing/providers/Microsoft.PolicyInsights/policyStates/latest/queryResults?api-version=2019-10-01", ""), http.StatusNotFound, "ResourceGroupNotFound")
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.PolicyInsights/policyStates/latest/queryResults?api-version=2019-10-01", ""), http.StatusMethodNotAllowed)
}

func TestAzurePolicyInsightsValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-policy")
	assignment := sub + "/providers/Microsoft.Authorization/policyAssignments/a"
	remediation := sub + "/providers/Microsoft.PolicyInsights/remediations/r" + azurePolicyInsightsTestAPIVersion
	attestation := sub + "/providers/Microsoft.PolicyInsights/attestations/a?api-version=2022-09-01"
	cases := []struct{ name, path, body string }{
		{"no assignment", remediation, `{"properties":{}}`},
		{"bad discovery mode", remediation, `{"properties":{"policyAssignmentId":"` + assignment + `","resourceDiscoveryMode":"All"}}`},
		{"bad failure threshold", remediation, `{"properties":{"policyAssignmentId":"` + assignment + `","failureThreshold":{"percentage":2}}}`},
		{"bad resource count", remediation, `{"properties":{"policyAssignmentId":"` + assignment + `","resourceCount":0}}`},
		{"attestation assignment", attestation, `{"properties":{"policyAssignmentId":"/subscriptions/x"}}`},
		{"attestation state", attestation, `{"properties":{"policyAssignmentId":"` + assignment + `","complianceState":"Exempt"}}`},
		{"attestation expiry", attestation, `{"properties":{"policyAssignmentId":"` + assignment + `","expiresOn":"tomorrow"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
