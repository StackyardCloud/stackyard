package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const azureSynapseWorkspaceType = "Microsoft.Synapse/workspaces"

// Microsoft.Synapse workspaces with their firewall rules, dedicated SQL
// pools and Apache Spark pools, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.Synapse"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "workspaces",
		ReadOnlyProperties: azureSynapseWorkspaceProperties,
		ValidateName:       azureSynapseValidateName,
		Validate:           azureSynapseValidateWorkspace,
		// The SQL administrator password is never returned.
		WriteOnlyProperties: []string{"sqlAdministratorLoginPassword"},
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "workspaces/firewallRules", Proxy: true, Validate: azureValidateFirewallRule})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns,
		Type:      "workspaces/sqlPools",
		Validate: func(payload map[string]any) string {
			if !strings.HasPrefix(azureARMString(azureARMObject(payload["sku"])["name"]), "DW") {
				return "sku.name must be a DW performance level, such as DW100c."
			}
			return ""
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := map[string]any{"creationDate": rec.CreatedAt.Format(time.RFC3339)}
			if _, present := rec.Properties["collation"]; !present {
				props["collation"] = "SQL_Latin1_General_CP1_CI_AS"
			}
			return props
		},
		StateProperties: map[string]any{"status": "Online"},
		Transitions:     map[string]map[string]any{"pause": {"status": "Paused"}, "resume": {"status": "Online"}},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns,
		Type:      "workspaces/bigDataPools",
		Validate:  azureSynapseValidateSparkPool,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"creationDate": rec.CreatedAt.Format(time.RFC3339)}
		},
	})
	registerAzureARMProviderAction(ns, "checkNameAvailability", azureSynapseCheckNameAvailability)
}

var (
	// Workspace names are 1-50 lowercase letters, digits or hyphens, and
	// cannot start or end with a hyphen.
	azureSynapseNamePattern  = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,48}[a-z0-9])?$`)
	azureSynapseNodeSizes    = []string{"Small", "Medium", "Large", "XLarge", "XXLarge", "XXXLarge"}
	azureSynapseSparkVersion = []string{"3.3", "3.4"}
)

func azureSynapseValidateName(name string) string {
	if !azureSynapseNamePattern.MatchString(name) || strings.HasSuffix(name, "-ondemand") {
		return "Workspace names must be 1-50 lowercase letters, digits or hyphens, cannot start or end with a hyphen and cannot end with '-ondemand'."
	}
	return ""
}

func azureSynapseValidateWorkspace(payload map[string]any) string {
	storage := azureARMObject(azureARMObject(payload["properties"])["defaultDataLakeStorage"])
	if strings.TrimSpace(azureARMString(storage["accountUrl"])) == "" || strings.TrimSpace(azureARMString(storage["filesystem"])) == "" {
		return "properties.defaultDataLakeStorage.accountUrl and filesystem are required."
	}
	return ""
}

func azureSynapseValidateSparkPool(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if raw, present := props["nodeSize"]; present && !slices.Contains(azureSynapseNodeSizes, azureARMString(raw)) {
		return "properties.nodeSize must be one of " + strings.Join(azureSynapseNodeSizes, ", ") + "."
	}
	if raw, present := props["sparkVersion"]; present && !slices.Contains(azureSynapseSparkVersion, azureARMString(raw)) {
		return "properties.sparkVersion must be one of " + strings.Join(azureSynapseSparkVersion, ", ") + "."
	}
	if count, present := props["nodeCount"]; present {
		if n, ok := count.(float64); !ok || n < 3 || n > 200 || n != float64(int(n)) {
			return "properties.nodeCount must be an integer from 3 to 200."
		}
	}
	return ""
}

// azureSynapseWorkspaceProperties sets the workspace endpoints, ID and
// managed resource group.
func azureSynapseWorkspaceProperties(rec *azureARMResource) map[string]any {
	name := strings.ToLower(rec.Name)
	props := map[string]any{
		"connectivityEndpoints": map[string]any{
			"web":         "https://web.azuresynapse.net?workspace=" + strings.ReplaceAll(rec.ID, "/", "%2f"),
			"dev":         "https://" + name + ".dev.azuresynapse.net",
			"sql":         name + ".sql.azuresynapse.net",
			"sqlOnDemand": name + "-ondemand.sql.azuresynapse.net",
		},
		"workspaceUID": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
	}
	if _, present := rec.Properties["managedResourceGroupName"]; !present {
		props["managedResourceGroupName"] = "synapseworkspace-managedrg-" + uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#rg")).String()
	}
	return props
}

// azureSynapseCheckNameAvailability compares workspace names across all
// subscriptions, because they are DNS names. Synapse answers with
// "available", like SQL.
func azureSynapseCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureSynapseValidateName(name); msg != "" {
		return http.StatusOK, map[string]any{"name": name, "available": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureSynapseWorkspaceType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"name": name, "available": false, "reason": "AlreadyExists", "message": "The specified workspace name '" + name + "' is already in use."}
	}
	return http.StatusOK, map[string]any{"name": name, "available": true}
}
