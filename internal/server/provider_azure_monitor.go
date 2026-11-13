package server

import (
	"regexp"
	"strings"
)

// Microsoft.Insights alerting, autoscale and diagnostic settings, served by
// the shared ARM handler. Each type uses its own API version in Azure; the
// handler accepts any.
func init() {
	const ns = "Microsoft.Insights"
	for _, typ := range []azureARMResourceType{
		{Type: "actionGroups", Validate: azureMonitorValidateActionGroup},
		{Type: "metricAlerts", Validate: azureMonitorValidateMetricAlert},
		{Type: "activityLogAlerts", Validate: azureMonitorValidateActivityLogAlert},
		{Type: "scheduledQueryRules", Validate: azureMonitorValidateScheduledQueryRule},
		{Type: "autoscalesettings", Validate: azureMonitorValidateAutoscale},
		{Type: "diagnosticSettings", Extension: true, Validate: azureMonitorValidateDiagnosticSetting},
	} {
		typ.Namespace = ns
		registerAzureARMResourceType(typ)
	}
}

// azureMonitorDurationPattern matches the ISO 8601 durations alert rules use,
// such as PT1M, PT1H or P1D. Every part is optional, so callers also reject
// the empty forms "P" and "PT" by length.
var azureMonitorDurationPattern = regexp.MustCompile(`^P(\d+D)?(T(\d+H)?(\d+M)?(\d+S)?)?$`)

func azureMonitorValidateActionGroup(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	shortName := azureARMString(props["groupShortName"])
	if shortName == "" || len(shortName) > 12 {
		return "properties.groupShortName is required and must be at most 12 characters."
	}
	if _, ok := props["enabled"].(bool); !ok {
		return "properties.enabled is required."
	}
	return ""
}

func azureMonitorValidateMetricAlert(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureMonitorRequireScopes(props); msg != "" {
		return msg
	}
	if severity, ok := props["severity"].(float64); !ok || severity < 0 || severity > 4 || severity != float64(int(severity)) {
		return "properties.severity must be an integer from 0 to 4."
	}
	if _, ok := props["enabled"].(bool); !ok {
		return "properties.enabled is required."
	}
	for _, field := range []string{"evaluationFrequency", "windowSize"} {
		if value := azureARMString(props[field]); len(value) < 3 || !azureMonitorDurationPattern.MatchString(value) {
			return "properties." + field + " must be an ISO 8601 duration such as PT5M."
		}
	}
	if azureARMString(azureARMObject(props["criteria"])["odata.type"]) == "" {
		return "properties.criteria with an odata.type is required."
	}
	return ""
}

func azureMonitorValidateActivityLogAlert(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureMonitorRequireScopes(props); msg != "" {
		return msg
	}
	if allOf, _ := azureARMObject(props["condition"])["allOf"].([]any); len(allOf) == 0 {
		return "properties.condition.allOf must contain at least one condition."
	}
	return ""
}

func azureMonitorValidateScheduledQueryRule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureMonitorRequireScopes(props); msg != "" {
		return msg
	}
	if allOf, _ := azureARMObject(props["criteria"])["allOf"].([]any); len(allOf) == 0 {
		return "properties.criteria.allOf must contain at least one condition."
	}
	return ""
}

func azureMonitorValidateAutoscale(payload map[string]any) string {
	if profiles, _ := azureARMObject(payload["properties"])["profiles"].([]any); len(profiles) == 0 || len(profiles) > 20 {
		return "properties.profiles must contain 1 to 20 profiles."
	}
	return ""
}

// azureMonitorValidateDiagnosticSetting requires a destination, as Azure does.
func azureMonitorValidateDiagnosticSetting(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, field := range []string{"workspaceId", "storageAccountId", "eventHubAuthorizationRuleId", "marketplacePartnerId"} {
		if strings.TrimSpace(azureARMString(props[field])) != "" {
			return ""
		}
	}
	return "At least one destination (workspaceId, storageAccountId, eventHubAuthorizationRuleId or marketplacePartnerId) is required."
}

func azureMonitorRequireScopes(props map[string]any) string {
	if scopes, _ := props["scopes"].([]any); len(scopes) == 0 {
		return "properties.scopes must contain at least one resource ID."
	}
	return ""
}
