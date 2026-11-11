# Azure Application Insights (`application-insights-v1`) Staged Plan

## Objective

Emulate the Azure Application Insights Data Access REST API (`v1`) with deterministic local behavior for:

- Analytics query execution
- Event lookup and event OData queries
- Metric retrieval and metric metadata
- Application metadata retrieval
- an Azure Go SDK-based example for `application-insights-v1`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-application-insights-v1` is not an ARM management-plane "Application Insights for Containers" service.
- It is the Application Insights data-access API hosted under the `api.applicationinsights.io` data-plane endpoint family.
- If the intended target was Azure Monitor Container Insights or a different ARM provider, that is a different service surface and should be planned separately.

## Primary References Reviewed

Primary entry page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/application-insights/`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/application-insights/query?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/events?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metrics?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metadata?view=rest-application-insights-v1`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/application-insights/query/execute?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/query/get?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/events/get?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/events/get-by-type?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/events/get-odata-metadata?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metrics/get?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metrics/get-metadata?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metrics/get-multiple?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metadata/get?view=rest-application-insights-v1`
- `https://learn.microsoft.com/en-us/rest/api/application-insights/metadata/post?view=rest-application-insights-v1`

## Service Naming

Canonical Stackyard naming:
- service key: `application_insights`
- display name: `Azure Application Insights`
- API version token: `v1`

Recommended repo paths:
- implementation: `internal/server/provider_azure_application_insights.go`
- tests: `internal/server/provider_azure_application_insights_test.go`
- example: `examples/azure/application-insights-v1`
- plan doc: `docs/application-insights-v1-plan.md`

## Surface Summary

The reviewed REST surface is split into 4 operation groups:

1. `Query`
2. `Events`
3. `Metrics`
4. `Metadata`

Total linked method pages reviewed: `10`

Method inventory:

`Query`
- `Execute`
- `Get`

`Events`
- `Get`
- `Get By Type`
- `Get Odata Metadata`

`Metrics`
- `Get`
- `Get Metadata`
- `Get Multiple`

`Metadata`
- `Get`
- `Post`

## Route Envelope

Documented route family:
- `/v1/apps/{appId}/query`
- `/v1/apps/{appId}/events/{eventType}`
- `/v1/apps/{appId}/events/{eventType}/{eventId}`
- `/v1/apps/{appId}/events/$metadata`
- `/v1/apps/{appId}/metrics/{metricId}`
- `/v1/apps/{appId}/metrics/metadata`
- `/v1/apps/{appId}/metrics`
- `/v1/apps/{appId}/metadata`

Stackyard emulation prefix:
- `/azure/application-insights/v1/apps/{appId}/...`

Important routing note:
- This API does not use the Azure ARM `management.azure.com` envelope.
- It also does not use an `api-version` query parameter.
- The version is part of the URL path: `/v1/apps/...`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- Security is OAuth2 with `user_impersonation` against the Application Insights data-plane resource.
- Success responses are immediate; no ARM-style LRO behavior is documented.
- Error responses use `errorResponse` with nested `errorInfo` and `errorDetail`.
- Most endpoints return JSON; `Events - Get Odata Metadata` returns XML EDMX.

### Query

`Query - Get`
- `GET /v1/apps/{appId}/query?query={query}`
- optional: `timespan`
- response: `queryResults`

`Query - Execute`
- `POST /v1/apps/{appId}/query`
- body:
  - required: `query`
  - optional: `applications`
  - optional: `timespan`
- response: `queryResults`

Modeling implications:
- deterministic Kusto table output is sufficient for the first slice
- `GET` and `POST` should share the same response builder
- cross-application `applications[]` can be accepted and echoed without implementing real federation

### Events

`Events - Get`
- `GET /v1/apps/{appId}/events/{eventType}/{eventId}`
- optional: `timespan`
- response: a single event document

