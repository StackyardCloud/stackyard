# Azure Search Management - Resource Manager - Operations (`search-management-resource-manager-operations-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Operations (`2025-05-01`) with deterministic local behavior for operation metadata discovery workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/operations?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/operations/list?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full ARM capability negotiation logic in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.Search/operations`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Operations endpoints documented for this API version:

- `GET /providers/Microsoft.Search/operations`

Common contract characteristics from operation pages:
- Successful response returns an operation list result object.
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Operations router surface.
- Recognize `/azure/providers/Microsoft.Search/operations`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests for the documented operation.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate method and path constraints for operations-list requests.

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Operation List Fixtures

- Implement deterministic operation list fixture payloads.
- Preserve stable ordering and object fields in responses.

Exit criteria:
- Typed success payloads are deterministic and contract-tested.

## Stage 3: Error and Pagination Fixtures

- Add deterministic authorization/validation error fixtures.
- Add deterministic pagination fixture behavior if operation list supports continuation.

Exit criteria:
- Negative and continuation-path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-operations-2025-05-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
