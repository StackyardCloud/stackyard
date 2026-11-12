package server

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"regexp"
	"strings"
)

// Microsoft.ContainerRegistry registries and their child resources, served by
// the shared ARM handler.
func init() {
	const ns = "Microsoft.ContainerRegistry"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "registries",
		ReadOnlyProperties: azureContainerRegistryProperties,
		ValidateName:       azureContainerRegistryValidateName,
		Validate:           azureContainerRegistryValidate,
		Actions: map[string]azureARMAction{
			"listcredentials": azureContainerRegistryListCredentials,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "registries/replications"})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "registries/webhooks", Validate: azureContainerRegistryValidateWebhook})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "registries/scopeMaps", Proxy: true, Validate: azureContainerRegistryValidateScopeMap})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "registries/tokens", Proxy: true})
}

var azureContainerRegistryNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]{5,50}$`)

func azureContainerRegistryValidateName(name string) string {
	if !azureContainerRegistryNamePattern.MatchString(name) {
		return "Registry names may contain only alpha numeric characters and must be between 5 and 50 characters."
	}
	return ""
}

func azureContainerRegistryValidate(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["sku"])["name"]) {
	case "Basic", "Standard", "Premium":
		return ""
	default:
		return "sku.name must be one of Basic, Standard, Premium."
	}
}

func azureContainerRegistryValidateWebhook(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	actions, _ := props["actions"].([]any)
	if strings.TrimSpace(azureARMString(props["serviceUri"])) == "" || len(actions) == 0 {
		return "properties.serviceUri and properties.actions are required."
	}
	return ""
}

func azureContainerRegistryValidateScopeMap(payload map[string]any) string {
	if actions, _ := azureARMObject(payload["properties"])["actions"].([]any); len(actions) == 0 {
		return "properties.actions is required."
	}
	return ""
}

func azureContainerRegistryProperties(id string) map[string]any {
	return map[string]any{"loginServer": strings.ToLower(id[strings.LastIndex(id, "/")+1:]) + ".azurecr.io"}
}

// azureContainerRegistryListCredentials returns admin credentials, which ACR
// only issues while adminUserEnabled is true. Passwords derive from the
// resource ID so they are stable across calls and restarts.
func azureContainerRegistryListCredentials(record *azureARMResource, _ map[string]any) (int, any) {
	if enabled, _ := record.Properties["adminUserEnabled"].(bool); !enabled {
		return http.StatusBadRequest, map[string]any{"error": map[string]any{
			"code":    "AdminUserDisabled",
			"message": "Admin user is disabled for registry '" + record.Name + "'.",
		}}
	}
	password := func(label string) map[string]any {
		sum := sha256.Sum256([]byte(azureARMKey(record.ID) + "#" + label))
		return map[string]any{"name": label, "value": base64.RawStdEncoding.EncodeToString(sum[:24])}
	}
	return http.StatusOK, map[string]any{
		"username":  record.Name,
		"passwords": []any{password("password"), password("password2")},
	}
}
