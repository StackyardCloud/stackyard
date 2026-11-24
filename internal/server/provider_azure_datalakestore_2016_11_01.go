package server

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Azure Data Lake Storage Gen1 (Microsoft.DataLakeStore 2016-11-01): accounts
// with their firewall rules, trusted identity providers and virtual network
// rules, served by the shared ARM handler, plus name availability, the
// regional capability and usage, and the operations list. No file system
// data plane runs.

const (
	azureDataLakeStore20161101APIVersion = "2016-11-01"
	azureDLSNamespace                    = "Microsoft.DataLakeStore"
	azureDLSAccountType                  = azureDLSNamespace + "/accounts"
	azureDLSMaxAccounts                  = 150
)

var azureDLSTiers = []string{"Consumption", "Commitment_1TB", "Commitment_10TB", "Commitment_100TB", "Commitment_500TB", "Commitment_1PB", "Commitment_5PB"}

func init() {
	ns := azureDLSNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts",
		ValidateName:       azureDataPlatAccountNameError,
		Validate:           azureDLSAccountValidate,
		ReadOnlyProperties: azureDLSAccountReadOnly,
		InlineChildren: map[string]string{
			"firewallRules":       "accounts/firewallRules",
			"trustedIdProviders":  "accounts/trustedIdProviders",
			"virtualNetworkRules": "accounts/virtualNetworkRules",
		},
		Transitions: map[string]map[string]any{"enablekeyvault": {"encryptionState": "Enabled"}},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/firewallRules", CreateStatus: http.StatusOK, Proxy: true,
		Validate: azureDataPlatIPv4Range,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/trustedIdProviders", CreateStatus: http.StatusOK, Proxy: true,
		Validate: azureDLSTrustedIDProviderValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/virtualNetworkRules", CreateStatus: http.StatusOK, Proxy: true,
		Validate: azureDLSVirtualNetworkRuleValidate,
	})
	registerAzureARMLocationAction(ns, "checkNameAvailability", azureDataPlatCheckName(azureDLSAccountType, azureDataPlatAccountNameError))
	registerAzureARMView(ns, "locations/usages", func(s *Server, scope string) []map[string]any {
		return []map[string]any{azureDataPlatAccountUsage(s, scope, azureDLSAccountType, azureDLSMaxAccounts)}
	})
	registerAzureARMView(ns, "operations", azureDataPlatOperations(ns, "Microsoft Data Lake Store",
		"accounts/read", "accounts/write", "accounts/delete", "accounts/enableKeyVault/action",
		"accounts/firewallRules/read", "accounts/firewallRules/write", "accounts/firewallRules/delete",
		"accounts/trustedIdProviders/read", "accounts/trustedIdProviders/write", "accounts/trustedIdProviders/delete",
		"accounts/virtualNetworkRules/read", "accounts/virtualNetworkRules/write", "accounts/virtualNetworkRules/delete",
		"locations/capability/read", "locations/usages/read", "locations/checkNameAvailability/action"))
}

func azureDLSAccountValidate(payload map[string]any) string {
	if msg := azureDataPlatAll(
		azureDataPlatEnum("encryptionState", "Enabled", "Disabled"),
		azureDataPlatEnum("encryptionConfig.type", "UserManaged", "ServiceManaged"),
		azureDataPlatEnum("firewallState", "Enabled", "Disabled"),
		azureDataPlatEnum("firewallAllowAzureIps", "Enabled", "Disabled"),
		azureDataPlatEnum("trustedIdProviderState", "Enabled", "Disabled"),
		azureDataPlatEnum("newTier", azureDLSTiers...),
	)(payload); msg != "" {
		return msg
	}
	config := azureARMObject(azureARMObject(payload["properties"])["encryptionConfig"])
	if azureARMString(config["type"]) == "UserManaged" {
		vault := azureARMObject(config["keyVaultMetaInfo"])
		for _, key := range []string{"keyVaultResourceId", "encryptionKeyName", "encryptionKeyVersion"} {
			if azureDataPlatEmpty(vault[key]) {
				return "properties.encryptionConfig.keyVaultMetaInfo." + key + " is required for UserManaged encryption."
			}
		}
	}
	return ""
}

// azureDLSAccountReadOnly fills in the server-owned account fields: its
// endpoint, ID and tier, with encryption on by default as for new accounts.
func azureDLSAccountReadOnly(rec *azureARMResource) map[string]any {
	tier := azureARMString(rec.Properties["newTier"])
	if tier == "" {
		tier = "Consumption"
	}
	props := map[string]any{
		"accountId":                   azureDataPlatGUID(rec.ID),
		"endpoint":                    rec.Name + ".azuredatalakestore.net",
		"state":                       "Active",
		"currentTier":                 tier,
		"creationTime":                rec.CreatedAt.Format(time.RFC3339),
		"lastModifiedTime":            rec.UpdatedAt.Format(time.RFC3339),
		"defaultGroup":                azureARMString(rec.Properties["defaultGroup"]),
		"encryptionProvisioningState": "Succeeded",
	}
	for key, value := range map[string]string{"encryptionState": "Enabled", "firewallState": "Disabled", "firewallAllowAzureIps": "Disabled", "trustedIdProviderState": "Disabled"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	if _, present := rec.Properties["encryptionConfig"]; !present {
		props["encryptionConfig"] = map[string]any{"type": "ServiceManaged"}
	}
	return props
}

func azureDLSTrustedIDProviderValidate(payload map[string]any) string {
	if msg := azureDataPlatRequired("idProvider")(payload); msg != "" {
		return msg
	}
	u, err := url.Parse(azureARMString(azureARMObject(payload["properties"])["idProvider"]))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "properties.idProvider must be an https URL."
	}
	return ""
}

func azureDLSVirtualNetworkRuleValidate(payload map[string]any) string {
	if msg := azureDataPlatRequired("subnetId")(payload); msg != "" {
		return msg
	}
	if !strings.Contains(strings.ToLower(azureARMString(azureARMObject(payload["properties"])["subnetId"])), "/providers/microsoft.network/virtualnetworks/") {
		return "properties.subnetId must be a virtual network subnet ID."
	}
	return ""
}

// handleAzureDataLakeStore20161101Router serves the routes the shared ARM
// handler doesn't: the regional capability, a single object rather than a
// list.
func (s *Server) handleAzureDataLakeStore20161101Router(w http.ResponseWriter, r *http.Request) bool {
	return s.serveAzureDataPlatCapability(w, r, azureDLSNamespace, azureDLSAccountType, azureDLSMaxAccounts)
}
