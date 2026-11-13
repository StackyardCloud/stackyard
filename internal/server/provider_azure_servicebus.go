package server

import (
	"net/http"
	"strings"
)

// Microsoft.ServiceBus namespaces, queues, topics, subscriptions and
// authorization rules, served by the shared ARM handler. Namespace naming,
// rules and keys are shared with Relay and Notification Hubs
// (azure_sas_namespaces.go).
func init() {
	const ns = "Microsoft.ServiceBus"
	ruleType := azureSASRuleType(ns, "namespaces/authorizationRules", false)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "namespaces",
		ReadOnlyProperties: azureSASNamespaceProperties,
		ValidateName:       azureSASValidateNamespaceName,
		Validate:           azureServiceBusValidateNamespace,
		DefaultChildren:    azureSASRootRule(ruleType),
	})
	registerAzureARMResourceType(ruleType)
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/queues", Proxy: true})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/topics", Proxy: true})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/topics/subscriptions", Proxy: true})
	registerAzureARMProviderAction(ns, "CheckNameAvailability", azureServiceBusCheckNameAvailability)
}

func azureServiceBusValidateNamespace(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["sku"])["name"]) {
	case "Basic", "Standard", "Premium":
		return ""
	default:
		return "sku.name must be one of Basic, Standard, Premium."
	}
}

// azureServiceBusCheckNameAvailability checks a namespace name; names are
// global DNS labels shared with Relay and Notification Hubs namespaces.
func azureServiceBusCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureSASValidateNamespaceName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "InvalidName", "message": msg}
	}
	if s.azureSASNamespaceNameTaken(name) {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "NameInUse", "message": "The specified service namespace is not available."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "reason": "None", "message": ""}
}
