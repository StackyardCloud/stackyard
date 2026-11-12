# Azure Confidential Ledger (`confidentialledger-2022-05-13`) Staged Plan

## Objective

Emulate Azure Confidential Ledger for API version `2022-05-13` with deterministic ARM-style behavior under `Microsoft.ConfidentialLedger`, including:

- provider `operations`
- `ledgers` create, get, patch, delete, list-by-resource-group, and list-by-subscription
- an Azure Go SDK example under `examples/azure/confidentialledger-2022-05-13`
- Azure coverage-script wiring for the new service key

The reviewed surface is small and centered on one primary ARM resource, so it is suitable for a near-complete implementation in a few stages.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/operation-groups?view=rest-resource-manager-confidentialledger-2022-05-13`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger?view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/operations/list?view=rest-resource-manager-confidentialledger-2022-05-13`

Linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/operations/list?view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger/create?view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger/delete?view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger/get?view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger/list-by-resource-group?view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger/list-by-subscription?tabs=HTTP&view=rest-resource-manager-confidentialledger-2022-05-13`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/confidentialledger/ledger/update?view=rest-resource-manager-confidentialledger-2022-05-13`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/confidentialledger/armconfidentialledger`

## Service Naming

Canonical Stackyard naming:
- service key: `confidentialledger`
- display name: `Azure Confidential Ledger`
- provider namespace: `Microsoft.ConfidentialLedger`
- API version: `2022-05-13`

Recommended repo paths:
- implementation: `internal/server/provider_azure_confidentialledger.go`
- tests: `internal/server/provider_azure_confidentialledger_test.go`
- example: `examples/azure/confidentialledger-2022-05-13`
- plan doc: `docs/confidentialledger-2022-05-13-plan.md`

## Surface Summary

The reviewed Learn surface exposes 2 operation groups:

1. `Operations`
2. `Ledger`

Total linked method pages reviewed: `7`

Method inventory:

`Operations`
- `List`

`Ledger`
- `Create`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.ConfidentialLedger/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ConfidentialLedger/ledgers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers/{ledgerName}`

Target query:
- `api-version=2022-05-13`

Important routing implications from the reviewed pages:

- provider operations are subscription-independent under `/providers/Microsoft.ConfidentialLedger/operations`
- list-by-subscription is under `/subscriptions/{subscriptionId}/providers/Microsoft.ConfidentialLedger/ledgers`
- core ledger CRUD and list-by-resource-group are resource-group scoped
- there are no nested child resources in this management-plane surface

## Contract Notes From Reviewed Pages

Shared management-plane traits:

- every reviewed method requires `api-version=2022-05-13`
- failures use `ErrorResponse`
- all routes are standard ARM OAuth2 endpoints in Learn; local Stackyard examples should continue to use the repo’s Azure shim transport

### Operations

`Operations.List`
- `GET /providers/Microsoft.ConfidentialLedger/operations`
- returns `ResourceProviderOperationList`

Modeling implication:
- implement first as a deterministic `value` list covering:
  - `Microsoft.ConfidentialLedger/ledgers/read`
  - `Microsoft.ConfidentialLedger/ledgers/write`
  - `Microsoft.ConfidentialLedger/ledgers/delete`

### Ledger

`Ledger.Create`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers/{ledgerName}`
- reviewed required request fields:
  - `location`
  - `properties.ledgerType`
- reviewed optional request fields include:
  - `properties.aadBasedSecurityPrincipals`
  - `properties.certBasedSecurityPrincipals`
  - `tags`
- responses:
  - `200` with `ConfidentialLedger`
  - `201` with `ConfidentialLedger`
  - asynchronous create is mentioned in examples and status descriptions

`Ledger.Update`
- `PATCH /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers/{ledgerName}`
- reviewed patchable fields:
  - `location`
  - `properties`
  - `tags`
- responses:
  - `200` with `ConfidentialLedger`
  - `201` async update

`Ledger.Get`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers/{ledgerName}`
- response: `ConfidentialLedger`

`Ledger.List By Resource Group`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers`
- optional query:
  - `$filter`, for example `ledgerType eq 'Public'`
- response: `ConfidentialLedgerList`

`Ledger.List By Subscription`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.ConfidentialLedger/ledgers`
- response: `ConfidentialLedgerList`

`Ledger.Delete`
- `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConfidentialLedger/ledgers/{ledgerName}`
- responses:
  - `200`
  - `202`
  - `204` if already absent

Important modeled fields visible on reviewed `ConfidentialLedger` pages:
- `id`
- `name`
- `type`
- `location`
- `tags`
- `systemData`
- `properties.ledgerName`
- `properties.ledgerUri`
- `properties.identityServiceUri`
- `properties.ledgerType`
- `properties.ledgerInternalNamespace`
- `properties.provisioningState`
- `properties.aadBasedSecurityPrincipals`
- `properties.certBasedSecurityPrincipals`

Important modeled principal fields:
- `aadBasedSecurityPrincipals[].principalId`
- `aadBasedSecurityPrincipals[].tenantId`
- `aadBasedSecurityPrincipals[].ledgerRoleName`
- `certBasedSecurityPrincipals[].cert`
- `certBasedSecurityPrincipals[].ledgerRoleName`

Modeling implication:
- one canonical `ConfidentialLedger` store is enough for the whole service
- request validation should require `location` and `properties.ledgerType`
- deterministic server-generated values should fill `id`, `type`, `systemData`, `properties.ledgerName`, `properties.ledgerUri`, `properties.identityServiceUri`, `properties.ledgerInternalNamespace`, and `properties.provisioningState`

## Non-Goals

