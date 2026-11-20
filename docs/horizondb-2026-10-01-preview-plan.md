# Azure HorizonDB (`horizondb-2026-10-01-preview`) Staged Plan

## Objective

Emulate the Azure HorizonDB resource provider (`Microsoft.HorizonDb`) on the shared ARM handler (`internal/server/azure_resource_manager.go`):

- **Clusters:** create (including point-in-time restore), replace, update, delete and failover, with their service-generated endpoints, private endpoint connections and private link resource.
- **Pools:** the cluster's pool with its nodes, firewall rules, maintenance events and power actions (start, stop, restart).
- **Settings:** parameter groups and their pool connections, the cluster's two authentication methods, Microsoft Entra administrators, log capture configuration and log files.
- **Regional catalogs:** the capability catalog and the default parameter groups.
- **Change data capture:** the cluster's capability, Microsoft Fabric destinations with their identity assignments, and Fabric Mirroring configurations.

No PostgreSQL runs. Every write completes at once and lifecycle states settle immediately.

## References

Microsoft Learn pages for the `2026-10-01-preview` API version, the only version Learn documents:

- `https://learn.microsoft.com/en-us/rest/api/horizondb/` (API overview)
- `https://learn.microsoft.com/en-us/rest/api/horizondb/operations/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/capabilities/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/capabilities/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/default-parameter-groups/list-by-subscription-and-region`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/default-parameter-groups/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/authentications/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-authentications/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-authentications/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-authentications/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/password-authentications/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/password-authentications/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/password-authentications/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/endpoints/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/endpoints/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/clusters/failover`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/log-capture-configurations/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/log-capture-configurations/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/log-capture-configurations/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/log-files/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/log-files/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/log-files/list-download-urls`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-administrators/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-administrators/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-administrators/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/microsoft-entra-administrators/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/firewall-rules/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/firewall-rules/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/firewall-rules/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/firewall-rules/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/maintenance-events/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/maintenance-events/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/maintenance-events/apply-now`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/maintenance-events/reschedule`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/nodes/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/nodes/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/restart`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/start`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/pools/stop`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/private-endpoint-connections/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/private-endpoint-connections/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/private-endpoint-connections/approve-or-reject`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/private-endpoint-connections/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/private-link-resources/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/private-link-resources/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/parameter-groups/list-connections`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-capabilities/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-capabilities/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-capabilities/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-capabilities/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-capabilities/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-destinations/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-destinations/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-destinations/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-destinations/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-destinations/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-destinations/activate-identity`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/identity-assignments/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/identity-assignments/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/identity-assignments/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/identity-assignments/delete`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-configurations/list`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-configurations/get`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-configurations/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-configurations/update`
- `https://learn.microsoft.com/en-us/rest/api/horizondb/change-data-capture-configurations/delete`

## Service Naming

- service key: `horizondb`
- display name: `Azure HorizonDB`
- implementation:
  - `internal/server/provider_azure_horizondb.go` (clusters, endpoints, private link, operations)
  - `internal/server/provider_azure_horizondb_pools.go` (pools, nodes, firewall rules, maintenance events, log files)
  - `internal/server/provider_azure_horizondb_settings.go` (parameter groups, catalogs, authentication, Entra administrators, log capture)
  - `internal/server/provider_azure_horizondb_cdc.go` (change data capture)
- shared handler: `NoPut`, `ValidatePut` (409 on PUT), `Deleted`, a server-aware `ValidateDelete`, `ETag` (an `etag` that changes with every write) and `IfMatchActions` (actions needing a matching `If-Match`) on `azureARMResourceType`
- tests: `internal/server/provider_azure_horizondb_test.go`, `internal/server/provider_azure_horizondb_pools_test.go`, `internal/server/provider_azure_horizondb_cdc_test.go`
- example: `examples/azure/horizondb-2026-10-01-preview/`

## Operations In Scope

| Resource | Operations |
|---|---|
| Operations | list |
| Clusters | create or replace, get, update, delete, list by resource group and subscription, failover |
| Endpoints | list, get |
| Authentications | list; password and Microsoft Entra methods: get, create or replace, update |
| Microsoft Entra administrators | create or replace, get, list, delete |
| Pools | create or replace, get, update, delete, list, start, stop, restart |
| Nodes | list, get |
| Firewall rules | create or replace, get, list, delete |
| Maintenance events | list (with `$filter`), get, apply now, reschedule |
| Private endpoint connections | approve or reject, get, list, delete |
| Private link resources | list, get |
| Log capture configurations | list, get, update |
| Log files | list, get, list download URLs |
| Parameter groups | create or replace, get, update, delete, list by resource group and subscription, list connections |
| Default parameter groups | list by subscription and region, get |
| Capabilities | list, get |
| Change data capture capability (`default`) | create or replace, get, update, delete, list |
| Change data capture destinations | create or replace, get, update, delete, list, activate identity |
| Identity assignments | create or replace, get, delete, list |
| Change data capture configurations | create or replace, get, update, delete, list |

