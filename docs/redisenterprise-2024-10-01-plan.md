# Azure Managed Redis (`redisenterprise-2024-10-01`) Staged Plan

## Objective

Emulate the Azure Managed Redis and Redis Enterprise control plane (`Microsoft.Cache/redisEnterprise`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2024-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/redis/redisenterprisecache/redis-enterprise`
- `https://learn.microsoft.com/en-us/rest/api/redis/redisenterprisecache/redis-enterprise/create`
- `https://learn.microsoft.com/en-us/rest/api/redis/redisenterprisecache/databases`
- `https://learn.microsoft.com/en-us/rest/api/redis/redisenterprisecache/databases/list-keys`

## Service Naming

- service key: `redisenterprise`
- display name: `Azure Managed Redis`
- implementation: `internal/server/provider_azure_redisenterprise.go`
- tests: `internal/server/provider_azure_redisenterprise_test.go`
- example: `examples/azure/redisenterprise-2024-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `redisEnterprise` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `redisEnterprise/databases` | proxy child | create/update, get, update, delete, list, `listKeys` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- cluster names are 1-60 letters, digits or single hyphens (400 `InvalidResourceName`)
- `sku.name` must be a Managed Redis (`Balanced_B*`, `MemoryOptimized_M*`, `ComputeOptimized_X*`, `FlashOptimized_A*`) or Enterprise (`Enterprise_E*`, `EnterpriseFlash_F*`) SKU with a size
- `hostName` is server-owned: `{name}.{region}.redis.azure.net`, or `{name}.{region}.redisenterprise.cache.azure.net` for Enterprise SKUs; `resourceState` is `Running`
- the only database name is `default`; `clientProtocol` and `clusteringPolicy` are checked against their enums; `port` defaults to 10000
- `listKeys` returns primary and secondary keys derived from the database ID

Out of scope for this stage:
- the Redis data plane (no Redis protocol listener)
- `regenerateKey`, `import`, `export`, `flush`, `forceUnlink` and geo-replication (`linkedDatabases`)
- rejecting a port change after creation, and capacity rules for Enterprise SKUs
- access policy assignments and private endpoint connections

## Quality Gates

- `go test ./internal/server -run TestAzureRedisEnterprise -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service redisenterprise`
- `python3 scripts/azure-io-contract-coverage.py --service redisenterprise`
- `python3 scripts/azure-doc-contract-coverage.py --service redisenterprise`
