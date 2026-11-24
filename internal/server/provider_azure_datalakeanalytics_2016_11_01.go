package server

import (
	"net/http"
	"slices"
	"strings"
	"time"
)

// Azure Data Lake Analytics (Microsoft.DataLakeAnalytics 2016-11-01):
// accounts with their Data Lake Store and storage accounts, compute
// policies and firewall rules, served by the shared ARM handler, plus name
// availability, the regional capability and the operations list. The job,
// pipeline, recurrence and U-SQL catalog data plane is in
// azure_datalakeanalytics_dataplane.go.

const (
	azureDataLakeAnalytics20161101APIVersion = "2016-11-01"
	azureDLANamespace                        = "Microsoft.DataLakeAnalytics"
	azureDLAAccountType                      = azureDLANamespace + "/accounts"
	azureDLAMaxAccounts                      = 150
)

var azureDLATiers = []string{"Consumption", "Commitment_100AUHours", "Commitment_500AUHours", "Commitment_1000AUHours", "Commitment_5000AUHours", "Commitment_10000AUHours", "Commitment_50000AUHours", "Commitment_100000AUHours", "Commitment_500000AUHours"}

func init() {
	ns := azureDLANamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts",
		ValidateName:       azureDataPlatAccountNameError,
		Validate:           azureDLAAccountValidate,
		ReadOnlyProperties: azureDLAAccountReadOnly,
		InlineChildren: map[string]string{
			"dataLakeStoreAccounts": "accounts/dataLakeStoreAccounts",
			"storageAccounts":       "accounts/storageAccounts",
			"computePolicies":       "accounts/computePolicies",
			"firewallRules":         "accounts/firewallRules",
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/dataLakeStoreAccounts", CreateStatus: http.StatusOK, Proxy: true, NoPatch: true,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if azureARMString(rec.Properties["suffix"]) != "" {
				return nil
			}
			return map[string]any{"suffix": "azuredatalakestore.net"}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/storageAccounts", CreateStatus: http.StatusOK, Proxy: true,
		Validate:            azureDataPlatRequired("accessKey"),
		WriteOnlyProperties: []string{"accessKey"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if azureARMString(rec.Properties["suffix"]) != "" {
				return nil
			}
			return map[string]any{"suffix": "blob.core.windows.net"}
		},
	})
	// ponytail: no blob containers are linked to a storage account, so the
	// list is empty and listSasTokens isn't served; link to the blob store
	// if a client needs real containers.
	registerAzureARMView(ns, "accounts/storageAccounts/containers", func(*Server, string) []map[string]any { return nil })
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/computePolicies", CreateStatus: http.StatusOK, Proxy: true,
		Validate: azureDLAComputePolicyValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accounts/firewallRules", CreateStatus: http.StatusOK, Proxy: true,
		Validate: azureDataPlatIPv4Range,
	})
	registerAzureARMLocationAction(ns, "checkNameAvailability", azureDataPlatCheckName(azureDLAAccountType, azureDataPlatAccountNameError))
	registerAzureARMView(ns, "operations", azureDataPlatOperations(ns, "Microsoft Data Lake Analytics",
		"accounts/read", "accounts/write", "accounts/delete",
		"accounts/dataLakeStoreAccounts/read", "accounts/dataLakeStoreAccounts/write", "accounts/dataLakeStoreAccounts/delete",
		"accounts/storageAccounts/read", "accounts/storageAccounts/write", "accounts/storageAccounts/delete",
		"accounts/computePolicies/read", "accounts/computePolicies/write", "accounts/computePolicies/delete",
		"accounts/firewallRules/read", "accounts/firewallRules/write", "accounts/firewallRules/delete",
		"locations/capability/read", "locations/checkNameAvailability/action"))
}

func azureDLAAccountValidate(payload map[string]any) string {
	if msg := azureDataPlatAll(
		azureDataPlatRequired("defaultDataLakeStoreAccount", "dataLakeStoreAccounts"),
		azureDataPlatEnum("firewallState", "Enabled", "Disabled"),
		azureDataPlatEnum("firewallAllowAzureIps", "Enabled", "Disabled"),
		azureDataPlatEnum("newTier", azureDLATiers...),
	)(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	stores, _ := props["dataLakeStoreAccounts"].([]any)
	defaultStore := azureARMString(props["defaultDataLakeStoreAccount"])
	if !slices.ContainsFunc(stores, func(item any) bool {
		return strings.EqualFold(azureARMString(azureARMObject(item)["name"]), defaultStore)
	}) {
		return "properties.defaultDataLakeStoreAccount must name one of properties.dataLakeStoreAccounts."
	}
	return ""
}

// azureDLAAccountReadOnly fills in the server-owned account fields and the
// documented defaults for limits the client left out.
func azureDLAAccountReadOnly(rec *azureARMResource) map[string]any {
	tier := azureARMString(rec.Properties["newTier"])
	if tier == "" {
		tier = "Consumption"
	}
	props := map[string]any{
		"accountId":                    azureDataPlatGUID(rec.ID),
		"endpoint":                     rec.Name + ".azuredatalakeanalytics.net",
		"state":                        "Active",
		"currentTier":                  tier,
		"creationTime":                 rec.CreatedAt.Format(time.RFC3339),
		"lastModifiedTime":             rec.UpdatedAt.Format(time.RFC3339),
		"systemMaxDegreeOfParallelism": 250,
		"systemMaxJobCount":            20,
		"maxDegreeOfParallelismPerJob": 32,
		"minPriorityPerJob":            1,
	}
	for key, value := range map[string]any{"maxDegreeOfParallelism": 30, "maxJobCount": 3, "queryStoreRetention": 30, "firewallState": "Disabled", "firewallAllowAzureIps": "Disabled"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureDLAComputePolicyValidate(payload map[string]any) string {
	if msg := azureDataPlatAll(
		azureDataPlatRequired("objectId", "objectType"),
		azureDataPlatEnum("objectType", "User", "Group", "ServicePrincipal"),
	)(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if props["maxDegreeOfParallelismPerJob"] == nil && props["minPriorityPerJob"] == nil {
		return "properties.maxDegreeOfParallelismPerJob or properties.minPriorityPerJob is required."
	}
	return ""
}

// handleAzureDataLakeAnalytics20161101Router serves the regional
// capability and the job, pipeline, recurrence and catalog data plane.
func (s *Server) handleAzureDataLakeAnalytics20161101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if strings.Contains(strings.ToLower(path), "/providers/microsoft.datalakeanalytics/") {
		return s.serveAzureDataPlatCapability(w, r, azureDLANamespace, azureDLAAccountType, azureDLAMaxAccounts)
	}
	return s.serveAzureDLADataPlane(w, r, path)
}
