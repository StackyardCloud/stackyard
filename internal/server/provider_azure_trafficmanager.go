package server

import (
	"net/http"
	"strings"
)

// Microsoft.Network Traffic Manager profiles and endpoints, served by the
// shared ARM handler. Other Microsoft.Network types keep their own handlers;
// the shared router only claims registered type paths.
func init() {
	const ns = "Microsoft.Network"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "trafficManagerProfiles",
		ReadOnlyProperties: azureTrafficManagerProfileProperties,
		Validate:           azureTrafficManagerValidateProfile,
	})
	for _, endpoint := range []struct{ typ, field string }{
		{"externalEndpoints", "target"},
		{"azureEndpoints", "targetResourceId"},
		{"nestedEndpoints", "targetResourceId"},
	} {
		field := endpoint.field
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns,
			Type:      "trafficManagerProfiles/" + endpoint.typ,
			Proxy:     true,
			Validate: func(payload map[string]any) string {
				if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])[field])) == "" {
					return "properties." + field + " is required."
				}
				return ""
			},
		})
	}
	registerAzureARMProviderAction(ns, "checkTrafficManagerNameAvailabilityV2", azureTrafficManagerCheckName)
	registerAzureARMProviderAction(ns, "checkTrafficManagerNameAvailability", azureTrafficManagerCheckName)
}

var azureTrafficManagerRoutingMethods = map[string]bool{
	"Performance": true, "Priority": true, "Weighted": true, "Geographic": true, "MultiValue": true, "Subnet": true,
}

func azureTrafficManagerValidateProfile(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch {
	case !strings.EqualFold(azureARMString(payload["location"]), "global"):
		return "Traffic Manager profiles must use location 'global'."
	case !azureTrafficManagerRoutingMethods[azureARMString(props["trafficRoutingMethod"])]:
		return "properties.trafficRoutingMethod must be one of Performance, Priority, Weighted, Geographic, MultiValue, Subnet."
	case strings.TrimSpace(azureARMString(azureARMObject(props["dnsConfig"])["relativeName"])) == "":
		return "properties.dnsConfig.relativeName is required."
	}
	return ""
}

// azureTrafficManagerProfileProperties derives the profile FQDN from its
// relative DNS name.
func azureTrafficManagerProfileProperties(_ string, properties map[string]any) map[string]any {
	relativeName := azureARMString(azureARMObject(properties["dnsConfig"])["relativeName"])
	return map[string]any{"dnsConfig": map[string]any{"fqdn": strings.ToLower(relativeName) + ".trafficmanager.net"}}
}

// azureTrafficManagerCheckName checks a relative DNS name, which is unique
// across all profiles in every subscription.
func azureTrafficManagerCheckName(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	result := map[string]any{"name": name, "type": azureARMString(payload["type"]), "nameAvailable": true}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, "Microsoft.Network/trafficManagerProfiles") &&
			strings.EqualFold(azureARMString(azureARMObject(rec.Properties["dnsConfig"])["relativeName"]), name)
	})) > 0 {
		result["nameAvailable"] = false
		result["reason"] = "AlreadyExists"
		result["message"] = "Domain name " + strings.ToLower(name) + ".trafficmanager.net already exists. Please choose a different DNS prefix."
	}
	return http.StatusOK, result
}
