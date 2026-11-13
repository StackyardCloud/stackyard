package server

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.OperationalInsights Log Analytics workspaces and their tables,
// saved searches and data exports, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.OperationalInsights"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "workspaces",
		ReadOnlyProperties: azureOperationalInsightsWorkspaceProperties,
		ValidateName:       azureOperationalInsightsValidateName,
		Validate:           azureOperationalInsightsValidateWorkspace,
		Actions:            map[string]azureARMAction{"sharedkeys": azureOperationalInsightsSharedKeys},
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "workspaces/tables", Proxy: true})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "workspaces/savedSearches", Proxy: true, Validate: azureOperationalInsightsValidateSavedSearch})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "workspaces/dataExports", Proxy: true, Validate: azureOperationalInsightsValidateDataExport})
}

var (
	azureOperationalInsightsNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{2,61}[a-zA-Z0-9]$`)
	azureOperationalInsightsSKUs        = map[string]bool{
		"Free": true, "Standard": true, "Premium": true, "PerNode": true, "PerGB2018": true,
		"Standalone": true, "CapacityReservation": true, "LACluster": true,
	}
)

func azureOperationalInsightsValidateName(name string) string {
	if !azureOperationalInsightsNamePattern.MatchString(name) {
		return "Workspace names must be 4-63 letters, digits or hyphens, and start and end with a letter or digit."
	}
	return ""
}

func azureOperationalInsightsValidateWorkspace(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if sku := azureARMString(azureARMObject(props["sku"])["name"]); sku != "" && !azureOperationalInsightsSKUs[sku] {
		return "properties.sku.name is not a Log Analytics pricing tier."
	}
	if raw, present := props["retentionInDays"]; present {
		if days, ok := raw.(float64); !ok || days < 30 || days > 730 || days != float64(int(days)) {
			return "properties.retentionInDays must be an integer from 30 to 730."
		}
	}
	return ""
}

func azureOperationalInsightsValidateSavedSearch(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, field := range []string{"category", "displayName", "query"} {
		if strings.TrimSpace(azureARMString(props[field])) == "" {
			return "properties.category, properties.displayName and properties.query are required."
		}
	}
	return ""
}

func azureOperationalInsightsValidateDataExport(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	tables, _ := props["tableNames"].([]any)
	if strings.TrimSpace(azureARMString(azureARMObject(props["destination"])["resourceId"])) == "" || len(tables) == 0 {
		return "properties.destination.resourceId and properties.tableNames are required."
	}
	return ""
}

// azureOperationalInsightsWorkspaceProperties derives a stable customerId
// (the workspace ID agents use) and defaults the pricing tier to PerGB2018.
func azureOperationalInsightsWorkspaceProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"customerId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#customer")).String(),
	}
	if azureARMString(azureARMObject(rec.Properties["sku"])["name"]) == "" {
		props["sku"] = map[string]any{"name": "PerGB2018"}
	}
	if _, present := rec.Properties["retentionInDays"]; !present {
		props["retentionInDays"] = 30
	}
	return props
}

func azureOperationalInsightsSharedKeys(record *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{
		"primarySharedKey":   azureARMDerivedKey(record.ID, "primary"),
		"secondarySharedKey": azureARMDerivedKey(record.ID, "secondary"),
	}
}
