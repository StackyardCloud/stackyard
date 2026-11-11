# Azure Billing (`billing-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Billing ARM management API for `2024-04-01` with deterministic local behavior for:

- provider-rooted billing utility routes such as `validateAddress`
- top-level `billingAccounts`
- subscription-scoped `billingProperty`
- account hierarchy resources such as `agreements`, `billingProfiles`, `invoiceSections`, `customers`, `departments`, and `enrollmentAccounts`
- account and child-scope access-control families such as `billingRoleDefinitions`, `billingRoleAssignments`, and `billingPermissions`
- commerce and ownership families such as `billingSubscriptions`, `products`, `paymentMethods`, `policies`, `invoices`, and `transactions`
- request and transfer families such as `billingRequests`, `transfers`, `recipientTransfers`, and `associatedTenants`
- commitment-purchase families such as `reservationOrders`, `reservations`, `savingsPlanOrders`, and `savingsPlans`
- an Azure Go SDK example under `examples/azure/billing-2024-04-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the ARM management plane under `Microsoft.Billing`.
- This is not a data plane and it is not a normal resource-group-based ARM provider.
- Most routes are provider-rooted under `/providers/Microsoft.Billing/...`, with one notable subscription-scoped branch for `billingProperty`.

Important modeling note:

- The Billing API is heavily conditioned on agreement type and purchase motion:
  - `Microsoft Customer Agreement`
  - `Microsoft Partner Agreement`
  - `Enterprise Agreement`
  - `Microsoft Online Services Program`
- The emulator should expose deterministic fixtures for those agreement families instead of trying to fully enforce every commercial rule from Azure.
- The first implementation should prioritize stable route coverage and plausible payloads over perfect agreement-policy simulation.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/billing/operation-groups?view=rest-billing-2024-04-01`

Operation-group and group-level pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/billing/address?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/agreements?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/associated-tenants?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/available-balances?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-accounts?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-permissions?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-profiles?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-property?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-requests?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-role-assignments?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-role-definition?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-subscriptions?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/customers?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/departments?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/enrollment-accounts?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/invoice-sections?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/invoices?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/payment-methods?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/policies?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/products?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/recipient-transfers?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/reservation-orders?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/reservations?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/savings-plan-orders?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/savings-plans?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/transactions?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/transfers?view=rest-billing-2024-04-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/billing/address/validate?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-accounts/list?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-accounts/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-accounts/update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-accounts/list-invoice-sections-by-create-subscription-permission?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-accounts/validate-payment-terms?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-profiles/create-or-update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-profiles/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-profiles/list-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/invoice-sections/create-or-update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/invoice-sections/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/customers/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/available-balances/get-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/available-balances/get-by-billing-profile?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-property/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-property/update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-role-definition/get-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-role-definition/list-by-billing-profile?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-role-assignments/get-by-customer?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-permissions/list-by-customer?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/payment-methods/get-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/payment-methods/list-by-billing-profile?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-subscriptions/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-subscriptions/update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-subscriptions/validate-move-eligibility?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/products/update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/invoices/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/invoices/get-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/transactions/list-by-billing-profile?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/billing-requests/create-or-update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/transfers/initiate?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/recipient-transfers/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/recipient-transfers/accept?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/associated-tenants/create-or-update?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/departments/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/enrollment-accounts/get?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/enrollment-accounts/get-by-department?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/reservation-orders/get-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/reservations/get-by-reservation-order?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/reservations/update-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/savings-plans/get-by-billing-account?view=rest-billing-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/billing/savings-plans/update-by-billing-account?view=rest-billing-2024-04-01`

Official Go SDK package reviewed:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/billing/armbilling`

## SDK Selection Note

