# Azure Synapse Analytics (`synapse-2021-06-01`) Staged Plan

## Objective

Emulate the Azure Synapse Analytics control plane (`Microsoft.Synapse`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): workspaces, their firewall rules, dedicated SQL pools and Apache Spark pools.

## References

Microsoft Learn pages for the `2021-06-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/synapse/workspaces/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/synapse/ip-firewall-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/synapse/sql-pools/create`
- `https://learn.microsoft.com/en-us/rest/api/synapse/sql-pools/pause`
- `https://learn.microsoft.com/en-us/rest/api/synapse/big-data-pools/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/synapse/operations/check-name-availability`

## Service Naming

- service key: `synapse`
- display name: `Azure Synapse Analytics`
- implementation: `internal/server/provider_azure_synapse.go`
- tests: `internal/server/provider_azure_synapse_test.go`
- example: `examples/azure/synapse-2021-06-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `workspaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `workspaces/firewallRules` | proxy child | create/update, get, delete, list |
| `workspaces/sqlPools` | tracked child | create, get, update (PATCH), delete, list, `pause`, `resume` |
| `workspaces/bigDataPools` | tracked child | create/update, get, update (PATCH), delete, list |
| `checkNameAvailability` | subscription action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- workspace names are 1-50 lowercase letters, digits or hyphens, cannot start or end with a hyphen and cannot end with `-ondemand` (400 `InvalidResourceName`)
- workspaces need `defaultDataLakeStorage.accountUrl` and `filesystem`; `sqlAdministratorLoginPassword` is accepted but never returned
- `connectivityEndpoints` (`web`, `dev`, `sql`, `sqlOnDemand`), `workspaceUID` and a default `managedResourceGroupName` are server-owned
- firewall rules need IPv4 start and end addresses with start not after end, as for the database flexible servers
- SQL pools need a `DW` SKU such as `DW100c`, default to collation `SQL_Latin1_General_CP1_CI_AS` and report `status` `Online`; `pause` and `resume` switch it between `Paused` and `Online`, and PUT and PATCH never change it
- Spark pools check `nodeSize` (`Small` to `XXXLarge`), `sparkVersion` (`3.3` or `3.4`) and `nodeCount` (3 to 200)
- `checkNameAvailability` answers with `available` and compares names across all subscriptions, because workspace endpoints are DNS names

Out of scope for this stage:
- the Synapse data plane (pipelines, notebooks, SQL and Spark execution)
- integration runtimes, managed private endpoints, Microsoft Entra administrators, keys, Data Explorer pools, SQL pool schemas, auditing, security and restore points
- long-running operation polling: create, pause and resume complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSynapse -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service synapse`
- `python3 scripts/azure-io-contract-coverage.py --service synapse`
- `python3 scripts/azure-doc-contract-coverage.py --service synapse`
