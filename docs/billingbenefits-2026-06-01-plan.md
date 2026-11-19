# Azure Billing Benefits (`billingbenefits-2026-06-01`) Staged Plan

## Objective

Emulate the GA savings plan API of Azure Billing Benefits (`Microsoft.BillingBenefits`): buying savings plans through savings plan order aliases, reading and listing the orders and plans a purchase creates, elevating on an order, updating and validating updates to a plan, validating purchases, the tenant savings plan list, and reservation order aliases. Nothing is billed: a purchase stores its order and plan and works out a payment schedule.

## References

Microsoft Learn pages, all on `2026-06-01`, the newest version Learn documents, except the purchase validation, which Learn documents only on `2022-11-01`. The emulator accepts any non-empty `api-version`:

- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/` (overview)
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order-alias/create`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order-alias/get`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order/get`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order/list`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan-order/elevate`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/get`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/list`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/list-all`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/update`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/savings-plan/validate-update`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/reservation-order-alias/create`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/reservation-order-alias/get`
- `https://learn.microsoft.com/en-us/rest/api/billingbenefits/validate-purchase/validate-purchase?view=rest-billingbenefits-2022-11-01`
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/savings-plan/savings-plan-compute-overview` (one- or three-year terms, hourly commitment)
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/savings-plan/buy-savings-plan` (monthly payments, `billingScopeId` formats)
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/savings-plan/scope-savings-plan` (scope options)
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/savings-plan/manage-savings-plan` (what can change after purchase)
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/savings-plan/renew-savings-plan` (automatic renewal)
- `https://learn.microsoft.com/en-us/azure/cost-management-billing/savings-plan/cancel-savings-plan` (purchases are final)
- `https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles` (Owner role ID)

## Service Naming

- service key: `billingbenefits`
- display name: `Azure Billing Benefits`
- implementation: `internal/server/provider_azure_billingbenefits.go`, `internal/server/provider_azure_billingbenefits_validate.go`
- tests: `internal/server/provider_azure_billingbenefits_test.go`
- example: `examples/azure/billingbenefits-2026-06-01/`

## Operations In Scope

All paths are at tenant scope, under `/providers/Microsoft.BillingBenefits`.

| Operation | Path |
|---|---|
| create and get a savings plan order alias | PUT, GET `/savingsPlanOrderAliases/{name}` |
| get and list savings plan orders | GET `/savingsPlanOrders[/{orderId}]` (`$expand=schedule` on a single order's GET) |
| elevate | POST `/savingsPlanOrders/{orderId}/elevate` |
| get, list and update savings plans | GET `/savingsPlanOrders/{orderId}/savingsPlans[/{planId}]`, PATCH `.../savingsPlans/{planId}` |
| validate a savings plan update | POST `.../savingsPlans/{planId}/validate` |
| list all savings plans | GET `/savingsPlans` |
| validate purchases | POST `/validate` |
| create and get a reservation order alias | PUT, GET `/reservationOrderAliases/{name}` |

## Behavior

Aliases, orders and plans are tenant types on the shared ARM handler. A small router (`handleAzureBillingBenefitsRouter`) adds what the shared handler can't: the async headers of alias purchases, `$expand=schedule` on an order, and the tenant savings plan list. Errors use the `{"error": {"code", "message"}}` payload with `InvalidRequestContent` and `InvalidResourceName` for rejected writes and `BadRequest` for rejected list queries.

- alias names match `^[a-zA-Z0-9_\-.]+$`. An alias purchase answers 201 with `Azure-AsyncOperation` and `Retry-After: 5`; the alias's `provisioningState` is `Created`, as Learn's samples show, and the operation it names (GET `/providers/Microsoft.BillingBenefits/operationResults/{id}`) is `Succeeded`. Each alias purchases once: a second PUT is rejected, and aliases have no DELETE or PATCH
- a savings plan purchase needs `sku.name`, `term` `P1Y` or `P3Y` (savings plans run one or three years), a `commitment` with grain `Hourly`, an amount above 0 and an uppercase ISO 4217 code, a `billingScopeId` of `/subscriptions/{guid}` or `/providers/Microsoft.Billing/billingAccounts/{id}/billingSubscriptions/{guid}`, and an `appliedScopeType`. `billingPlan` is `P1M` for monthly payments or omitted to pay up front; `renew` is a boolean (default false)
- scopes: `Single` needs `appliedScopeProperties.subscriptionId` or `resourceGroupId` (the group in the subscription when both are given), `ManagementGroup` needs `managementGroupId` and a `tenantId` GUID, and `Shared` takes neither
- a purchase creates a `Succeeded` order (GUID name) and one savings plan, both with the purchase's SKU. The plan's benefit, effective and purchase times are the purchase time and it expires a term later; a billing-subscription scope gives the plan and order its `billingAccountId`. A new plan's utilization is 0 for 1, 7 and 30 days
- `$expand=schedule` adds `planInformation`: the total is the hourly commitment for 8,760 hours a year of the term (Learn's sample: USD 0.001 an hour for three years is 26.28), paid in one payment up front or, for `P1M`, in 12 or 36 equal monthly payments from the purchase date. Payments due by now are `Succeeded` with a payment date; the rest are `Scheduled`, and `nextPaymentDueDate` is the first of them
- elevate returns a role assignment granting the caller the built-in Owner role (`8e3af657-a8ff-443c-a75c-2fe8c4bcb635`) at the order
- orders and plans come only from purchases: PUT is rejected, and purchases are final, so neither has DELETE. Orders have no PATCH
- a plan update changes only `appliedScopeType` and `appliedScopeProperties` (together), `displayName` and `renew`/`renewProperties`; the commitment, term and billing frequency are fixed after purchase, so other properties are rejected. A new scope replaces the old one, and `Shared` clears the scope properties. `renewProperties.purchaseProperties` is checked like a purchase, but only the fields it sets, since a renewal inherits the rest; it needs `renew` true, and turning `renew` off drops it
- validate (update and purchase) takes a non-empty `benefits` array and answers 200 with one result per item: `valid: true`, or `valid: false` with the reason and `reasonCode` `InvalidRequestContent`
- the tenant savings plan list takes `$filter` with `eq` comparisons of the properties Learn lists, each optionally in parentheses (which can't group several), joined by `and` and `or` (`and` binds tighter); other operators and properties are rejected, and plans are never `archived`. `$orderby` takes one of those properties with `asc` or `desc`; `take` (at least 1) and `$skiptoken` (at least 0) page the list, with a `nextLink` while plans remain; `selectedState` keeps plans in that display state; `refreshSummary` must be `true` or `false`. `additionalProperties` holds the `summary` counts by display state of the plans `$filter` selects
- a reservation order alias needs `sku.name`, a reservable `reservedResourceType`, `term` `P1M`, `P1Y`, `P3Y` or `P5Y`, a whole `quantity` of at least 1, a `billingScopeId` as above and an `appliedScopeType`; `billingPlan` is `P1M` or omitted, `renewAction` `CreateNew` or `RenewExisting`, and `instanceFlexibility` `On` or `Off`. Its optional `location` is kept, and `reservationOrderId` names a `Microsoft.Capacity` reservation order

Out of scope for this stage:
- the preview APIs: discounts, credits, conditional credits, MACCs, free services, sources and catalog groups
- the operations list
- creating the `Microsoft.Capacity` reservation order a reservation order alias names (Capacity purchases take `P1Y`, `P3Y` or `P5Y` terms, while Billing Benefits also takes `P1M`)
- plan expiry and renewal over time, utilization from usage, and `$expand=renewProperties` (renewal settings are always returned once set)
