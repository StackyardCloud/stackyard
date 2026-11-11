# Azure Data Catalog Data Asset (`data-catalog-data-asset`) Staged Plan

## Objective

Emulate Azure Data Catalog Data Asset (`api-version=2016-03-30`) with deterministic local behavior for:

- `Annotate`
- `Register Or Update`
- `Get With Annotations`
- `Search`
- `Delete`
- `Update Annotation`
- `Delete Annotation`
- an Azure Go SDK example under `examples/azure/datacatalog/data-catalog-data-asset`
- Azure coverage-script wiring for the new service key

Important lifecycle note from reviewed docs:
- Azure Data Catalog was retired on **May 15, 2024**.
- This plan still targets the published REST contract so existing client integrations can be emulated in Stackyard.

## Primary References Reviewed

Primary operation-group page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/datacatalog/data-catalog-data-asset`

Method sections reviewed on that page:
- `Annotate method`
- `Register or Update method`
- `GET with Annotations method`
- `Search method`
- `Delete method`
- `Update Annotation method`
- `Delete Annotation method`

Reference detail:
- the Data Asset page exposes all methods as sections on a single page rather than separate per-method URLs.

SDK approach selected for implementation:
- use Azure Go SDK core runtime pipeline (`github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime` + `azcore/policy`) for the example

SDK package note:
- no stable service-specific typed Go client package was identified for this retired Data Catalog surface, so the example should use the Azure Go SDK runtime pipeline directly.

## Service Naming

Canonical Stackyard naming:
- service key: `datacatalog_data_asset`
- display name: `Azure Data Catalog Data Asset`
- API version: `2016-03-30`

Recommended repo paths:
- implementation: `internal/server/provider_azure_datacatalog_data_asset.go`
- tests: `internal/server/provider_azure_datacatalog_data_asset_test.go`
- example: `examples/azure/datacatalog/data-catalog-data-asset`
- plan doc: `docs/datacatalog-data-catalog-data-asset-plan.md`

## Surface Summary

Reviewed method inventory:

- `Annotate` (`POST /catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}`)
- `Register Or Update` (`POST /catalogs/{catalog_name}/views/{view_name}`)
- `GET With Annotations` (`GET /catalogs/{catalog_name}/views/{view_name}/{view_item_id}`)
- `Search` (`GET /catalogs/{catalog_name}/search/search`)
- `Delete` (`DELETE /catalogs/{catalog_name}/views/{view_name}/{view_item_id}`)
- `Update Annotation` (`PUT /catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}` and `PUT /catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}/{nested_non_singleton_view_item_id}`)
- `Delete Annotation` (`DELETE /catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}` and `DELETE /catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}/{nested_non_singleton_view_item_id}`)

Total methods reviewed: `7`

## Route Envelope

Documented route family:
- `POST {endpoint}/catalogs/{catalog_name}/views/{view_name}?api-version=2016-03-30`
- `GET {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}?api-version=2016-03-30`
- `DELETE {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}?api-version=2016-03-30`
- `POST {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}?api-version=2016-03-30`
- `PUT {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}?api-version=2016-03-30`
- `PUT {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}/{nested_non_singleton_view_item_id}?api-version=2016-03-30`
- `DELETE {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}?api-version=2016-03-30`
- `DELETE {endpoint}/catalogs/{catalog_name}/views/{view_name}/{view_item_id}/{nested_view_name}/{nested_non_singleton_view_item_id}?api-version=2016-03-30`
- `GET {endpoint}/catalogs/{catalog_name}/search/search?searchTerms={searchTerms}&count={count}&startPage={startPage}&api-version=2016-03-30`

Stackyard route envelope:
- same paths with `/azure` prefix, for example:
  - `POST /azure/catalogs/{catalog_name}/views/{view_name}?api-version=2016-03-30`
  - `GET /azure/catalogs/{catalog_name}/search/search?searchTerms=...&api-version=2016-03-30`

## Contract Notes From Reviewed Pages

Shared traits:
- `catalog_name` is required in the URI (supports `DefaultCatalog`)
- `api-version` is required and documented as `2016-03-30`
- responses are JSON payloads (except `204` deletes with no content body)

`Register Or Update`:
- documented success statuses:
  - `201 Created` when asset is created
  - `200 OK` when asset identity already exists and asset is updated
- documented error:
  - `412 Precondition Failed` when provided ETag does not match
- response includes `id` and `Location` header

`Annotate`:
- documented success statuses:
  - `201 Created` when annotation is created
  - `200 OK` when annotation is updated
- documented error:
  - `412 Precondition Failed` on ETag mismatch
- response includes `id` and `Location` header

`GET With Annotations`:
- documented success status: `200 OK`
- supports `Accept` metadata mode with `adc.metadata` values:
  - `none`
  - `minimal`
  - `full`
- response includes data-asset envelope with `roles`, `effectiveRights`, `properties`, and `annotations`

`Search`:
- documented success status: `200 OK`
- documented client errors include invalid query parameter values:
  - `400 BadRequest`
- query parameters include:
  - `searchTerms` (required)
  - `startPage` (integer > 0)
  - `count` (1-100)
  - `view` (supported value `DataSource`)
  - `facets` (optional)
- response is a `Search Response` envelope with query metadata and results

`Delete`:
- documented success status: `204 NoContent` (delete if exists)
- documented error:
  - `412 Precondition Failed` on ETag mismatch

`Update Annotation`:
- documented success status: `200 OK`
- documented error:
  - `412 Precondition Failed` on ETag mismatch

`Delete Annotation`:
- documented success status: `204 NoContent` (delete if exists)
- documented error:
  - `412 Precondition Failed` on ETag mismatch

## Staged Implementation Plan

### Stage 0: Coverage And Planning Wiring

- add this plan doc
- add aliases for `datacatalog_data_asset` to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this doc in `SERVICE_PLAN_DOCS`

Exit criteria:
- coverage scripts discover the new service key
- doc coverage maps this service to this plan doc

### Stage 1: Router Skeleton And Strict API Version

Implement:
- route matcher for `/azure/catalogs/{catalog_name}/views/*` and `/azure/catalogs/{catalog_name}/search/search`
- strict `api-version=2016-03-30` gate
- method ownership for all documented methods
- `405` for unsupported methods on owned routes

Exit criteria:
- routes are owned by Azure provider router
- invalid/missing API version returns `400`

### Stage 2: Data Asset Lifecycle And Annotation Storage

Implement:
- in-memory store keyed by `{catalog, view, view_item_id}`
- register/update identity behavior with deterministic `view_item_id`
- core asset payload fixture (`id`, `etag`, `timestamp`, `roles`, `effectiveRights`, `properties`)
- nested annotation collections (singleton and non-singleton) with deterministic IDs

Exit criteria:
- `POST views/{view}` can create and update a data asset
- `GET views/{view}/{id}` returns asset plus annotations
- annotation create/update paths return `id` + `Location`

### Stage 3: Search + Delete + Concurrency Semantics

Implement:
- `search/search` query validation and paged response
- delete-if-exists behavior for asset and annotation deletes
- ETag mismatch handling (`412`) for register/update, annotate/update, and delete calls with conditions
- `Accept` metadata mode parsing (`none|minimal|full`) for get-with-annotations

Exit criteria:
- search returns deterministic query envelope and results
- stale ETag/If-Match cases return modeled `412`
- delete-if-exists paths return `204`

### Stage 4: Tests

Add focused provider tests for:
- register/get/annotate/update/search/delete lifecycle
- create vs update status codes (`201` and `200`)
- ETag mismatch failures (`412`)
- invalid query parameters and API version failures (`400`)
- unsupported method handling (`405`)

Exit criteria:
- `go test ./internal/server -run 'TestAzureDataCatalogDataAsset' -count=1` passes

### Stage 5: Example

Add `examples/azure/datacatalog/data-catalog-data-asset`:
- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example flow:
- register/update an asset
- get with annotations metadata
- annotate and update an expert annotation
- search for the asset
- delete annotation and delete asset

SDK approach:
- use `azcore/runtime` pipeline with a shared-key style policy for local emulator auth

Exit criteria:
- `go build ./examples/azure/datacatalog/data-catalog-data-asset` passes

### Stage 6: Validation

Run:
- `go test ./internal/server -run 'TestAzureDataCatalogDataAsset' -count=1`
- `go build ./examples/azure/datacatalog/data-catalog-data-asset`
- `python3 scripts/azure-contract-coverage.py --service datacatalog_data_asset --require-service datacatalog_data_asset --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datacatalog_data_asset --require-service datacatalog_data_asset --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datacatalog_data_asset --require-service datacatalog_data_asset --fail-on any`

Success target:
- no `NotImplemented` responses for the seven documented data-asset methods
- coverage gates pass for the new service key
