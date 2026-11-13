# Azure Cache for Redis (`redis-2024-11-01`) Staged Plan

## Objective

Emulate the Azure Cache for Redis control plane (`Microsoft.Cache/redis`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2024-11-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/redis/redis`
- `https://learn.microsoft.com/en-us/rest/api/redis/redis/create`
- `https://learn.microsoft.com/en-us/rest/api/redis/redis/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/redis/firewall-rules`

## Service Naming

- service key: `redis`
- display name: `Azure Cache for Redis`
- implementation: `internal/server/provider_azure_redis.go`
- tests: `internal/server/provider_azure_redis_test.go`
- example: `examples/azure/redis-2024-11-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `redis` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listKeys` |
| `redis/firewallRules` | proxy child | create/update, get, delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- cache names are 1-63 letters, digits or single hyphens, starting and ending with a letter or digit (400 `InvalidResourceName`)
- the SKU lives at `properties.sku`, not top-level `sku`: `Basic`/`Standard` need family `C` and capacity 0-6, `Premium` needs family `P` and capacity 1-5 (400 `InvalidRequestContent`)
- `hostName` (`{lower(name)}.redis.cache.windows.net`), `port` 6379 and `sslPort` 6380 are server-owned
- `listKeys` returns primary and secondary keys derived from the resource ID
- firewall rules require IPv4 `startIP` and `endIP`

Out of scope for this stage:
- the Redis data plane (no Redis protocol listener)
- `regenerateKey`, `forceReboot`, `importData`/`exportData`, `checkNameAvailability`
- patch schedules, linked servers, private endpoint connections and access policies
- Azure Managed Redis (`Microsoft.Cache/redisEnterprise`, issue #423)

## Quality Gates

- `go test ./internal/server -run TestAzureRedis -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service redis`
- `python3 scripts/azure-io-contract-coverage.py --service redis`
- `python3 scripts/azure-doc-contract-coverage.py --service redis`