- The official Azure Go SDK package for this surface is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/billing/armbilling`.
- The package index shows broad typed coverage across the families needed for this service, including:
  - `AddressClient`
  - `AccountsClient`
  - `AgreementsClient`
  - `AssociatedTenantsClient`
  - `AvailableBalancesClient`
  - `CustomersClient`
  - `DepartmentsClient`
  - `EnrollmentAccountsClient`
  - `InvoiceSectionsClient`
  - `InvoicesClient`
  - `PaymentMethodsClient`
  - `PermissionsClient`
  - `PoliciesClient`
  - `ProductsClient`
  - `RequestsClient`
  - `ReservationOrdersClient`
  - `ReservationsClient`
  - `RoleAssignmentsClient`
  - `RoleDefinitionsClient`
  - `SubscriptionsClient`
  - `TransactionsClient`
  - `TransfersClient`
  - `RecipientTransfersClient`
- The future example should prefer typed `armbilling` clients and only fall back to `azcore/runtime` if a reviewed `2024-04-01` route is missing or awkward in the generated package.

## Service Naming

Canonical Stackyard naming:

- service key: `billing`
- display name: `Azure Billing`
- API version token: `2024-04-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_billing.go`
- tests: `internal/server/provider_azure_billing_test.go`
- example: `examples/azure/billing-2024-04-01`
- plan doc: `docs/billing-2024-04-01-plan.md`

Recommended coverage aliases:

- `billing`
- `azure_billing`
- `azure-billing`
- `microsoft_billing`
- `rest_billing_2024_04_01`
- `billing_2024_04_01`

## API Surface Summary

The reviewed pages expose four main route shapes:

- provider-rooted utility and global routes:
  - `/providers/Microsoft.Billing/validateAddress`
  - `/providers/Microsoft.Billing/billingAccounts`
  - `/providers/Microsoft.Billing/billingRequests/{billingRequestName}`
  - `/providers/Microsoft.Billing/transfers/{transferName}`
- provider-rooted account hierarchy routes:
  - `/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/...`
- deeply nested commercial subresources:
  - `billingProfiles`
  - `invoiceSections`
  - `customers`
  - `paymentMethods` or `paymentMethodLinks`
  - `billingSubscriptions`
  - `products`
  - `agreements`
  - `associatedTenants`
  - `reservationOrders`
  - `savingsPlanOrders`
- subscription-scoped property route:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Billing/billingProperty/default`

Modeling implications:

- this service is mostly not ARM tracked-resource CRUD in the normal `resourceGroups/{rg}/providers/...` pattern
- the emulator should keep canonical stores keyed by:
  - billing account ID
  - billing profile ID under account
  - invoice section ID under profile
  - customer ID under profile
  - subscription ID under account or profile
  - request or transfer ID for top-level workflow objects
- many routes are read/list or action-oriented rather than resource lifecycle routes
- many list operations support rich query options such as `filter`, `orderBy`, `top`, `skip`, `count`, `search`, or `expand`
- a coverage-first implementation should support those parameters permissively even if the backing filter logic is minimal at first

## Contract Notes From Reviewed Method Pages

Shared traits:

- failures use Billing `ErrorResponse`
- list routes generally return `value` and often `nextLink`
- many routes are asynchronous or can return `202 Accepted`
- resource names and IDs are often constrained by agreement-specific patterns
- payload shape is highly polymorphic across agreement types, so representative fixture documents are more useful than overfitting to one agreement family

Important route semantics visible in the reviewed pages:

- `billingAccounts` are list/get/update plus action routes, not full create/delete resources
- `billingProfiles` and `invoiceSections` support create/update and behave more like ARM resources
- `billingProperty` is anchored on subscription ID and is not part of the billing-account hierarchy
- `billingRequests`, `transfers`, and `recipientTransfers` are workflow objects, not ordinary child resources
- `reservations`, `reservationOrders`, `savingsPlanOrders`, and `savingsPlans` are effectively inventory and patchable purchase artifacts under a billing account

## Recommended State Model

Seed deterministic fixtures for:

- one MCA-style billing account
- one MPA-style billing account
- one EA-style billing account
- one MOSP-style subscription billing property

For each account family, keep nested maps for:

- `agreements`
- `billingProfiles`
- `invoiceSections`
- `customers`
- `paymentMethods`
- `policies`
- `products`
- `billingSubscriptions`
- `billingRequests`
- `transfers`
- `associatedTenants`
- `departments`
- `enrollmentAccounts`
- `reservationOrders`
- `reservations`
- `savingsPlanOrders`
- `savingsPlans`

Cross-cutting read-only fixture stores:

- `transactions`
- `invoices`
- `availableBalances`
- `roleDefinitions`
- `permissions`

## Staged Implementation Plan

### Stage 1: Route Skeleton, Shared Models, and Coverage Wiring

Create the initial provider and route registration for:

