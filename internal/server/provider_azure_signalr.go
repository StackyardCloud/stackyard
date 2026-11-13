package server

import (
	"net/http"
	"strings"
)

const azureSignalRType = "Microsoft.SignalRService/SignalR"

// Microsoft.SignalRService/SignalR services and replicas, served by the
// shared ARM handler. Web PubSub keeps its own handler in the same
// namespace, which also serves the shared location-scoped name check.
func init() {
	const ns = "Microsoft.SignalRService"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "SignalR",
		ReadOnlyProperties: azureSignalRProperties,
		ValidateName:       azureSignalRValidateName,
		Validate:           azureSignalRValidate,
		Actions:            map[string]azureARMAction{"listkeys": azureSignalRListKeys},
	})
	// Replicas are tracked resources in their own region.
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "SignalR/replicas", Validate: azureSignalRValidate})
}

func azureSignalRValidateName(name string) string {
	if !azureWebPubSubValidName(name) {
		return "Resource names must be 3-63 characters, start with a letter, and end with a letter or digit."
	}
	return ""
}

func azureSignalRValidate(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["sku"])["name"]) {
	case "Free_F1", "Standard_S1", "Premium_P1", "Premium_P2":
	default:
		return "sku.name must be one of Free_F1, Standard_S1, Premium_P1, Premium_P2."
	}
	switch azureARMString(payload["kind"]) {
	case "", "SignalR", "RawWebSockets":
		return ""
	default:
		return "kind must be SignalR or RawWebSockets."
	}
}

func azureSignalRHostName(rec *azureARMResource) string {
	return strings.ToLower(rec.Name) + ".service.signalr.net"
}

func azureSignalRProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"hostName":       azureSignalRHostName(rec),
		"hostNamePrefix": strings.ToLower(rec.Name),
		"publicPort":     443,
		"serverPort":     443,
		"version":        "1.0",
	}
}

func azureSignalRListKeys(record *azureARMResource, _ map[string]any) (int, any) {
	primary, secondary := azureARMDerivedKey(record.ID, "primary"), azureARMDerivedKey(record.ID, "secondary")
	connection := func(key string) string {
		return "Endpoint=https://" + azureSignalRHostName(record) + ";AccessKey=" + key + ";Version=1.0;"
	}
	return http.StatusOK, map[string]any{
		"primaryKey":                primary,
		"secondaryKey":              secondary,
		"primaryConnectionString":   connection(primary),
		"secondaryConnectionString": connection(secondary),
	}
}

// azureSignalRNameAvailability checks a service name; host names are global,
// so it compares across all subscriptions and regions.
func (s *Server) azureSignalRNameAvailability(name string) map[string]any {
	if msg := azureSignalRValidateName(name); msg != "" {
		return map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureSignalRType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name is already taken. Please try a different name."}
	}
	return map[string]any{"nameAvailable": true}
}
