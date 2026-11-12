# Azure Consumption (`consumption-2024-08-01`) Staged Plan

## Objective

Emulate the Azure Consumption ARM management API for `2024-08-01` with deterministic local behavior for:

- provider `operations`
- scope-based reporting families such as `budgets`, `charges`, `marketplaces`, `priceSheets`, `tags`, and `usageDetails`
- billing-account and billing-profile families such as `balances`, `credits`, `events`, and `lots`
- reservation reporting families such as `reservationRecommendations`, `reservationRecommendationDetails`, `reservationTransactions`, `reservationDetails`, and `reservationSummaries`
- management-group `aggregatedCost`
- an Azure Go SDK example under `examples/azure/consumption-2024-08-01`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed Learn surface is the ARM management plane under `Microsoft.Consumption`, not Cost Management exports and not the separate `Microsoft.Billing` resource provider already represented elsewhere in the repo.
- This API is mostly read-only reporting plus one mutable ARM-style family, `budgets`.
- The surface is heavily scope-dependent. Routes are split across:
  - generic `{scope}` resource IDs
  - `providers/Microsoft.Billing/billingAccounts/...`
  - management groups
  - reservation-order and reservation identifiers
- Several Learn pages are authorization-gated, but they still expose enough route, parameter, request-body, response, and example metadata to stage the emulator cleanly.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/consumption/operation-groups?view=rest-consumption-2024-08-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/consumption/aggregated-cost?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/balances?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/budgets?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/charges?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/credits?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/events?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/lots?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/marketplaces?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/price-sheet?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservation-recommendation-details?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservation-recommendations?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservation-transactions?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservations-details?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservations-summaries?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/tags?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/usage-details?view=rest-consumption-2024-08-01`

Representative linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/consumption/operations/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/budgets/create-or-update?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/budgets/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/charges/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/marketplaces/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/usage-details/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/price-sheet/get?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/price-sheet/get-by-billing-period?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/credits/get?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/events/list-by-billing-account?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/lots/list-by-customer?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/tags/get?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservation-recommendations/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservation-recommendation-details/get?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservation-transactions/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservations-details/list?view=rest-consumption-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/consumption/reservations-summaries/list?view=rest-consumption-2024-08-01`

## SDK Selection Note

