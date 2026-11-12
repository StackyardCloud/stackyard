# Azure Device Registry Resource Manager (`deviceregistry_2026_04_01`) Staged Plan

## Objective

Emulate Azure Device Registry Resource Manager REST API (`api-version=2026-04-01`) with deterministic local behavior for:

- asset endpoint profile, asset, namespace, and schema lifecycle surfaces
- namespace-scoped device/discovered-device/discovered-asset operations
- namespace action/migration and operation-status polling flows
- provider operations and billing container read/list workflows
- an Azure Go SDK example under `examples/azure/deviceregistry-2026-04-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/operation-groups?view=rest-deviceregistry-2026-04-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/billing-containers?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-devices?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-assets?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-devices?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/operation-status?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/operations?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-versions?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schemas?view=rest-deviceregistry-2026-04-01`

Method pages reviewed directly: all linked methods from all operation groups (`58` total).

Asset Endpoint Profiles method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles/list-by-subscription?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/asset-endpoint-profiles/update?view=rest-deviceregistry-2026-04-01`

Assets method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/list-by-subscription?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/update?view=rest-deviceregistry-2026-04-01`

Billing Containers method pages (`2`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/billing-containers/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/billing-containers/list-by-subscription?view=rest-deviceregistry-2026-04-01`

Namespace Assets method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets/execute-action?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-assets/update?view=rest-deviceregistry-2026-04-01`

Namespace Devices method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-devices/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-devices/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-devices/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-devices/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-devices/update?view=rest-deviceregistry-2026-04-01`

Namespace Discovered Assets method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-assets/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-assets/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-assets/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-assets/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-assets/update?view=rest-deviceregistry-2026-04-01`

Namespace Discovered Devices method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-devices/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-devices/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-devices/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-devices/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespace-discovered-devices/update?view=rest-deviceregistry-2026-04-01`

Namespaces method pages (`7`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/list-by-subscription?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/migrate?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/namespaces/update?view=rest-deviceregistry-2026-04-01`

Operation Status method pages (`1`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/operation-status/get?view=rest-deviceregistry-2026-04-01`

Operations method pages (`1`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/operations/list?view=rest-deviceregistry-2026-04-01`

Schema Registries method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries/list-by-resource-group?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries/list-by-subscription?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-registries/update?view=rest-deviceregistry-2026-04-01`

Schema Versions method pages (`4`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-versions/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-versions/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-versions/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schema-versions/list-by-schema?view=rest-deviceregistry-2026-04-01`

Schemas method pages (`4`):

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schemas/create-or-replace?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schemas/delete?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schemas/get?view=rest-deviceregistry-2026-04-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/schemas/list-by-schema-registry?view=rest-deviceregistry-2026-04-01`

SDK parity cross-check used:

- `https://learn.microsoft.com/en-us/rest/api/deviceregistry/assets/create-or-replace?view=rest-deviceregistry-2026-04-01` (Go tab uses `armdeviceregistry`)
- `https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/deviceregistry/armdeviceregistry`

## Surface Summary

Reviewed operation groups and methods:

- Asset Endpoint Profiles (6)
- Assets (6)
- Billing Containers (2)
- Namespace Assets (6)
- Namespace Devices (5)
- Namespace Discovered Assets (5)
- Namespace Discovered Devices (5)
- Namespaces (7)
- Operation Status (1)
- Operations (1)
- Schema Registries (6)
- Schema Versions (4)
- Schemas (4)

Total reviewed methods: `58`.

## Service Naming

Canonical Stackyard naming:

- service key: `deviceregistry_2026_04_01`
- display name: `Azure Device Registry Resource Manager`
- API version: `2026-04-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_deviceregistry_2026_04_01.go`
- tests: `internal/server/provider_azure_deviceregistry_2026_04_01_test.go`
- example: `examples/azure/deviceregistry-2026-04-01`
- plan doc: `docs/deviceregistry-2026-04-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DeviceRegistry/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/billingContainers`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/locations/{location}/operationStatuses/{operationId}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredAssets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/discoveredDevices`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}/schemaVersions`

Representative routes:

- `GET /providers/Microsoft.DeviceRegistry/operations?api-version=2026-04-01`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.DeviceRegistry/locations/{location}/operationStatuses/{operationId}?api-version=2026-04-01`
- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/assets/{assetName}/executeAction?api-version=2026-04-01`
- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/migrate?api-version=2026-04-01`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assetEndpointProfiles/{assetEndpointProfileName}?api-version=2026-04-01`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/assets/{assetName}?api-version=2026-04-01`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/namespaces/{namespaceName}/devices/{deviceName}?api-version=2026-04-01`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}/schemaVersions/{schemaVersionName}?api-version=2026-04-01`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}/schemas/{schemaName}?api-version=2026-04-01`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceRegistry/schemaRegistries/{schemaRegistryName}?api-version=2026-04-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2026-04-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- non-success responses are documented as `Other Status Codes | ErrorResponse`
- list-style methods return pageable list envelopes with `nextLink` semantics

Observed response patterns:

- `200 OK` for read/list and many update/action completions
- `10` methods include `201 Created` (create-or-replace success)
- `20` methods include `202 Accepted` (delete/update/action LRO acceptance)
- `10` methods include `204 No Content` (idempotent delete/no-op delete)
- many `201`/`202` responses document LRO headers (`Azure-AsyncOperation` and/or `Location`, plus `Retry-After`)
- `operation-status/get` provides explicit operation polling endpoint

Selected request-body requirements from reviewed pages:

- `asset-endpoint-profiles/create-or-replace` requires `extendedLocation, location, properties.endpointProfileType, properties.targetAddress`
- `assets/create-or-replace` requires `extendedLocation, location, properties.assetEndpointProfileRef`
- `namespace-assets/create-or-replace` requires `extendedLocation, location, properties.deviceRef`
- `namespace-assets/execute-action` requires `managementActionName, managementGroupName`
- `namespace-devices/create-or-replace` requires `location`
- `namespace-discovered-assets/create-or-replace` requires `extendedLocation, location, properties.deviceRef, properties.discoveryId, properties.version`
- `namespace-discovered-devices/create-or-replace` requires `extendedLocation, location, properties.discoveryId, properties.version`
- `namespaces/create-or-replace` requires `location`
- `schema-registries/create-or-replace` requires `location, properties.namespace, properties.storageAccountContainerUrl`
- `schema-versions/create-or-replace` requires `properties.schemaContent`
- `schemas/create-or-replace` requires `properties.format, properties.schemaType`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `deviceregistry_2026_04_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- add Azure Go SDK example scaffold under `examples/azure/deviceregistry-2026-04-01`

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Device Registry management paths to provider stub
- example compiles and includes calls for the full reviewed method set

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for all `Microsoft.DeviceRegistry` management-plane families
- strict `api-version=2026-04-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Device Registry route families are deterministically owned
- version and method failures are stable and testable

### Stage 2: Core Top-Level Resources

Implement operation groups:

- Operations
- Billing Containers
- Asset Endpoint Profiles
- Assets
- Namespaces
- Operation Status

Behavior:

- deterministic ARM envelopes (`id`, `name`, `type`, `location`, `tags`, `properties`, `systemData`)
- deterministic list ordering and optional paging tokens
- deterministic async envelopes/headers for create-update-delete LRO flows

Exit criteria:

- no `NotImplemented` for top-level resource groups

### Stage 3: Namespace-Scoped Resource Surfaces

Implement operation groups:

- Namespace Assets
- Namespace Devices
- Namespace Discovered Assets
- Namespace Discovered Devices

Behavior:

- deterministic parent-child scoping under namespace
- deterministic execute-action and migrate operation records
- stable update/delete idempotency semantics

Exit criteria:

- all namespace-scoped methods return modeled responses

### Stage 4: Schema and Registry Surfaces

Implement operation groups:

- Schema Registries
- Schemas
- Schema Versions

Behavior:

- deterministic nested schema store keyed by `{schemaRegistryName, schemaName, schemaVersionName}`
- deterministic schema-content payload handling and retrieval
- stable list and deletion semantics across registry/schema/version scopes

Exit criteria:

- all schema methods return modeled responses

### Stage 5: Tests

Add provider tests for:

- api-version and method gating
- top-level and namespace-scoped lifecycle behaviors
- schema registry/schema/schema-version nesting and constraints
- LRO-style status/header behavior (`Location`, `Azure-AsyncOperation`, `Retry-After`)
- deterministic error envelopes for invalid inputs and not-found resources

Exit criteria:

- `go test ./internal/server -run 'TestAzureDeviceRegistry20260401' -count=1` passes

### Stage 6: Example and Validation

Maintain `examples/azure/deviceregistry-2026-04-01` with Azure Go SDK runtime calls spanning all reviewed methods.

Validation:

- run example against local Stackyard Azure endpoint
- verify route ownership, status codes, and envelopes align with reviewed docs

Exit criteria:

- example executes end-to-end against emulated routes once Stage 1+ handlers are implemented
- coverage scripts and plan mapping include the new service key
