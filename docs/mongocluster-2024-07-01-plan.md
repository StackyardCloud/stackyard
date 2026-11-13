# Azure Cosmos DB for MongoDB vCore (`mongocluster-2024-07-01`) Staged Plan

## Objective

Emulate the Azure Cosmos DB for MongoDB vCore control plane (`Microsoft.DocumentDB/mongoClusters`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2024-07-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/mongocluster/mongo-clusters`
- `https://learn.microsoft.com/en-us/rest/api/mongocluster/mongo-clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/mongocluster/mongo-clusters/list-connection-strings`
- `https://learn.microsoft.com/en-us/rest/api/mongocluster/mongo-clusters/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/mongocluster/firewall-rules`

## Service Naming

- service key: `mongocluster`
- display name: `Azure Cosmos DB for MongoDB vCore`
- implementation: `internal/server/provider_azure_mongocluster.go`
- tests: `internal/server/provider_azure_mongocluster_test.go`
- example: `examples/azure/mongocluster-2024-07-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `mongoClusters` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listConnectionStrings` |
| `mongoClusters/firewallRules` | proxy child | create/update, get, delete, list |
| `locations/{location}/checkMongoClusterNameAvailability` | action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-40 lowercase letters, digits or hyphens and cannot start or end with a hyphen (400 `InvalidResourceName`)
- new clusters (`createMode` empty or `Default`) need `administrator.userName` and `administrator.password`; `serverVersion` must be `5.0`, `6.0` or `7.0` (default `7.0`); `highAvailability.targetMode` must be `Disabled`, `SameZone` or `ZoneRedundantPreferred`
- `administrator.password` is accepted on PUT and PATCH but never stored or returned
- `connectionString` (`mongodb+srv://<user>:<password>@{name}.global.mongocluster.cosmos.azure.com/...`, with placeholders as in Azure) and `clusterStatus` are server-owned; `listConnectionStrings` returns it as `GlobalReadWrite`
- firewall rules need IPv4 `startIpAddress` and `endIpAddress` with start not greater than end, as for the flexible server providers
- `checkMongoClusterNameAvailability` compares names across all subscriptions and regions, because host names are global

Out of scope for this stage:
- the MongoDB wire protocol (no database engine runs)
- the other `Microsoft.DocumentDB` resources (Cosmos DB database accounts)
- replicas, promotion, private endpoints and users

## Quality Gates

- `go test ./internal/server -run TestAzureMongoCluster -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service mongocluster`
- `python3 scripts/azure-io-contract-coverage.py --service mongocluster`
- `python3 scripts/azure-doc-contract-coverage.py --service mongocluster`
