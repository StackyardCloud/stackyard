# Azure Data Manager for Agriculture (`data-manager-for-agri-resourcemanager-2023-06-01-preview`) Staged Plan

## Objective

Emulate Azure Data Manager for Agriculture Resource Manager for API version `2023-06-01-preview` with deterministic ARM-style behavior under `Microsoft.AgFoodPlatform`, including:

- provider `operations`
- top-level `farmBeats` resource lifecycle
- `checkNameAvailability`
- async `operationResults`
- `dataConnectors`
- available and installed `extensions`
- `privateEndpointConnections` and `privateLinkResources`
- solution discovery plus installed `solutions`
- an Azure Go SDK example under `examples/azure/data-manager-for-agri-resourcemanager-2023-06-01-preview`
- Azure coverage-script wiring for the new service key

The service surface is broad enough that implementation should be staged around the core `farmBeats` resource and then expanded into child collections and action-style control-plane routes.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/operation-groups?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`

Primary service overview reviewed:
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/`

Linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/operations/list?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/data-manager-for-agriculture-resources/create-or-update?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/data-manager-for-agriculture-resources/get?tabs=HTTP&view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview&viewFallbackFrom=rest-data-manager-for-agri-resourcemanager-2021-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/data-manager-for-agriculture-resources/list-by-resource-group?tabs=HTTP&view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/check-name-availability/check-name-availability?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/data-connectors/get?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/data-connectors/delete?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/operation-results/get?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/extensions/get?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/extensions/delete?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/private-endpoint-connections/create-or-update?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/private-endpoint-connections/get?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/private-link-resources/list-by-resource?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/resourcemanager/solutions/get?view=rest-data-manager-for-agri-resourcemanager-2023-06-01-preview`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/agrifood/armagrifood`

## Service Naming

Canonical Stackyard naming:
- service key: `data_manager_for_agri`
- display name: `Azure Data Manager for Agriculture`
- provider namespace: `Microsoft.AgFoodPlatform`
- API version: `2023-06-01-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_data_manager_for_agri.go`
- tests: `internal/server/provider_azure_data_manager_for_agri_test.go`
- example: `examples/azure/data-manager-for-agri-resourcemanager-2023-06-01-preview`
- plan doc: `docs/data-manager-for-agri-resourcemanager-2023-06-01-preview-plan.md`

Recommended future coverage aliases:
- `data_manager_for_agri`
- `datamanagerforagri`
- `data-manager-for-agri`
- `agrifood`
- `agfoodplatform`
- `microsoft_agfoodplatform`
- `data_manager_for_agri_resourcemanager_2023_06_01_preview`

## Surface Summary

The reviewed control-plane documentation exposes these operation families:

1. `Data Manager for Agriculture Resource`
2. `Available Extensions`
3. `Management of Extensions`
4. `Check Name Availability`
5. `Operations`
6. `Async Operation Result`
7. `Private Endpoint Connections`
8. `Private Link Resources`
9. `Solutions Discoverability`
10. `Solutions`
11. `DataConnectors`

Important route-shape observations from the reviewed pages:

- The top-level managed resource is still routed as `farmBeats` even though the model type is `DataManagerForAgriculture`.
- Most nested resources are anchored under:
  `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}/...`
- `checkNameAvailability` is subscription-scoped and not nested under a resource group.
- `operationResults` is location-scoped under the provider namespace, not nested under a specific `farmBeats` instance.
- `extensions`, `solutions`, and `dataConnectors` are child collections beneath a `farmBeats` resource, with list/get/install-or-create/update/delete style semantics.
- Private networking support is split between mutable `privateEndpointConnections` and read-only `privateLinkResources`.

## Route Envelope

Primary ARM routes:

- `/azure/providers/Microsoft.AgFoodPlatform/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AgFoodPlatform/checkNameAvailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AgFoodPlatform/locations/{location}/operationResults/{operationResultsId}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}/dataConnectors`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}/extensions`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}/privateLinkResources`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AgFoodPlatform/farmBeats/{dataManagerForAgricultureResourceName}/solutions`

Target query:
- `api-version=2023-06-01-preview`

## Contract Notes From Reviewed Pages

### Operations

`Operations.List`
- `GET /providers/Microsoft.AgFoodPlatform/operations`
- returns a provider operation catalog with standard ARM operation metadata

Modeling implication:
- implement first as a deterministic `value` list covering every collection the emulator exposes

### Data Manager for Agriculture Resource

Reviewed pages show standard ARM CRUD/list/update semantics for the top-level managed resource.

Important fields surfaced in the reviewed pages:
- `location`
- `tags`
- `identity.type`
- `properties.instanceUri`
- `properties.provisioningState`
- `properties.publicNetworkAccess`
- `properties.sensorIntegration.enabled`
- `properties.sensorIntegration.provisioningState`
- `properties.privateEndpointConnections`

Modeling implication:
- use `farmBeats` as the persisted route segment, but return `DataManagerForAgriculture`-shaped payloads
- the first implementation should require `location` and generate deterministic `instanceUri`, provisioning state, and `systemData`
- `HEAD` support is likely worth adding early because ARM coverage frequently probes it even when Learn emphasizes `GET`

### Check Name Availability

Reviewed page shows:
- `POST /subscriptions/{subscriptionId}/providers/Microsoft.AgFoodPlatform/checkNameAvailability`
- body with `name` and `type`
- `200 OK` returns `nameAvailable`, optional `reason`, and optional `message`

Modeling implication:
- implement deterministic duplicate-name checking against the in-memory `farmBeats` store
- support `AlreadyExists` and `Invalid` paths instead of returning generic errors

### Async Operation Result

Reviewed page shows:
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.AgFoodPlatform/locations/{location}/operationResults/{operationResultsId}`
- returns `ArmAsyncOperation`

