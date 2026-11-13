package server

import (
	"net/http"
	"strings"
)

// Microsoft.NotificationHubs namespaces, notification hubs and their
// authorization rules, served by the shared ARM handler. Naming, rules and
// keys are shared with the Service Bus family (azure_sas_namespaces.go).
func init() {
	const ns = "Microsoft.NotificationHubs"
	ruleType := azureSASRuleType(ns, "namespaces/authorizationRules", false)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns,
		Type:      "namespaces",
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := azureSASNamespaceProperties(rec)
			if azureARMString(rec.Properties["namespaceType"]) == "" {
				props["namespaceType"] = "NotificationHub"
			}
			return props
		},
		ValidateName:    azureSASValidateNamespaceName,
		Validate:        azureNotificationHubsValidateNamespace,
		DefaultChildren: azureSASRootRule(ruleType),
	})
	registerAzureARMResourceType(ruleType)
	// Notification hubs are tracked resources with their own location.
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/notificationHubs"})
	// Hub connection strings never carry EntityPath; clients pass the hub name separately.
	registerAzureARMResourceType(azureSASRuleType(ns, "namespaces/notificationHubs/authorizationRules", false))
	registerAzureARMProviderAction(ns, "checkNamespaceAvailability", azureNotificationHubsCheckNamespaceAvailability)
}

func azureNotificationHubsValidateNamespace(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["sku"])["name"]) {
	case "Free", "Basic", "Standard":
	default:
		return "sku.name must be one of Free, Basic, Standard."
	}
	switch azureARMString(azureARMObject(payload["properties"])["namespaceType"]) {
	case "", "NotificationHub", "Messaging":
		return ""
	default:
		return "properties.namespaceType must be NotificationHub or Messaging."
	}
}

// azureNotificationHubsCheckNamespaceAvailability answers with the API's own
// spelling, "isAvailiable", which SDKs deserialize.
func azureNotificationHubsCheckNamespaceAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	available := azureSASValidateNamespaceName(name) == "" && !s.azureSASNamespaceNameTaken(name)
	return http.StatusOK, map[string]any{
		"name":         name,
		"type":         "Microsoft.NotificationHubs/namespaces/checkNamespaceAvailability",
		"isAvailiable": available,
	}
}
