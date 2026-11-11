# Azure Extended Location (`extendedlocation_2021_08_15`) Staged Plan

## Objective

Implement an emulated Azure Extended Location Resource Manager surface for API version `2021-08-15`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/extendedlocation-2021-08-15`

This plan is based on full traversal of the Extended Location operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/operation-groups?view=rest-extendedlocation-2021-08-15`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations?view=rest-extendedlocation-2021-08-15`

## Reviewed Methods

- custom-locations:
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/create-or-update?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/delete?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/get?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/list-by-resource-group?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/list-by-subscription?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/list-enabled-resource-types?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/list-operations?view=rest-extendedlocation-2021-08-15`
  - `https://learn.microsoft.com/en-us/rest/api/extendedlocation/custom-locations/update?view=rest-extendedlocation-2021-08-15`

Total reviewed methods: `8`.

## Service Naming

- service key: `extendedlocation_2021_08_15`
- display name: `Azure Extended Location`
- API version: `2021-08-15`

Recommended repo paths:

- implementation: `internal/server/provider_azure_extendedlocation_2021_08_15.go`
- tests: `internal/server/provider_azure_extendedlocation_2021_08_15_test.go`
- example: `examples/azure/extendedlocation-2021-08-15`
- plan doc: `docs/extendedlocation-2021-08-15-plan.md`

## Route Envelope

Management-plane route families include:

- `/azure/providers/Microsoft.ExtendedLocation/...`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ExtendedLocation/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/...`

Representative routes:

- `/azure/providers/Microsoft.ExtendedLocation/operations?api-version=2021-08-15`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ExtendedLocation/customLocations?api-version=2021-08-15`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}?api-version=2021-08-15`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ExtendedLocation/customLocations/{resourceName}/enabledResourceTypes?api-version=2021-08-15`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2021-08-15`
- reviewed methods document `Other Status Codes` with `ErrorResponse` envelopes

Observed response patterns:

- `200` appears in `7` methods
- `201` appears in `1` methods
- `202` appears in `1` methods
- `204` appears in `1` methods

Request-body coverage from reviewed pages:

- `2` methods include request bodies
- body-bearing methods:
  - `custom-locations/create-or-update`
  - `custom-locations/update`

Operation-group method counts:

- custom-locations: `8`

## SDK Reference

- Azure Go SDK runtime/pipeline used by the example:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_extendedlocation_2021_08_15.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `extendedlocation_2021_08_15 -> extendedlocation-2021-08-15-plan.md`

### Stage 1: Request Envelope Validation

- validate required path/query parameters per method
- validate create/update request body requirements for custom locations
- normalize `ErrorResponse` payloads for invalid requests

### Stage 2: Route Families

- implement the `customLocations` family incrementally:
  - create-or-update / get / update / delete
  - list-by-resource-group / list-by-subscription
  - list-enabled-resource-types
  - list-operations

### Stage 3: Deterministic Fixtures + State

- add deterministic fixtures for all list/get/create/update/delete paths
- maintain in-memory state keyed by custom location ARM ID
- include deterministic enabled-resource-types fixture results

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_extendedlocation_2021_08_15_test.go` covering:
  - route matching and API version validation
  - success paths for all methods
  - negative scenarios for path/query/body validation
  - long-running delete (`202`/`204`) semantics

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `extendedlocation_2021_08_15`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/extendedlocation-2021-08-15/main.go` executes representative calls across the custom locations surface, including create/get/list/update/delete plus operations and enabled-resource-types listing.
