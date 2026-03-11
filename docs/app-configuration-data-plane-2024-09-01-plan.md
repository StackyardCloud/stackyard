# Azure App Configuration Data Plane (`data-plane-2024-09-01`) Staged Plan

## Objective

Emulate Azure App Configuration data-plane APIs (`2024-09-01`) with deterministic local behavior for key-values, keys, labels, revisions, snapshots, locks, and operation status queries.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/operation-groups?view=rest-data-plane-appconfiguration-2024-09-01`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-key-values?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-keys?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-labels?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-revisions?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-snapshot?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-snapshots?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/create-snapshot?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/delete-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/delete-lock?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-key-values?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-keys?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-labels?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-operation-details?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-revisions?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-snapshot?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-snapshots?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/put-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/put-lock?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/update-snapshot?view=rest-data-plane-appconfiguration-2024-09-01`

Representative method-page references:
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/check-key-value/check-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/get-key-value/get-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/put-key-value/put-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/delete-key-value/delete-key-value?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/create-snapshot/create-snapshot?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/update-snapshot/update-snapshot?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/put-lock/put-lock?view=rest-data-plane-appconfiguration-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/appconfiguration/delete-lock/delete-lock?view=rest-data-plane-appconfiguration-2024-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate persisted state/version semantics for every key-value and snapshot field in this stage.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/appconfiguration/kv`
- `/azure/appconfiguration/kv/{key}`
- `/azure/appconfiguration/keys`
- `/azure/appconfiguration/labels`
- `/azure/appconfiguration/revisions`
- `/azure/appconfiguration/snapshots`
- `/azure/appconfiguration/snapshots/{snapshot}`
- `/azure/appconfiguration/locks/{key}`
- `/azure/appconfiguration/operations`

Target API version:
- `api-version=2024-09-01`

## API Surface and Contract Notes

Operation groups reviewed:
- 21 operation-group pages linked from the operation-groups root page.

Method pages reviewed:
- 21 method pages linked from those operation-group pages.
- Crawl output captured at `/tmp/app_configuration_data_plane_2024_09_01_methods.json`.

Observed contract characteristics:
- Data-plane endpoint family rooted at `{store}.azconfig.io`.
- Methods span `HEAD`, `GET`, `PUT`, `PATCH`, and `DELETE`.
- Conditional/check semantics are represented by `HEAD` operations on collection and resource endpoints.
- Non-success responses include typed error contracts documented per operation.

## Stage 0: Contract Skeleton

- Add dedicated App Configuration data-plane router surface.
- Recognize route ownership for key-values, collections, snapshots, locks, and operation-details paths.
- Add route-recognition tests spanning each operation family.

Exit criteria:
- Route ownership and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating from discovered methods (`HEAD/GET/PUT/PATCH/DELETE`).
- Return deterministic invalid-request payloads for malformed API version input.

Exit criteria:
- Invalid query/method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/app-configuration/data-plane-2024-09-01`.
- Exercise representative check/get/put/delete/patch workflows across all data-plane route families.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add App Configuration data-plane aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for canonical service key `app_configuration_data_plane`.

Exit criteria:
- Coverage tooling resolves `app_configuration_data_plane` and versioned aliases consistently.