- do not emulate the Confidential Ledger data-plane endpoints in this management-plane service
- do not implement real attestation, identity issuance, or certificate verification
- do not emulate long-running ARM polling beyond minimal accepted response headers if needed
- do not implement deep semantic validation for every principal object beyond the documented minimum fields

## Stage 0: Coverage Naming And Route Claiming

- Add a dedicated Azure Confidential Ledger router for `Microsoft.ConfidentialLedger`
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Recommended aliases:
  - `confidentialledger`
  - `confidential_ledger`
  - `confidential-ledger`
  - `azure_confidential_ledger`
  - `azure_confidentialledger`
  - `confidentialledger_2022_05_13`
  - `rest_resource_manager_confidentialledger_2022_05_13`
  - `microsoft_confidentialledger`
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route recognition tests for:
  - provider operations
  - subscription ledger listing
  - resource-group ledger list and item routes

Exit criteria:
- coverage scripts discover the new canonical service key
- Azure routing claims all Confidential Ledger management paths deterministically

## Stage 1: Shared Validation And Error Shape

- Validate `api-version=2022-05-13`
- Enforce supported methods:
  - `GET`, `PUT`, `PATCH`, `DELETE`
- Return deterministic ARM-style error payloads for:
  - unsupported methods
  - invalid or missing `api-version`
  - malformed JSON
  - missing required `Create` fields
  - ledger not found

Validation rules to implement first:
- `location` required on `PUT`
- `properties.ledgerType` required on `PUT`
- `PATCH` requires at least one updatable field

Exit criteria:
- negative tests cover method validation, malformed bodies, and not-found behavior
- contract and IO harnesses stop reporting generic `NotImplemented` or route misses

## Stage 2: Provider Operations

- Implement `Operations.List` with a deterministic `value` array of provider operations
- Keep the response small but representative:
  - `name`
  - `display.resource`
  - `display.operation`
  - `display.description`
  - `isDataAction`

Exit criteria:
- provider operations page matches the documented route and typed envelope

## Stage 3: Core Ledger Lifecycle

- Implement persistent in-memory `ConfidentialLedger` resources for:
  - `Create`
  - `Get`
  - `List By Resource Group`
  - `List By Subscription`
- Preserve key request fields from the caller:
  - `location`
  - `tags`
  - `properties.ledgerType`
  - `properties.aadBasedSecurityPrincipals`
  - `properties.certBasedSecurityPrincipals`
- Stamp deterministic server fields:
  - `id`
  - `name`
  - `type`
  - `systemData`
  - `etag`
  - `properties.ledgerName`
  - `properties.ledgerUri`
  - `properties.identityServiceUri`
  - `properties.ledgerInternalNamespace`
  - `properties.provisioningState`

Recommended first-pass state behavior:
- `PUT` returns `201` for create and `200` for replace
- new resources default to `properties.provisioningState = "Succeeded"`
- keep `ledgerType` and principal arrays shallow but stable for round-tripping in tests and example code

Exit criteria:
- all core read and list methods return typed `ConfidentialLedger` or `ConfidentialLedgerList` payloads
- no core ledger route falls through to generic Azure handlers

## Stage 4: Patch And Delete Semantics

- Implement `Update` as shallow merge semantics for:
  - `location`
  - `tags`
  - `properties`
- Update `systemData.lastModified*` fields deterministically
- Implement `Delete`
  - prefer deterministic `200` or `202` on existing resources
  - return `204` if the ledger is already absent, matching the reviewed page

Pragmatic implementation note:
- if the example is materially simpler with synchronous behavior, the emulator can still return `200` for `PATCH` and `DELETE`, as long as the route no longer returns `NotImplemented` and the typed response body remains correct

Exit criteria:
- patch and delete are stable under repeated calls
- example and coverage harnesses do not stall on mutation endpoints

## Stage 5: Azure Go SDK Example

- Add `examples/azure/confidentialledger-2022-05-13`
- Use the Azure Go SDK package family for Confidential Ledger ARM management:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/confidentialledger/armconfidentialledger`
- Follow the repo’s Azure example pattern:
  - `azsdkshim.StaticTokenCredential{}`
  - `arm.ClientOptions`
  - `cloud.ResourceManager` endpoint override to Stackyard
  - `azsdkshim.NewTransport("/azure", account, subscriptionKey)`

Recommended example flow:
- create `OperationsClient` and list provider operations
- create `LedgersClient`
- create one ledger using the minimum documented required fields plus one AAD principal
- get the ledger
- list ledgers by resource group
- list ledgers by subscription
- patch tags or principals
- delete the ledger

Exit criteria:
- the example compiles cleanly
- the example exercises all implemented route families without special-case hacks

## Stage 6: Coverage And Verification

- Update service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Map this plan doc in `SERVICE_PLAN_DOCS`
- Add focused tests for:
  - operations list
  - create/get/list
  - patch
  - delete idempotency
  - invalid body and missing required fields
  - invalid `api-version`

Expected verification commands after implementation:
- `go test ./internal/server -run 'TestAzureConfidentialLedger' -count=1`
- `go build ./examples/azure/confidentialledger-2022-05-13`
- `python3 scripts/azure-contract-coverage.py --service confidentialledger --require-service confidentialledger --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service confidentialledger --require-service confidentialledger --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service confidentialledger --require-service confidentialledger --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service confidentialledger --require-service confidentialledger --fail-on any`

Exit criteria:
- all Confidential Ledger endpoints are discovered by the coverage scripts
- no Confidential Ledger route returns `NotImplemented`
- all targeted coverage commands succeed

## Implementation Order Recommendation

Recommended merge order:

1. Stage 0 and Stage 1
2. Stage 2 and Stage 3
3. Stage 4
4. Stage 5 and Stage 6

This order gets the ARM route envelope and validation in place first, then completes the single-resource lifecycle before adding the example and coverage polish.
