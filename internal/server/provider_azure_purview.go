package server

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const azurePurviewAccountType = "Microsoft.Purview/accounts"

// Microsoft.Purview accounts, served by the shared ARM handler. The catalog,
// scanning and Atlas Kafka endpoints are reported but not served.
func init() {
	const ns = "Microsoft.Purview"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "accounts",
		ReadOnlyProperties: azurePurviewAccountProperties,
		ValidateName:       azurePurviewValidateName,
		Validate:           azurePurviewValidateAccount,
		Actions: map[string]azureARMAction{
			"listkeys": func(rec *azureARMResource, _ map[string]any) (int, any) {
				endpoint := "Endpoint=sb://atlas-" + azurePurviewAccountID(rec) + ".servicebus.windows.net/;SharedAccessKeyName=AlternateSharedAccessKey;SharedAccessKey="
				return http.StatusOK, map[string]any{
					"atlasKafkaPrimaryEndpoint":   endpoint + azureARMDerivedKey(rec.ID, "purview-kafka-primary"),
					"atlasKafkaSecondaryEndpoint": endpoint + azureARMDerivedKey(rec.ID, "purview-kafka-secondary"),
				}
			},
			"addrootcollectionadmin": func(_ *azureARMResource, payload map[string]any) (int, any) {
				if strings.TrimSpace(azureARMString(payload["objectId"])) == "" {
					return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "objectId is required.")
				}
				return http.StatusOK, nil
			},
		},
	})
	registerAzureARMProviderAction(ns, "checkNameAvailability", func(s *Server, payload map[string]any) (int, any) {
		return s.azureIoTNameAvailability(payload, azurePurviewAccountType, azurePurviewValidateName)
	})
}

var azurePurviewNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]$`)

func azurePurviewValidateName(name string) string {
	if !azurePurviewNamePattern.MatchString(name) {
		return "Account names must be 3-63 letters, digits or hyphens, and start and end with a letter or digit."
	}
	return ""
}

func azurePurviewValidateAccount(payload map[string]any) string {
	if identity, present := payload["identity"]; present {
		if kind := azureARMString(azureARMObject(identity)["type"]); kind != "SystemAssigned" && kind != "UserAssigned" && kind != "None" {
			return "identity.type must be SystemAssigned, UserAssigned or None."
		}
	}
	if msg := azureEnumProperty("publicNetworkAccess", "Enabled", "Disabled", "NotSpecified")(payload); msg != "" {
		return msg
	}
	return azureEnumProperty("managedResourcesPublicNetworkAccess", "Enabled", "Disabled", "NotSpecified")(payload)
}

func azurePurviewAccountID(rec *azureARMResource) string {
	return strings.ReplaceAll(uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(), "-", "")
}

// azurePurviewAccountProperties reports the account's endpoints and its
// managed resources, which Stackyard does not create.
func azurePurviewAccountProperties(rec *azureARMResource) map[string]any {
	name := strings.ToLower(rec.Name)
	managedGroup := azureARMString(rec.Properties["managedResourceGroupName"])
	if managedGroup == "" {
		managedGroup = "managed-rg-" + name
	}
	managedPrefix := "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + managedGroup + "/providers/"
	props := map[string]any{
		"accountStatus":            map[string]any{"accountProvisioningState": "Succeeded"},
		"friendlyName":             rec.Name,
		"createdAt":                rec.CreatedAt.Format(time.RFC3339),
		"managedResourceGroupName": managedGroup,
		"endpoints": map[string]any{
			"catalog":  "https://" + name + ".purview.azure.com/catalog",
			"guardian": "https://" + name + ".purview.azure.com/guardian",
			"scan":     "https://" + name + ".purview.azure.com/scan",
		},
		"managedResources": map[string]any{
			"resourceGroup":     "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + managedGroup,
			"storageAccount":    managedPrefix + "Microsoft.Storage/storageAccounts/scan" + azurePurviewAccountID(rec)[:16],
			"eventHubNamespace": managedPrefix + "Microsoft.EventHub/namespaces/atlas-" + azurePurviewAccountID(rec),
		},
	}
	if _, present := rec.Properties["publicNetworkAccess"]; !present {
		props["publicNetworkAccess"] = "Enabled"
	}
	return props
}
