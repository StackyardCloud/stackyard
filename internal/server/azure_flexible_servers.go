package server

import (
	"bytes"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

// Shared rules for the PostgreSQL and MySQL flexible server providers, which
// have the same resource model: servers with databases and firewall rules,
// start/stop/restart, and global DNS names checked by checkNameAvailability.

type azureFlexibleServerFlavor struct {
	Namespace      string // e.g. "Microsoft.DBforPostgreSQL"
	DNSSuffix      string // e.g. ".postgres.database.azure.com"
	Versions       []string
	DefaultVersion string
	// DatabaseDefaults are charset and collation for new databases.
	DatabaseDefaults map[string]any
	// SystemDatabases are created with every server.
	SystemDatabases []string
	ReservedLogins  map[string]bool
}

var (
	azureFlexibleServerNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
	azureFlexibleServerTiers       = map[string]bool{"Burstable": true, "GeneralPurpose": true, "MemoryOptimized": true}
)

func (f azureFlexibleServerFlavor) serverType() string { return f.Namespace + "/flexibleServers" }

func registerAzureFlexibleServer(f azureFlexibleServerFlavor) {
	databases := azureARMResourceType{
		Namespace:          f.Namespace,
		Type:               "flexibleServers/databases",
		Proxy:              true,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any { return azureFlexibleServerDatabaseDefaults(f, rec) },
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:           f.Namespace,
		Type:                "flexibleServers",
		ReadOnlyProperties:  func(rec *azureARMResource) map[string]any { return azureFlexibleServerProperties(f, rec) },
		ValidateName:        azureFlexibleServerValidateName,
		Validate:            func(payload map[string]any) string { return azureFlexibleServerValidate(f, payload) },
		WriteOnlyProperties: []string{"administratorLoginPassword"},
		// start and stop own the state, so a stopped server stays stopped until started.
		StateProperties: map[string]any{"state": "Ready"},
		Transitions: map[string]map[string]any{
			"start":   {"state": "Ready"},
			"stop":    {"state": "Stopped"},
			"restart": {},
		},
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			children := make([]*azureARMResource, 0, len(f.SystemDatabases))
			for _, name := range f.SystemDatabases {
				children = append(children, azureARMChild(parent, databases, name, map[string]any{}))
			}
			return children
		},
	})
	registerAzureARMResourceType(databases)
	registerAzureARMResourceType(azureARMResourceType{Namespace: f.Namespace, Type: "flexibleServers/firewallRules", Proxy: true, Validate: azureValidateFirewallRule})
	check := func(s *Server, payload map[string]any) (int, any) { return s.azureFlexibleServerCheckName(f, payload) }
	registerAzureARMProviderAction(f.Namespace, "checkNameAvailability", check)
	registerAzureARMProviderAction(f.Namespace, "locations/checkNameAvailability", check)
}

func azureFlexibleServerValidateName(name string) string {
	if !azureFlexibleServerNamePattern.MatchString(name) {
		return "Server names must be 3-63 lowercase letters, digits or hyphens, and cannot start or end with a hyphen."
	}
	return ""
}

func azureFlexibleServerValidate(f azureFlexibleServerFlavor, payload map[string]any) string {
	sku := azureARMObject(payload["sku"])
	if azureARMString(sku["name"]) == "" || !azureFlexibleServerTiers[azureARMString(sku["tier"])] {
		return "sku.name is required and sku.tier must be Burstable, GeneralPurpose or MemoryOptimized."
	}
	props := azureARMObject(payload["properties"])
	if version := azureARMString(props["version"]); version != "" && !slices.Contains(f.Versions, version) {
		return "properties.version must be one of " + strings.Join(f.Versions, ", ") + "."
	}
	switch azureARMString(props["createMode"]) {
	case "", "Default", "Create":
	default:
		// Restores and replicas copy the source server's administrator.
		return ""
	}
	if azureARMString(azureARMObject(props["authConfig"])["passwordAuth"]) == "Disabled" {
		return ""
	}
	login := azureARMString(props["administratorLogin"])
	if login == "" || azureARMString(props["administratorLoginPassword"]) == "" {
		return "properties.administratorLogin and properties.administratorLoginPassword are required."
	}
	if f.ReservedLogins[strings.ToLower(login)] || strings.HasPrefix(strings.ToLower(login), "pg_") {
		return "properties.administratorLogin '" + login + "' is a reserved name."
	}
	return ""
}

// azureFlexibleServerProperties sets the host name and default version.
func azureFlexibleServerProperties(f azureFlexibleServerFlavor, rec *azureARMResource) map[string]any {
	props := map[string]any{"fullyQualifiedDomainName": rec.Name + f.DNSSuffix}
	if azureARMString(rec.Properties["version"]) == "" {
		props["version"] = f.DefaultVersion
	}
	return props
}

func azureFlexibleServerDatabaseDefaults(f azureFlexibleServerFlavor, rec *azureARMResource) map[string]any {
	props := map[string]any{}
	for k, v := range f.DatabaseDefaults {
		if azureARMString(rec.Properties[k]) == "" {
			props[k] = v
		}
	}
	return props
}

// azureFlexibleServerCheckName checks a server name; host names are global,
// so it compares across all subscriptions and regions.
func (s *Server) azureFlexibleServerCheckName(f azureFlexibleServerFlavor, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if msg := azureFlexibleServerValidateName(name); msg != "" {
		return http.StatusOK, map[string]any{"name": name, "nameAvailable": false, "reason": "Invalid", "message": msg}
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, f.serverType()) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"name": name, "nameAvailable": false, "reason": "AlreadyExists", "message": "Specified server name is already used."}
	}
	return http.StatusOK, map[string]any{"name": name, "nameAvailable": true, "message": ""}
}

// azureValidateFirewallRule checks an IPv4 start/end range, shared by the
// database providers' firewallRules.
func azureValidateFirewallRule(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	start := net.ParseIP(azureARMString(props["startIpAddress"])).To4()
	end := net.ParseIP(azureARMString(props["endIpAddress"])).To4()
	if start == nil || end == nil {
		return "properties.startIpAddress and properties.endIpAddress must be IPv4 addresses."
	}
	if bytes.Compare(start, end) > 0 {
		return "properties.startIpAddress must not be greater than properties.endIpAddress."
	}
	return ""
}
