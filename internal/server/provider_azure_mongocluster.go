package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const azureMongoClusterType = "Microsoft.DocumentDB/mongoClusters"

// Microsoft.DocumentDB mongoClusters (Azure Cosmos DB for MongoDB vCore) and
// their firewall rules, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.DocumentDB"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:           ns,
		Type:                "mongoClusters",
		ReadOnlyProperties:  azureMongoClusterProperties,
		ValidateName:        azureMongoClusterValidateName,
		Validate:            azureMongoClusterValidate,
		WriteOnlyProperties: []string{"administrator.password"},
		Actions:             map[string]azureARMAction{"listconnectionstrings": azureMongoClusterListConnectionStrings},
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "mongoClusters/firewallRules", Proxy: true, Validate: azureValidateFirewallRule})
	registerAzureARMProviderAction(ns, "locations/checkMongoClusterNameAvailability", azureMongoClusterCheckName)
}

var azureMongoClusterNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

func azureMongoClusterValidateName(name string) string {
	if !azureMongoClusterNamePattern.MatchString(name) {
		return "Cluster names must be 3-40 lowercase letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureMongoClusterValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if v := azureARMString(props["serverVersion"]); v != "" && !slices.Contains([]string{"5.0", "6.0", "7.0"}, v) {
		return "properties.serverVersion must be 5.0, 6.0 or 7.0."
	}
	if mode := azureARMString(azureARMObject(props["highAvailability"])["targetMode"]); mode != "" && !slices.Contains([]string{"Disabled", "SameZone", "ZoneRedundantPreferred"}, mode) {
		return "properties.highAvailability.targetMode must be Disabled, SameZone or ZoneRedundantPreferred."
	}
	switch azureARMString(props["createMode"]) {
	case "", "Default":
	default:
		// Restores and replicas copy the source cluster's administrator.
		return ""
	}
	admin := azureARMObject(props["administrator"])
	if azureARMString(admin["userName"]) == "" || azureARMString(admin["password"]) == "" {
		return "properties.administrator.userName and properties.administrator.password are required."
	}
	return ""
}

func azureMongoClusterHost(rec *azureARMResource) string {
	return rec.Name + ".global.mongocluster.cosmos.azure.com"
}

func azureMongoClusterProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		// Like Azure, the connection string carries placeholders, not credentials.
		"connectionString": "mongodb+srv://<user>:<password>@" + azureMongoClusterHost(rec) + "/?tls=true&authMechanism=SCRAM-SHA-256&retrywrites=false&maxIdleTimeMS=120000",
		"clusterStatus":    "Ready",
	}
	if azureARMString(rec.Properties["serverVersion"]) == "" {
		props["serverVersion"] = "7.0"
	}
	return props
}

func azureMongoClusterListConnectionStrings(record *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{"connectionStrings": []any{map[string]any{
		"connectionString": azureARMString(record.Properties["connectionString"]),
		"description":      "default connection string",
		"name":             "GlobalReadWrite",
	}}}
}

// azureMongoClusterCheckName checks a cluster name; host names are global,
// so it compares across all subscriptions and regions.
func azureMongoClusterCheckName(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureMongoClusterValidateName(name); msg != "" {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureMongoClusterType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "Cluster name '" + name + "' is already in use."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}
