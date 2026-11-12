# Azure Quantum (`azurequantum-resourcemanager-2025-12-15-preview`) Staged Plan

## Objective

Emulate the Azure Quantum ARM management API for `2025-12-15-preview` with deterministic local behavior for:

- provider `operations`
- location-scoped `offerings`
- global `checkNameAvailability`
- top-level `workspaces` lifecycle and discovery routes
- workspace key-management routes
- an Azure Go SDK example under `examples/azure/azurequantum-resourcemanager-2025-12-15-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for `Microsoft.Quantum`.
- This is distinct from the Azure Quantum workspace data plane already modeled separately in Stackyard.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2025-12-15-preview`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/operation-groups?view=rest-azurequantum-resourcemanager-2025-12-15-preview`

Operation-group and method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/operations/list?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/offerings/list?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/check-name-availability?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/create-or-update?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/get?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/list-by-resource-group?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/list-by-subscription?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/list-keys?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/regenerate-keys?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/update-tags?view=rest-azurequantum-resourcemanager-2025-12-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/resourcemanager/workspaces/delete?view=rest-azurequantum-resourcemanager-2025-12-15-preview`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/quantum/armquantum`

If the preview surface is ahead of the released typed package:
- keep the example typed where possible
- use `azcore/arm` plus a small raw request helper only for preview-only gaps

## Service Naming

Canonical Stackyard naming:
- service key: `quantum`
- display name: `Azure Quantum`
- provider namespace: `Microsoft.Quantum`
- API version: `2025-12-15-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_quantum.go`
- tests: `internal/server/provider_azure_quantum_test.go`
- example: `examples/azure/azurequantum-resourcemanager-2025-12-15-preview`
- plan doc: `docs/azurequantum-resourcemanager-2025-12-15-preview-plan.md`

Recommended coverage aliases:
- `quantum`
- `azure_quantum`
- `azurequantum`
- `quantum_workspace`
- `quantum_workspaces`
- `quantum-workspaces`
- `microsoft_quantum`
- `azurequantum_resourcemanager_2025_12_15_preview`
- `rest_azurequantum_resourcemanager_2025_12_15_preview`

## Surface Summary

The reviewed `2025-12-15-preview` management surface exposes 3 operation groups:

1. `Operations`
2. `Offerings`
3. `Workspaces`

Broad route families identified from the reviewed pages:

- provider discovery:
  - `/providers/Microsoft.Quantum/operations`
- location-scoped catalog routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Quantum/locations/{location}/offerings`
- global name check:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Quantum/checkNameAvailability`
- top-level ARM resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}`
- subscription and resource-group listing:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Quantum/workspaces`
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces`
- workspace action routes:
  - `/listKeys`
  - `/regenerateKeys`

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2025-12-15-preview`
- top-level resources use `Microsoft.Quantum`

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.Quantum/operations?api-version=2025-12-15-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Quantum/locations/{location}/offerings?api-version=2025-12-15-preview`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.Quantum/checkNameAvailability?api-version=2025-12-15-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}?api-version=2025-12-15-preview`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}?api-version=2025-12-15-preview`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}?api-version=2025-12-15-preview`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}?api-version=2025-12-15-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Quantum/workspaces?api-version=2025-12-15-preview`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces?api-version=2025-12-15-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}/listKeys?api-version=2025-12-15-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}/regenerateKeys?api-version=2025-12-15-preview`

Important routing implications from the reviewed pages:

- `workspaceName` follows standard Azure resource-name constraints and should be validated lightly rather than left completely free-form
- key routes are actions on the workspace resource, not independent child resources
- `checkNameAvailability` is provider-scoped, not resource-group-scoped
- delete is ARM-style and should prefer accepted/no-content semantics over a generic `200`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style JSON error envelopes
- list routes return `value`
- top-level `workspaces` return standard ARM resource fields:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `identity`
  - `systemData`
  - `properties.provisioningState`

### Operations

`Operations.List`
- provider route under `/providers/Microsoft.Quantum/operations`
- returns standard provider operation metadata

Modeling implication:
- expose a deterministic `value` list including at least:
  - `Microsoft.Quantum/workspaces/read`
  - `Microsoft.Quantum/workspaces/write`
  - `Microsoft.Quantum/workspaces/delete`
  - `Microsoft.Quantum/workspaces/listKeys/action`
  - `Microsoft.Quantum/workspaces/regenerateKeys/action`
  - `Microsoft.Quantum/locations/offerings/read`

### Offerings

Reviewed method pages:
- `List`

Key traits from the reviewed page:

- route is location-scoped under subscription
- response is a catalog-style list, not an ARM tracked resource collection
- offerings include target/provider capabilities, SKU-like information, and pricing or quota dimensions

Modeling implications:
- return a deterministic `value` list of 2-3 representative offerings
- each offering should include stable fields that examples and coverage can inspect, such as:
  - `id`
  - `providerId`
  - `providerSku`
  - `displayName`
  - `targets`
  - `quotaDimensions`
  - `pricingDimensions`
- do not overfit to every optional nested field in the first pass

### Workspaces

Reviewed method pages:
- `Check Name Availability`
- `Create Or Update`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `List Keys`
- `Regenerate Keys`
- `Update Tags`
- `Delete`

Key traits from the reviewed pages:

- `Check Name Availability`
  - request body includes `name` and `type`
  - response includes an availability boolean plus reason/message when unavailable

- `Create Or Update`
  - `PUT /.../workspaces/{workspaceName}`
  - required request field:
    - `location`
  - common request fields include:
    - `tags`
    - `identity`
    - `properties.providers`
    - `properties.storageAccount`
    - `properties.workspaceDescription`
    - `properties.apiKeyEnabled`
  - returns a full ARM resource envelope

- `Get`
  - returns workspace metadata plus provider-specific properties

- `Update Tags`
  - patch-style partial update focused on `tags`
  - should not require the full create payload

- `List Keys`
  - returns workspace access keys and connection strings or endpoint-aligned auth material

- `Regenerate Keys`
  - mutates either the primary or secondary key based on the request payload
  - should preserve the untouched key

- `Delete`
  - ARM-style delete with accepted/no-content semantics

Modeling implications:
- persist one canonical `Workspace` store keyed by ARM ID
- preserve caller-provided `tags`, `identity`, and a lightweight `properties` subset
- synthesize stable values for:
  - `properties.provisioningState`
  - `properties.endpointUri`
  - `properties.workspaceId`
  - `properties.apiKeyEnabled`
- seed deterministic primary and secondary keys during create so `listKeys` always returns data
- implement `regenerateKeys` as a state mutation on the stored workspace record
- implement `HEAD` on the workspace resource path even if the docs foreground `GET`, because ARM existence probes commonly appear in coverage/examples

## Non-Goals

- do not emulate the Azure Quantum data-plane jobs or sessions in this service
- do not implement real provider-side quota enforcement from the offerings catalog
- do not model full ARM async-operation polling unless coverage/examples require it
- do not enforce deep validation for every nested provider capability field on create
- do not implement RBAC or real key-auth behavior behind `listKeys` results

## Stage 0: Coverage Naming And Route Claiming

- Add a dedicated Azure Quantum ARM router for `Microsoft.Quantum`
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add the aliases listed above
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route recognition tests for:
  - provider operations
  - location offerings
  - `checkNameAvailability`
  - workspace CRUD/list routes
  - workspace key action routes

Exit criteria:
- coverage scripts can target the new service key without alias misses
- router tests prove requests resolve into the new provider instead of `NotImplemented`

## Stage 1: Provider Operations, Offerings, And Name Check

- Implement:
  - `GET /providers/Microsoft.Quantum/operations`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Quantum/locations/{location}/offerings`
  - `POST /subscriptions/{subscriptionId}/providers/Microsoft.Quantum/checkNameAvailability`
