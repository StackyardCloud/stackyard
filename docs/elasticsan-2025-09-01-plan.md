# Azure Elastic SAN (`elasticsan_2025_09_01`) Staged Plan

## Objective

Emulate Azure Elastic SAN Resource Manager REST API (`api-version=2025-09-01`) with deterministic local behavior for:

- elastic SAN lifecycle (create/get/list/update/delete)
- private endpoint connection lifecycle
- private link resources and SKU discovery
- volume group lifecycle
- volume snapshot lifecycle
- volume lifecycle plus `preBackup` and `preRestore` actions
- provider operations discovery endpoint
- an Azure Go SDK example under `examples/azure/elasticsan-2025-09-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/operation-groups?view=rest-elasticsan-2025-09-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/operations?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-endpoint-connections?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-link-resources?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/skus?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-groups?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes?view=rest-elasticsan-2025-09-01`

Method pages reviewed directly: all linked methods from all operation groups (`29` total).

Elastic Sans methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/create?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/delete?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/get?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/list-by-resource-group?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/list-by-subscription?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/elastic-sans/update?view=rest-elasticsan-2025-09-01`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/operations/list?view=rest-elasticsan-2025-09-01`

Private Endpoint Connections methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-endpoint-connections/create?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-endpoint-connections/delete?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-endpoint-connections/get?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-endpoint-connections/list?view=rest-elasticsan-2025-09-01`

Private Link Resources methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/private-link-resources/list-by-elastic-san?view=rest-elasticsan-2025-09-01`

Skus methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/skus/list?view=rest-elasticsan-2025-09-01`

Volume Groups methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-groups/create?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-groups/delete?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-groups/get?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-groups/list-by-elastic-san?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-groups/update?view=rest-elasticsan-2025-09-01`

Volume Snapshots methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots/create?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots/delete?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots/get?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volume-snapshots/list-by-volume-group?view=rest-elasticsan-2025-09-01`

Volumes methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/create?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/delete?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/get?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/list-by-volume-group?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/pre-backup?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/pre-restore?view=rest-elasticsan-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/elasticsan/volumes/update?view=rest-elasticsan-2025-09-01`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan`

## Surface Summary

Reviewed operation groups and methods:

- Elastic Sans (6)
- Operations (1)
- Private Endpoint Connections (4)
- Private Link Resources (1)
- Skus (1)
- Volume Groups (5)
- Volume Snapshots (4)
- Volumes (7)

Total reviewed methods: `29`.

## Service Naming

Canonical Stackyard naming:

- service key: `elasticsan_2025_09_01`
- display name: `Azure Elastic SAN`
- API version: `2025-09-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_elasticsan_2025_09_01.go`
- tests: `internal/server/provider_azure_elasticsan_2025_09_01_test.go`
- example: `examples/azure/elasticsan-2025-09-01`
- plan doc: `docs/elasticsan-2025-09-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.ElasticSan/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ElasticSan/elasticSans`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ElasticSan/skus`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections/{privateEndpointConnectionName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateLinkResources`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/snapshots/{snapshotName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes/{volumeName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/preBackup`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/preRestore`

Representative routes:

- `GET /azure/providers/Microsoft.ElasticSan/operations?api-version=2025-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}?api-version=2025-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/volumes/{volumeName}?api-version=2025-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/preBackup?api-version=2025-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}/volumegroups/{volumeGroupName}/preRestore?api-version=2025-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ElasticSan/elasticSans/{elasticSanName}?api-version=2025-09-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-09-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods document non-success paths as `Other Status Codes | ErrorResponse`

Observed response patterns:

- `200` appears in `29` methods
- `201` appears in `5` methods
- `202` appears in `10` methods
- `204` appears in `5` methods

Request-body coverage from reviewed pages:

- `10` methods include request bodies
- body-bearing methods:
  - `ElasticSans_Create`
  - `ElasticSans_Update`
  - `PrivateEndpointConnections_Create`
  - `VolumeGroups_Create`
  - `VolumeGroups_Update`
  - `VolumeSnapshots_Create`
  - `Volumes_Create`
  - `Volumes_PreBackup`
  - `Volumes_PreRestore`
  - `Volumes_Update`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add stage-0 provider skeleton at `internal/server/provider_azure_elasticsan_2025_09_01.go`
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key resolves as `elasticsan_2025_09_01`
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.ElasticSan` route families for this API version
- enforce `api-version=2025-09-01` for handled routes
- preserve behavior for unrelated Azure providers

### Stage 2: Provider Ops + Read/List Surfaces

- implement `Operations_List`
- implement elastic SAN read/list routes
- implement private endpoint connection read/list routes
- implement private link resources and SKU list routes
- implement volume group, snapshot, and volume read/list routes

### Stage 3: Mutating Surfaces

- implement `ElasticSans_Create`, `ElasticSans_Update`, `ElasticSans_Delete`
- implement `PrivateEndpointConnections_Create`, `PrivateEndpointConnections_Delete`
- implement `VolumeGroups_Create`, `VolumeGroups_Update`, `VolumeGroups_Delete`
- implement `VolumeSnapshots_Create`, `VolumeSnapshots_Delete`
- implement `Volumes_Create`, `Volumes_Update`, `Volumes_Delete`
- implement `Volumes_PreBackup` and `Volumes_PreRestore`
- support deterministic `200`/`201`/`202`/`204` status behavior per method contract

### Stage 4: Payload and Query Validation

- validate required payload fields for body-bearing methods:
  - elastic SAN size/SKU configuration
  - private endpoint connection state
  - volume group encryption/protocol/network ACL shape
  - snapshot source IDs
  - volume size and managed-by metadata
  - pre-backup volume names and pre-restore disk snapshot IDs
- normalize and validate resource name constraints from URI parameters

### Stage 5: Tests

- add provider tests `internal/server/provider_azure_elasticsan_2025_09_01_test.go` covering:
  - route ownership and API-version validation
  - elastic SAN lifecycle
  - private endpoint connection lifecycle
  - private link resources and SKU discovery routes
  - volume group/snapshot/volume lifecycle
  - pre-backup and pre-restore actions
  - deterministic error envelopes for contract violations
