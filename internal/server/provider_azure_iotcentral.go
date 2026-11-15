package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

const azureIoTCentralAppType = "Microsoft.IoTCentral/iotApps"

// Microsoft.IoTCentral applications, served by the shared ARM handler.
// Applications report created at once; no IoT Central app is served at the
// subdomain.
func init() {
	const ns = "Microsoft.IoTCentral"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "iotApps",
		ReadOnlyProperties: azureIoTCentralAppProperties,
		ValidateName:       azureIoTCentralValidateName,
		Validate:           azureIoTCentralValidateApp,
	})
	registerAzureARMProviderAction(ns, "checkNameAvailability", func(s *Server, payload map[string]any) (int, any) {
		return s.azureIoTNameAvailability(payload, azureIoTCentralAppType, azureIoTCentralValidateName)
	})
	registerAzureARMProviderAction(ns, "checkSubdomainAvailability", azureIoTCentralSubdomainAvailability)
}

// Names and subdomains are both DNS labels.
var azureIoTCentralNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])$`)

func azureIoTCentralValidateName(name string) string {
	if !azureIoTCentralNamePattern.MatchString(name) {
		return "Names must be 2-63 lowercase letters, digits or hyphens, and start and end with a letter or digit."
	}
	return ""
}

func azureIoTCentralValidateApp(payload map[string]any) string {
	if !slices.Contains([]string{"ST0", "ST1", "ST2"}, azureARMString(azureARMObject(payload["sku"])["name"])) {
		return "sku.name must be ST0, ST1 or ST2."
	}
	if identity, present := payload["identity"]; present {
		if kind := azureARMString(azureARMObject(identity)["type"]); kind != "SystemAssigned" && kind != "None" {
			return "identity.type must be SystemAssigned or None."
		}
	}
	if msg := azureIoTCentralValidateName(azureARMString(azureARMObject(payload["properties"])["subdomain"])); msg != "" {
		return "properties.subdomain is invalid: " + msg
	}
	return azureEnumProperty("publicNetworkAccess", "Enabled", "Disabled")(payload)
}

// azureIoTCentralAppProperties sets the application ID and state, and
// defaults the display name and template.
func azureIoTCentralAppProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"applicationId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"state":         "created",
	}
	for key, value := range map[string]string{"displayName": rec.Name, "template": "iotc-pnp-preview@1.0.0", "publicNetworkAccess": "Enabled"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

// azureIoTCentralSubdomainAvailability checks a subdomain across all
// subscriptions, because it is the application's DNS name.
func azureIoTCentralSubdomainAvailability(s *Server, payload map[string]any) (int, any) {
	subdomain := strings.TrimSpace(azureARMString(payload["name"]))
	if subdomain == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureIoTCentralValidateName(subdomain); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureIoTCentralAppType) && strings.EqualFold(azureARMString(rec.Properties["subdomain"]), subdomain)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The subdomain '" + subdomain + "' is already taken."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true, "message": ""}
}
