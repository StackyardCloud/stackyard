package server

import (
	"net/http"
	"regexp"
	"strings"
)

// azureServiceBusRuleType is shared by the registration and the namespace's
// default RootManageSharedAccessKey rule.
var azureServiceBusRuleType = azureARMResourceType{
	Namespace: "Microsoft.ServiceBus",
	Type:      "namespaces/authorizationRules",
	Proxy:     true,
	Validate:  azureServiceBusValidateRule,
	Actions:   map[string]azureARMAction{"listkeys": azureServiceBusListKeys},
}

// Microsoft.ServiceBus namespaces, queues, topics, subscriptions and
// authorization rules, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.ServiceBus"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "namespaces",
		ReadOnlyProperties: azureServiceBusNamespaceProperties,
		ValidateName:       azureServiceBusValidateNamespaceName,
		Validate:           azureServiceBusValidateNamespace,
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			return []*azureARMResource{azureARMChild(parent, azureServiceBusRuleType, "RootManageSharedAccessKey",
				map[string]any{"rights": []any{"Listen", "Manage", "Send"}})}
		},
	})
	registerAzureARMResourceType(azureServiceBusRuleType)
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/queues", Proxy: true})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/topics", Proxy: true})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "namespaces/topics/subscriptions", Proxy: true})
	registerAzureARMProviderAction(ns, "CheckNameAvailability", azureServiceBusCheckNameAvailability)
}

var azureServiceBusNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{4,48}[a-zA-Z0-9]$`)

func azureServiceBusValidateNamespaceName(name string) string {
	lower := strings.ToLower(name)
	if !azureServiceBusNamePattern.MatchString(name) || strings.HasSuffix(lower, "-sb") || strings.HasSuffix(lower, "-mgmt") {
		return "Namespace names must be 6-50 letters, digits or hyphens, start with a letter, end with a letter or digit, and not end with -sb or -mgmt."
	}
	return ""
}

func azureServiceBusValidateNamespace(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["sku"])["name"]) {
	case "Basic", "Standard", "Premium":
		return ""
	default:
		return "sku.name must be one of Basic, Standard, Premium."
	}
}

// azureServiceBusValidateRule requires Listen/Send/Manage rights, where
// Manage also needs Listen and Send.
func azureServiceBusValidateRule(payload map[string]any) string {
	rights, _ := azureARMObject(payload["properties"])["rights"].([]any)
	granted := map[string]bool{}
	for _, right := range rights {
		value := azureARMString(right)
		if value != "Listen" && value != "Send" && value != "Manage" {
			return "properties.rights may only contain Listen, Send and Manage."
		}
		granted[value] = true
	}
	switch {
	case len(granted) == 0:
		return "properties.rights is required."
	case granted["Manage"] && (!granted["Listen"] || !granted["Send"]):
		return "Manage rights require Listen and Send."
	}
	return ""
}

func azureServiceBusNamespaceProperties(id string, _ map[string]any) map[string]any {
	return map[string]any{
		"serviceBusEndpoint": "https://" + strings.ToLower(id[strings.LastIndex(id, "/")+1:]) + ".servicebus.windows.net:443/",
		"status":             "Active",
	}
}

// azureServiceBusListKeys returns a rule's keys and connection strings. The
// record ID ends in .../namespaces/{namespace}/authorizationRules/{rule}.
func azureServiceBusListKeys(record *azureARMResource, _ map[string]any) (int, any) {
	parts := strings.Split(record.ID, "/")
	host := strings.ToLower(parts[len(parts)-3]) + ".servicebus.windows.net"
	primary, secondary := azureARMDerivedKey(record.ID, "primary"), azureARMDerivedKey(record.ID, "secondary")
	connection := func(key string) string {
		return "Endpoint=sb://" + host + "/;SharedAccessKeyName=" + record.Name + ";SharedAccessKey=" + key
	}
	return http.StatusOK, map[string]any{
		"keyName":                   record.Name,
		"primaryKey":                primary,
		"secondaryKey":              secondary,
		"primaryConnectionString":   connection(primary),
		"secondaryConnectionString": connection(secondary),
	}
}

// azureServiceBusCheckNameAvailability checks a namespace name; names are
// global DNS labels, so it compares across all subscriptions.
func azureServiceBusCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureServiceBusValidateNamespaceName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "InvalidName", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.ServiceBus/namespaces") && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "NameInUse", "message": "The specified service namespace is not available."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "reason": "None", "message": ""}
}
