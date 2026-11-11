# Azure NetApp Files (`netapp-2025-12-01`) Staged Plan

## Objective

Emulate the Azure NetApp Files ARM management API for `2025-12-01` with deterministic local behavior for:

- provider `operations`
- top-level `netAppAccounts` lifecycle and discovery routes
- nested `capacityPools`
- deep storage branches such as `volumes`, `snapshots`, `subvolumes`, `volumeQuotaRules`, and `volumeGroups`
- backup and vault families such as `backupPolicies`, `backupVaults`, `backups`, and scoped backup list routes
- account-, region-, and subscription-scoped utility endpoints such as name/quota/file-path checks, usages, quota limits, and region info
- an Azure Go SDK example under `examples/azure/netapp-2025-12-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure NetApp Files under `Microsoft.NetApp`.
- This is not the data path for NFS/SMB clients and not an ONTAP administration surface.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2025-12-01`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/netapp/operation-groups?view=rest-netapp-2025-12-01`

Representative operation-group and method pages reviewed across the surface:
- `https://learn.microsoft.com/en-us/rest/api/netapp/operations/list?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/accounts/create-or-update?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/accounts/update?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/netapp-resource?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/pools/create-or-update?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/volumes?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/volumes/update?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/snapshots/get?view=rest-netapp-2025-12-01`
- `https://learn.microsoft.com/en-us/rest/api/netapp/volume-groups?view=rest-netapp-2025-12-01`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/netapp/armnetapp/v10`

SDK selection note:
- The closest official Go SDK for the reviewed `2025-12-01` surface is the current preview/staged `armnetapp` generation line.
- The package already exposes clients for the major management families visible on the operation-groups page, including accounts, pools, volumes, snapshots, volume groups, resource checks, backup policies, backups, and quota-related clients.
- Plan assumption: prefer typed `armnetapp` clients where available, and use raw ARM requests only if a reviewed route is missing from the generated package version chosen by the repo at implementation time.

## Service Naming

Canonical Stackyard naming:
- service key: `netapp`
- display name: `Azure NetApp Files`
- provider namespace: `Microsoft.NetApp`
- API version: `2025-12-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_netapp.go`
- tests: `internal/server/provider_azure_netapp_test.go`
- example: `examples/azure/netapp-2025-12-01`
- plan doc: `docs/netapp-2025-12-01-plan.md`

Recommended coverage aliases:
- `netapp`
- `netapp_files`
- `netapp-files`
- `azure_netapp`
- `azure_netapp_files`
- `microsoft_netapp`
- `rest_netapp_2025_12_01`
- `netapp_2025_12_01`

## Surface Summary

The reviewed `2025-12-01` management surface is large and resource-dense. The operation-groups page enumerates:

1. `Operations`
2. `Accounts`
3. `Backup Policies`
4. `Backup Vaults`
5. `Backups`
6. `Backups Under Account`
7. `Backups Under Backup Vault`
8. `Backups Under Volume`
9. `NetApp Resource`
10. `NetApp Resource Quota Limits`
11. `NetApp Resource Quota Limits Account`
12. `NetApp Resource Region Infos`
13. `NetApp Resource Usages`
14. `Pools`
15. `Ransomware Reports`
16. `Snapshot Policies`
17. `Snapshots`
18. `Subvolumes`
19. `Volume Groups`
20. `Volume Quota Rules`
21. `Volumes`

This is too broad to implement as one flat slice. The implementation should be staged by nesting depth and operator usefulness:

- provider and utility checks first
- account and pool lifecycle second
- volume tree third
- backup and quota families after that
- specialized tails such as ransomware reports only once the core hierarchy is stable

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2025-12-01`
- all routes sit under the `Microsoft.NetApp` provider namespace

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.NetApp/operations?api-version=2025-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.NetApp/netAppAccounts/{accountName}?api-version=2025-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.NetApp/netAppAccounts/{accountName}/capacityPools/{poolName}?api-version=2025-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.NetApp/netAppAccounts/{accountName}/capacityPools/{poolName}/volumes/{volumeName}?api-version=2025-12-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.NetApp/netAppAccounts/{accountName}/capacityPools/{poolName}/volumes/{volumeName}/snapshots/{snapshotName}?api-version=2025-12-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.NetApp/netAppAccounts/{accountName}/volumeGroups?api-version=2025-12-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.NetApp/locations/{location}/checkNameAvailability?api-version=2025-12-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.NetApp/locations/{location}/checkQuotaAvailability?api-version=2025-12-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.NetApp/locations/{location}/checkFilePathAvailability?api-version=2025-12-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.NetApp/locations/{location}/regionInfos?api-version=2025-12-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.NetApp/locations/{location}/quotaLimits?api-version=2025-12-01`

Important routing implications from the reviewed pages:
- the hierarchy is deeply nested and path parsing must preserve:
  - `accountName`
  - `poolName`
  - `volumeName`
  - child resource names such as snapshot, subvolume, quota rule, and backup identifiers
- many create and update operations are ARM long-running operations with `202 Accepted`, `Location`, and `Azure-AsyncOperation`
- utility endpoints are not all under resource groups; some are subscription-location scoped
- account- and backup-vault-scoped backup list routes overlap with the deep volume backup hierarchy and should share a canonical backup store

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- resources return standard ARM fields:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `etag`
  - `systemData`
  - `properties.*`

### Operations

`Operations.List`
- route: `/providers/Microsoft.NetApp/operations`
- returns `OperationListResult`
- sample payload includes:
  - `name`
  - `display.provider`
  - `display.resource`
  - `display.operation`
  - `display.description`
  - `origin`
  - `serviceSpecification`

Modeling implication:
- expose a deterministic `value` list covering every resource family implemented in each stage, plus the major utility and action routes

### Accounts

Reviewed method pages:
- `Create Or Update`
- `Update`

Key traits from the reviewed pages:
- top-level route:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.NetApp/netAppAccounts/{accountName}`
- create/update requires `location`
- properties can include:
  - `activeDirectories`
  - `encryption`
  - `nfsV4IDDomain`
