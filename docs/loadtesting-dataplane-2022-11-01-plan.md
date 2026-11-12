# Azure App Testing Load Testing Data Plane (`loadtesting-dataplane-2022-11-01`) Staged Plan

## Objective

Emulate the Azure App Testing Load Testing data-plane API for `2022-11-01` with deterministic local behavior for:

- test administration under `/tests`
- test file upload, list, metadata lookup, and deletion
- test app components and server metrics configuration
- test-run creation, retrieval, listing, stopping, and deletion under `/test-runs`
- test-run app components and server metrics configuration
- test-run artifact metadata lookup
- test-run metric namespaces, metric definitions, metric dimension values, and metric value queries
- an Azure Go SDK-based example under `examples/azure/loadtesting-dataplane-2022-11-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-loadtesting-dataplane-2022-11-01` is not the ARM `Microsoft.LoadTestService` management plane already implemented in this repo.
- It is the load-testing service data plane rooted at a service endpoint, with `/tests` and `/test-runs` resources rather than `/subscriptions/.../providers/...`.
- This plan intentionally treats it as a separate Stackyard service from ARM `loadtesting`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/operation-groups?view=rest-loadtesting-dataplane-2022-11-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run?view=rest-loadtesting-dataplane-2022-11-01`

Representative linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/list-tests?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/get-test?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/create-or-update-test?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/list-test-files?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/upload-test-file?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/create-or-update-app-components?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-administration/create-or-update-server-metrics-config?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/create-or-update-test-run?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/get-test-run?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/list-test-runs?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/stop-test-run?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/get-test-run-file?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/list-metric-definitions?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/list-metrics?view=rest-loadtesting-dataplane-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/dataplane/load-test-run/create-or-update-server-metrics-config?view=rest-loadtesting-dataplane-2022-11-01`

SDK selection note:
- I did not find a stable service-specific typed Azure Go SDK package for this data-plane surface during review.
- Plan assumption: the example should use the official Azure Go SDK core runtime pipeline (`azcore/runtime` and `azcore/policy`) rather than inventing a non-existent typed client.

## Service Naming

Canonical Stackyard naming:
- service key: `loadtesting_data_plane`
- display name: `Azure App Testing Load Testing Data Plane`
- API version token: `2022-11-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_loadtesting_data_plane.go`
- tests: `internal/server/provider_azure_loadtesting_data_plane_test.go`
- example: `examples/azure/loadtesting-dataplane-2022-11-01`
- plan doc: `docs/loadtesting-dataplane-2022-11-01-plan.md`

Recommended coverage aliases:
- `loadtesting_data_plane`
- `loadtesting_dataplane`
- `load-testing-data-plane`
- `load-testing-dataplane`
- `azure_loadtesting_data_plane`
- `azure_loadtesting_dataplane`
- `app_testing_load_testing_data_plane`
- `rest_loadtesting_dataplane_2022_11_01`
- `loadtesting_2022_11_01_dataplane`

## Surface Summary

The reviewed REST surface is split into 2 operation groups:

1. `Load Test Administration`
2. `Load Test Run`

Method families identified from the reviewed pages:

`Load Test Administration`
- `List Tests`
- `Get Test`
- `Create Or Update Test`
- `Delete Test`
- `List Test Files`
- `Get Test File`
- `Upload Test File`
- `Delete Test File`
- `Get App Components`
- `Create Or Update App Components`
- `Get Server Metrics Config`
- `Create Or Update Server Metrics Config`

`Load Test Run`
- `List Test Runs`
- `Get Test Run`
- `Create Or Update Test Run`
- `Delete Test Run`
- `Stop Test Run`
- `Get Test Run File`
- `Get App Components`
- `Create Or Update App Components`
- `Get Server Metrics Config`
- `Create Or Update Server Metrics Config`
- `List Metric Namespaces`
- `List Metric Definitions`
- `List Metric Dimension Values`
- `List Metrics`

## Route Envelope