## Behavior

### Clusters

- Names are 1 to 63 lowercase letters, digits and hyphens, starting and ending with a letter or digit.
- `postgreSqlVersion` is 17 or 18 and `backup.geoRedundancy` is required for a new cluster. `creationMode` defaults to `Create`, `network.publicAccess` to `Disabled`, `backup.retentionDays` (1–35) to 7, and `dataEncryption` to `SystemManaged`.
- `postgreSqlVersion`, `creationMode`, `pointInTimeUtc`, `creationSourceClusterResourceId`, `creationSourceClusterLocation` and `backup.geoRedundancy` are fixed at creation: a replacement may omit or repeat them, and changing one is rejected. PATCH accepts only the update model (`network`, `backup.retentionDays`, `dataEncryption`, `aiModelManagement`, `tags`, `identity`).
- **Point-in-time restore** needs `pointInTimeUtc` and an existing source cluster, and must omit `postgreSqlVersion`; the version and geo-redundancy come from the source. Same-location restores need a point between the source's creation and now. Cross-location restores need `creationSourceClusterLocation`, which must be the source's location, and the response omits `pointInTimeUtc`. Restore-only properties are rejected for `Create`.
- **Data encryption:** the type can't switch after creation. `AzureKeyVault` needs a versionless key URI and a user-assigned identity that is also in `identity.userAssignedIdentities`, needs geo-redundancy `Disabled`, and an update may change the key or the identity, not both. `SystemManaged` takes neither. Customer-managed keys report `primaryEncryptionKeyStatus` `Valid`.
- **Identity:** omitting it keeps the assignments; `None` clears them; assigned identities get deterministic `principalId` and `clientId` values.
- `readWritePool` names the cluster's pool once it has one. Failover needs that pool `Ready` with at least two nodes, otherwise 409.
- Each cluster starts with both authentication methods `Disabled` and `PostgreSqlServer` log capture `Enabled` for 7 days. Deleting the cluster deletes everything under it.
- **Endpoints:** the cluster's read-write endpoint is named after the cluster (`<cluster>.<clusterid>.<region>.horizondb.azure.com`), and each pool adds a read-only endpoint named after it (`<pool>.<cluster>...`).

### Pools, nodes and maintenance

- One pool per cluster (409 otherwise), named differently from its cluster.
- `computeModel` (`Provisioned`, AMD or Intel, generation 1–3 and a vCores size from the capability catalog: 2, 4, 8, 16, 32, 48, 64, 96, 128 or 192), `nodeCount` (1–16) and `highAvailability.mode` are required. `Disabled` goes with one node, `AnyZone` or `ZoneRedundant` with 2–16. PATCH merges first and then checks the effective count and mode.
- A new pool gets the cluster's default parameter group (`default-postgresql-{version}-provisioned`, applied `Immediate`) and may only name that one. Later updates may switch to a customer parameter group of the same PostgreSQL version.
- Start, stop and restart answer 200 with the pool. Start and stop are no-ops in their target state, restart needs a `Ready` pool, and anything else is 409. A stopped pool lists no nodes, and replacing a pool keeps its state.
- Nodes `node-01`…`node-NN`: `node-01` holds the read-write role, and zone-redundant pools spread across zones 1–3.
- **Firewall rules:** dotted-decimal IPv4 start and end with start ≤ end, and a description of up to 1024 characters.
- **Maintenance:** each pool starts with one planned, deferrable event a week out. Reschedule moves it to a future time after its current start and no later than its deferral deadline, and `rescheduledFromUtc` keeps the original start. Apply now starts it. Both need a planned or rescheduled event (otherwise 409) and answer with the action response. `$filter` supports `properties/status eq '…'` clauses joined with `or`.
- Deleting a pool deletes its firewall rules and maintenance events. Deleting the last pool rejects the cluster's private endpoint connections with a description saying so.

### Settings