`Events - Get By Type`
- `GET /v1/apps/{appId}/events/{eventType}`
- optional query parameters include:
  - `timespan`
  - `$filter`
  - `$search`
  - `$orderby`
  - `$select`
  - `$skip`
  - `$top`
  - `$format`
  - `$count`
  - `$apply`
- response: event collection / OData-style result

`Events - Get Odata Metadata`
- `GET /v1/apps/{appId}/events/$metadata`
- response content type: `application/xml;charset=utf-8`

Documented event-type enum includes:
- `traces`
- `customEvents`
- `pageViews`
- `requests`
- `dependencies`
- `exceptions`
- `availabilityResults`
- `$all`

Modeling implications:
- return deterministic event IDs and timestamped rows
- implement minimal OData-style list envelope with `value` and optional `@odata.count`
- return a fixed EDMX document for `$metadata`

### Metrics

`Metrics - Get`
- `GET /v1/apps/{appId}/metrics/{metricId}`
- optional query parameters:
  - `timespan`
  - `interval`
  - `aggregation`
  - `segment`
  - `top`
  - `orderby`
  - `filter`
- response: `metricsResult`

`Metrics - Get Metadata`
- `GET /v1/apps/{appId}/metrics/metadata`
- response: object keyed by metric ID with supported aggregations and group-by fields

`Metrics - Get Multiple`
- `POST /v1/apps/{appId}/metrics`
- request body: `metricsPostBodySchema[]`
- response: `metricsResults[]`

Documented standard metric IDs include examples such as:
- `requests/count`
- `requests/duration`
- `requests/failed`
- `pageViews/count`
- `dependencies/count`
- `exceptions/count`
- `availabilityResults/availabilityPercentage`

Modeling implications:
- start with a small deterministic supported metric set
- support both singleton and batch metric retrieval using the same underlying fixture builder
- keep segment/group-by behavior shallow in the first slice

### Metadata

`Metadata - Get`
- `GET /v1/apps/{appId}/metadata`
- response: `metadataResults`

`Metadata - Post`
- `POST /v1/apps/{appId}/metadata`
- response: `metadataResults`

Documented metadata payload includes:
- `tables`
- `tableGroups`
- `applications`
- optional function metadata in definitions

Modeling implications:
- `GET` and `POST` can share one deterministic metadata fixture
- metadata should align with the query/events/metrics fixtures so columns and names are internally consistent

## Non-Goals

- Do not implement Azure ARM Application Insights component management in this service.
- Do not implement live Kusto semantics.
- Do not implement live OAuth token validation beyond existing repo conventions for Azure data-plane shims.
- Do not implement full OData filtering semantics in the first slice.
- Do not conflate this API with Azure Monitor Container Insights or Application Insights component ARM APIs.

## Planned Repo Shape

Implementation:
- `internal/server/provider_azure_application_insights.go`

Tests:
- `internal/server/provider_azure_application_insights_test.go`

Example:
- `examples/azure/application-insights-v1/main.go`
- `examples/azure/application-insights-v1/Dockerfile`
- `examples/azure/application-insights-v1/docker-compose.yml`

Coverage wiring:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Example Strategy

Because this surface is data-plane and not ARM, the example should not use an ARM client factory.

Planned example approach:
- use Azure Go SDK primitives for request pipeline/auth shape
- reuse the repo's `azsdkshim` transport override pattern
- issue typed or thin-wrapper `GET`/`POST` requests against `/azure/application-insights/v1/apps/{appId}/...`

Pragmatic implementation note:
- Based on the reviewed docs, this surface does not present an obvious modern generated Go client analogous to ARM providers.
- The implementation plan therefore assumes a lightweight Azure Go SDK example built on `azcore` pipeline pieces plus repo shim transport, rather than inventing a fake ARM client.
- If a suitable first-party generated client is identified during implementation, the example can be upgraded to that client without changing the route plan.

