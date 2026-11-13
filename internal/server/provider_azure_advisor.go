package server

import (
	"net/http"
	"slices"
)

// Microsoft.Advisor configurations at subscription and resource group
// scope, and recommendations. The emulator never generates recommendations.
func init() {
	const ns = "Microsoft.Advisor"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:    ns,
		Type:         "configurations",
		Extension:    true,
		ValidateName: azureSingletonName("default"),
		Validate:     azureAdvisorValidateConfiguration,
	})
	registerAzureARMView(ns, "recommendations", func(*Server, string) []map[string]any { return nil })
	registerAzureARMProviderAction(ns, "generateRecommendation", func(*Server, map[string]any) (int, any) { return http.StatusAccepted, nil })
}

var azureAdvisorCPUThresholds = []string{"5", "10", "15", "20", "100"}

func azureAdvisorValidateConfiguration(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if raw, present := props["lowCpuThreshold"]; present && !slices.Contains(azureAdvisorCPUThresholds, azureARMString(raw)) {
		return "properties.lowCpuThreshold must be one of 5, 10, 15, 20 or 100."
	}
	if raw, present := props["exclude"]; present {
		if _, ok := raw.(bool); !ok {
			return "properties.exclude must be a boolean."
		}
	}
	return ""
}
