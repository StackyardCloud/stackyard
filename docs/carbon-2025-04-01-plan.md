# Azure Carbon (`carbon-2025-04-01`) Staged Plan

## Objective

Emulate the Azure Carbon optimization management API for `2025-04-01` with deterministic local behavior for:

- provider `operations`
- carbon emission report queries
- carbon emission data available date range queries
- an Azure Go SDK example under `examples/azure/carbon-2025-04-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the Azure Carbon optimization management plane under `Microsoft.Carbon`.
- This is not a tracked ARM resource provider with resource-group CRUD.
- The reviewed `2025-04-01` surface is provider-rooted and query-oriented:
  - `GET /providers/Microsoft.Carbon/operations`
  - `POST /providers/Microsoft.Carbon/carbonEmissionReports`
  - `POST /providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange`

Important modeling note:

- The emulator should not invent `accounts`, `profiles`, or other persistent ARM resources that do not appear in the reviewed surface.
- The useful deterministic behavior is in validated query payload handling and stable synthesized report output.
- Because both query routes are provider-rooted, service state can stay small and largely fixture-driven rather than CRUD-driven.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/carbon/operation-groups?view=rest-carbon-2025-04-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/carbon/operations?view=rest-carbon-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/carbon/carbon-service?view=rest-carbon-2025-04-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/carbon/operations/list?view=rest-carbon-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/carbon/carbon-service/query-carbon-emission-reports?view=rest-carbon-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/carbon/carbon-service/query-carbon-emission-data-available-date-range?view=rest-carbon-2025-04-01`

