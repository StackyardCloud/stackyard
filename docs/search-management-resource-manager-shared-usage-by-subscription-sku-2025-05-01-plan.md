# Azure Search Management - Resource Manager - Usage By Subscription Sku (`search-management-resource-manager-shared-usage-by-subscription-sku-2025-05-01`) Staged Plan

## Objective

Emulate Azure Search Management Resource Manager Usage By Subscription Sku (`2025-05-01`) with deterministic local behavior for subscription+region SKU quota usage lookup.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/usage-by-subscription-sku?view=rest-searchmanagement-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/searchmanagement/usage-by-subscription-sku/usage-by-subscription-sku?view=rest-searchmanagement-2025-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement dynamic quota accounting from real control-plane state in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Search/locations/{location}/usages/{skuName}`

Target API version:
- `api-version=2025-05-01`

## API Surface and Contract Notes

Usage By Subscription Sku operations documented for this API version:

- `GET /subscriptions/{subscriptionId}/providers/Microsoft.Search/locations/{location}/usages/{skuName}`

Common contract characteristics from operation pages:
- Required identifiers include `subscriptionId`, `location`, and `skuName`.
- Success response returns `QuotaUsageResult` with fields such as `name`, `currentValue`, `limit`, `unit`, and `id`.
- Error model uses `CloudError` for non-success responses.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Management Usage By Subscription Sku router surface.
- Recognize documented ARM route under `/azure/subscriptions/.../providers/Microsoft.Search/locations/.../usages/...`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests for the documented operation.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate path identifiers (`subscriptionId`, `location`, `skuName`).
- Validate method constraints for read-only usage lookup.

Exit criteria:
- Invalid requests return deterministic `400` style contracts.

## Stage 2: Deterministic Usage Fixtures

- Implement deterministic `QuotaUsageResult` fixture payloads keyed by location+sku.
- Preserve stable payload shape and value constraints.

Exit criteria:
- Success-path behavior is deterministic and contract-tested.

## Stage 3: Error and Edge Fixtures

- Add deterministic not-found and validation-error fixtures.
- Add deterministic boundary fixtures (quota reached, near-capacity, zero usage).

Exit criteria:
- Negative and edge contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-management/search-management-resource-manager-shared-usage-by-subscription-sku-2025-05-01`.
- Update Azure coverage alias scripts for usage-by-subscription-sku and shared-usage naming variants.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
