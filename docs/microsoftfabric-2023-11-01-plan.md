# Azure Microsoft Fabric (`microsoftfabric_2023_11_01`) Staged Plan

## Objective

Implement an emulated Azure Microsoft Fabric Resource Manager surface for API version `2023-11-01`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/microsoftfabric-2023-11-01`

This plan is based on full traversal of the Microsoft Fabric operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/operation-groups?view=rest-microsoftfabric-2023-11-01`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities?view=rest-microsoftfabric-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/operations?view=rest-microsoftfabric-2023-11-01`

## Reviewed Methods

- fabric-capacities:
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/check-name-availability?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/create-or-update?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/delete?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/get?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/list-by-resource-group?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/list-by-subscription?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/list-skus?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/list-skus-for-capacity?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/resume?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/suspend?view=rest-microsoftfabric-2023-11-01`
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/fabric-capacities/update?view=rest-microsoftfabric-2023-11-01`
- operations:
  - `https://learn.microsoft.com/en-us/rest/api/microsoftfabric/operations/list?view=rest-microsoftfabric-2023-11-01`

Total reviewed methods: `12`.

## Service Naming

- service key: `microsoftfabric_2023_11_01`
- display name: `Azure Microsoft Fabric`
- API version: `2023-11-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_microsoftfabric_2023_11_01.go`
- tests: `internal/server/provider_azure_microsoftfabric_2023_11_01_test.go`
- example: `examples/azure/microsoftfabric-2023-11-01`
- plan doc: `docs/microsoftfabric-2023-11-01-plan.md`

## Route Envelope

Management-plane route families include:

- `/azure/providers/Microsoft.Fabric/...`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Fabric/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/...`

Representative routes:

- `/azure/providers/Microsoft.Fabric/operations?api-version=2023-11-01`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Fabric/skus?api-version=2023-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}?api-version=2023-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Fabric/capacities/{capacityName}/resume?api-version=2023-11-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2023-11-01`
- reviewed methods document `Other Status Codes` with `ErrorResponse` envelopes

Observed response patterns:

- `200` appears in `11` methods
- `201` appears in `1` methods
- `202` appears in `4` methods
- `204` appears in `1` methods

Request-body coverage from reviewed pages:

- `3` methods include request bodies
- body-bearing methods:
  - `fabric-capacities/check-name-availability`
  - `fabric-capacities/create-or-update`
  - `fabric-capacities/update`

Operation-group method counts:

- fabric-capacities: `11`
- operations: `1`

## SDK Reference

- Azure Go SDK runtime/pipeline used by the example:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_microsoftfabric_2023_11_01.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `microsoftfabric_2023_11_01 -> microsoftfabric-2023-11-01-plan.md`

### Stage 1: Request Envelope Validation

- validate required path/query parameters per method
- validate create/update/check-name request body requirements
- normalize `ErrorResponse` payloads for invalid requests

### Stage 2: Route Families

- implement operation groups incrementally:
  - fabric-capacities
  - operations

### Stage 3: Deterministic Fixtures + State

- add deterministic fixtures for capacity CRUD/list, skus, suspend/resume
- maintain in-memory state keyed by capacity ARM ID
- include deterministic long-running operation responses for `202` methods

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_microsoftfabric_2023_11_01_test.go` covering:
  - route matching and API version validation
  - success paths for each method
  - negative scenarios for body/param validation
  - async delete/suspend/resume semantics

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `microsoftfabric_2023_11_01`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/microsoftfabric-2023-11-01/main.go` executes representative calls across both operation groups, including create/get/list/update/delete plus suspend/resume and SKU/operations discovery calls.
