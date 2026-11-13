package server

import (
	"net/http"
	"strings"
)

// Microsoft.Relay namespaces, hybrid connections, WCF relays and their
// authorization rules, served by the shared ARM handler. Naming, rules and
// keys are shared with the Service Bus family (azure_sas_namespaces.go).
func init() {
	const ns = "Microsoft.Relay"
	ruleType := azureSASRuleType(ns, "namespaces/authorizationRules", false)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "namespaces",
		ReadOnlyProperties: azureSASNamespaceProperties,
		ValidateName:       azureSASValidateNamespaceName,
		Validate: func(payload map[string]any) string {
			if azureARMString(azureARMObject(payload["sku"])["name"]) != "Standard" {
				return "sku.name must be Standard."
			}
			return ""
		},
		DefaultChildren: azureSASRootRule(ruleType),
	})
	registerAzureARMResourceType(ruleType)
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/hybridConnections", Proxy: true})
	registerAzureARMResourceType(azureSASRuleType(ns, "namespaces/hybridConnections/authorizationRules", true))
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns,
		Type:      "namespaces/wcfRelays",
		Proxy:     true,
		Validate: func(payload map[string]any) string {
			if relayType := azureARMString(azureARMObject(payload["properties"])["relayType"]); relayType != "NetTcp" && relayType != "Http" {
				return "properties.relayType must be NetTcp or Http."
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureSASRuleType(ns, "namespaces/wcfRelays/authorizationRules", true))
	registerAzureARMProviderAction(ns, "checkNameAvailability", azureRelayCheckNameAvailability)
}

func azureRelayCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
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