- account create may return `201 Created` with async headers

Modeling implications:
- persist canonical account state keyed by ARM ID
- implement `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-subscription, and list-by-resource-group
- keep the nested directory/encryption structures shallow but stable

### NetApp Resource Utility Endpoints

Reviewed method page:
- `NetApp Resource`

Key traits from the reviewed page:
- this group includes:
  - `Check File Path Availability`
  - `Check Name Availability`
  - `Check Quota Availability`
  - `Query Network Sibling Set`
  - `Query Region Info`
  - `Update Network Sibling Set`

Modeling implications:
- implement the three availability checks early, because they are small and commonly referenced by SDK examples
- model region/network sibling responses as deterministic static fixtures rather than full backend calculations

### Pools

Reviewed method page:
- `Pools - Create Or Update`

Key traits from the reviewed page:
- route:
  - `/netAppAccounts/{accountName}/capacityPools/{poolName}`
- request requires:
  - `location`
- properties include:
  - `size`
  - `serviceLevel`
  - `qosType`
- create may return `201 Created` with async headers

Modeling implications:
- use a canonical child store under accounts
- support `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-account
- validate the pool name pattern and basic required fields

### Volumes

Reviewed method pages:
- `Volumes`
- `Volumes - Update`

Key traits from the reviewed pages:
- route:
  - `/netAppAccounts/{accountName}/capacityPools/{poolName}/volumes/{volumeName}`
- operations include:
  - create/update and patch
  - delete
  - replication, revert, pool change, file-lock break, and other action routes
- the volume surface is the largest and most action-heavy branch

Modeling implications:
- split implementation into:
  - core volume CRUD/list/get
  - essential action placeholders
  - specialized replication and migration tails later
- keep first-pass volume properties focused on contract-critical fields such as:
  - `creationToken`
  - `serviceLevel`
  - `usageThreshold`
  - `protocolTypes`
  - `subnetId`
  - `provisioningState`

### Snapshots

Reviewed method page:
- `Snapshots - Get`

Key traits from the reviewed page:
- route:
  - `/netAppAccounts/{accountName}/capacityPools/{poolName}/volumes/{volumeName}/snapshots/{snapshotName}`
- snapshots are a child resource under a volume

Modeling implications:
- seed snapshots deterministically from the volume tree or create them explicitly in a later stage
- prioritize `GET`, `LIST`, `CREATE`, `DELETE`, and basic restore action coverage before advanced snapshot-policy integration

### Volume Groups

Reviewed method page:
- `Volume Groups`

Key traits from the reviewed page:
- route family hangs under the account:
  - `/netAppAccounts/{accountName}/volumeGroups/{volumeGroupName}`
- operations include:
  - create
  - delete
  - get
  - list by account
- volume groups are account-scoped containers that refer to one or more volumes

Modeling implications:
- represent volume groups with a lightweight relationship to existing volume fixtures
- do not require full cross-resource orchestration in the first pass; deterministic embedded volume references are enough

## Recommended Stages

### Stage 0: Wiring and Coverage Discovery

Goal:
- make the new service discoverable in coverage and reserve the route family

Work:
- add `netapp` aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- add the provider entry point in `internal/server/http.go`
- add the plan doc itself
- add a focused test scaffold

Exit criteria:
- coverage scripts recognize `netapp`
- provider routing claims `Microsoft.NetApp` requests

### Stage 1: Operations + Utility Checks

Goal:
- cover the provider root and the utility endpoints with the highest leverage

Work:
- implement `GET /providers/Microsoft.NetApp/operations`
- implement:
  - `checkNameAvailability`
  - `checkQuotaAvailability`
  - `checkFilePathAvailability`
  - `regionInfos`
  - `usages`
  - `quotaLimits`
- add stable success and validation behavior for location-scoped routes

Exit criteria:
- provider root and utility endpoints are green

### Stage 2: Accounts

Goal:
- establish the top-level resource and ARM patterns for the service

