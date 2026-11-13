package server

import (
	"net/http"
	"regexp"
	"strings"
	"time"
)

const azureDigitalTwinsType = "Microsoft.DigitalTwins/digitalTwinsInstances"

// Microsoft.DigitalTwins instances and their endpoints, served by the shared
// ARM handler.
func init() {
	const ns = "Microsoft.DigitalTwins"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "digitalTwinsInstances",
		ReadOnlyProperties: azureDigitalTwinsProperties,
		ValidateName:       azureDigitalTwinsValidateName,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns,
		Type:      "digitalTwinsInstances/endpoints",
		Proxy:     true,
		Validate:  azureDigitalTwinsValidateEndpoint,
		// Endpoint secrets are accepted but never returned.
		WriteOnlyProperties: []string{
			"connectionStringPrimaryKey", "connectionStringSecondaryKey",
			"primaryConnectionString", "secondaryConnectionString",
			"accessKey1", "accessKey2",
		},
	})
	registerAzureARMProviderAction(ns, "locations/checkNameAvailability", azureDigitalTwinsCheckNameAvailability)
}

var azureDigitalTwinsNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]$`)

// azureDigitalTwinsEndpointKeys names the key each endpoint type needs for
// key-based authentication.
var azureDigitalTwinsEndpointKeys = map[string]string{
	"EventHub":   "connectionStringPrimaryKey",
	"ServiceBus": "primaryConnectionString",
	"EventGrid":  "accessKey1",
}

func azureDigitalTwinsValidateName(name string) string {
	if !azureDigitalTwinsNamePattern.MatchString(name) {
		return "Instance names must be 3-63 letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureDigitalTwinsValidateEndpoint(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	endpointType := azureARMString(props["endpointType"])
	key, known := azureDigitalTwinsEndpointKeys[endpointType]
	if !known {
		return "properties.endpointType must be EventHub, EventGrid or ServiceBus."
	}
	switch azureARMString(props["authenticationType"]) {
	case "", "KeyBased":
		if azureARMString(props[key]) == "" {
			return "properties." + key + " is required for key-based " + endpointType + " endpoints."
		}
		if endpointType == "EventGrid" && azureARMString(props["TopicEndpoint"]) == "" {
			return "properties.TopicEndpoint is required for EventGrid endpoints."
		}
	case "IdentityBased":
		if endpointType == "EventGrid" {
			return "EventGrid endpoints only support key-based authentication."
		}
		if azureARMString(props["endpointUri"]) == "" || azureARMString(props["entityPath"]) == "" {
			return "properties.endpointUri and properties.entityPath are required for identity-based endpoints."
		}
	default:
		return "properties.authenticationType must be KeyBased or IdentityBased."
	}
	return ""
}

// azureDigitalTwinsProperties sets the API host name and timestamps.
// ponytail: the host uses the location name; Azure uses a short region code
// (wus2), so add a code table if clients parse it.
func azureDigitalTwinsProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"hostName":        strings.ToLower(rec.Name) + ".api." + strings.ToLower(rec.Location) + ".digitaltwins.azure.net",
		"createdTime":     rec.CreatedAt.Format(time.RFC3339),
		"lastUpdatedTime": rec.UpdatedAt.Format(time.RFC3339),
	}
}

// azureDigitalTwinsCheckNameAvailability checks an instance name; host names
// are global, so it compares across all subscriptions and regions.
func azureDigitalTwinsCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureDigitalTwinsValidateName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureDigitalTwinsType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name '" + name + "' is already taken."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}
