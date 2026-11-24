package server

import (
	"net/netip"
	"strings"
)

// Azure Durable Task Scheduler (Microsoft.DurableTask, api-version 2026-02-01)
// on the shared ARM handler: schedulers, task hubs, the scheduler's retention
// policy, private endpoint connections, private link resources and the
// operations list. No orchestrations run.
const (
	dtaskMaxAllowlist  = 100
	dtaskMaxRetentions = 10
	dtaskMaxRetention  = 90 // days
)

func init() {
	const ns = "Microsoft.DurableTask"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "schedulers",
		Validate: dtaskValidateScheduler,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			region := strings.ToLower(strings.ReplaceAll(rec.Location, " ", ""))
			return map[string]any{"endpoint": "https://" + strings.ToLower(rec.Name) + "." + region + ".durabletask.io"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "schedulers/taskHubs", Proxy: true, NoPatch: true,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"dashboardUrl": "https://dashboard.durabletask.io/subscriptions/" + rec.SubscriptionID + "/schedulers/" +
				strings.Split(rec.ID, "/")[8] + "/taskhubs/" + rec.Name}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "schedulers/retentionPolicies", Proxy: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The retention policy must be named 'default'."
			}
			return ""
		},
		Validate: dtaskValidateRetention,
	})
	registerAzureARMPrivateLink(ns, "schedulers", "scheduler", "privatelink.durabletask.io")
	registerAzureARMOperations(ns, "Microsoft Durable Task", [][2]string{
		{"schedulers", "schedulers"}, {"schedulers/taskHubs", "task hubs"}, {"schedulers/retentionPolicies", "retention policies"},
	})
}

func dtaskValidateScheduler(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	allow, ok := props["ipAllowlist"].([]any)
	if !ok || len(allow) > dtaskMaxAllowlist {
		return "properties.ipAllowlist is missing or holds more than 100 entries."
	}
	for _, item := range allow {
		entry, _ := item.(string)
		if _, err := netip.ParsePrefix(entry); err != nil {
			if _, err := netip.ParseAddr(entry); err != nil {
				return "ipAllowlist entry '" + entry + "' is not an IP address or CIDR range."
			}
		}
	}
	sku := azureARMObject(props["sku"])
	switch azureARMString(sku["name"]) {
	case "Dedicated":
		if capacity, ok := sku["capacity"].(float64); ok && (capacity < 1 || capacity > 3 || capacity != float64(int64(capacity))) {
			return "properties.sku.capacity must be 1, 2 or 3."
		}
	case "Consumption":
	default:
		return "properties.sku.name must be Dedicated or Consumption."
	}
	return ""
}

func dtaskValidateRetention(payload map[string]any) string {
	policies, ok := azureARMObject(payload["properties"])["retentionPolicies"].([]any)
	if !ok || len(policies) > dtaskMaxRetentions {
		return "properties.retentionPolicies is missing or holds too many entries."
	}
	seen := map[string]bool{}
	for _, item := range policies {
		policy := azureARMObject(item)
		days, ok := policy["retentionPeriodInDays"].(float64)
		if !ok || days < 0 || days > dtaskMaxRetention || days != float64(int64(days)) {
			return "retentionPeriodInDays must be an integer between 0 and 90."
		}
		state := azureARMString(policy["orchestrationState"])
		switch state {
		case "", "Completed", "Failed", "Terminated", "Canceled":
		default:
			return "orchestrationState must be Completed, Failed, Terminated or Canceled."
		}
		if seen[state] {
			return "Each orchestrationState may appear in only one retention policy."
		}
		seen[state] = true
	}
	return ""
}
