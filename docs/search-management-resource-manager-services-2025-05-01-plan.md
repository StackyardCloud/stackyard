# Azure Search Management - Resource Manager - Services (`search-management-resource-manager-services-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Services (`2025-05-01`) with deterministic local behavior for search service lifecycle and enumeration workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/check-name-availability?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/create-or-update?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/delete?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/get?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/list-by-resource-group?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/list-by-subscription?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/update?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/services/upgrade?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement real ARM provisioning orchestration in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Search/checkNameAvailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Search/searchServices`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/upgrade`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Services operations documented for this API version:

- `POST /subscriptions/{subscriptionId}/providers/Microsoft.Search/checkNameAvailability`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}`
- `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.Search/searchServices`
- `PATCH /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}`
- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/upgrade`

Common contract characteristics from operation pages:
- Paths mix subscription-scoped and resource-group-scoped operations.
- Create/update, delete, and upgrade can involve long-running operation semantics.
- List operations use service-list result envelopes (`value`, optional continuation).
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Services router surface.
- Recognize all documented Services routes/methods under `/azure/subscriptions/.../providers/Microsoft.Search/...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate path identifiers (`subscriptionId`, `resourceGroupName`, `searchServiceName`).
- Validate operation-specific payload shape (check-name, create/update, patch).

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Service Resource Fixtures

- Implement deterministic fixtures for create/get/update/delete service flows.
- Implement deterministic list-by-rg/list-by-subscription fixtures with stable ordering.
- Implement deterministic check-name and upgrade acceptance fixtures.

Exit criteria:
- Success-path behavior is deterministic and contract-tested.

## Stage 3: Error and LRO Fixtures

- Add deterministic not-found/conflict/validation fixtures.
- Add deterministic long-running-operation status fixtures for create/delete/upgrade paths.

Exit criteria:
- Negative and LRO-path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-services-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
