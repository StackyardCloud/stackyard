# Azure Cosmos DB (`cosmosdb-2024-11-15`) Staged Plan

## Objective

Emulate the Azure Cosmos DB control plane for database accounts (`Microsoft.DocumentDB/databaseAccounts`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Mongo vCore clusters in the same namespace are covered by `docs/mongocluster-2024-07-01-plan.md`.

## References

Microsoft Learn pages for the `2024-11-15` API version:

- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts/list-connection-strings`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/database-accounts/check-name-exists`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/sql-resources`
- `https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/mongo-db-resources`

## Service Naming

- service key: `cosmosdb`
- display name: `Azure Cosmos DB`
- implementation: `internal/server/provider_azure_cosmosdb.go`
- tests: `internal/server/provider_azure_cosmosdb_test.go`
- example: `examples/azure/cosmosdb-2024-11-15/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `databaseAccounts` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listKeys`, `readonlykeys` (POST), `listConnectionStrings` |
| `databaseAccounts/sqlDatabases` | proxy child | create/update, get, delete, list |
| `databaseAccounts/sqlDatabases/containers` | proxy child | create/update, get, delete, list |
| `databaseAccounts/mongodbDatabases` | proxy child | create/update, get, delete, list |
| `databaseAccounts/mongodbDatabases/collections` | proxy child | create/update, get, delete, list |
| `databaseAccountNames/{name}` | tenant-scoped HEAD | 200 when the name is taken, 404 when it is free |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- account names are 3-44 lowercase letters, digits or hyphens and cannot start or end with a hyphen (400 `InvalidResourceName`)
- `kind` is `GlobalDocumentDB` (default), `MongoDB` or `Parse`; `properties.databaseAccountOfferType` must be `Standard`
- `properties.locations` needs at least one region, each with a `locationName` and a unique `failoverPriority`, and one region with priority 0 (the write region)
- `consistencyPolicy.defaultConsistencyLevel` must be `Eventual`, `Session`, `BoundedStaleness`, `Strong` or `ConsistentPrefix` and defaults to `Session`
- `documentEndpoint` (`{name}.documents.azure.com` on port 443), `writeLocations` and `readLocations` are server-owned and derived from `locations`, with regional endpoints such as `{name}-eastus`
- `listKeys` returns four 88-character keys derived from the resource ID; `readonlykeys` returns only the read-only pair; `listConnectionStrings` returns `AccountEndpoint={endpoint};AccountKey={key};` for each key
- SQL and MongoDB databases, containers and collections need `properties.resource.id`; containers also need `partitionKey.paths` starting with `/`
- the name check compares account names across all subscriptions

Out of scope for this stage:
- the Cosmos DB data plane (documents, queries and the MongoDB wire protocol)
- Cassandra, Gremlin and Table resources, throughput settings, stored procedures, triggers and UDFs, role definitions and assignments, backups, restores, failover and key regeneration
- checks that MongoDB resources are only created on `MongoDB` accounts, or that `resource.id` matches the resource name

## Quality Gates

- `go test ./internal/server -run TestAzureCosmosDB -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service cosmosdb`
- `python3 scripts/azure-io-contract-coverage.py --service cosmosdb`
- `python3 scripts/azure-doc-contract-coverage.py --service cosmosdb`