- Return deterministic provider operations and offering catalogs
- Implement name-check rules such as:
  - obvious collisions with existing workspace names
  - basic invalid-name rejection
  - accepted names returning `nameAvailable: true`

Exit criteria:
- all discovery and preflight routes return modeled success shapes
- no reviewed non-workspace route falls through to `NotImplemented`

## Stage 2: Core `workspaces` Lifecycle

- Implement:
  - `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}`
  - `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}`
  - `HEAD /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}`
  - `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Quantum/workspaces`
  - `PATCH /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}`
  - `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Quantum/workspaces/{workspaceName}`
- Persist deterministic ARM metadata:
  - `id`
  - `type`
  - `systemData`
  - `properties.provisioningState`
  - `properties.endpointUri`
  - `properties.workspaceId`
- Seed deterministic access keys during create

Exit criteria:
- all core workspace methods return modeled success shapes
- repeated get/list/patch/delete flows are deterministic
- no workspace route returns `NotImplemented`

## Stage 3: Workspace Key Actions

- Implement:
  - `POST /.../workspaces/{workspaceName}/listKeys`
  - `POST /.../workspaces/{workspaceName}/regenerateKeys`
- Model stable response fields such as:
  - `primaryKey`
  - `secondaryKey`
  - connection strings or endpoint-key pairings if surfaced by the reviewed method pages
- Support regenerate payloads that target only one key at a time

Exit criteria:
- `listKeys` returns non-empty deterministic keys
- `regenerateKeys` mutates only the requested key and preserves the other
- key action routes are fully claimed by the new provider

## Stage 4: Tests And Contract Tightening

- Add focused tests in `internal/server/provider_azure_quantum_test.go` covering:
  - provider operations
  - offerings listing
  - name availability flows
  - workspace create/get/head/list/patch/delete
  - list-keys and regenerate-keys behavior
  - 404 behavior for missing resources
  - lightweight validation failures for malformed names or missing required fields
- Add deterministic assertions for:
  - `systemData`
  - ARM IDs and types
  - list ordering
  - endpoint URI generation
  - key mutation behavior

Exit criteria:
- focused server tests cover the full reviewed surface implemented in prior stages
- coverage scripts can exercise the service without route misses or placeholder empty payloads

## Stage 5: Azure Go SDK Example

- Add example under:
  - `examples/azure/azurequantum-resourcemanager-2025-12-15-preview`
- Prefer the official typed management SDK package:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/quantum/armquantum`
- Example flow should cover:
  - check name availability
  - create a workspace
  - get/list/update/delete the workspace
  - list offerings for a location
  - list keys and regenerate one key
- If the typed SDK does not expose every preview method, keep the example mostly typed and use a small raw ARM request helper only for the missing preview route

Exit criteria:
- example builds cleanly in repo CI conventions
- example demonstrates at least one discovery flow, one workspace lifecycle flow, and one key-management flow

## Validation Plan

After implementation, validate with:

- targeted server tests:
  - `go test ./internal/server -run 'TestAzureQuantum' -count=1`
- example build:
  - `go build ./examples/azure/azurequantum-resourcemanager-2025-12-15-preview`
- coverage checks:
  - `python3 scripts/azure-contract-coverage.py --service quantum --require-service quantum --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service quantum --require-service quantum --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service quantum --require-service quantum --fail-on any`
  - `python3 scripts/azure-endpoint-coverage.py --service quantum --require-service quantum --fail-on any`

Success target:
- all reviewed routes return modeled success or modeled service errors
- no reviewed Azure Quantum ARM route returns `NotImplemented`
