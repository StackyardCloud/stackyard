# Azure Search Management - Resource Manager - Network Security Perimeter Configurations (`search-management-resource-manager-network-security-parameter-configurations-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Network Security Perimeter Configurations (`2025-05-01`) with deterministic local behavior for perimeter configuration listing, retrieval, and reconcile flows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/network-security-perimeter-configurations?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/network-security-perimeter-configurations/get?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/network-security-perimeter-configurations/reconcile?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement real Azure control-plane reconciliation in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/networkSecurityPerimeterConfigurations`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/networkSecurityPerimeterConfigurations/{nspConfigName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/networkSecurityPerimeterConfigurations/{nspConfigName}/reconcile`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Network Security Perimeter Configurations operations documented for this API version:

- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/networkSecurityPerimeterConfigurations`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/networkSecurityPerimeterConfigurations/{nspConfigName}`
- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/networkSecurityPerimeterConfigurations/{nspConfigName}/reconcile`

Common contract characteristics from operation pages:
- `subscriptionId`, `resourceGroupName`, `searchServiceName`, and `nspConfigName` are required identifiers.
- List returns a paged shape (`value`, optional `nextLink`).
- Get returns a typed network security perimeter configuration resource.
- Reconcile is asynchronous and returns `202 Accepted` with `Location` header.
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Network Security Perimeter Configurations router surface.
- Recognize documented ARM routes under `/azure/subscriptions/.../providers/Microsoft.Search/searchServices/.../networkSecurityPerimeterConfigurations...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate ARM resource identifiers (`subscriptionId`, `resourceGroupName`, `searchServiceName`, `nspConfigName`).
- Validate reconcile operation shape and method constraints.

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Resource Fixtures

- Implement deterministic list/get fixtures for perimeter configurations.
- Implement deterministic reconcile acceptance fixture (`202` + stable `Location` header).

Exit criteria:
- Operations return deterministic typed payloads/header contracts.

## Stage 3: Error and Async Fixtures

- Add deterministic not-found and validation conflict fixtures.
- Add deterministic asynchronous polling/error fixture branches for reconcile workflows.

Exit criteria:
- Negative path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-network-security-parameter-configurations-2025-05-01`.
- Update Azure coverage alias scripts for perimeter and parameter service naming variants.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
