# Azure Billing Benefits (`billingbenefits-2022-11-01`) Staged Plan

## Objective

Emulate the Azure Billing Benefits ARM management API for `2022-11-01` with deterministic local behavior for:

- provider `operations`
- `reservationOrderAliases`
- `savingsPlanOrderAliases`
- `savingsPlanOrders`
- `savingsPlans`
- an Azure Go SDK example under `examples/azure/billingbenefits-2022-11-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the ARM management plane under `Microsoft.BillingBenefits`.
- This is distinct from the broader `Microsoft.Billing` provider already implemented in the repo.
- It is also distinct from the legacy Reserved VM Instances REST surface; this plan is scoped only to the `billingbenefits` `2022-11-01` routes verified from Microsoft Learn.

Important modeling note:

- The surface is centered on benefit purchase aliases and savings-plan lifecycle management.
- The emulator should model purchase and update workflows deterministically, including async-style create semantics for aliases, rather than trying to reproduce live commerce-side billing behavior.
- The first implementation should prioritize stable route coverage, correct resource envelopes, and coherent provisioning-state transitions.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/operation-groups?view=rest-billingbenefits-2022-11-01`

Operation-group and group-level pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/operations?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/reservation-order-alias?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order-alias?view=rest-billingbenefits-2022-11-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/operations/list?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/reservation-order-alias/create?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/reservation-order-alias/get?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order/list?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order/get?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order/elevate?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/get?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/list?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/list-all?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/update?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/validate-update?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order-alias/create?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order-alias/get?view=rest-billingbenefits-2022-11-01`

Official Go SDK package reviewed:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/billingbenefits/armbillingbenefits/v2`

## SDK Selection Note

