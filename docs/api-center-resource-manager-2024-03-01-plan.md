# Azure API Center Resource Manager (`resource-manager-2024-03-01`) Staged Plan

## Objective

Emulate Azure API Center management-plane APIs (`2024-03-01`) with deterministic local behavior for service lifecycle, workspaces, APIs, API versions, API definitions, deployments, environments, metadata schemas, and provider operations.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/operation-groups?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/api-definitions?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/api-versions?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/apis?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/deployments?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/environments?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/metadata-schemas?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/operations?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/services?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/workspaces?view=rest-resource-manager-apicenter-2024-03-01`

Representative method references:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/operations/list?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/services/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/services/export-metadata-schema?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/workspaces/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/apis/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/api-versions/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/api-definitions/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/api-definitions/export-specification?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/api-definitions/import-specification?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/deployments/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/environments/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/apicenter/metadata-schemas/create-or-update?view=rest-resource-manager-apicenter-2024-03-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full asynchronous ARM provisioning internals in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.ApiCenter/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ApiCenter/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ApiCenter/*`

Target API version:
- `api-version=2024-03-01`

## API Surface and Contract Notes

Operation groups reviewed:
- 9 operation-group pages (`api-definitions`, `api-versions`, `apis`, `deployments`, `environments`, `metadata-schemas`, `operations`, `services`, `workspaces`).

Method pages reviewed:
- 45 method pages linked from those operation-group pages.
- Review covered all method pages including CRUD/list/head methods, metadata export/import operations, and provider operations.

Common contract characteristics from method pages:
- ARM management-plane pathing under `providers/Microsoft.ApiCenter`.
- Success status families include `200`, `201`, `202`, and `204` depending on operation semantics.
- Non-success responses use typed ARM error payloads documented under `Other Status Codes`.
- API includes both long-running style action endpoints and synchronous list/read operations.

## Stage 0: Contract Skeleton

- Add dedicated Azure API Center resource-manager router surface.
- Recognize provider/subscription/resource-group envelopes for `Microsoft.ApiCenter`.
- Add route-recognition tests spanning all operation groups.

Exit criteria:
- Route ownership and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating for API Center resource-manager routes.
- Return deterministic invalid-request payloads for malformed API version input.

Exit criteria:
- Invalid query/method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized API Center resource-manager routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/api-center/resource-manager-2024-03-01`.
- Exercise representative workflows across operations, services, workspaces, APIs, versions, definitions, deployments, environments, and metadata schemas.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add API Center resource-manager aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for the `api_center_resource_manager` service key.

Exit criteria:
- Coverage tooling resolves `api_center_resource_manager` and versioned aliases consistently.
