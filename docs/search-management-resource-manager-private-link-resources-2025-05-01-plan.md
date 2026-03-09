# Azure Search Management - Resource Manager - Private Link Resources (`search-management-resource-manager-private-link-resources-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Private Link Resources (`2025-05-01`) with deterministic local behavior for supported private link resource discovery.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-link-resources?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-link-resources/list-supported?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full private link provisioning workflows in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateLinkResources`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Private Link Resources operations documented for this API version:

- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateLinkResources`

Common contract characteristics from operation pages:
- Required path identifiers include `subscriptionId`, `resourceGroupName`, and `searchServiceName`.
- Success response returns `PrivateLinkResourcesResult` with `value` and optional `nextLink`.
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Private Link Resources router surface.
- Recognize documented ARM route under `/azure/subscriptions/.../providers/Microsoft.Search/searchServices/.../privateLinkResources`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests for the documented operation.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate ARM identifiers (`subscriptionId`, `resourceGroupName`, `searchServiceName`).

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Resource Fixtures

- Implement deterministic `PrivateLinkResourcesResult` fixture payloads.
- Preserve stable ordering and continuation behavior for `nextLink`.

Exit criteria:
- Success response shape is deterministic and contract-tested.

## Stage 3: Error and Pagination Fixtures

- Add deterministic authorization/validation error fixtures.
- Add deterministic pagination edge-case fixtures for continuation paths.

Exit criteria:
- Negative and continuation-path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-private-link-resources-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