- The official Azure Go SDK package for this surface is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/billingbenefits/armbillingbenefits/v2`.
- The reviewed package surface aligns well with the verified API groups and examples:
  - `OperationsClient`
  - `ReservationOrderAliasClient`
  - `SavingsPlanClient`
  - `SavingsPlanOrderAliasClient`
  - `SavingsPlanOrderClient`
  - `RPClient.ValidatePurchase`
- The future example should prefer typed `armbillingbenefits/v2` clients directly; no raw `azcore/runtime` fallback should be necessary for the verified `2022-11-01` surface.

## Service Naming

Canonical Stackyard naming:

- service key: `billingbenefits`
- display name: `Azure Billing Benefits`
- API version token: `2022-11-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_billingbenefits.go`
- tests: `internal/server/provider_azure_billingbenefits_test.go`
- example: `examples/azure/billingbenefits-2022-11-01`
- plan doc: `docs/billingbenefits-2022-11-01-plan.md`

Recommended coverage aliases:

- `billingbenefits`
- `billing_benefits`
- `billing-benefits`
- `azure_billingbenefits`
- `azure_billing_benefits`
- `microsoft_billingbenefits`
- `rest_billingbenefits_2022_11_01`
- `billingbenefits_2022_11_01`

## API Surface Summary

The reviewed pages expose four major route families:

- provider operations:
  - `/providers/Microsoft.BillingBenefits/operations`
- alias resources:
  - `/providers/Microsoft.BillingBenefits/reservationOrderAliases/{reservationOrderAliasName}`
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrderAliases/{savingsPlanOrderAliasName}`
- savings plan orders:
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrders`
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}`
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}/elevate`
- savings plans:
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}/savingsPlans`
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}/savingsPlans/{savingsPlanId}`
  - `/providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}/savingsPlans/{savingsPlanId}/validate`
  - `/providers/Microsoft.BillingBenefits/savingsPlans`

Modeling implications:

- this service is provider-rooted and does not use resource groups
- alias resources should be modeled as creation-entry points that lead to a stable downstream order ID
- savings plan orders and savings plans should share canonical storage so alias-created resources resolve through their order and plan routes without duplication
- list routes support query options such as `$filter`, `$orderby`, `$skiptoken`, `selectedState`, `take`, and `refreshSummary`
- the emulator can initially parse those parameters permissively while returning deterministic filtered or unfiltered fixtures

## Contract Notes From Reviewed Method Pages

Shared traits:

- failures use ARM-style `ErrorResponse`
- many responses expose `provisioningState` values such as:
  - `Creating`
  - `PendingBilling`
  - `ConfirmedBilling`
  - `Created`
  - `Succeeded`
  - `Cancelled`
  - `Expired`
  - `Failed`
- alias create routes can return:
  - `200 OK`
  - `201 Created` with async headers such as `Azure-AsyncOperation` and `Retry-After`
- `savingsPlan` update can return `200 OK` or `202 Accepted`

Data-shape traits visible across the reviewed pages:

- purchase requests are centered on:
  - `sku`
  - `term`
  - `billingPlan`
  - `billingScopeId`
  - `appliedScopeType`
  - `appliedScopeProperties`
  - `displayName`
  - `renew`
- reservation aliases also include reserved-resource fields like:
  - `reservedResourceType`
  - `reservedResourceProperties.instanceFlexibility`
- savings plans center on:
  - `commitment`
  - `renewProperties`
  - benefit timing
  - billing/account/profile/customer projection fields

## Recommended State Model

Keep canonical stores for:

- provider operations
- reservation order aliases keyed by alias name
- savings plan order aliases keyed by alias name
- savings plan orders keyed by order ID
- savings plans keyed by `{orderID}/{planID}`

Recommended seeded fixtures:

- one reservation order alias resolving to one stable reservation-order ID
- one savings plan order alias resolving to one stable savings-plan-order ID
- one savings plan order with:
  - order-level summary state
  - display name
  - term
  - commitment summary
- two savings plans under the seeded order so both order-scoped list and global list routes return non-empty data

Recommended workflow behavior:

- alias `Create`:
  - creates or idempotently updates the alias resource
  - ensures a backing order exists
  - returns `201` on first create and `200` on replay
- savings plan `Update`:
  - mutates a plan’s display name, scope, renew settings, or commitment metadata
  - updates `provisioningState` and timestamps deterministically
- savings plan `Validate Update`:
  - returns a validation payload without mutating stored state

## Staged Implementation Plan

### Stage 1: Route Skeleton, Shared Models, and Coverage Wiring

Create the initial provider and route registration for:

- `GET /providers/Microsoft.BillingBenefits/operations`
- `PUT` and `GET` for `reservationOrderAliases`
- `PUT` and `GET` for `savingsPlanOrderAliases`

Include:

- shared provider-root route parsing
- shared error helpers
- canonical alias and order storage
- initial `billingbenefits` coverage aliases in the Azure coverage scripts

### Stage 2: Savings Plan Orders

Implement:

- `GET /providers/Microsoft.BillingBenefits/savingsPlanOrders`
- `GET /providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}`
- `POST /providers/Microsoft.BillingBenefits/savingsPlanOrders/{savingsPlanOrderId}/elevate`

Recommended outcome:

- alias-created orders resolve through the order routes
- list and get share the same canonical order fixtures
- `elevate` returns a deterministic success payload or updated state marker

### Stage 3: Savings Plans

Implement:

- order-scoped list
- global list-all
- get by `{orderId}/{planId}`
- patch update
- validate update

Recommended outcome:

- global and order-scoped list routes are backed by the same savings-plan store
- update and validate use the same request schema but only update mutates state
- `$expand` on `Get` can be accepted and surfaced into richer output for seeded fields like `renewProperties`

### Stage 4: Query and Filtering Fidelity

Improve list semantics for:

- `$filter`
- `$orderby`
- `$skiptoken`
- `take`
- `selectedState`
- `refreshSummary`

Recommended simplification:

- implement deterministic subset handling for the most common fields:
  - display name
  - provisioning state
  - term
  - applied scope type
- accept unsupported query values permissively unless they are structurally invalid

### Stage 5: Example and Coverage Finalization

Add the example under `examples/azure/billingbenefits-2022-11-01` using the official `armbillingbenefits/v2` package.

Recommended example flow:

1. list provider operations
2. create a reservation order alias
3. create a savings plan order alias
4. get the created savings plan order alias
5. list savings plan orders
6. get a savings plan order
7. list savings plans in the order
8. get one savings plan
9. validate and then update the savings plan

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Testing Strategy

Add focused provider tests for:

- operations list
- reservation order alias create/get idempotency
- savings plan order alias create/get idempotency
- savings plan order list/get/elevate
- savings plan list/list-all/get/update/validate
- query-parameter acceptance for list routes
- negative paths for invalid alias names, missing order IDs, and missing plan IDs

Coverage-oriented assertions should verify:

- every documented route family resolves to a concrete handler
- no verified route falls through to `NotImplemented`
- alias-created resources are visible through their downstream order and plan routes

## Coverage Script Updates To Make During Implementation

Add a new Azure service entry for:

- service key: `billingbenefits`
- plan doc: `docs/billingbenefits-2022-11-01-plan.md`
- example directory: `examples/azure/billingbenefits-2022-11-01`

Include alias mapping for:

- `billingbenefits`
- `billing_benefits`
- `azure_billingbenefits`
- `microsoft_billingbenefits`
- `rest_billingbenefits_2022_11_01`

## Validation Targets For The Future Implementation

At implementation time, validate with:

- `go test ./internal/server -run 'TestAzureBillingBenefits' -count=1`
- `go build ./examples/azure/billingbenefits-2022-11-01`
- `python3 scripts/azure-contract-coverage.py --service billingbenefits --require-service billingbenefits --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service billingbenefits --require-service billingbenefits --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service billingbenefits --require-service billingbenefits --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service billingbenefits --require-service billingbenefits --fail-on any`