Modeling implication:
- mint stable operation IDs for mutating calls that need long-running-operation compatibility
- even if writes complete immediately, store synthetic successful operation records so pollers and coverage can resolve them

### DataConnectors

Reviewed pages show a child collection under `farmBeats` with at least get and delete operations, and the control-plane overview states create, update, and delete support.

Important fields surfaced in the reviewed `get` page:
- `properties.credentials.kind`
- `properties.credentials.clientId`
- `properties.credentials.clientSecret`
- `properties.credentials.apiKey`
- `eTag`
- `systemData`

Modeling implication:
- stage this after `farmBeats`
- represent connector credentials structurally but keep secrets synthetic and non-sensitive
- validate the connector name and parent existence, but avoid deep partner-specific credential semantics in the first pass

### Available Extensions and Management of Extensions

Reviewed pages show two related but distinct surfaces:

- available extension catalog
- installed extension management under a specific `farmBeats`

Important implications from the reviewed pages and SDK:
- `ExtensionsClient` handles install/uninstall/update/list-by-farmbeats
- `FarmBeatsExtensionsClient` exposes catalog lookup/list behavior

Modeling implication:
- keep a static provider-wide extension catalog
- keep a separate per-`farmBeats` installed-extension store
- reuse the same canonical extension IDs across both surfaces so installs are deterministic

### Private Endpoint Connections and Private Link Resources

Reviewed pages show:
- `privateEndpointConnections` supports get, list, create-or-update approval, and likely delete
- `privateLinkResources` supports list and get semantics under a `farmBeats` resource

Important fields surfaced:
- `properties.privateLinkServiceConnectionState.status`
- `properties.privateLinkServiceConnectionState.description`
- `properties.provisioningState`
- `properties.groupIds`

Modeling implication:
- `privateLinkResources` can be a mostly static list
- `privateEndpointConnections` should be a mutable child store tied to the parent `farmBeats` resource
- approval/rejection should mutate connection-state fields instead of inventing a separate action route

### Solutions Discoverability and Solutions

The control-plane overview separates:
- provider or service-level discoverability of available solutions
- installed solutions under a specific `farmBeats`

The reviewed `Solutions.Get` page confirms installed solutions use:
- `/farmBeats/{resourceName}/solutions/{solutionId}`

Modeling implication:
- keep a static catalog of discoverable solutions
- keep a per-parent installed-solution store similar to extensions
- first-pass install/update/delete behavior can be deterministic and synchronous as long as returned shapes remain ARM-like

## Azure Go SDK Notes