Representative example calls:
- query via `GET`
- query via `POST`
- event-by-type query
- single event lookup
- metrics metadata lookup
- single metric retrieval
- batch metric retrieval
- app metadata via `GET`

## Coverage Naming and Aliases

Recommended aliases to add:
- `application_insights`
- `applicationinsights`
- `application_insights_v1`
- `application-insights-v1`
- `azure_application_insights`
- `app_insights`

Recommended doc mapping:
- `application_insights -> application-insights-v1-plan.md`

## Staged Implementation Plan

### Stage 0: Contract Skeleton

- Add a dedicated Azure Application Insights router.
- Recognize the `/azure/application-insights/v1/apps/*` family.
- Route all four operation groups through one provider entrypoint.
- Add route-recognition tests for representative `GET` and `POST` paths.

Exit criteria:
- Route ownership is deterministic and contract-tested.

### Stage 1: Shared Validation and Error Foundation

- Validate required `appId` path parts.
- Validate required `query` for `Query - Get` and `Query - Execute`.
- Validate required `eventType`, `eventId`, and `metricId` path segments where applicable.
- Return consistent data-plane `errorResponse` payloads for malformed requests.
- Preserve XML content type for `events/$metadata`.

Exit criteria:
- Invalid requests fail with deterministic, modeled error contracts.

### Stage 2: Metadata and Shared Schema Fixtures

- Implement deterministic application metadata fixtures.
- Return a stable `metadataResults` shape for both `GET` and `POST`.
- Define a shared table/schema fixture used by query and event responses.
- Add tests for metadata `GET` and `POST`.

Exit criteria:
- Metadata endpoints are green and act as the schema source of truth.

### Stage 3: Query Endpoints

- Implement `Query - Get`.
- Implement `Query - Execute`.
- Return deterministic `queryResults.tables[]` with stable columns and rows.
- Support optional `timespan` passthrough in the response fixture or request echo fields if helpful for debugging.

Exit criteria:
- Both query methods succeed with modeled results and matching test coverage.

### Stage 4: Events Endpoints

- Implement `Events - Get`.
- Implement `Events - Get By Type`.
- Implement `Events - Get Odata Metadata`.
- Add deterministic event fixtures for the documented event types.
- Return a fixed EDMX XML document for `$metadata`.

Exit criteria:
- Event list, single event, and OData metadata paths are deterministic and tested.

### Stage 5: Metrics Endpoints

- Implement `Metrics - Get`.
- Implement `Metrics - Get Metadata`.
- Implement `Metrics - Get Multiple`.
- Build a small stable metrics catalog with deterministic numeric values.
- Reuse fixtures across singleton and batched metric retrieval.

Exit criteria:
- All metrics methods return modeled data with no `NotImplemented` behavior.

### Stage 6: Azure Go SDK Example

- Add `examples/azure/application-insights-v1`.
- Demonstrate representative reads across query, events, metrics, and metadata.
- Keep example transport compatible with local Stackyard routing under `/azure/application-insights/v1`.

Exit criteria:
- Example compiles and exercises the new service surface end to end.

### Stage 7: Coverage Wiring

- Add the service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add the plan-doc mapping in doc coverage.
- Verify the service resolves cleanly through Azure coverage discovery.

Exit criteria:
- Coverage scripts recognize `application_insights` and include it in reporting.

## Validation Targets For Implementation

When the implementation phase happens, the expected validation loop is:

- `go test ./internal/server -run 'TestAzureApplicationInsights' -count=1`
- `go build ./examples/azure/application-insights-v1`
- `python3 scripts/azure-contract-coverage.py --service application_insights --require-service application_insights --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service application_insights --require-service application_insights --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service application_insights --require-service application_insights --fail-on any`

Success criteria:
- all recognized Application Insights endpoints return deterministic success or modeled validation errors
- no routed Application Insights endpoint returns `NotImplemented`
- coverage discovery can resolve `application_insights`
