# Azure Database for MySQL flexible server (`mysql-2023-12-30`) Staged Plan

## Objective

Emulate the Azure Database for MySQL flexible server control plane (`Microsoft.DBforMySQL/flexibleServers`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Rules shared with PostgreSQL flexible servers live in `internal/server/azure_flexible_servers.go`.

## References

Microsoft Learn pages for the `2023-12-30` API version:

- `https://learn.microsoft.com/en-us/rest/api/mysql/flexibleserver/servers`
- `https://learn.microsoft.com/en-us/rest/api/mysql/flexibleserver/servers/create`
- `https://learn.microsoft.com/en-us/rest/api/mysql/flexibleserver/servers/stop`
- `https://learn.microsoft.com/en-us/rest/api/mysql/flexibleserver/databases`
- `https://learn.microsoft.com/en-us/rest/api/mysql/flexibleserver/firewall-rules`
- `https://learn.microsoft.com/en-us/rest/api/mysql/flexibleserver/check-name-availability`

## Service Naming

- service key: `mysql`
- display name: `Azure Database for MySQL flexible server`
- implementation: `internal/server/provider_azure_mysql.go`
- tests: `internal/server/provider_azure_mysql_test.go`
- example: `examples/azure/mysql-2023-12-30/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `flexibleServers` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `start`, `stop`, `restart` |
| `flexibleServers/databases` | proxy child | create/update, get, delete, list |
| `flexibleServers/firewallRules` | proxy child | create/update, get, delete, list |
| `checkNameAvailability` | action | `locations/{location}/checkNameAvailability` and the subscription-scoped form without a location |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). The server, firewall rule, state and name rules match PostgreSQL flexible servers (see `docs/postgresql-2024-08-01-plan.md`). What differs:

- `properties.version` must be `5.7` or `8.0.21` and defaults to `8.0.21`
- reserved logins also include `sa`
- `fullyQualifiedDomainName` is `{name}.mysql.database.azure.com`; MySQL and PostgreSQL names are checked separately
- every server starts with the `information_schema`, `mysql`, `performance_schema` and `sys` databases; databases default to charset `utf8mb4` and collation `utf8mb4_0900_ai_ci`

Out of scope for this stage:
- the MySQL wire protocol (no database engine runs)
- server configurations (parameters), Microsoft Entra administrators, backups, replicas, failover and private endpoints
- long-running operation polling: start, stop and restart complete synchronously with 200

## Quality Gates

- `go test ./internal/server -run TestAzureMySQL -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service mysql`
- `python3 scripts/azure-io-contract-coverage.py --service mysql`
- `python3 scripts/azure-doc-contract-coverage.py --service mysql`
