# Azure Database for PostgreSQL flexible server (`postgresql-2024-08-01`) Staged Plan

## Objective

Emulate the Azure Database for PostgreSQL flexible server control plane (`Microsoft.DBforPostgreSQL/flexibleServers`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Rules shared with MySQL flexible servers live in `internal/server/azure_flexible_servers.go`.

## References

Microsoft Learn pages for the `2024-08-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/servers`
- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/servers/create`
- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/servers/stop`
- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/databases`
- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/firewall-rules`
- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/postgresql/flexibleserver/check-name-availability-with-location`

## Service Naming

- service key: `postgresql`
- display name: `Azure Database for PostgreSQL flexible server`
- implementation: `internal/server/provider_azure_postgresql.go`
- tests: `internal/server/provider_azure_postgresql_test.go`
- example: `examples/azure/postgresql-2024-08-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `flexibleServers` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `start`, `stop`, `restart` |
| `flexibleServers/databases` | proxy child | create/update, get, delete, list |
| `flexibleServers/firewallRules` | proxy child | create/update, get, delete, list |
| `checkNameAvailability` | action | subscription-scoped and `locations/{location}/checkNameAvailability` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-63 lowercase letters, digits or hyphens and cannot start or end with a hyphen (400 `InvalidResourceName`)
- `sku.name` is required and `sku.tier` must be `Burstable`, `GeneralPurpose` or `MemoryOptimized`; `properties.version` must be 11 to 16 and defaults to `16`
- new servers (`createMode` empty, `Default` or `Create`) need `administratorLogin` and `administratorLoginPassword` unless `authConfig.passwordAuth` is `Disabled`; reserved logins such as `azure_superuser`, `admin` or `root`, and logins starting with `pg_`, are rejected
- `administratorLoginPassword` is accepted on PUT and PATCH but never stored or returned
- `fullyQualifiedDomainName` (`{name}.postgres.database.azure.com`) and `state` are server-owned; `stop` sets `state` to `Stopped`, `start` sets it back to `Ready`, and the state survives updates; `restart` succeeds without changes
- every server starts with the `azure_maintenance`, `azure_sys` and `postgres` databases; databases default to charset `UTF8` and collation `en_US.utf8`
- firewall rules need IPv4 `startIpAddress` and `endIpAddress` with start not greater than end
- `checkNameAvailability` compares names across all subscriptions and regions, because host names are global, and answers `Invalid` or `AlreadyExists`

Out of scope for this stage:
- the PostgreSQL wire protocol (no database engine runs)
- server configurations (parameters), Microsoft Entra administrators, backups, replicas, virtual endpoints, migrations and private endpoints
- long-running operation polling: start, stop and restart complete synchronously with 200

## Quality Gates

- `go test ./internal/server -run TestAzurePostgreSQL -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service postgresql`
- `python3 scripts/azure-io-contract-coverage.py --service postgresql`
- `python3 scripts/azure-doc-contract-coverage.py --service postgresql`
