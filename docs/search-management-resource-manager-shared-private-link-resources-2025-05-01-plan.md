# Azure Search Management - Resource Manager - Shared Private Link Resources (`search-management-resource-manager-shared-private-link-resources-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Shared Private Link Resources (`2025-05-01`) with deterministic local behavior for create/update, delete, get, and list workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/shared-private-link-resources?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/shared-private-link-resources/create-or-update?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/shared-private-link-resources/delete?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/shared-private-link-resources/get?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/shared-private-link-resources/list-by-search-service?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement real private endpoint approval orchestration in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/sharedPrivateLinkResources`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Shared Private Link Resources operations documented for this API version:

- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}`
- `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/sharedPrivateLinkResources`

Common contract characteristics from operation pages:
- Required path identifiers include `subscriptionId`, `resourceGroupName`, `searchServiceName`, and optional `sharedPrivateLinkResourceName`.
- Create/update and delete can return long-running style statuses (`202 Accepted`) in addition to synchronous statuses.
- List returns a typed collection envelope with `value` and possible continuation.
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Shared Private Link Resources router surface.
- Recognize documented ARM routes under `/azure/subscriptions/.../providers/Microsoft.Search/searchServices/.../sharedPrivateLinkResources...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate ARM identifiers (`subscriptionId`, `resourceGroupName`, `searchServiceName`, `sharedPrivateLinkResourceName`).
- Validate create/update payload shape (`privateLinkResourceId`, `groupId`, `requestMessage`).

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Resource Fixtures

- Implement deterministic create/get/list/delete fixtures for shared private link resources.
- Return stable payload shape for resource and list responses.

Exit criteria:
- Success-path behavior is deterministic and contract-tested.

## Stage 3: Error and LRO Fixtures

- Add deterministic not-found/conflict/validation fixtures.
- Add deterministic long-running operation fixtures for create/delete flows.

Exit criteria:
- Negative and LRO-path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-shared-private-link-resources-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
