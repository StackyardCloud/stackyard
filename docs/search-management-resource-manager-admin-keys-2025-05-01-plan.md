# Azure Search Management - Resource Manager - Admin Keys (`search-management-resource-manager-admin-keys-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Admin Keys (`2025-05-01`) with deterministic local behavior for admin key retrieval and regeneration workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/admin-keys?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/admin-keys/get?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/admin-keys/regenerate?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement Azure RBAC or real key rotation in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/listAdminKeys`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/regenerateAdminKey/{keyKind}`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Admin Keys operations documented for this API version:

- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/listAdminKeys`
- `POST /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Search/searchServices/{searchServiceName}/regenerateAdminKey/{keyKind}`

Common contract characteristics from operation pages:
- Resource identifiers include `subscriptionId`, `resourceGroupName`, and `searchServiceName`.
- Regenerate operation accepts `keyKind` values of `primary` or `secondary`.
- Success response shape is `AdminKeyResult` with `primaryKey` and `secondaryKey`.
- Error model uses the `CloudError` envelope for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Admin Keys router surface.
- Recognize documented ARM Admin Keys routes under `/azure/subscriptions/.../providers/Microsoft.Search/searchServices/...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate ARM resource identifiers (`subscriptionId`, `resourceGroupName`, `searchServiceName`).
- Validate `keyKind` enum for regenerate operation (`primary`, `secondary`).

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Admin Key Fixtures

- Implement deterministic `AdminKeyResult` fixture payloads for list and regenerate operations.
- Preserve stable output shape for contract and SDK-level tests.

Exit criteria:
- Both operations return deterministic typed payloads under stable fixtures.

## Stage 3: Error and Policy Fixtures

- Add deterministic not-found and authorization-style fixture branches.
- Add deterministic failure fixture branches for resilience tests.

Exit criteria:
- Negative path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-admin-keys-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
