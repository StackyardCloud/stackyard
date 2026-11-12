# Azure Data Catalog Custom Data Source (`data-catalog-custom-data-source`) Staged Plan

## Objective

Emulate Azure Data Catalog Custom Data Source (`api-version=2016-03-30`) with deterministic local behavior for:

- `Enumerate` custom data source protocols
- `Register` custom data source protocols
- data source protocol validation and duplicate handling (`ImmutableViewItem`)
- an Azure Go SDK example under `examples/azure/datacatalog/data-catalog-custom-data-source`
- Azure coverage-script wiring for the new service key

Important lifecycle note from reviewed docs:
- Azure Data Catalog was retired on **May 15, 2024**.
- This plan still targets the published REST contract so existing client integrations can be emulated in Stackyard.

## Primary References Reviewed

Primary operation-group page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/datacatalog/data-catalog-custom-data-source`

Method sections reviewed on that page:
- `Enumerate method`
- `Register method`

Specification page reviewed for payload rules:
- `https://learn.microsoft.com/en-us/azure/data-catalog/data-catalog-developer-concepts`
- reviewed section: `Custom data source protocol specification`

Reference detail:
- the Custom Data Source page exposes both methods as sections on a single page rather than separate per-method URLs.

SDK approach selected for implementation:
- use Azure Go SDK core runtime pipeline (`github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime` + `azcore/policy`) for the example

SDK package note:
- no stable service-specific typed Go client package was identified for this retired Data Catalog surface, so the example should use the Azure Go SDK runtime pipeline directly.

## Service Naming

Canonical Stackyard naming:
- service key: `datacatalog_custom_data_source`
- display name: `Azure Data Catalog Custom Data Source`
- API version: `2016-03-30`

Recommended repo paths:
- implementation: `internal/server/provider_azure_datacatalog_custom_data_source.go`
- tests: `internal/server/provider_azure_datacatalog_custom_data_source_test.go`
- example: `examples/azure/datacatalog/data-catalog-custom-data-source`
- plan doc: `docs/datacatalog-data-catalog-custom-data-source-plan.md`

## Surface Summary

Reviewed method inventory:

- `Enumerate` (`GET /catalogs/{catalog_name}/dataSourceProtocols`)
- `Register` (`POST /catalogs/{catalog_name}/dataSourceProtocols`)

Total methods reviewed: `2`

## Route Envelope

Documented route family:
- `GET {endpoint}/catalogs/{catalog_name}/dataSourceProtocols?api-version=2016-03-30`
- `POST {endpoint}/catalogs/{catalog_name}/dataSourceProtocols?api-version=2016-03-30`

Stackyard route envelope:
- `GET /azure/catalogs/{catalog_name}/dataSourceProtocols?api-version=2016-03-30`
- `POST /azure/catalogs/{catalog_name}/dataSourceProtocols?api-version=2016-03-30`

## Contract Notes From Reviewed Pages

Shared traits:
- `catalog_name` is required in the URI (supports `DefaultCatalog`)
- `api-version` is required and documented as `2016-03-30`
- responses are JSON payloads

`Enumerate`:
- returns status `200` and a `value` array of protocol definitions
- each protocol entry includes:
  - `namespace`
  - `name`
  - `identityProperties`
  - `identitySets`
  - `id`
  - `timestamp`
- docs note `nextLink` may be present for pagination

`Register`:
- returns success when a new protocol is created
- documented errors:
  - `400 BadRequest` for invalid protocol specification
  - `400 BadRequest` with code `ImmutableViewItem` when protocol already exists
- response header includes `Location` in the documented example

Validation rules from the reviewed developer-concepts specification:
- `namespace`:
  - 1-255 chars total
  - dot-separated parts
  - each part 1-255 chars, starts with a letter, letters/numbers only
- `name`:
  - 1-255 chars
  - starts with a letter
  - letters, numbers, dash only
- `identityProperties`:
  - required, 1-20 entries
  - `name` required, 1-100 chars, starts with a letter, letters/numbers only
  - `type` required, supported values include `bool`, `boolean`, `byte`, `guid`, `int`, `integer`, `long`, `string`, `url`
  - `ignoreCase` valid only for `string`
  - `urlPathSegmentsIgnoreCase` valid only for `url`
- `identitySets`:
  - required, 1-20 entries
  - each set references only defined `identityProperties`
  - duplicates are not allowed within a set

## Staged Implementation Plan

### Stage 0: Coverage And Planning Wiring

- add this plan doc
- add aliases for `datacatalog_custom_data_source` to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this doc in `SERVICE_PLAN_DOCS`

Exit criteria:
- coverage scripts discover the new service key
- doc coverage maps this service to this plan doc

### Stage 1: Router Skeleton And Strict API Version

Implement:
- route matcher for `/azure/catalogs/{catalog_name}/dataSourceProtocols`
- strict `api-version=2016-03-30` gate
- method ownership for `GET` and `POST`
- `405` for unsupported methods on owned route

Exit criteria:
- route is owned by Azure provider router
- invalid/missing API version returns `400`

### Stage 2: Enumerate + Register Core Behavior

Implement:
- in-memory store keyed by `{catalog, namespace, name}`
- `Enumerate` returns deterministic `value` list
- `Register` creates immutable protocol records and returns JSON + `Location` header

Exit criteria:
- first register succeeds
- enumerate reflects created records for the target catalog
- duplicate register returns `400` with `ImmutableViewItem`

### Stage 3: Specification-Driven Validation

Implement server-side validation aligned with reviewed spec:
- namespace/name shape rules
- identity property count and field rules
- supported identity property types
- `ignoreCase` / `urlPathSegmentsIgnoreCase` constraints
- identity set membership rules

Exit criteria:
- malformed payloads return deterministic `400 BadRequest`
- valid payloads produce stable protocol fixtures

### Stage 4: Tests

Add focused provider tests for:
- enumerate/register lifecycle
- duplicate register (`ImmutableViewItem`)
- invalid payloads (`BadRequest`)
- invalid/missing API version
- unsupported method handling (`405`)

Exit criteria:
- `go test ./internal/server -run 'TestAzureDataCatalogCustomDataSource' -count=1` passes

### Stage 5: Example

Add `examples/azure/datacatalog/data-catalog-custom-data-source`:
- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example flow:
- enumerate protocols
- register a protocol
- enumerate again
- attempt duplicate register and assert modeled `ImmutableViewItem`

SDK approach:
- use `azcore/runtime` pipeline with a shared-key style policy for local emulator auth

Exit criteria:
- `go build ./examples/azure/datacatalog/data-catalog-custom-data-source` passes

### Stage 6: Validation

Run:
- `go test ./internal/server -run 'TestAzureDataCatalogCustomDataSource' -count=1`
- `go build ./examples/azure/datacatalog/data-catalog-custom-data-source`
- `python3 scripts/azure-contract-coverage.py --service datacatalog_custom_data_source --require-service datacatalog_custom_data_source --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datacatalog_custom_data_source --require-service datacatalog_custom_data_source --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datacatalog_custom_data_source --require-service datacatalog_custom_data_source --fail-on any`

Success target:
- no `NotImplemented` responses for the two documented custom-data-source methods
- coverage gates pass for the new service key
