package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Microsoft.Devices IoT Hub Device Provisioning Service instances, served by
// the shared ARM handler. Keys and the name check follow IoT Hub
// (provider_azure_iothub.go).
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          "Microsoft.Devices",
		Type:               "provisioningServices",
		ReadOnlyProperties: azureDPSProperties,
		ValidateName:       azureDPSValidateName,
		Validate:           azureDPSValidate,
		// Keys are only returned by listkeys.
		WriteOnlyProperties: []string{"authorizationPolicies"},
		Actions: map[string]azureARMAction{"listkeys": func(record *azureARMResource, _ map[string]any) (int, any) {
			return http.StatusOK, azureIoTPolicyKeys(record, []azureIoTPolicy{{"provisioningserviceowner", "ServiceConfig, DeviceConnect, EnrollmentWrite"}})
		}},
	})
	registerAzureARMProviderAction("Microsoft.Devices", "checkProvisioningServiceNameAvailability", func(s *Server, payload map[string]any) (int, any) {
		return s.azureIoTNameAvailability(payload, "Microsoft.Devices/provisioningServices", azureDPSValidateName)
	})
}

var (
	azureDPSNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,62}[a-zA-Z0-9]$`)
	// azureDPSKeyPattern matches the key in a linked hub's connection string.
	azureDPSKeyPattern  = regexp.MustCompile(`(?i)(SharedAccessKey=)[^;]*`)
	azureDPSHostPattern = regexp.MustCompile(`(?i)HostName=([^;]+)`)
)

func azureDPSValidateName(name string) string {
	if !azureDPSNamePattern.MatchString(name) {
		return "Provisioning service names must be 3-64 letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureDPSValidate(payload map[string]any) string {
	if azureARMString(azureARMObject(payload["sku"])["name"]) != "S1" {
		return "sku.name must be S1."
	}
	props := azureARMObject(payload["properties"])
	if policy := azureARMString(props["allocationPolicy"]); policy != "" && !slices.Contains([]string{"Hashed", "GeoLatency", "Static"}, policy) {
		return "properties.allocationPolicy must be Hashed, GeoLatency or Static."
	}
	hubs, _ := props["iotHubs"].([]any)
	for _, raw := range hubs {
		hub := azureARMObject(raw)
		if !azureDPSHostPattern.MatchString(azureARMString(hub["connectionString"])) || azureARMString(hub["location"]) == "" {
			return "each properties.iotHubs entry needs a connectionString with a HostName and a location."
		}
	}
	return ""
}

// azureDPSProperties sets the host names, a stable ID scope and the default
// allocation policy, and masks the keys in linked hubs' connection strings
// as Azure does.
func azureDPSProperties(rec *azureARMResource) map[string]any {
	sum := sha256.Sum256([]byte(azureARMKey(rec.ID) + "#idScope"))
	props := map[string]any{
		"serviceOperationsHostName":  strings.ToLower(rec.Name) + ".azure-devices-provisioning.net",
		"deviceProvisioningHostName": "global.azure-devices-provisioning.net",
		"idScope":                    "0ne" + strings.ToUpper(hex.EncodeToString(sum[:4])),
		"state":                      "Active",
	}
	if azureARMString(rec.Properties["allocationPolicy"]) == "" {
		props["allocationPolicy"] = "Hashed"
	}
	if hubs, ok := rec.Properties["iotHubs"].([]any); ok {
		masked := make([]any, 0, len(hubs))
		for _, raw := range hubs {
			hub := azureARMMerge(azureARMObject(raw), nil)
			connection := azureARMString(hub["connectionString"])
			hub["connectionString"] = azureDPSKeyPattern.ReplaceAllString(connection, "${1}****")
			if m := azureDPSHostPattern.FindStringSubmatch(connection); m != nil {
				hub["name"] = m[1]
			}
			masked = append(masked, hub)
		}
		props["iotHubs"] = masked
	}
	return props
}
