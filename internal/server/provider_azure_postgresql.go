package server

// Microsoft.DBforPostgreSQL flexible servers, databases and firewall rules,
// served by the shared ARM handler (rules in azure_flexible_servers.go).
func init() {
	registerAzureFlexibleServer(azureFlexibleServerFlavor{
		Namespace:        "Microsoft.DBforPostgreSQL",
		DNSSuffix:        ".postgres.database.azure.com",
		Versions:         []string{"11", "12", "13", "14", "15", "16"},
		DefaultVersion:   "16",
		DatabaseDefaults: map[string]any{"charset": "UTF8", "collation": "en_US.utf8"},
		SystemDatabases:  []string{"azure_maintenance", "azure_sys", "postgres"},
		ReservedLogins:   map[string]bool{"azure_superuser": true, "azure_pg_admin": true, "admin": true, "administrator": true, "root": true, "guest": true, "public": true},
	})
}
