package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
)

const azureSQLServerType = "Microsoft.Sql/servers"

// Microsoft.Sql logical servers, databases, elastic pools and firewall
// rules, served by the shared ARM handler.
func init() {
	const ns = "Microsoft.Sql"
	// Databases and elastic pools are tracked resources in the server's region.
	databases := azureARMResourceType{
		Namespace:          ns,
		Type:               "servers/databases",
		ReadOnlyProperties: azureSQLDatabaseProperties,
		ValidateName:       azureSQLValidateDatabaseName,
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:           ns,
		Type:                "servers",
		ReadOnlyProperties:  azureSQLServerProperties,
		ValidateName:        azureSQLValidateServerName,
		Validate:            azureSQLValidateServer,
		WriteOnlyProperties: []string{"administratorLoginPassword"},
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			master := azureARMChild(parent, databases, "master", map[string]any{})
			master.Location = parent.Location
			master.SKU = map[string]any{"name": "System", "tier": "System", "capacity": 0}
			return []*azureARMResource{master}
		},
	})
	registerAzureARMResourceType(databases)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "servers/elasticPools",
		ReadOnlyProperties: func(*azureARMResource) map[string]any { return map[string]any{"state": "Ready"} },
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "servers/firewallRules", Proxy: true, Validate: azureValidateFirewallRule})
	registerAzureARMProviderAction(ns, "checkNameAvailability", azureSQLCheckNameAvailability)
}

var (
	azureSQLServerNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	azureSQLReservedLogins    = []string{"admin", "administrator", "sa", "root", "dbmanager", "loginmanager", "dbo", "guest", "public"}
)

func azureSQLValidateServerName(name string) string {
	if !azureSQLServerNamePattern.MatchString(name) {
		return "Server names must be 1-63 lowercase letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureSQLValidateDatabaseName(name string) string {
	if len(name) > 128 || strings.ContainsAny(name, `<>*%&:\/?`) || strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") {
		return "Database names must be at most 128 characters, cannot contain <>*%&:\\/? and cannot end with a period or space."
	}
	return ""
}

func azureSQLValidateServer(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if tls := azureARMString(props["minimalTlsVersion"]); tls != "" && !slices.Contains([]string{"1.0", "1.1", "1.2", "1.3", "None"}, tls) {
		return "properties.minimalTlsVersion must be 1.0, 1.1, 1.2, 1.3 or None."
	}
	if entraOnly, _ := azureARMObject(props["administrators"])["azureADOnlyAuthentication"].(bool); entraOnly {
		return ""
	}
	login := azureARMString(props["administratorLogin"])
	if login == "" || azureARMString(props["administratorLoginPassword"]) == "" {
		return "properties.administratorLogin and properties.administratorLoginPassword are required unless properties.administrators.azureADOnlyAuthentication is true."
	}
	if slices.Contains(azureSQLReservedLogins, strings.ToLower(login)) {
		return "properties.administratorLogin '" + login + "' is a reserved name."
	}
	return ""
}

func azureSQLServerProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"fullyQualifiedDomainName": rec.Name + ".database.windows.net",
		"state":                    "Ready",
	}
	if azureARMString(rec.Properties["version"]) == "" {
		props["version"] = "12.0"
	}
	return props
}

// azureSQLDatabaseProperties derives a stable databaseId and reports the
// service objective from the SKU, defaulting to General Purpose Gen5 2 vCores.
func azureSQLDatabaseProperties(rec *azureARMResource) map[string]any {
	objective := azureARMString(rec.SKU["name"])
	if objective == "" {
		objective = "GP_Gen5_2"
	}
	props := map[string]any{
		"status":                        "Online",
		"databaseId":                    uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#database")).String(),
		"currentServiceObjectiveName":   objective,
		"requestedServiceObjectiveName": objective,
	}
	if azureARMString(rec.Properties["collation"]) == "" {
		props["collation"] = "SQL_Latin1_General_CP1_CI_AS"
	}
	return props
}

// azureSQLCheckNameAvailability checks a server name; host names are global,
// so it compares across all subscriptions. SQL answers with "available".
func azureSQLCheckNameAvailability(s *Server, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureSQLValidateServerName(name); msg != "" {
		return http.StatusOK, map[string]any{"name": name, "available": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureSQLServerType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"name": name, "available": false, "reason": "AlreadyExists", "message": "The specified server name is already in use."}
	}
	return http.StatusOK, map[string]any{"name": name, "available": true}
}