Official Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/carbonoptimization/armcarbonoptimization`

## SDK Selection Note

- The reviewed method pages include official Go samples using `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/carbonoptimization/armcarbonoptimization`.
- The generated examples indicate the future Stackyard example should use:
  - `armcarbonoptimization.NewClientFactory`
  - `NewOperationsClient`
  - `NewCarbonServiceClient`
- Because this surface is small and provider-rooted, the example should stay typed and not fall back to raw `azcore/runtime` unless the local SDK package version is missing the exact `2025-04-01` client methods.

## Service Naming

Canonical Stackyard naming:

- service key: `carbon`
- display name: `Azure Carbon`
- API version token: `2025-04-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_carbon.go`
- tests: `internal/server/provider_azure_carbon_test.go`
- example: `examples/azure/carbon-2025-04-01`
- plan doc: `docs/carbon-2025-04-01-plan.md`

Recommended coverage aliases:

- `carbon`
- `azure_carbon`
- `microsoft_carbon`
- `carbon_2025_04_01`
- `rest_carbon_2025_04_01`

## API Surface Summary

The reviewed pages expose only two functional route families plus provider discovery:

- provider operations:
  - `/providers/Microsoft.Carbon/operations`
- provider-rooted report query:
  - `/providers/Microsoft.Carbon/carbonEmissionReports`
- provider-rooted available-date query:
  - `/providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange`

Modeling implications:

- there is no resource-group scope, subscription path segment, or child-resource tree to persist
- the main implementation challenge is request validation and deterministic report shaping
- `carbonEmissionReports` should support multiple report request shapes without exposing random or unstable output
- paging for detailed reports should be deterministic when `pageSize` and `skipToken` are used

## Contract Notes From Reviewed Method Pages

Shared traits:

- failures use ARM-style `ErrorResponse`
- responses use standard management-plane error shapes even though the service is provider-rooted
- OAuth scope is standard Azure ARM `user_impersonation`

### Operations

Reviewed method page:

- `List`

Key traits from the reviewed page:

- route:
  - `/providers/Microsoft.Carbon/operations`
- returns `OperationListResult`
- the sample response shows only two functional operations beyond discovery:
  - carbon emission reports
  - carbon emission data available date range

Modeling implications:

- return a very small deterministic operation set aligned to the routes actually implemented
- do not pad the operation list with imaginary resource CRUD verbs

### Carbon Emission Reports

Reviewed method page:

- `Query Carbon Emission Reports`

Key traits from the reviewed page:

- route:
  - `/providers/Microsoft.Carbon/carbonEmissionReports`
- method:
  - `POST`
- request body is a discriminated query shape keyed by `reportType`
- reviewed request families include:
  - `Overall Summary Report`
  - `Monthly Summary Report`
  - `Item Details Report`
  - `Top Items Summary Report`
  - `Top Items Monthly Summary Report`
- common request fields include:
  - `subscriptionList`
  - `carbonScopeList`
  - `dateRange`
  - optional `locationList`
  - optional `resourceGroupUrlList`
  - optional `resourceTypeList`
- additional constraints visible in the reviewed page:
  - `subscriptionList` max length is `100`
  - `pageSize` range is `1..5000`
  - `topItems` range is `1..10`
  - `Item Details Report` and `Top Items Summary Report` support only one month at a time, so the start and end dates should be equal

Modeling implications:

- the emulator should validate `reportType` and reject unsupported combinations with a structured error
- use a deterministic seeded emissions dataset keyed by subscription, month, scope, category, resource type, resource group, and location
- generate:
  - aggregate summaries for overall and monthly reports
  - stable sorted rows for item-details and top-items reports
- implement deterministic `skipToken` behavior for item-detail pagination instead of returning one giant fixed page

### Carbon Emission Data Available Date Range

Reviewed method page:

- `Query Carbon Emission Data Available Date Range`

Key traits from the reviewed page:

- route:
  - `/providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange`
- method:
  - `POST`
- response type:
  - `CarbonEmissionDataAvailableDateRange`
- response fields:
  - `startDate`
  - `endDate`

Modeling implications:

- this route can be seeded as a small read-only fixture
- the example in the reviewed page returns a simple date range, so the emulator should keep this route straightforward and stable

## Recommended State Model

Keep a compact provider-scoped state model:

- one seeded operation list
- one seeded available-date-range fixture
- one deterministic synthetic emissions dataset from which all report variants are derived

Recommended seeded emissions dimensions:

- subscription ID
- month
- emission scope
- location
- resource group URL
- resource type
- resource ID

Recommended implementation behavior:

- normalize incoming filter values to the lowercase conventions described in the docs
- derive summary and top-N responses from the same canonical emissions rows
- encode pagination state in a deterministic `skipToken` based on the filtered row offset rather than storing opaque server sessions

## Staged Implementation Plan

### Stage 1: Route Skeleton and Coverage Wiring

Create the initial provider and route registration for:

- `GET /providers/Microsoft.Carbon/operations`
- `POST /providers/Microsoft.Carbon/carbonEmissionReports`
- `POST /providers/Microsoft.Carbon/queryCarbonEmissionDataAvailableDateRange`

Include:

- initial `carbon` coverage aliases in the Azure coverage scripts
- a provider-specific request parser and error helpers
- server state registration in `internal/server/http.go`

### Stage 2: Provider Operations

Implement:

- `Operations.List`

Recommended outcome:

- the returned operation list exactly matches the emulator’s supported Carbon surface
- tests verify the provider name, route, and action descriptions remain stable

### Stage 3: Available Date Range Query

Implement:

- `CarbonService.QueryCarbonEmissionDataAvailableDateRange`

Recommended outcome:

- the route returns a fixed but realistic `startDate` and `endDate`
- the response is stable across repeated calls and independent of unrelated service state

### Stage 4: Carbon Emission Reports Query

Implement:

- `CarbonService.QueryCarbonEmissionReports`

Recommended outcome:

- supported `reportType` variants return coherent shapes
- the same input payload always yields the same sorted output
- invalid payload combinations return structured management-plane errors
- pagination works for detail-style reports where the request uses `pageSize` and `skipToken`

### Stage 5: Example, Tests, and Final Coverage Validation

Add:

- focused provider tests in `internal/server/provider_azure_carbon_test.go`
- the Azure Go SDK example in `examples/azure/carbon-2025-04-01`
- service documentation mapping in the coverage scripts

Recommended example flow:

- create the client factory
- list provider operations
- query the available date range
- submit one report query, preferably a summary request and one filtered detail request

## Testing Strategy

Target focused tests rather than broad generic route smoke tests.

Minimum test coverage should include:

- `operations` returns the expected Carbon provider actions
- available-date-range returns the seeded start and end values
- summary report requests succeed with valid payloads
- detail report requests validate `pageSize`, `skipToken`, and one-month constraints
- unsupported `reportType` or malformed filters return non-`200` structured errors

Recommended targeted cases:

- `Overall Summary Report` with two subscriptions
- `Monthly Summary Report` across multiple months
- `Item Details Report` with pagination
- `Top Items Summary Report` with `topItems`
- invalid uppercase or malformed resource-group URL input normalization behavior if the implementation chooses strict validation

## Coverage Script Updates To Make During Implementation

Update the following scripts:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for:

- `carbon`
- `azure_carbon`
- `microsoft_carbon`
- `carbon_2025_04_01`
- `rest_carbon_2025_04_01`

Add the plan doc mapping:

- `carbon`: `carbon-2025-04-01-plan.md`

## Validation Targets For The Future Implementation

After implementation, validate with:

- `go test ./internal/server -run 'TestAzureCarbon' -count=1`
- `go build ./examples/azure/carbon-2025-04-01`
- `python3 scripts/azure-contract-coverage.py --service carbon --require-service carbon --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service carbon --require-service carbon --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service carbon --require-service carbon --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service carbon --require-service carbon --fail-on any`

Expected quality bar:

- no Carbon endpoint falls through to `NotImplemented`
- the example builds cleanly against the chosen Azure Go SDK package
- the coverage scripts recognize `carbon` and map it to this plan doc