Documented route family:
- `GET {endpoint}/tests?api-version=2022-11-01`
- `GET {endpoint}/tests/{testId}?api-version=2022-11-01`
- `PATCH {endpoint}/tests/{testId}?api-version=2022-11-01`
- `DELETE {endpoint}/tests/{testId}?api-version=2022-11-01`
- `GET {endpoint}/tests/{testId}/files?api-version=2022-11-01`
- `GET {endpoint}/tests/{testId}/files/{fileName}?api-version=2022-11-01`
- `PUT {endpoint}/tests/{testId}/files/{fileName}?api-version=2022-11-01`
- `PATCH {endpoint}/tests/{testId}/app-components?api-version=2022-11-01`
- `PATCH {endpoint}/tests/{testId}/server-metrics-config?api-version=2022-11-01`
- `GET {endpoint}/test-runs?api-version=2022-11-01`
- `GET {endpoint}/test-runs/{testRunId}?api-version=2022-11-01`
- `PATCH {endpoint}/test-runs/{testRunId}?api-version=2022-11-01`
- `DELETE {endpoint}/test-runs/{testRunId}?api-version=2022-11-01`
- `POST {endpoint}/test-runs/{testRunId}:stop?api-version=2022-11-01`
- `GET {endpoint}/test-runs/{testRunId}/files/{fileName}?api-version=2022-11-01`
- `PATCH {endpoint}/test-runs/{testRunId}/app-components?api-version=2022-11-01`
- `PATCH {endpoint}/test-runs/{testRunId}/server-metrics-config?api-version=2022-11-01`
- `GET {endpoint}/test-runs/{testRunId}/metric-namespaces?api-version=2022-11-01`
- `GET {endpoint}/test-runs/{testRunId}/metric-definitions?api-version=2022-11-01&metricNamespace={metricNamespace}`
- `GET {endpoint}/test-runs/{testRunId}/metric-dimensions/{dimensionName}?api-version=2022-11-01&metricName={metricName}&metricNamespace={metricNamespace}`
- `POST {endpoint}/test-runs/{testRunId}/metrics?api-version=2022-11-01&metricname={metricname}&metricNamespace={metricNamespace}&timespan={timespan}`

Stackyard emulation prefix should remain distinct from ARM `loadtesting`.

