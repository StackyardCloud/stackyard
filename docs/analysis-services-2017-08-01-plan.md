# Azure Analysis Services (`analysis-services-2017-08-01`) Staged Plan

## Objective

Emulate Azure Analysis Services management-plane APIs (`2017-08-01`) with deterministic local behavior for provider operations, server lifecycle, gateway actions, SKU discovery, and long-running operation status/result lookups.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/operation-groups?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/operations?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers?view=rest-analysisservices-2017-08-01`

Method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/operations/list?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/check-name-availability?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/create?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/delete?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/dissociate-gateway?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/get-details?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list-by-resource-group?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list-gateway-status?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list-operation-results?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list-operation-statuses?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list-skus-for-existing?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/list-skus-for-new?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/resume?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/suspend?view=rest-analysisservices-2017-08-01`
- `https://learn.microsoft.com/en-us/rest/api/analysisservices/servers/update?view=rest-analysisservices-2017-08-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full Analysis Services engine internals in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.AnalysisServices/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AnalysisServices/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AnalysisServices/*`

Target API version:
- `api-version=2017-08-01`

## API Surface and Contract Notes

Operation groups reviewed:
- 2 operation-group pages (`operations`, `servers`).

Method pages reviewed:
- 16 method pages linked from those operation groups.

Common contract characteristics from method pages:
- ARM management-plane pathing under `providers/Microsoft.AnalysisServices`.
- Success status families include `200`, `201`, `202`, and `204` depending on operation type.
- Non-success contracts are documented with typed error payloads in `Other Status Codes`.
- API includes synchronous read/list calls plus long-running create/update/delete and action endpoints.

## Stage 0: Contract Skeleton

- Add dedicated Azure Analysis Services router surface.
- Recognize provider/subscription/resource-group ARM envelopes for `Microsoft.AnalysisServices`.
- Add route-recognition tests spanning all documented method families.

Exit criteria:
- Route ownership and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating for Analysis Services management-plane routes.
- Return deterministic invalid-request payloads for malformed API version input.

Exit criteria:
- Invalid query/method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized Analysis Services routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/analysis-services-2017-08-01`.
- Exercise representative workflows for operations list, name availability checks, server lifecycle, gateway actions, SKU listing, and operation status/result reads.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add Analysis Services aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for the `analysis_services` service key.

Exit criteria:
- Coverage tooling resolves `analysis_services` and versioned aliases consistently.