- The Azure Go SDK has a dedicated package for this surface: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption`.
- The package exposes typed clients for a large part of the reviewed surface, including:
  - `OperationsClient`
  - `BudgetsClient`
  - `UsageDetailsClient`
  - `ChargesClient`
  - `BalancesClient`
  - `CreditsClient`
  - `EventsClient`
  - `LotsClient`
  - `PriceSheetClient`
  - `TagsClient`
  - reservation clients
  - `AggregatedCostClient`
- Practical constraint:
  - the published `armconsumption` package visible on `pkg.go.dev` is older than the exact `2024-08-01` Learn surface, so the future example should prefer typed `armconsumption` clients where they overlap and use narrow `azcore/runtime` fallback only if a reviewed `2024-08-01` route is missing from the generated package revision available in the repo.

## Service Naming

Canonical Stackyard naming:

- service key: `consumption`
- display name: `Azure Consumption`
- API version token: `2024-08-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_consumption.go`
- tests: `internal/server/provider_azure_consumption_test.go`
- example: `examples/azure/consumption-2024-08-01`
- plan doc: `docs/consumption-2024-08-01-plan.md`

Recommended coverage aliases:

- `consumption`
- `azure_consumption`
- `microsoft_consumption`
- `consumption_2024_08_01`
- `rest_consumption_2024_08_01`

## API Surface Summary

The reviewed pages expose several distinct route families.

Provider-level:

- `/providers/Microsoft.Consumption/operations`

Generic scope-based routes using `{scope}`:

- `/{scope}/providers/Microsoft.Consumption/budgets`
- `/{scope}/providers/Microsoft.Consumption/budgets/{budgetName}`
- `/{scope}/providers/Microsoft.Consumption/charges`
- `/{scope}/providers/Microsoft.Consumption/marketplaces`
- `/{scope}/providers/Microsoft.Consumption/tags`
- `/{scope}/providers/Microsoft.Consumption/usageDetails`
- `/{scope}/providers/Microsoft.Consumption/reservationRecommendations`
- `/{scope}/providers/Microsoft.Consumption/reservationRecommendationDetails`
- `/{scope}/providers/Microsoft.Consumption/reservationDetails`
- `/{scope}/providers/Microsoft.Consumption/reservationSummaries`

Management-group-specific routes:

- `/providers/Microsoft.Management/managementGroups/{managementGroupId}/providers/Microsoft.Consumption/aggregatedcost`
- billing-period variant for the same family

Billing-account, billing-profile, and customer families under `Microsoft.Billing`:

- balances
- credits
- events
- lots
- pricesheets
- charges and usage-details examples on billing-account, billing-profile, invoice-section, and customer scopes
- reservation-transactions examples on billing-account and billing-profile scopes

Reservation-order scoped routes:

- reservation transactions
- reservation details
- reservation summaries
- reservation-order-and-reservation variants

Modeling implications:

- this is not a single clean resource tree; it is a reporting API collection with many parallel scope patterns
- `budgets` is the only core CRUD family and should be treated as the first durable ARM-style resource
- most other families are deterministic list/get responses driven by scope parsing and query parameters
- many routes support paging or filtering and should emit realistic `nextLink` or `properties.nextLink` shapes where the docs show them

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use `ErrorResponse`
- most list routes return `value` plus optional `nextLink`
- many routes allow several scope shapes, so path recognition must key off the suffix after the scope rather than hard-coding only subscription paths

### Provider Operations

From `Operations - List`:

- route:
  - `GET /providers/Microsoft.Consumption/operations?api-version=2024-08-01`
- response:
  - `value`
  - each operation includes `name`, `display`, and `id`
- the sample operation list confirms broad families such as:
  - `balances`
  - `budgets`
  - `charges`
  - `credits`
  - `events`
  - `operationresults`
  - `pricesheets`
  - `reservationDetails`
  - `reservationTransactions`
  - `tags`

Implementation implication:

- the provider operation fixture should cover every family included in the staged emulator, not just budgets and usage

### Budgets

From `Budgets - Create Or Update` and `List`:

- routes:
  - `PUT {scope}/providers/Microsoft.Consumption/budgets/{budgetName}`
  - `GET {scope}/providers/Microsoft.Consumption/budgets`
  - `GET/DELETE {scope}/providers/Microsoft.Consumption/budgets/{budgetName}`
- create/update request requires:
  - `properties.amount`
  - `properties.category`
  - `properties.timeGrain`
  - `properties.timePeriod`
- optional but important fields:
  - `eTag`
  - `properties.filter`
  - `properties.notifications`
- response includes:
  - `currentSpend`
  - `forecastSpend`
  - `eTag`

Implementation implication:

- `budgets` should be the anchor mutable family for this service, with deterministic spend snapshots and patch-free full `PUT` semantics

### Charges, Marketplaces, Tags, Usage Details

From the reviewed list methods:

- routes:
  - `GET {scope}/providers/Microsoft.Consumption/charges`
  - `GET {scope}/providers/Microsoft.Consumption/marketplaces`
  - `GET {scope}/providers/Microsoft.Consumption/tags`
  - `GET {scope}/providers/Microsoft.Consumption/usageDetails`
- common query traits:
  - filter expressions
  - date windows
  - `top` and `skiptoken`
  - scope-dependent grouping such as `$apply` for charges
- `usageDetails` has an explicit product note:
  - Microsoft recommends new integrations move to Cost Details APIs over time

Implementation implication:

- these should be modeled as deterministic report lists keyed by normalized scope plus filter/query inputs, not as stored resources

### Balances, Credits, Events, Lots, Price Sheet

From the reviewed group and method pages:

- balances:
  - billing-account scoped current and billing-period variants
- credits:
  - billing-account and billing-profile scoped `balanceSummary`
- events:
  - billing-account and billing-profile list flows with date filters
- lots:
  - list by billing account, billing profile, and customer
- pricesheet:
  - direct subscription get
  - billing-period get
  - async download by billing-account period

Implementation implication:

- this family is a good second slice after budgets because it is read-heavy and deterministic
- the async pricesheet download route should be staged separately from the synchronous get/list routes

### Aggregated Cost

From `Aggregated Cost`:

- routes are management-group specific, not generic `{scope}`
- current and billing-period variants are both `GET`

Implementation implication:

- this should be modeled as its own small stage rather than mixed into the generic scope router

### Reservation Recommendation and Reservation Reporting Families

From the reviewed reservation pages:

- recommendation list:
  - `GET {resourceScope}/providers/Microsoft.Consumption/reservationRecommendations`
  - supports filtering on `scope`, `resourceType`, and `lookBackPeriod`
- recommendation details:
  - `GET {resourceScope}/providers/Microsoft.Consumption/reservationRecommendationDetails`
  - query parameters include:
    - `scope`
    - `region`
    - `term`
    - `lookBackPeriod`
    - `product`
- reservation details and summaries:
  - support date windows
  - support reservation-order and reservation filters
  - docs explicitly warn about ARM payload-size limits and possible 400/504 on large date ranges
- reservation transactions:
  - billing-account and billing-profile families with similar date-window and paging concerns

Implementation implication:

- these should be implemented as deterministic read-only report generators
- the emulator should include size-aware or date-range-aware validation so the routes do not look like trivial static stubs

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the new service shell and alias wiring:

- reserve service key `consumption`
- add provider registration in `internal/server/http.go`
- add provider scaffolding in `internal/server/provider_azure_consumption.go`
- add coverage aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Acceptance target:

- the service is discoverable in coverage scripts and doc mapping
- unimplemented routes are isolated to this provider instead of falling through to unrelated handlers

### Stage 1: Shared Scope Parsing and Provider Operations

Implement the shared route primitives:

- provider `operations`
- generic `{scope}` parsing for subscription, resource-group, management-group, billing-account, billing-profile, invoice-section, customer, department, and enrollment-account examples
- common query parsing for:
  - `top`
  - `skiptoken`
  - `filter`
  - date parameters

Acceptance target:

- provider operations return documented operation families
- scope normalization is reused across the reporting handlers

### Stage 2: Budgets

Implement the durable mutable family:

- `PUT/GET/DELETE budget`
- `GET budgets`
- deterministic `currentSpend` and `forecastSpend`
- eTag handling
- validation for required fields and basic date/time-grain rules

Acceptance target:

- budget lifecycle works end to end
- list and get reflect stored state

### Stage 3: Core Scope Reports

Implement the generic scope list/report families:

- `charges`
- `marketplaces`
- `tags`
- `usageDetails`

Acceptance target:

- read routes succeed across at least subscription and resource-group scope variants
- filters and paging tokens produce stable deterministic differences

### Stage 4: Billing-Scoped Read Families

Implement the `Microsoft.Billing`-anchored families:

- `balances`
- `credits`
- `events`
- `lots`
- synchronous `pricesheets` get routes

Acceptance target:

- billing-account, billing-profile, and customer route variants return realistic typed payloads
- the repo coverage gates no longer see these routes as `NotImplemented`

### Stage 5: Aggregated Cost and Reservation Recommendation Families

Implement the analytic read routes:

- management-group `aggregatedCost`
- `reservationRecommendations`
- `reservationRecommendationDetails`

Acceptance target:

- recommendation filters and what-if parameters are validated
- management-group cost aggregation returns deterministic billing-period-aware results

### Stage 6: Reservation Detail / Summary / Transaction Families

Implement the deeper reservation reporting slice:

- `reservationTransactions`
- `reservationDetails`
- `reservationSummaries`
- reservation-order and reservation-specific variants

Acceptance target:

- route variants by resource scope and reservation scope all respond successfully
- emulator includes deterministic safeguards for too-large date ranges to mirror documented payload-size constraints

### Stage 7: Async Pricesheet Download Tail

Implement the asynchronous tail:

- `pricesheets/downloadByBillingAccountPeriod`
- operation-style download response payload

Acceptance target:

- synchronous pricesheet get and async download can both be exercised by the example and tests

### Stage 8: Azure Go SDK Example

Add the example under `examples/azure/consumption-2024-08-01`:

- prefer `armconsumption` typed clients for operations, budgets, and one or more reporting families
- use runtime fallback only for any reviewed `2024-08-01` route absent from the typed package revision available in the repo
- include:
  - `main.go`
  - `Dockerfile`
  - `docker-compose.yml`

Acceptance target:

- `go build ./examples/azure/consumption-2024-08-01` succeeds
- example demonstrates at least:
  - `operations`
  - budget create/get/list
  - one scope report
  - one billing-scoped report

### Stage 9: Tests and Coverage Closure

Add focused provider contract tests for:

- provider operations
- budgets lifecycle
- scope report routes
- billing-scoped routes
- reservation recommendation routes
- validation failures for bad scopes, missing required fields, and malformed date/query combinations

Acceptance target:

- `go test ./internal/server -run 'TestAzureConsumption' -count=1` passes
- coverage scripts pass for `consumption`
- endpoint coverage reports no `NotImplemented` for the implemented surface

## Recommended First Implementation Slice

The first implementation PR should cover enough of the service to clear the most visible documented families without taking on every reporting tail at once:

- provider `operations`
- `budgets` CRUD/list
- generic-scope `charges`, `marketplaces`, `tags`, and `usageDetails`
- billing-scoped `balances`, `credits`, `events`, `lots`, and synchronous `pricesheets`
- one management-group `aggregatedCost` path
- `reservationRecommendations` and `reservationRecommendationDetails`
- example and coverage wiring

This gives the repo a credible and broad `Microsoft.Consumption` footprint quickly, while keeping the heavier reservation transaction/detail families for a follow-up merge.