Recommended Stackyard route family:
- `/azure/loadtesting-data-plane/2022-11-01/tests`
- `/azure/loadtesting-data-plane/2022-11-01/tests/{testId}`
- `/azure/loadtesting-data-plane/2022-11-01/tests/{testId}/files`
- `/azure/loadtesting-data-plane/2022-11-01/tests/{testId}/files/{fileName}`
- `/azure/loadtesting-data-plane/2022-11-01/tests/{testId}/app-components`
- `/azure/loadtesting-data-plane/2022-11-01/tests/{testId}/server-metrics-config`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}:stop`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/files/{fileName}`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/app-components`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/server-metrics-config`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/metric-namespaces`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/metric-definitions`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/metric-dimensions/{dimensionName}`
- `/azure/loadtesting-data-plane/2022-11-01/test-runs/{testRunId}/metrics`

Important routing note:
- This API does not use the Azure ARM `subscriptions/.../providers/...` envelope.
- It still requires `api-version` on every route.
- Upstream docs model `{endpoint}` as the data-plane service URL associated with a load test resource, so Stackyard should emulate a stable data-plane root and keep service state local to the emulator.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- security is OAuth2 with Microsoft Entra ID
- errors use `Azure.Core.Foundations.ErrorResponse`
- test and test-run IDs require the pattern `^[a-z0-9_-]*$`
- create-or-update methods use `PATCH` with `application/merge-patch+json`
- list routes commonly return paged `value` payloads
- file upload and download routes are separate from test/test-run metadata

### Load Test Administration

`List Tests`
- `GET /tests`
- optional filters include:
  - `orderby`
  - `search`
  - `lastModifiedStartTime`
  - `lastModifiedEndTime`
  - `maxpagesize`
- response type: paged test list

`Get Test`
- `GET /tests/{testId}`
- response type: `Test`

`Create Or Update Test`
- `PATCH /tests/{testId}`
- request body is merge-patch JSON
- reviewed request fields include test metadata and certificate references
- response type: `Test`

`Delete Test`
- `DELETE /tests/{testId}`

Modeling implications:
- keep one canonical test store keyed by `testId`
- allow patch-style create/update semantics
- preserve high-value metadata fields while keeping validation shallow on the first pass
- implement predictable filtering and ordering for `List Tests`

### Test Files

`List Test Files`
- `GET /tests/{testId}/files`
- response type: `PagedTestFileInfo`

`Get Test File`
- `GET /tests/{testId}/files/{fileName}`
- response type: `FileInfo`-style metadata, not binary content

`Upload Test File`
- `PUT /tests/{testId}/files/{fileName}`
- request body is `application/octet-stream`
- optional query parameter: `fileType`
- file size note in docs: max 50 MB

`Delete Test File`
- `DELETE /tests/{testId}/files/{fileName}`

Modeling implications:
- store uploaded file bytes and metadata per `(testId, fileName)`
- keep file content retrieval separate from file metadata because the reviewed surface exposes metadata, not test-file content download, on admin routes
- model file type deterministically from the query or filename extension

### Test App Components And Server Metrics Config

`Create Or Update App Components`
- `PATCH /tests/{testId}/app-components`
- request body contains `components`, keyed by fully qualified Azure resource ID

`Get App Components`
- `GET /tests/{testId}/app-components`

`Create Or Update Server Metrics Config`
- `PATCH /tests/{testId}/server-metrics-config`
- request body contains `metrics`, keyed by metric ID

`Get Server Metrics Config`
- `GET /tests/{testId}/server-metrics-config`

Modeling implications:
- keep `appComponents` and `serverMetricsConfig` as first-class nested collections on each test
- use ARM resource IDs and metric IDs as opaque keys
- return stable merged state after repeated patch calls

### Test Runs

`List Test Runs`
- `GET /test-runs`
- optional filters include search, ordering, status, and date ranges
- response type: paged `TestRun` list

`Get Test Run`
- `GET /test-runs/{testRunId}`
- response type: `TestRun`
- reviewed response fields include status, artifacts, statistics, pass/fail results, and timestamps

`Create Or Update Test Run`
- `PATCH /test-runs/{testRunId}`
- optional query parameter: `oldTestRunId`
- creates and starts a test run

`Stop Test Run`
- `POST /test-runs/{testRunId}:stop`
- returns updated `TestRun`

`Delete Test Run`
- `DELETE /test-runs/{testRunId}`

Modeling implications:
- keep a dedicated run store keyed by `testRunId`
- persist `testId` linkage and synthetic run lifecycle state
- implement deterministic status transitions such as:
  - `ACCEPTED`
  - `NOTSTARTED`
  - `EXECUTING`
  - `DONE`
  - `CANCELLING`
  - `CANCELLED`
- support rerun semantics when `oldTestRunId` is provided by cloning the earlier run’s test linkage and configuration

### Test Run App Components, Server Metrics Config, And Files

`Get Test Run File`
- `GET /test-runs/{testRunId}/files/{fileName}`
- returns `TestRunFileInfo`

`Get App Components`
- `GET /test-runs/{testRunId}/app-components`

`Create Or Update App Components`
- `PATCH /test-runs/{testRunId}/app-components`

`Get Server Metrics Config`
- `GET /test-runs/{testRunId}/server-metrics-config`

`Create Or Update Server Metrics Config`
- `PATCH /test-runs/{testRunId}/server-metrics-config`

Modeling implications:
- seed each run with synthetic output artifact metadata even if no real execution happens
- keep run-scoped app components and metrics config separate from test-scoped state, but allow easy seeding from the parent test on run creation

### Metrics

`List Metric Namespaces`
- returns namespaces available for a test run

`List Metric Definitions`
- `GET /test-runs/{testRunId}/metric-definitions`
- requires `metricNamespace`

`List Metric Dimension Values`
- route is dimension-name scoped
- requires at least `metricName` and `metricNamespace`

`List Metrics`
- `POST /test-runs/{testRunId}/metrics`
- requires:
  - `metricname`
  - `metricNamespace`
  - `timespan`
- optional:
  - `aggregation`
  - `interval`
- request body may include dimension filters
- response type: paged time-series metrics

Modeling implications:
- return a small deterministic metrics catalog per run
- generate time-series values from the run ID, metric name, and requested timespan so repeated queries are stable
- support simple dimension filtering for the most common dimensions such as sampler name, result, or instance

## Non-Goals

- do not execute real JMeter or Locust workloads
- do not implement real Azure Monitor integration or dynamic metric backends
- do not enforce every upstream date-range, sort-order, or max-page-size rule in the first pass
- do not implement full certificate validation, secret resolution, or identity-based authorization
- do not model every preview field from newer data-plane versions not present in `2022-11-01`

## Stage 0: Coverage Naming And Route Claiming

- Add a dedicated Azure Load Testing data-plane router
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add the aliases listed above
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route recognition tests for:
  - `/tests`
  - `/tests/{testId}`
  - test file routes
  - `/test-runs`
  - `/test-runs/{testRunId}`
  - metric query routes

Exit criteria:
- the new service key resolves cleanly across the Azure coverage scripts
- router tests prove the new data-plane requests do not fall through to `NotImplemented`

## Stage 1: Core Test Administration

- Implement:
  - `GET /tests`
  - `GET /tests/{testId}`
  - `PATCH /tests/{testId}`
  - `DELETE /tests/{testId}`
- Persist deterministic `Test` resources with:
  - IDs
  - display names
  - descriptions
  - created/updated timestamps
  - load-test resource reference

Exit criteria:
- the core test administration routes return modeled success payloads
- repeated patch/get/list/delete flows are deterministic

## Stage 2: Test Files, App Components, And Server Metrics Config

- Implement:
  - `GET /tests/{testId}/files`
  - `GET /tests/{testId}/files/{fileName}`
  - `PUT /tests/{testId}/files/{fileName}`
  - `DELETE /tests/{testId}/files/{fileName}`
  - `GET /tests/{testId}/app-components`
  - `PATCH /tests/{testId}/app-components`
  - `GET /tests/{testId}/server-metrics-config`
  - `PATCH /tests/{testId}/server-metrics-config`
- Preserve uploaded file bytes and metadata
- Keep deterministic nested-state merging for app components and server metrics config

Exit criteria:
- all reviewed test child routes return typed, non-placeholder payloads
- no test-file or nested test-config route returns `NotImplemented`

## Stage 3: Core Test Runs

- Implement:
  - `GET /test-runs`
  - `GET /test-runs/{testRunId}`
  - `PATCH /test-runs/{testRunId}`
  - `POST /test-runs/{testRunId}:stop`
  - `DELETE /test-runs/{testRunId}`
- Seed run artifacts and summary statistics on run creation
- Support rerun-clone behavior through `oldTestRunId`

Exit criteria:
- runs have stable lifecycle state and summary payloads
- stop/delete behavior is deterministic and coverage-friendly

## Stage 4: Run Files, Run Config, And Metrics

- Implement:
  - `GET /test-runs/{testRunId}/files/{fileName}`
  - `GET /test-runs/{testRunId}/app-components`
  - `PATCH /test-runs/{testRunId}/app-components`
  - `GET /test-runs/{testRunId}/server-metrics-config`
  - `PATCH /test-runs/{testRunId}/server-metrics-config`
  - `GET /test-runs/{testRunId}/metric-namespaces`
  - `GET /test-runs/{testRunId}/metric-definitions`
  - `GET /test-runs/{testRunId}/metric-dimensions/{dimensionName}`
  - `POST /test-runs/{testRunId}/metrics`
- Generate deterministic metric catalogs and time-series payloads

Exit criteria:
- metric and artifact routes return modeled success payloads with stable shapes
- no reviewed run metrics route falls through to `NotImplemented`

## Stage 5: Tests And Contract Tightening

- Add focused tests in `internal/server/provider_azure_loadtesting_data_plane_test.go` covering:
  - test CRUD/list flows
  - test-file upload/list/get/delete
  - test app-components and server-metrics-config patch/get
  - run create/get/list/stop/delete
  - run file and nested config retrieval
  - metric namespace/definition/value queries
  - 404 and validation failures for malformed IDs or missing required parameters

Exit criteria:
- focused provider tests cover the full reviewed route surface implemented above
- data-plane coverage can run without shape mismatches or placeholder responses

## Stage 6: Azure Go SDK Example

- Add example under:
  - `examples/azure/loadtesting-dataplane-2022-11-01`
- Use the official Azure Go SDK core runtime:
  - `azcore/runtime`
  - `azcore/policy`
- Example flow should cover:
  - create or update a test
  - upload and list a test file
  - set app components and server metrics config
  - create a test run
  - get and stop the run
  - query metric namespaces or metrics

Exit criteria:
- example builds cleanly in repo CI conventions
- example demonstrates both admin and run-side data-plane routes

## Validation Plan

After implementation, validate with:

- targeted server tests:
  - `go test ./internal/server -run 'TestAzureLoadTestingDataPlane' -count=1`
- example build:
  - `go build ./examples/azure/loadtesting-dataplane-2022-11-01`
- coverage checks:
  - `python3 scripts/azure-contract-coverage.py --service loadtesting_data_plane --require-service loadtesting_data_plane --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service loadtesting_data_plane --require-service loadtesting_data_plane --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service loadtesting_data_plane --require-service loadtesting_data_plane --fail-on any`
  - `python3 scripts/azure-endpoint-coverage.py --service loadtesting_data_plane --require-service loadtesting_data_plane --fail-on any`

Success target:
- all reviewed data-plane routes return modeled success or modeled service errors
- no reviewed Load Testing data-plane route returns `NotImplemented`
- the example compiles and exercises the new provider cleanly
