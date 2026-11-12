# Azure API Management Resource Manager (`resource-manager-2024-05-01`) Staged Plan

## Objective

Emulate Azure API Management management-plane APIs (`2024-05-01`) with deterministic local behavior for service lifecycle, API configuration, products, users, policies, notifications, gateway resources, and workspace-scoped management workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/operation-groups?view=rest-apimanagement-2024-05-01`

Representative operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-management-operations?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-management-service?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/apis?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-operation?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/product?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/user?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/gateway?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/workspace?view=rest-apimanagement-2024-05-01`

Representative method-page references:
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-management-operations/list?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-management-service/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-management-service/apply-network-configuration-updates?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/apis/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-operation/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/api-operation-policy/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/product/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/user/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/named-value/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/gateway/create-or-update?view=rest-apimanagement-2024-05-01`
- `https://learn.microsoft.com/en-us/rest/api/apimanagement/workspace/create-or-update?view=rest-apimanagement-2024-05-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full asynchronous provisioning internals for all API Management LRO paths in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.ApiManagement/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ApiManagement/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ApiManagement/*`

Target API version:
- `api-version=2024-05-01`

## API Surface and Contract Notes

Operation groups reviewed:
- 141 operation-group pages linked from the API Management operation-groups root page.

Method pages reviewed:
- 613 method pages linked from those operation-group pages.
- Method-page crawl included extraction of route and status signatures for each method page.

Common contract characteristics from method pages:
- ARM management-plane pathing under `providers/Microsoft.ApiManagement`.
- Broad method coverage across `GET`, `PUT`, `PATCH`, `DELETE`, `POST`, and `HEAD`.
- Success status families include `200`, `201`, `202`, and `204` depending on operation semantics.
- Non-success responses use typed ARM error payloads documented under `Other Status Codes`.

## Stage 0: Contract Skeleton

- Add dedicated Azure API Management resource-manager router surface.
- Recognize provider/subscription/resource-group ARM envelopes for `Microsoft.ApiManagement`.
- Add route-recognition tests spanning representative families across service, API, product, user, gateway, and workspace resources.

Exit criteria:
- Route ownership and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating for API Management resource-manager routes.
- Return deterministic invalid-request payloads for malformed API version input.

Exit criteria:
- Invalid query/method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized API Management routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/api-management/resource-manager-2024-05-01`.
- Exercise representative workflows across operations, service, APIs, operations policies, products, users, named values, gateways, and workspaces.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add API Management resource-manager aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for the `api_management_resource_manager` service key.

Exit criteria:
- Coverage tooling resolves `api_management_resource_manager` and versioned aliases consistently.
