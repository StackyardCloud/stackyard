package server

// Microsoft.DBforMySQL flexible servers, databases and firewall rules,
// served by the shared ARM handler (rules in azure_flexible_servers.go).
func init() {
	registerAzureFlexibleServer(azureFlexibleServerFlavor{
		Namespace:        "Microsoft.DBforMySQL",
		DNSSuffix:        ".mysql.database.azure.com",
		Versions:         []string{"5.7", "8.0.21"},
		DefaultVersion:   "8.0.21",
		DatabaseDefaults: map[string]any{"charset": "utf8mb4", "collation": "utf8mb4_0900_ai_ci"},
		SystemDatabases:  []string{"information_schema", "mysql", "performance_schema", "sys"},
		ReservedLogins:   map[string]bool{"azure_superuser": true, "admin": true, "administrator": true, "root": true, "guest": true, "public": true, "sa": true},
	})
}
