package server

import (
	"net/http"
	"slices"
	"strings"
	"time"
)

// Microsoft.PolicyInsights remediations, attestations and policy state
// queries, served by the shared ARM handler. Stackyard evaluates no policy:
// remediations deploy nothing and the state queries return no records.
func init() {
	const ns = "Microsoft.PolicyInsights"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "remediations", Extension: true,
		Validate:           azurePolicyInsightsValidateRemediation,
		ReadOnlyProperties: azurePolicyInsightsRemediationProperties,
		// ponytail: cancel is accepted at any time, even after the
		// remediation finished; nothing ever runs.
		Transitions: map[string]map[string]any{"cancel": {"provisioningState": "Canceled"}},
		Actions: map[string]azureARMAction{"listdeployments": func(*azureARMResource, map[string]any) (int, any) {
			return http.StatusOK, map[string]any{"value": []any{}}
		}},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "attestations", Extension: true,
		Validate:           azurePolicyInsightsValidateAttestation,
		ReadOnlyProperties: azurePolicyInsightsAttestationProperties,
	})
	for _, path := range []string{"policyStates/latest/queryResults", "policyStates/default/queryResults", "policyEvents/default/queryResults"} {
		collection, version, _ := strings.Cut(path, "/")
		version, _, _ = strings.Cut(version, "/")
		registerAzureARMScopeAction(ns, path, func(_ *Server, scope string, _ map[string]any) (int, any) {
			return http.StatusOK, map[string]any{
				"@odata.context": "https://management.azure.com" + scope + "/providers/" + ns + "/" + collection + "/$metadata#" + version,
				"@odata.count":   0,
				"value":          []any{},
			}
		})
	}
	registerAzureARMScopeAction(ns, "policyStates/latest/summarize", azurePolicyInsightsSummarize)
}

func azurePolicyInsightsAssignmentID(props map[string]any) string {
	if !strings.Contains(strings.ToLower(azureARMString(props["policyAssignmentId"])), "/providers/microsoft.authorization/policyassignments/") {
		return "properties.policyAssignmentId must be a policy assignment ID."
	}
	return ""
}

func azurePolicyInsightsValidateRemediation(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azurePolicyInsightsAssignmentID(props); msg != "" {
		return msg
	}
	if msg := azureEnumProperty("resourceDiscoveryMode", "ExistingNonCompliant", "ReEvaluateCompliance")(payload); msg != "" {
		return msg
	}
	if raw, present := azureARMObject(props["failureThreshold"])["percentage"]; present {
		if pct, ok := raw.(float64); !ok || pct < 0 || pct > 1 {
			return "properties.failureThreshold.percentage must be a number from 0 to 1."
		}
	}
	for _, key := range []string{"resourceCount", "parallelDeployments"} {
		if raw, present := props[key]; present {
			if n, ok := raw.(float64); !ok || n != float64(int(n)) || n < 1 || n > 50000 {
				return "properties." + key + " must be an integer from 1 to 50000."
			}
		}
	}
	return ""
}

// azurePolicyInsightsRemediationProperties reports a remediation that found
// nothing to deploy, with the discovery mode default.
func azurePolicyInsightsRemediationProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"createdOn":        rec.CreatedAt.Format(time.RFC3339),
		"lastUpdatedOn":    rec.UpdatedAt.Format(time.RFC3339),
		"deploymentStatus": map[string]any{"totalDeployments": 0, "successfulDeployments": 0, "failedDeployments": 0},
	}
	if _, present := rec.Properties["resourceDiscoveryMode"]; !present {
		props["resourceDiscoveryMode"] = "ExistingNonCompliant"
	}
	return props
}

var azurePolicyInsightsComplianceStates = []string{"Compliant", "NonCompliant", "Unknown"}

func azurePolicyInsightsValidateAttestation(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azurePolicyInsightsAssignmentID(props); msg != "" {
		return msg
	}
	if msg := azureEnumProperty("complianceState", azurePolicyInsightsComplianceStates...)(payload); msg != "" {
		return msg
	}
	for _, key := range []string{"expiresOn", "assessmentDate"} {
		if raw, present := props[key]; present {
			if _, err := time.Parse(time.RFC3339, azureARMString(raw)); err != nil {
				return "properties." + key + " must be an RFC 3339 time."
			}
		}
	}
	return ""
}

func azurePolicyInsightsAttestationProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"lastComplianceStateChangeAt": rec.UpdatedAt.Format(time.RFC3339)}
	if state := azureARMString(rec.Properties["complianceState"]); !slices.Contains(azurePolicyInsightsComplianceStates, state) {
		props["complianceState"] = "Unknown"
	}
	return props
}

// azurePolicyInsightsSummarize returns an empty compliance summary.
func azurePolicyInsightsSummarize(_ *Server, scope string, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{
		"@odata.context": "https://management.azure.com" + scope + "/providers/Microsoft.PolicyInsights/policyStates/$metadata#summary",
		"@odata.count":   1,
		"value": []any{map[string]any{
			"@odata.id": nil,
			"results": map[string]any{
				"queryResultsUri":       "https://management.azure.com" + scope + "/providers/Microsoft.PolicyInsights/policyStates/latest/queryResults",
				"nonCompliantResources": 0,
				"nonCompliantPolicies":  0,
				"resourceDetails":       []any{},
				"policyDetails":         []any{},
				"policyGroupDetails":    []any{},
			},
			"policyAssignments": []any{},
		}},
	}
}