The closest official Azure Go SDK package is:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/agrifood/armagrifood`

Important caveat from the reviewed package:
- the package appears to cover the older control-plane subset well, including `FarmBeatsModelsClient`, `OperationsClient`, `LocationsClient`, `ExtensionsClient`, `FarmBeatsExtensionsClient`, `PrivateEndpointConnectionsClient`, and `PrivateLinkResourcesClient`
- the package does not obviously expose the full newer `2023-06-01-preview` surface such as every `dataConnectors` or `solutions` operation

Example implication:
- the example should use `armagrifood` for the overlapping official clients
- if the repo wants one example to touch newer uncovered routes, use the Azure SDK pipeline plus direct REST requests rather than inventing a fake typed client

## Planned Repo Shape

Implementation:
- add `azureDataManagerForAgriStore` to the shared server state in `internal/server/http.go`
- add a dedicated provider file `internal/server/provider_azure_data_manager_for_agri.go`
- register the provider from the main Azure route dispatcher in `internal/server/http.go`

Tests:
- add focused route and shape coverage in `internal/server/provider_azure_data_manager_for_agri_test.go`
- include list/get/create/update/delete tests for the top-level resource and at least one child collection
- include explicit tests for `checkNameAvailability`, `operationResults`, and one approval-style `privateEndpointConnections` mutation

Example:
- add `examples/azure/data-manager-for-agri-resourcemanager-2023-06-01-preview/main.go`
- prefer an example that creates a `farmBeats` resource, checks name availability, lists operations, and exercises one child collection

Coverage wiring to add during implementation:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended future doc/alias mapping:
- map `data_manager_for_agri` to this plan doc and to the `Microsoft.AgFoodPlatform` route family

## Staged Implementation Plan

### Stage 0: Service Claiming and Coverage Wiring

Goal:
- reserve the new Azure service key and ensure the provider stops falling through to generic `NotImplemented`

Scope:
- add service aliases in all Azure coverage scripts
- add plan-doc mapping in the doc coverage script
- add a provider dispatch stub in `http.go`

Exit criteria:
- coverage can target `data_manager_for_agri`
- unmatched routes for this namespace are routed into the new provider instead of the generic fallback

### Stage 1: Shared ARM Helpers, Operations, and Name Availability

Goal:
- land the provider skeleton and the lowest-risk provider-wide endpoints

Scope:
- shared ID parsing and validation for `Microsoft.AgFoodPlatform`
- `Operations.List`
- `CheckNameAvailability`
- deterministic `systemData`, provisioning-state, and async operation record helpers

Exit criteria:
- provider discovery is green
- name checks reflect existing in-memory resources

### Stage 2: Core `farmBeats` Resource

Goal:
- make the top-level resource lifecycle coverage-complete

Scope:
- create or update
- get
- head
- list by resource group
- list by subscription
- patch update
- delete

Implementation notes:
- require `location`
- synthesize `instanceUri`
- keep `publicNetworkAccess`, `sensorIntegration`, and `identity` shallow but stable

Exit criteria:
- all reviewed top-level `farmBeats` endpoints succeed without `NotImplemented`

### Stage 3: Async Operation Results

Goal:
- support polling and long-running-operation compatibility for the mutating endpoints

Scope:
- store successful synthetic operation records keyed by location and operation ID
- `OperationResults.Get`

Exit criteria:
- any operation-result coverage path resolves deterministically

### Stage 4: `dataConnectors`

Goal:
- cover the child CRUD surface that motivated the `2023-06-01-preview` control-plane update

Scope:
- create or update
- get
- list by `farmBeats`
- patch update
- delete

Implementation notes:
- support at least the credential discriminator shapes surfaced in the docs
- keep secrets synthetic and stable

Exit criteria:
- `dataConnectors` endpoints are all successful and return modeled payloads

### Stage 5: Extensions and Solutions

Goal:
- implement the catalog/install split for both extensions and solutions

Scope:
- available extension catalog list/get
- installed extension create/get/list/update/delete
- discoverable solutions list/get
- installed solutions create/get/list/update/delete

Implementation notes:
- use static catalog fixtures with deterministic IDs like `publisher.extension` and `publisher.solution`
- installed-resource state should hang off the parent `farmBeats` resource

Exit criteria:
- extension and solution coverage no longer falls through or returns empty shells

### Stage 6: Private Networking

Goal:
- cover the remaining child families under the managed resource

Scope:
- `privateEndpointConnections` get/list/create-or-update/delete
- `privateLinkResources` get/list

Implementation notes:
- keep private-link resources read-only and static
- keep connection approval state mutable and modeled

Exit criteria:
- private networking routes return deterministic ARM-shaped payloads

### Stage 7: Example and End-to-End Validation

Goal:
- prove the service is usable through the official Azure Go SDK where available and through coverage gates

Scope:
- add the example under `examples/azure/data-manager-for-agri-resourcemanager-2023-06-01-preview`
- prefer the official `armagrifood` clients for operations, name availability, and `farmBeats`
- use raw REST calls only for preview routes not surfaced by the typed package

Validation target after implementation:
- `go test ./internal/server -run 'TestAzureDataManagerForAgri' -count=1`
- `go build ./examples/azure/data-manager-for-agri-resourcemanager-2023-06-01-preview`
- `python3 scripts/azure-contract-coverage.py --service data_manager_for_agri --require-service data_manager_for_agri --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service data_manager_for_agri --require-service data_manager_for_agri --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service data_manager_for_agri --require-service data_manager_for_agri --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service data_manager_for_agri --require-service data_manager_for_agri --fail-on any`

## Non-Goals For First Implementation

To keep the first merge tractable, do not attempt all of the following at once:

- full ARM long-running-operation header fidelity beyond stable operation-result polling
- exhaustive validation of every credential subtype and extension/solution publisher-specific payload
- private endpoint lifecycle realism beyond deterministic approval-state mutation
- data-plane `farmbeats.azure.net` APIs, which are a separate service surface and should remain out of scope for this control-plane provider

## Recommended First Merge Slice

The best first merge is:

1. Stage 0
2. Stage 1
3. Stage 2
4. Stage 3
5. one narrow child collection, preferably `dataConnectors`

That sequence claims the namespace, satisfies the most common ARM coverage routes, and avoids starting with the more bespoke extension/solution/private-network branches.
