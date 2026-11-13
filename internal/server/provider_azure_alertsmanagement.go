package server

import (
	"fmt"
	"slices"
	"strings"
)

// Microsoft.AlertsManagement alert processing rules and Prometheus rule
// groups, plus the fired-alerts list. The emulator never fires alerts.
func init() {
	const ns = "Microsoft.AlertsManagement"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "actionRules",
		Validate:           azureAlertProcessingRuleValidate,
		ReadOnlyProperties: azureAlertRuleEnabledByDefault,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "prometheusRuleGroups",
		Validate:           azurePrometheusRuleGroupValidate,
		ReadOnlyProperties: azureAlertRuleEnabledByDefault,
	})
	registerAzureARMView(ns, "alerts", func(*Server, string) []map[string]any { return nil })
}

// azureAlertRuleEnabledByDefault turns a rule on unless the client says otherwise.
func azureAlertRuleEnabledByDefault(rec *azureARMResource) map[string]any {
	if _, present := rec.Properties["enabled"]; present {
		return nil
	}
	return map[string]any{"enabled": true}
}

var azureAlertActionTypes = []string{"AddActionGroups", "RemoveAllActionGroups"}

func azureAlertProcessingRuleValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if scopes, _ := props["scopes"].([]any); len(scopes) == 0 {
		return "properties.scopes must list at least one scope."
	}
	actions, _ := props["actions"].([]any)
	if len(actions) == 0 {
		return "properties.actions must list at least one action."
	}
	for i, raw := range actions {
		action := azureARMObject(raw)
		actionType := azureARMString(action["actionType"])
		if !slices.Contains(azureAlertActionTypes, actionType) {
			return fmt.Sprintf("properties.actions[%d].actionType must be AddActionGroups or RemoveAllActionGroups.", i)
		}
		if ids, _ := action["actionGroupIds"].([]any); actionType == "AddActionGroups" && len(ids) == 0 {
			return fmt.Sprintf("properties.actions[%d].actionGroupIds must list at least one action group.", i)
		}
	}
	return ""
}

func azurePrometheusRuleGroupValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if scopes, _ := props["scopes"].([]any); len(scopes) == 0 {
		return "properties.scopes must list the Azure Monitor workspace."
	}
	rules, _ := props["rules"].([]any)
	if len(rules) == 0 {
		return "properties.rules must list at least one rule."
	}
	for i, raw := range rules {
		rule := azureARMObject(raw)
		record, alert := azureARMString(rule["record"]), azureARMString(rule["alert"])
		if (record == "") == (alert == "") {
			return fmt.Sprintf("properties.rules[%d] must set exactly one of record or alert.", i)
		}
		if strings.TrimSpace(azureARMString(rule["expression"])) == "" {
			return fmt.Sprintf("properties.rules[%d].expression is required.", i)
		}
	}
	return ""
}
