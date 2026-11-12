# Azure Compute Fleet (`computefleet-2024-11-01`) Staged Plan

## Objective

Emulate Azure Compute Fleet for API version `2024-11-01` with deterministic ARM-style behavior under `Microsoft.AzureFleet`, including:

- provider `operations`
- `fleets` create, get, patch, delete, list-by-resource-group, and list-by-subscription
- `fleets/{fleetName}/virtualMachineScaleSets` listing
- an Azure Go SDK example under `examples/azure/computefleet-2024-11-01`
- Azure coverage-script wiring for the new service key

The reviewed surface is compact. It is a good candidate for a near-complete implementation in a few stages, with most effort concentrated in one resource model, one child list route, and ARM-style validation.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/computefleet/operation-groups?view=rest-computefleet-2024-11-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/operations/list?view=rest-computefleet-2024-11-01`

Linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/computefleet/operations/list?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/create-or-update?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/delete?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/get?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/list-by-resource-group?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/list-by-subscription?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/list-virtual-machine-scale-sets?view=rest-computefleet-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/computefleet/fleets/update?view=rest-computefleet-2024-11-01`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computefleet/armcomputefleet`

## Service Naming

Canonical Stackyard naming:
- service key: `compute_fleet`
- display name: `Azure Compute Fleet`
- provider namespace: `Microsoft.AzureFleet`
- API version: `2024-11-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_compute_fleet.go`
- tests: `internal/server/provider_azure_compute_fleet_test.go`
- example: `examples/azure/computefleet-2024-11-01`
- plan doc: `docs/computefleet-2024-11-01-plan.md`

## Surface Summary

The reviewed Learn surface exposes 2 operation groups:

1. `Operations`
2. `Fleets`

Total linked method pages reviewed: `8`

Method inventory:

`Operations`
- `List`

`Fleets`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `List Virtual Machine Scale Sets`
- `Update`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.AzureFleet/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AzureFleet/fleets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{fleetName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{fleetName}/virtualMachineScaleSets`

Target query:
- `api-version=2024-11-01`

Important routing implications from the reviewed pages:

- provider operations are subscription-independent under `/providers/Microsoft.AzureFleet/operations`
- list-by-subscription is under `/subscriptions/{subscriptionId}/providers/Microsoft.AzureFleet/fleets`
- core fleet CRUD and list-by-resource-group are resource-group scoped
- the only child route in this surface is `virtualMachineScaleSets`, which is read-only and fleet-scoped

## Contract Notes From Reviewed Pages

Shared management-plane traits:

- every reviewed method requires `api-version=2024-11-01`
- failures use `Error Response`
- all routes are standard ARM OAuth2 endpoints in Learn; local Stackyard examples should continue to use the repo’s Azure shim transport

### Operations

`Operations.List`
- `GET /providers/Microsoft.AzureFleet/operations`
- returns `Operation List Result`

Modeling implication:
- implement first as a deterministic `value` list covering:
  - `Microsoft.AzureFleet/fleets/read`
  - `Microsoft.AzureFleet/fleets/write`
  - `Microsoft.AzureFleet/fleets/delete`
  - `Microsoft.AzureFleet/fleets/virtualMachineScaleSets/read`

### Fleets

`Fleets.Create Or Update`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{fleetName}`
- reviewed required request fields:
  - `location`
  - `properties.computeProfile`
  - `properties.vmSizesProfile`
- reviewed optional fields include:
  - `identity`
  - `plan`
  - `properties.additionalLocationsProfile`
  - `properties.regularPriorityProfile`
  - `properties.spotPriorityProfile`
  - `properties.vmAttributes`
  - `tags`
  - `zones`
- responses:
  - `200` with `Fleet`
  - `201` with `Fleet`
  - `Retry-After` header may be present

`Fleets.Update`
- `PATCH /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{fleetName}`
- reviewed patchable fields:
  - `identity`
  - `plan`
  - `properties`
  - `tags`
- responses:
  - `200` with `Fleet`
  - `202` accepted with `Location` and `Retry-After`

`Fleets.Get`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{fleetName}`
- response: `Fleet`

`Fleets.List By Resource Group`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets`
- response: `Fleet List Result`

`Fleets.List By Subscription`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.AzureFleet/fleets`
- response: `Fleet List Result`

