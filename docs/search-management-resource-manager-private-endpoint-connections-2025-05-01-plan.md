# Azure Search Management - Resource Manager - Private Endpoint Connections (`search-management-resource-manager-private-endpoint-connections-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Private Endpoint Connections (`2025-05-01`) with deterministic local behavior for listing, getting, updating, and deleting private endpoint connections.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-endpoint-connections?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-endpoint-connections/list-by-service?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-endpoint-connections/get?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-endpoint-connections/update?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/private-endpoint-connections/delete?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full Azure Private Link lifecycle control in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateEndpointConnections/{privateEndpointConnectionName}`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Private Endpoint Connections operations documented for this API version:

- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateEndpointConnections`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateEndpointConnections/{privateEndpointConnectionName}`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateEndpointConnections/{privateEndpointConnectionName}`
- `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/privateEndpointConnections/{privateEndpointConnectionName}`

Common contract characteristics from operation pages:
- Required path identifiers include `subscriptionId`, `resourceGroupName`, `searchServiceName`, and optionally `privateEndpointConnectionName`.
- List returns `PrivateEndpointConnectionListResult` with `value` and optional continuation.
- Get/Update/Delete return a `PrivateEndpointConnection` resource on success.
- Delete operation explicitly documents `404 Not Found` when the target resource is missing.
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Private Endpoint Connections router surface.
- Recognize documented ARM routes under `/azure/subscriptions/.../providers/Microsoft.Search/searchServices/.../privateEndpointConnections...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate ARM identifiers (`subscriptionId`, `resourceGroupName`, `searchServiceName`, `privateEndpointConnectionName`).
- Validate update request body shape (`privateLinkServiceConnectionState` status/description).

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Resource Fixtures

- Implement deterministic list/get/update/delete private endpoint connection fixtures.
- Return stable payload shape for `PrivateEndpointConnection` and `PrivateEndpointConnectionListResult`.

Exit criteria:
- Success-path behavior is deterministic and contract-tested.

## Stage 3: Error and State Fixtures

- Add deterministic not-found and validation-error fixtures.
- Add deterministic state-transition fixtures for update/delete flows (e.g., approved/rejected/disconnected semantics).

Exit criteria:
- Negative path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-private-endpoint-connections-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