- **Private endpoint connections:** approval needs a pool (409 `PrivateEndpointRequiresPool`), while rejection is always allowed. `groupIds` is the cluster name, and the private link resource carries the `privatelink.horizondb.azure.com` zone.
- **Password authentication:** setting the login or password needs `state` `Enabled` in the same request, and disabling can't carry a password. Enabling for the first time needs `administratorLogin` and `administratorLoginPassword` (a PUT that sends the login needs the password), which can't change afterwards and can't be sent by PATCH. A PATCH that only resets the password needs the method enabled (409 otherwise). The password is write-only.
- **Microsoft Entra authentication:** takes only `state`.
- **Microsoft Entra administrators:** named by a GUID object ID. They need `principalName` (up to 256 characters) and `principalType`, and `tenantId` defaults to the subscription's tenant. These properties can't change afterwards.
- **Log capture:** PATCH-only `state` and `retentionDays` (1–35). Each pool adds a server log file that belongs to its cluster, and only `listDownloadUrls` returns its signed URL, valid for one hour.
- **Parameter groups:**
  - Names are 2 to 63 letters, digits and hyphens.
  - `postgreSqlVersion` is required and fixed.
  - Overrides must name writable catalog parameters once each, with values that fit their type (Boolean, Integer and Enumeration ranges, Numeric bounds, String patterns).
  - Responses list the whole catalog with `valueSource`, and a GET body can be sent back as is.
  - PUT replaces the overrides; PATCH overlays the ones it sends on the stored ones by name.
  - Deleting a group that pools use is rejected with 409. `connections` lists the pools that use it.
- **Catalogs:** the default groups (one per PostgreSQL version) and capabilities (recommending version 17 on AMD) are the same in every location.

### Change data capture

- Names are 1 to 63 lowercase letters, digits and hyphens; the capability is always `default`. Every change data capture resource returns an `etag` that changes with each write.
- `desiredState` (`Enabled` or `Disabled`) is required on every PUT; PATCH accepts only `desiredState` (plus `autoUpgradeExtension` on the capability). Server-owned properties a client sends are ignored.
- **Capability:** reports the extension (`Ready` once enabled), `ComputeBased` replication, prerequisites (the read-write endpoint is `Ready` once the cluster has a pool) and the cluster's destination and configuration counts. Destinations and configurations need it (409 `ChangeDataCaptureNotConfigured`), and it can only be deleted after them (409).
- **Destinations:** `destination` needs `destinationType` `Fabric`, `protocol` `Native` and a `workspaceId`, and can't change afterwards. An enabled destination is `WaitingForIdentity` until an identity assignment is activated, then `Enabled`. It reports how many configurations use it, and can only be deleted after them and its identity assignments (409).
- **Identity assignments:** `identityResourceId` must exactly match a key in the cluster's `identity.userAssignedIdentities` and can't change afterwards. Validation succeeds at once, the principal ID is the one the cluster identity reports, and new assignments are `Inactive`. An active assignment can't be deleted (409), and the cluster can't drop an identity an assignment uses (400).
- **Activate identity:** needs an `If-Match` header matching the destination's `etag` (400 when missing, 412 when stale, `*` matches any) and an assignment under that destination. It makes that assignment `Active`, the previous one `Inactive`, and reports both; repeating it for the active assignment succeeds without change.
- **Configurations:** `databaseName`, `usageProfile` (`Mirroring`) and `destinationResourceId` are required and fixed. The destination must be under the same cluster and `Enabled` when the configuration is created. A configuration gets a stable `logicalStreamId`, an `Active` topology binding on its cluster, and a `NotConnected` replication session while enabled.

## Out Of Scope

- **Timing:** asynchronous provisioning states and `Azure-AsyncOperation` polling. Every write completes at once, and deletes answer 204.
- **Failover:** moving the read-write role. `node-01` keeps it after failover.
- **Parameter sync:** `PendingReboot` sync status. Parameter changes apply at once, so pools stay `InSync`.
- **Identity PATCH merging:** PATCH replaces `identity.userAssignedIdentities` the way PUT does.
- **Parameter catalog:** only a subset of the engine parameters.
- **Log files:** files beyond each pool's first one. Disabling capture doesn't stop log files from being created.
- **Restore scope:** restoring from deleted source clusters.
- **Change data capture sessions:** no Fabric session ever connects, writer moves never supersede a configuration, and `drainExistingSessions` has no effect. `If-Match` is honored only on activate identity.

## Quality Gates

- `go test ./internal/server -run TestAzureHorizonDB -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service horizondb --require-service horizondb --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service horizondb --require-service horizondb --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service horizondb --require-service horizondb --fail-on any`
- the example under `examples/azure/horizondb-2026-10-01-preview/` against a local server