`Fleets.Delete`
- `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{fleetName}`
- responses:
  - `202` with `Location` and `Retry-After`
  - `204` when the resource does not exist

Important modeled fields visible on reviewed `Fleet` pages:
- `id`
- `name`
- `type`
- `location`
- `identity`
- `plan`
- `tags`
- `zones`
- `systemData`
- `properties.computeProfile`
- `properties.vmSizesProfile`
- `properties.additionalLocationsProfile`
- `properties.regularPriorityProfile`
- `properties.spotPriorityProfile`
- `properties.vmAttributes`
- `properties.provisioningState`
- `properties.timeCreated`
- `properties.uniqueId`

Modeling implication:
- one canonical `Fleet` resource store is enough for the whole service
- `Create Or Update` should validate the minimum required fields from the reviewed page and preserve the caller’s request body shape where feasible
- deterministic server-generated values should fill `id`, `type`, `systemData`, `properties.provisioningState`, `properties.timeCreated`, and `properties.uniqueId`

### List Virtual Machine Scale Sets

`Fleets.List Virtual Machine Scale Sets`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureFleet/fleets/{name}/virtualMachineScaleSets`
- response: `Virtual Machine Scale Set List Result`

Important modeled fields from the reviewed page:
- `value[].name`
- `value[].id`
- `value[].type`
- `value[].operationStatus`
- `value[].error`
- `nextLink`

Modeling implication:
- this route can be derived from the stored fleet instead of using a second persistent collection
- for the first pass, generate a stable synthetic VMSS list from the fleet name and priority profile data
- `operationStatus` should map to deterministic fleet lifecycle states such as `Creating`, `Updating`, or `Succeeded`

## Non-Goals

- do not emulate actual VM placement, capacity auctions, eviction, or pricing logic
- do not implement a real long-running operation controller for every mutation in the first pass
- do not validate the entire nested `computeProfile.baseVirtualMachineProfile` schema deeply
- do not emulate child resources beyond the documented `virtualMachineScaleSets` list route

## Stage 0: Coverage Naming And Route Claiming

- Add a dedicated Azure Compute Fleet router for `Microsoft.AzureFleet`
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Recommended aliases:
  - `compute_fleet`
  - `computefleet`
  - `compute-fleet`
  - `azure_compute_fleet`
  - `azure_computefleet`
  - `computefleet_2024_11_01`
  - `rest_computefleet_2024_11_01`
  - `microsoft_azurefleet`
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route recognition tests for:
  - provider operations
  - subscription fleet listing
  - resource-group fleet list and item routes
  - `virtualMachineScaleSets` child listing

Exit criteria:
- coverage scripts discover the new canonical service key
- Azure routing claims all Compute Fleet paths deterministically

## Stage 1: Shared Validation And Error Shape

- Validate `api-version=2024-11-01`
- Enforce supported methods:
  - `GET`, `PUT`, `PATCH`, `DELETE`
- Return deterministic ARM-style error payloads for:
  - unsupported methods
  - invalid or missing `api-version`
  - malformed JSON
  - missing required `Create Or Update` fields
  - fleet not found

Validation rules to implement first:
- `location` required on `PUT`
- `properties.computeProfile` required on `PUT`
- `properties.vmSizesProfile` required on `PUT`
- `PATCH` requires at least one updatable field

Exit criteria:
- negative tests cover method validation, malformed bodies, and not-found behavior
- contract and IO harnesses stop reporting generic `NotImplemented` or route misses

## Stage 2: Operations And Read-Only Discovery

- Implement `Operations.List` with a deterministic `value` array of provider operations
- Keep the response small but representative:
  - `name`
  - `display.provider`
  - `display.resource`
  - `display.operation`
  - `display.description`
  - `origin`
  - `isDataAction`

Exit criteria:
- provider operations page matches the documented route and typed envelope

## Stage 3: Core Fleet Lifecycle

- Implement persistent in-memory `Fleet` resources for:
  - `Create Or Update`
  - `Get`
  - `List By Resource Group`
  - `List By Subscription`
- Preserve key request fields from the caller:
  - `identity`
  - `plan`
  - `tags`
  - `zones`
  - `properties.computeProfile`
  - `properties.vmSizesProfile`
  - `properties.additionalLocationsProfile`
  - `properties.regularPriorityProfile`
  - `properties.spotPriorityProfile`
  - `properties.vmAttributes`
- Stamp deterministic server fields:
  - `id`
  - `name`
  - `type`
  - `systemData`
  - `etag`
  - `properties.provisioningState`
  - `properties.timeCreated`
  - `properties.uniqueId`

Recommended first-pass state behavior:
- `PUT` returns `201` for create and `200` for replace
- new resources default to `properties.provisioningState = "Succeeded"`
- store enough of `computeProfile` so the example can round-trip a real SDK request body

Exit criteria:
- all core read and list methods return typed `Fleet` or `Fleet List Result` payloads
- no fleet CRUD path falls through to generic Azure handlers

## Stage 4: Patch And Delete Semantics

- Implement `Update` as shallow merge semantics for:
  - `identity`
  - `plan`
  - `tags`
  - `properties`
- Update `systemData.lastModified*` fields deterministically
- Implement `Delete`
  - prefer deterministic `202` accepted on existing resources with `Location` and `Retry-After`
  - return `204` if the fleet is already absent, matching the reviewed page

Pragmatic implementation note:
- if the Azure Go SDK example is materially simpler with synchronous behavior, the emulator can still accept `PATCH` with `200` and `DELETE` with `202`, as long as the route no longer returns `NotImplemented` and typed headers are present where expected

Exit criteria:
- patch and delete are stable under repeated calls
- example and coverage harnesses do not stall on mutation endpoints

## Stage 5: Child `virtualMachineScaleSets` Listing

- Implement `List Virtual Machine Scale Sets`
- Derive a deterministic list from each stored fleet

Suggested synthetic child modeling:
- `name`: `<fleetName>-vmss-1`
- `id`: fleet child ID under `.../fleets/{fleetName}/virtualMachineScaleSets/{vmssName}`
- `type`: `Microsoft.AzureFleet/fleets/virtualMachineScaleSets`
- `operationStatus`: mirror the fleet’s current provisioning state
- `error`: omit on success; include a deterministic object only for explicit failure-test fixtures if needed later

Exit criteria:
- child list returns `Virtual Machine Scale Set List Result`
- at least one typed child item is present for created fleets

## Stage 6: Azure Go SDK Example

- Add `examples/azure/computefleet-2024-11-01`
- Use `armcomputefleet`
- Follow the repo’s Azure example pattern:
  - `azsdkshim.StaticTokenCredential{}`
  - `arm.ClientOptions`
  - `cloud.ResourceManager` endpoint override to Stackyard
  - `azsdkshim.NewTransport("/azure", account, subscriptionKey)`

Recommended example flow:
- create `OperationsClient` and list provider operations
- create `FleetsClient`
- create or update one fleet using the minimum documented required fields
- get the fleet
- list fleets by resource group
- list fleets by subscription
- list child virtual machine scale sets
- patch tags or one priority profile
- delete the fleet

Exit criteria:
- the example compiles cleanly
- the example exercises all implemented route families without special-case hacks

## Stage 7: Coverage And Verification

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
  - child VMSS listing
  - invalid body and missing required fields
  - invalid `api-version`

Expected verification commands after implementation:
- `go test ./internal/server -run 'TestAzureComputeFleet' -count=1`
- `go build ./examples/azure/computefleet-2024-11-01`
- `python3 scripts/azure-contract-coverage.py --service compute_fleet --require-service compute_fleet --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service compute_fleet --require-service compute_fleet --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service compute_fleet --require-service compute_fleet --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service compute_fleet --require-service compute_fleet --fail-on any`

Exit criteria:
- all Compute Fleet endpoints are discovered by the coverage scripts
- no Compute Fleet route returns `NotImplemented`
- all targeted coverage commands succeed

## Implementation Order Recommendation

Recommended merge order:

1. Stage 0 and Stage 1
2. Stage 2 and Stage 3
3. Stage 4 and Stage 5
4. Stage 6 and Stage 7

This order keeps the first landing small and makes the service visible to coverage quickly, before adding the SDK example and child list polishing.
