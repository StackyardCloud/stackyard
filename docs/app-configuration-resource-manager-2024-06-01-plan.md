# Azure App Configuration Resource Manager (`resource-manager-2024-06-01`) Staged Plan

## Objective

Emulate Azure App Configuration management-plane APIs (`2024-06-01`) with deterministic local behavior for configuration stores, key-values, operations, private endpoint connections, private link resources, replicas, and snapshots.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/operation-groups?view=rest-appconfiguration-2024-06-01`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/key-values?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/operations?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/private-endpoint-connections?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/private-link-resources?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/replicas?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/snapshots?view=rest-appconfiguration-2024-06-01`

Representative method-page references:
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/operations/list?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/operations/check-name-availability?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/create?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/update?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/list-keys?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/regenerate-key?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/configuration-stores/purge-deleted?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/key-values/create-or-update?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/private-endpoint-connections/create-or-update?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/private-link-resources/list-by-configuration-store?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/replicas/create?view=rest-appconfiguration-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/appconfiguration/snapshots/create?view=rest-appconfiguration-2024-06-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full asynchronous provisioning internals in this stage.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.AppConfiguration/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AppConfiguration/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppConfiguration/*`

Target API version:
- `api-version=2024-06-01`

## API Surface and Contract Notes

Operation groups reviewed:
- 7 operation-group pages linked from the operation-groups root page.

Method pages reviewed:
- 29 method pages linked from those operation-group pages.
- Crawl output captured at `/tmp/app_configuration_resource_manager_2024_06_01_methods.json`.

Observed contract characteristics:
- ARM management-plane pathing under `providers/Microsoft.AppConfiguration`.
- Methods span `GET`, `POST`, `PUT`, `PATCH`, and `DELETE`.
- Success status families include `200`, `201`, and `202` depending on operation semantics.
- Non-success responses use typed ARM error payloads documented under `Other Status Codes`.

## Stage 0: Contract Skeleton

- Add dedicated App Configuration resource-manager router surface.
- Recognize provider/subscription/resource-group ARM envelopes for `Microsoft.AppConfiguration`.
- Add route-recognition tests spanning representative operation families.

Exit criteria:
- Route ownership and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating from the discovered method set.
- Return deterministic invalid-request payload for malformed `api-version`.

Exit criteria:
- Invalid query/method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized App Configuration routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/app-configuration/resource-manager-2024-06-01`.
- Exercise representative workflows across configuration stores, key-values, keys, private endpoint connections, private link resources, replicas, snapshots, and purge-deleted actions.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add App Configuration resource-manager aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for canonical service key `app_configuration_resource_manager`.

Exit criteria:
- Coverage tooling resolves `app_configuration_resource_manager` and versioned aliases consistently.