Work:
- implement `netAppAccounts`:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list-by-subscription
  - list-by-resource-group
- validate account names and required location
- seed minimal default account properties used by downstream children

Exit criteria:
- account lifecycle is deterministic and test-covered

### Stage 3: Capacity Pools

Goal:
- add the next layer of the hierarchy

Work:
- implement `capacityPools` under accounts:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list-by-account
- validate `size`, `serviceLevel`, and `qosType` minimally

Exit criteria:
- accounts and pools together cover a stable nested ARM tree

### Stage 4: Core Volumes

Goal:
- implement the main storage unit used by most higher-order families

Work:
- implement `volumes`:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list-by-pool
- model essential properties and a small set of common actions if coverage requires them
- keep replication and external-migration actions as explicit follow-up work unless directly exercised

Exit criteria:
- core volume CRUD/list/get behavior is green without `NotImplemented`

### Stage 5: Snapshots, Subvolumes, and Quota Rules

Goal:
- complete the most common deep child branches

Work:
- implement:
  - `snapshots`
  - `subvolumes`
  - `volumeQuotaRules`
- prefer deterministic list/get/create/delete first
- add restore or metadata actions only when coverage demonstrates the need

Exit criteria:
- the deep volume subtree covers the common child-resource paths

### Stage 6: Volume Groups and Backup Families

Goal:
- cover the remaining resource groups most likely to be used in management automation

Work:
- implement:
  - `volumeGroups`
  - `backupPolicies`
  - `backupVaults`
  - `backups`
  - list-by-account, list-by-backup-vault, and list-by-volume variants
- keep a shared canonical backup record so the multiple scoped list routes agree

Exit criteria:
- account-level storage management coverage is broad enough for the main NetApp Files management stories

### Stage 7: Example and Validation

Goal:
- make the new emulator easy to verify and consume

Work:
- add `examples/azure/netapp-2025-12-01`
- use the official Azure Go SDK package:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/netapp/armnetapp/v10`
- demonstrate:
  - operations list
  - account create/get
  - pool create/get
  - volume create/get
  - one utility check such as `checkNameAvailability`
- finalize coverage aliases and doc mappings
- run targeted service coverage plus repo validation expected for new Azure services

Exit criteria:
- example builds cleanly
- all targeted service endpoints pass the contract, IO, doc, and endpoint coverage scripts

## Implementation Notes

Suggested in-memory layout:

- top-level maps keyed by ARM ID for:
  - accounts
  - pools
  - volumes
  - snapshots
  - subvolumes
  - volume quota rules
  - volume groups
  - backup policies
  - backup vaults
  - backups
- utility fixtures keyed by subscription and location for:
  - usages
  - quota limits
  - region info
  - network sibling sets

Suggested helper structure:

- shared ARM helpers:
  - validate `api-version`
  - parse subscription, resource group, account, pool, and volume identifiers
  - emit ARM errors
- resource helpers:
  - canonical ARM ID/type builders
  - ETag generation
  - nested list filtering
  - async accepted responses for long-running creates and updates

Response-shape strategy:

- prioritize contract-critical required fields and stable nesting over exhaustive reproduction of every optional NetApp property
- keep large property bags compact but realistic, especially for volumes and backups
- return deterministic LRO headers where the reviewed pages document `201` or `202` plus `Location` / `Azure-AsyncOperation`

## Testing Strategy

Add focused tests in `internal/server/provider_azure_netapp_test.go` for:

- operations list
- utility checks:
  - name availability
  - quota availability
  - file path availability
- account create/get/head/patch/delete/list
- pool create/get/head/patch/delete/list
- volume create/get/head/patch/delete/list
- snapshot get/list and at least one create/delete path
- volume group get/list/create/delete
- invalid API version handling
- missing required fields such as location
- missing parent-resource and child-resource handling

Coverage validation target after implementation:

- `python3 scripts/azure-contract-coverage.py --service netapp --require-service netapp --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service netapp --require-service netapp --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service netapp --require-service netapp --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service netapp --require-service netapp --fail-on any`

## Risks and Gaps

- Some Microsoft Learn pages for this service version are intermittently authorization-gated in the browser surface, so implementation may need to confirm tail details from generated SDK types when docs are thin.
- The `2025-12-01` Go SDK package line appears to still be in preview or beta form, so the example may need to pin the closest supported version rather than an exact GA tag.
- The full volume action surface is large; the initial implementation should focus on CRUD/list/get plus a few high-signal actions rather than trying to emulate every replication and migration verb in one pass.

## Definition of Done

The staged implementation should be considered complete when:

- `netapp` is recognized by the Azure coverage scripts
- `Microsoft.NetApp` routes return modeled responses instead of `NotImplemented`
- accounts, pools, and volumes form a deterministic nested ARM hierarchy
- the key utility endpoints and at least one deep child branch are implemented and test-covered
- the Azure Go SDK example builds and exercises the local emulator
- the targeted Azure coverage scripts for `netapp` pass cleanly
