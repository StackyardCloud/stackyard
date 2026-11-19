# Azure Cosmos DB for PostgreSQL (`postgresqlhsc-2023-03-02-preview`) Staged Plan

## Objective

Emulate Azure Cosmos DB for PostgreSQL cluster management (`Microsoft.DBforPostgreSQL/serverGroupsv2`, formerly Hyperscale (Citus)) on the shared ARM handler (`internal/server/azure_resource_manager.go`): clusters with their servers, firewall rules, roles, server parameters and private endpoint connections, plus start, stop, restart, read replica promotion, point-in-time restore and the name availability check. No PostgreSQL runs.

## References

Microsoft Learn pages for the `2023-03-02-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/` (property values, password rules)
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/create`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/update`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/delete`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/list`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/start`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/stop`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/restart`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/clusters/promote-read-replica`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/servers/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/servers/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/get-coordinator`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/get-node`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/list-by-server`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/update-on-coordinator`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/configurations/update-on-node`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/firewall-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/firewall-rules/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/firewall-rules/delete`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/firewall-rules/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/roles/create`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/roles/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/roles/delete`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/roles/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/private-endpoint-connections/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/private-endpoint-connections/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/private-endpoint-connections/delete`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/private-endpoint-connections/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/private-link-resources/get`
- `https://learn.microsoft.com/en-us/rest/api/postgresqlhsc/private-link-resources/list-by-cluster`
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/resources-compute` (compute and storage options)
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/reference-limits` (name length, storage scaling, burstable, Entra ID and geo-backup limits)
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/reference-versions` (PostgreSQL versions, retirement dates, Citus versions)
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/howto-scale-grow` (worker nodes and storage can't decrease)
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/concepts-read-replicas` (replica rules and promotion)
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/concepts-backup` (35-day restore window)
- `https://learn.microsoft.com/en-us/azure/cosmos-db/postgresql/reference-parameters` (server parameters)

## Service Naming

- service key: `postgresqlhsc`
- display name: `Azure Cosmos DB for PostgreSQL`
- implementation: `internal/server/provider_azure_postgresqlhsc.go`, `internal/server/provider_azure_postgresqlhsc_validate.go`
- tests: `internal/server/provider_azure_postgresqlhsc_test.go`
- example: `examples/azure/postgresqlhsc-2023-03-02-preview/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `Microsoft.DBforPostgreSQL/serverGroupsv2` | create, get, update (PATCH), delete, list by resource group and subscription; POST `start`, `stop`, `restart`, `promote` |
| `serverGroupsv2/{name}/servers` | get, list |
| `serverGroupsv2/{name}/configurations` | get, list |
| `serverGroupsv2/{name}/servers/{server}/configurations` | list |
| `serverGroupsv2/{name}/coordinatorConfigurations`, `nodeConfigurations` | get, update (PUT) |
| `serverGroupsv2/{name}/firewallRules` | create or update, get, delete, list |
| `serverGroupsv2/{name}/roles` | create, get, delete, list |
| `serverGroupsv2/{name}/privateEndpointConnections` | create or update, get, delete, list |
| `serverGroupsv2/{name}/privateLinkResources` | get, list |
| `/subscriptions/{id}/providers/Microsoft.DBforPostgreSQL/checkNameAvailability` | POST, shared with flexible servers and chosen by `type` |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- cluster names are 3 to 40 lowercase letters, numbers and hyphens, not only numbers and not starting or ending with a hyphen; they're unique across subscriptions, and `checkNameAvailability` with the `serverGroupsv2` type reports invalid and used names
- a new cluster needs `administratorLoginPassword` (unless `authConfig.passwordAuth` is `disabled`), `coordinatorServerEdition`, `coordinatorStorageQuotaInMb`, `coordinatorVCores` and `nodeCount`; the password is never returned. As with flexible servers, every PUT of a cluster that isn't a restore or replica must carry the password, since the handler can't tell a create from an update before validating; PATCH needs none
- passwords, for the cluster and for roles, are 8 to 128 characters with uppercase and lowercase letters, numbers and non-alphanumeric characters, and can't contain three or more consecutive alphanumeric characters of the role name (`citus` for the cluster). Learn's own examples use `password`, which these rules reject
- `postgresqlVersion` defaults to `16`; versions are 11 to 16, and a new cluster can't use a version past its Learn retirement date. `citusVersion` defaults to the version Learn lists for the PostgreSQL version (Citus 12 for 14 to 16, 11 for 13) and its major version must match
- compute and storage follow the Learn options. Single node (`nodeCount` 0): `BurstableMemoryOptimized` with 1 vCore or `BurstableGeneralPurpose` with 2, on 32, 64 or 128 GiB, or `GeneralPurpose` with 2 to 64 vCores on 128 GiB to 2 TiB (1 and 2 TiB need 8 vCores). Multi-node (2 to 20 workers): a `GeneralPurpose` coordinator with 4 to 96 vCores on 128 GiB to 32 TiB, and `MemoryOptimized` workers with 4 to 104 vCores on 0.5 to 32 TiB
- burstable clusters have no high availability, geo-redundant backup, read replicas or version upgrades
- worker node count and storage can only grow; PostgreSQL and Citus versions can only be upgraded, and only to versions that aren't retired (a version upgrade without `citusVersion` takes the new version's Citus)
- `databaseName` (default `citus`), `enableGeoBackup`, `dataEncryption`, `pointInTimeUTC`, `sourceResourceId` and `sourceLocation` can only be set at creation; PATCH takes `ClusterForUpdate`'s properties only, and a PUT that omits settings keeps them
- customer-managed key encryption needs a key URI and one of the cluster's user-assigned identities, and can't be combined with geo-redundant backup
- Microsoft Entra ID authentication needs PostgreSQL 14 or later, and at least one authentication method must be enabled; `aadAuthEnabled` and `passwordEnabled` report them
- a read replica (`sourceResourceId` without `pointInTimeUTC`) copies its source's version, compute, storage and node count; its source must exist, can't be a replica or burstable, and has at most five replicas. A replica's storage and node count can't change, but its compute can. `sourceLocation` must be the source's region
- a restore (`sourceResourceId` with `pointInTimeUTC`) copies its source's settings; the point must be within the last 35 days, after the source was created and not in the future. `earliestRestoreTime` reports the start of that window
- creation-only IDs and regions compare without regard to case, so a PUT that echoes them back in other casing is accepted
- `promote` turns a read replica into an independent cluster (it loses its source), optionally enabling geo-redundant backup (not with a customer-managed key), and answers 202; restored and regular clusters can't be promoted
- `start` and `stop` set `state` to `Ready` or `Stopped` and answer 200; `restart` answers 202
- servers are `{cluster}-c` (coordinator) and `{cluster}-w{n}` (workers), with host names under `.postgres.database.azure.com`; they, `serverNames` and the private link resources (`coordinator`, `worker-{n}`) follow the node count
- server parameters exist for both roles from creation, can be read and set but not deleted, and report `system-default` or `user-override`; values must be one of the allowed values or within the allowed integer range. The cluster-wide view lists each parameter's coordinator and worker values, and a server's view lists its role's values
- firewall rules take IPv4 start and end addresses with the start no greater than the end
- roles need a password unless they use an external Microsoft Entra identity (`objectId` and a `principalType` of `user` or `servicePrincipal`; groups aren't supported), which needs a cluster with Entra ID authentication; password roles need password authentication. `roleType` defaults to `user`
- private endpoint connections take a connection state of `Pending`, `Approved` or `Rejected` and are listed on the cluster
- deletes of firewall rules, roles and private endpoint connections answer 204, which Learn lists for those operations

Out of scope for this stage:
- running PostgreSQL, connecting to servers and replicating data
- `readReplicas` on the source cluster: replicas aren't children of their source, so the list isn't kept
- server parameters beyond the subset in `azurePGParameters`, and `requiresRestart`
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzurePG -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service postgresqlhsc`
- `python3 scripts/azure-io-contract-coverage.py --service postgresqlhsc`
- `python3 scripts/azure-doc-contract-coverage.py --service postgresqlhsc`