- `POST /providers/Microsoft.Billing/validateAddress`
- `GET /providers/Microsoft.Billing/billingAccounts`
- `GET /providers/Microsoft.Billing/billingAccounts/{billingAccountName}`
- `PATCH /providers/Microsoft.Billing/billingAccounts/{billingAccountName}`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.Billing/billingProperty/default`
- `PATCH /subscriptions/{subscriptionId}/providers/Microsoft.Billing/billingProperty/default`

Include:

- shared error helpers
- shared billing-account ID validation helpers
- permissive query parsing helpers
- initial `billing` coverage aliases in the Azure coverage scripts

### Stage 2: Billing Account Hierarchy Foundation

Implement the core account-nested read/write families that the rest of the surface depends on:

- `agreements`
- `billingProfiles`
- `invoiceSections`
- `customers`
- `availableBalance`

Recommended outcome:

- one canonical account tree with account -> profile -> invoice section/customer nesting
- create/update/get/list for `billingProfiles` and `invoiceSections`
- get/list for `agreements`, `customers`, and `availableBalance`

### Stage 3: Access Control and Policy Surface

Implement the agreement-scope authorization families:

- `billingRoleDefinitions`
- `billingRoleAssignments`
- `billingPermissions`
- `policies`

This stage should cover the repeated scope variants:

- billing account
- billing profile
- customer
- invoice section
- department
- enrollment account
- subscription where applicable

Recommended simplification:

- use a shared scope parser and a shared role-assignment store keyed by scope string
- seed a small deterministic set of role definitions and permissions rather than deriving them dynamically

### Stage 4: Commerce and Ownership Resources

Implement the high-traffic business objects:

- `paymentMethods`
- `billingSubscriptions`
- `products`
- `invoices`
- `transactions`

Important action routes from the reviewed pages:

- subscription `cancel`, `delete`, `move`, `merge`, `split`, `validateMoveEligibility`
- product `move` and `validateMoveEligibility`
- invoice download or amend routes

Recommended first-pass behavior:

- support all reviewed routes
- keep mutation semantics deterministic and shallow
- store movement and cancellation state transitions directly on the affected subscription or product fixture

### Stage 5: Requests, Transfers, and Tenant Collaboration

Implement workflow-style routes:

- `billingRequests`
- `transfers`
- `recipientTransfers`
- `associatedTenants`

These routes should model:

- creation or initiation
- get and list
- accept, decline, or cancel transitions where applicable

Recommended simplification:

- treat transfer/request state changes as explicit finite-state transitions on stored workflow objects

### Stage 6: EA and Commitment Purchase Tails

Implement the deeper but still documented families:

- `departments`
- `enrollmentAccounts`
- `reservationOrders`
- `reservations`
- `savingsPlanOrders`
- `savingsPlans`

Recommended priority inside this stage:

1. read/list routes
2. patch or update routes
3. optional expand handling

These are important for full endpoint coverage, but they should not block the earlier account/profile/subscription work.

### Stage 7: Example and Coverage Finalization

Add the example under `examples/azure/billing-2024-04-01` using the official `armbilling` package.

Recommended example flow:

1. validate an address
2. list billing accounts
3. get one billing account
4. list billing profiles under the account
5. get available balance for the account or profile
6. list subscriptions or products under the account
7. list invoices or transactions for a profile

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Testing Strategy

Add focused provider tests for:

- account list/get/update and billing-property routes
- profile and invoice-section lifecycle
- customer and available-balance reads
- role-definition, role-assignment, permission, and policy scope variants
- subscription and product action routes
- transfer and request state transitions
- reservations and savings plan read/update routes

Coverage-oriented assertions should verify:

- every documented route family for the implemented stages resolves to a concrete handler
- unsupported agreement families return modeled Billing errors instead of `NotImplemented`
- list routes consistently shape `value` and `nextLink`

## Coverage Script Updates To Make During Implementation

Add a new Azure service entry for:

- service key: `billing`
- plan doc: `docs/billing-2024-04-01-plan.md`
- example directory: `examples/azure/billing-2024-04-01`

Include alias mapping for:

- `billing`
- `azure_billing`
- `microsoft_billing`
- `rest_billing_2024_04_01`

## Validation Targets For The Future Implementation

At implementation time, validate with:

- `go test ./internal/server -run 'TestAzureBilling' -count=1`
- `go build ./examples/azure/billing-2024-04-01`
- `python3 scripts/azure-contract-coverage.py --service billing --require-service billing --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service billing --require-service billing --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service billing --require-service billing --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service billing --require-service billing --fail-on any`
