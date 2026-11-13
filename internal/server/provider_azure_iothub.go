package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const azureIoTHubType = "Microsoft.Devices/IotHubs"

// Microsoft.Devices IoT hubs and the consumer groups of their built-in
// Event Hub-compatible endpoint, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.Devices"
	endpoints := azureARMResourceType{
		Namespace: ns,
		Type:      "IotHubs/eventHubEndpoints",
		Proxy:     true,
		ValidateName: func(name string) string {
			if !strings.EqualFold(name, "events") {
				return "The only Event Hub-compatible endpoint is 'events'."
			}
			return ""
		},
	}
	consumerGroups := azureARMResourceType{Namespace: ns, Type: "IotHubs/eventHubEndpoints/ConsumerGroups", Proxy: true, Validate: azureIoTHubValidateConsumerGroup}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "IotHubs",
		ReadOnlyProperties: azureIoTHubProperties,
		ValidateName:       azureIoTHubValidateName,
		Validate:           azureIoTHubValidate,
		// Keys are only returned by listkeys.
		WriteOnlyProperties: []string{"authorizationPolicies"},
		Actions:             map[string]azureARMAction{"listkeys": azureIoTHubListKeys},
		// The built-in endpoint and its $Default consumer group always exist.
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			events := azureARMChild(parent, endpoints, "events", map[string]any{})
			return []*azureARMResource{events, azureARMChild(events, consumerGroups, "$Default", map[string]any{"name": "$Default"})}
		},
	})
	registerAzureARMResourceType(endpoints)
	registerAzureARMResourceType(consumerGroups)
	registerAzureARMProviderAction(ns, "checkNameAvailability", azureIoTHubCheckNameAvailability)
}

var (
	azureIoTHubNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,48}[a-zA-Z0-9]$`)
	azureIoTHubSKUs        = []string{"F1", "S1", "S2", "S3", "B1", "B2", "B3"}
	// azureIoTHubPolicies are the shared access policies every hub starts with.
	azureIoTHubPolicies = []azureIoTPolicy{
		{"iothubowner", "RegistryWrite, ServiceConnect, DeviceConnect"},
		{"service", "ServiceConnect"},
		{"device", "DeviceConnect"},
		{"registryRead", "RegistryRead"},
		{"registryReadWrite", "RegistryWrite"},
	}
)

type azureIoTPolicy struct{ keyName, rights string }

func azureIoTHubValidateName(name string) string {
	if !azureIoTHubNamePattern.MatchString(name) {
		return "IoT hub names must be 3-50 letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureIoTHubValidate(payload map[string]any) string {
	if !slices.Contains(azureIoTHubSKUs, azureARMString(azureARMObject(payload["sku"])["name"])) {
		return "sku.name must be one of " + strings.Join(azureIoTHubSKUs, ", ") + "."
	}
	events := azureARMObject(azureARMObject(azureARMObject(payload["properties"])["eventHubEndpoints"])["events"])
	if raw, present := events["partitionCount"]; present {
		if n, ok := raw.(float64); !ok || n < 2 || n > 128 || n != float64(int(n)) {
			return "properties.eventHubEndpoints.events.partitionCount must be an integer from 2 to 128."
		}
	}
	if raw, present := events["retentionTimeInDays"]; present {
		if n, ok := raw.(float64); !ok || n < 1 || n > 7 || n != float64(int(n)) {
			return "properties.eventHubEndpoints.events.retentionTimeInDays must be an integer from 1 to 7."
		}
	}
	return ""
}

func azureIoTHubValidateConsumerGroup(payload map[string]any) string {
	if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])["name"])) == "" {
		return "properties.name is required."
	}
	return ""
}

// azureIoTHubProperties sets the host name and the built-in endpoint's
// partitions, path and namespace, keeping the client's partition count
// and retention.
func azureIoTHubProperties(rec *azureARMResource) map[string]any {
	events := azureARMObject(azureARMObject(rec.Properties["eventHubEndpoints"])["events"])
	// PATCH skips Validate, so only a valid count sizes partitionIds.
	partitions := 4
	if n, ok := events["partitionCount"].(float64); ok && n >= 2 && n <= 128 && n == float64(int(n)) {
		partitions = int(n)
	}
	ids := make([]any, partitions)
	for i := range ids {
		ids[i] = strconv.Itoa(i)
	}
	sum := sha256.Sum256([]byte(azureARMKey(rec.ID)))
	builtIn := map[string]any{
		"partitionCount": partitions,
		"partitionIds":   ids,
		"path":           strings.ToLower(rec.Name),
		"endpoint":       "sb://iothub-ns-" + strings.ToLower(rec.Name) + "-" + hex.EncodeToString(sum[:4]) + ".servicebus.windows.net/",
	}
	if _, present := events["retentionTimeInDays"]; !present {
		builtIn["retentionTimeInDays"] = 1
	}
	return map[string]any{
		"hostName":          strings.ToLower(rec.Name) + ".azure-devices.net",
		"state":             "Active",
		"eventHubEndpoints": map[string]any{"events": builtIn},
	}
}

// azureIoTPolicyKeys lists policies with keys derived from the resource ID,
// in the SharedAccessSignatureAuthorizationRule shape IoT Hub and DPS share.
func azureIoTPolicyKeys(record *azureARMResource, policies []azureIoTPolicy) map[string]any {
	value := make([]any, 0, len(policies))
	for _, p := range policies {
		value = append(value, map[string]any{
			"keyName":      p.keyName,
			"primaryKey":   azureARMDerivedKey(record.ID, p.keyName+"#primary"),
			"secondaryKey": azureARMDerivedKey(record.ID, p.keyName+"#secondary"),
			"rights":       p.rights,
		})
	}
	return map[string]any{"value": value}
}

func azureIoTHubListKeys(record *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, azureIoTPolicyKeys(record, azureIoTHubPolicies)
}

// azureIoTNameAvailability checks a Microsoft.Devices name; host names are
// global, so it compares across all subscriptions and regions.
func (s *Server) azureIoTNameAvailability(payload map[string]any, fullType string, validate func(string) string) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := validate(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, fullType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name '" + name + "' is already taken."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "message": ""}
}

func azureIoTHubCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
	return s.azureIoTNameAvailability(payload, azureIoTHubType, azureIoTHubValidateName)
}
