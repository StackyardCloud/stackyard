# Azure Reservations (`reservations-2022-11-01`) Staged Plan

## Objective

Emulate Azure Reservations (`Microsoft.Capacity`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): pricing and purchasing reservation orders, and updating, splitting, merging, archiving and returning their reservations. Nothing is billed and reservations never apply to running resources; prices come from a flat sample rate.

## References

Microsoft Learn pages for the `2022-11-01` API version and the reservation policies it enforces:

- `https://learn.microsoft.com/en-us/rest/api/reserved-vm-instances/reservation-order/purchase`
- `https://learn.microsoft.com/en-us/rest/api/reserved-vm-instances/reservation/update`
- `https://learn.microsoft.com/en-us/rest/api/reserved-vm-instances/reservation/split`
- `https://learn.microsoft.com/en-us/rest/api/reserved-vm-instances/reservation/merge`
- `https://learn.microsoft.com/en-us/rest/api/reserved-vm-instances/calculate-refund/post`
- `https://learn.microsoft.com/en-us/rest/api/reserved-vm-instances/return/post`
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/reservations/manage-reserved-vm-instance`
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/reservations/exchange-and-refund-azure-reservations`

## Service Naming

- service key: `reservations`
- display name: `Azure Reservations`
- implementation: `internal/server/provider_azure_reservations.go`
- tests: `internal/server/provider_azure_reservations_test.go`
- example: `examples/azure/reservations-2022-11-01/`

## Operations In Scope

All paths are under `/providers/Microsoft.Capacity` (tenant scope) unless noted.

| Resource | Operations |
|---|---|
| `calculatePrice` | POST: price a purchase and get the reservation order ID to buy it under |
| `reservationOrders` | purchase (PUT, 200), get, list; POST `split`, `merge`, `calculateRefund`, `return`, `changeDirectory` |
| `reservationOrders/{order}/reservations` | get, list, update (PATCH); POST `archive`, `unarchive`, `availableScopes` |
| `reservations` | list across orders |
| `/subscriptions/{id}/providers/Microsoft.Capacity/catalogs` | list (sample catalog) |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- a purchase needs a sku name, a reservable resource type, a subscription or billing profile as billing scope, a term (`P1Y`, `P3Y`, `P5Y`), a quantity of at least 1 and a scope type; order IDs are GUIDs, and an order can be purchased only once
- scopes must agree with their type: `Single` takes one subscription or resource group, `Shared` none, and `ManagementGroup` a management group and its tenant; PATCH checks the same and clears the scopes when changing to `Shared`
- the order returns its dates, term, plan and original quantity, while its reservation carries the purchase details; reservations are created only by purchases, splits and merges (a PUT is rejected)
- PATCH takes the update model: scope, instance flexibility, display name (as `name`), renewal and review date
- split divides a reservation in effect into two whose quantities add up to its own, and leaves the source `Split`; merge combines two or more distinct reservations of the same SKU, location, term and billing plan, and leaves the sources `Merged`; the order lists every reservation
- a return quotes the `sessionId` that `calculateRefund` gave for the same reservation and quantity; returning all units cancels the reservation, and the order is cancelled once none of its reservations is in effect
- refunds are prorated by the whole days left in the term, can't exceed USD 50,000 per billing scope in 12 months (`calculateRefund` reports `RefundLimitExceeded` and the return is rejected), and aren't available for Red Hat and SUSE Linux plans
- orders and reservations have no DELETE (405); a scope change replaces the old scope fields; split and merge results don't carry their sources' split or merge history
- prices are a flat USD 100 per unit per month; `changeDirectory` reports success but moves nothing, since the emulator has one tenant

Out of scope for this stage:
- exchanges (`calculateExchange`, `exchange`), revisions and applied reservations
- catalog filters, renewals, expiry and utilization
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureReservations -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service reservations`
- `python3 scripts/azure-io-contract-coverage.py --service reservations`
- `python3 scripts/azure-doc-contract-coverage.py --service reservations`
