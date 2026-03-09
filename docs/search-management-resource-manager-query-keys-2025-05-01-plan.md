# Azure Search Management - Resource Manager - Query Keys (`search-management-resource-manager-query-keys-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Query Keys (`2025-05-01`) with deterministic local behavior for create, delete, and list query key workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/query-keys?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/query-keys/create?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/query-keys/delete?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/query-keys/list-by-search-service?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement real key material generation/rotation in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/createQueryKey/{name}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/deleteQueryKey/{key}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/listQueryKeys`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Query Keys operations documented for this API version:

- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/createQueryKey/{name}`
- `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/deleteQueryKey/{key}`
- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/listQueryKeys`

Common contract characteristics from operation pages:
- Required identifiers include `subscriptionId`, `resourceGroupName`, `searchServiceName`, and operation-specific tokens (`name`, `key`).
- Create returns a typed `QueryKey` payload on success.
- Delete can return success without payload and documents not-found style behavior.
- List returns `QueryKeysListResult` (`value` collection).
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Query Keys router surface.
- Recognize documented ARM routes under `/azure/subscriptions/.../providers/Microsoft.Search/searchServices/...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate ARM identifiers and operation tokens (`name`, `key`).
- Validate method constraints for create/delete/list endpoints.

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Query Key Fixtures

- Implement deterministic fixtures for create/list/delete query key flows.
- Return stable `QueryKey` and `QueryKeysListResult` payload shapes.

Exit criteria:
- Success-path behavior is deterministic and contract-tested.

## Stage 3: Error and Lifecycle Fixtures

- Add deterministic not-found and validation-error fixtures.
- Add deterministic duplicate/create-conflict fixtures for repeated key-name workflows.

Exit criteria:
- Negative path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-query-keys-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
