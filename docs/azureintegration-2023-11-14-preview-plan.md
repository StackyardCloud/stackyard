# Azure Integration Environment (`azureintegration-2023-11-14-preview`) Staged Plan

## Objective

Emulate the Azure Integration Environment ARM management API for `2023-11-14-preview` with deterministic local behavior for:

- provider `operations`
- top-level `spaces` lifecycle and discovery routes
- nested `applications`
- nested `application resources`
- nested `business processes`
- nested `business process versions`
- space-scoped `infrastructure resources`
- application-scoped business-process development artifact actions
- an Azure Go SDK example under `examples/azure/azureintegration-2023-11-14-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure Integration Environment under `Microsoft.IntegrationSpaces`.
- This is not a Logic Apps runtime/data plane and not a messaging/event-processing data plane.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2023-11-14-preview`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/operation-groups?view=rest-azureintegration-2023-11-14-preview`

Operation-group and representative method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/spaces?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/spaces/create-or-update?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/spaces/patch?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/applications/patch?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/application-resources/create-or-update?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/application-resources/get?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/application-resources/patch?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/business-processes/create-or-update?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/business-processes/get?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/business-processes/patch?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/business-process-versions/get?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/business-process-versions/list-by-business-process?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/applications/get-business-process-development-artifact?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/applications/save-business-process-development-artifact?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/applications/validate-business-process-development-artifact?view=rest-azureintegration-2023-11-14-preview`
- `https://learn.microsoft.com/en-us/rest/api/azureintegration/infrastructure-resources/patch?view=rest-azureintegration-2023-11-14-preview`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/integrationspaces/armintegrationspaces`

SDK selection note:
- The official `armintegrationspaces` package exists and already exposes typed clients such as `SpacesClient`, `ApplicationsClient`, `ApplicationResourcesClient`, `BusinessProcessesClient`, `BusinessProcessVersionsClient`, and `InfrastructureResourcesClient`.
- Plan assumption: prefer typed `armintegrationspaces` clients for the example and implementation guidance instead of falling back to raw ARM requests.

## Service Naming

Canonical Stackyard naming:
- service key: `azureintegration`
- display name: `Azure Integration Environment`
- provider namespace: `Microsoft.IntegrationSpaces`
- API version: `2023-11-14-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_integration_environment.go`
- tests: `internal/server/provider_azure_integration_environment_test.go`
- example: `examples/azure/azureintegration-2023-11-14-preview`
- plan doc: `docs/azureintegration-2023-11-14-preview-plan.md`

Recommended coverage aliases:
- `azureintegration`
- `azure_integration`
- `azure-integration`
- `integration_environment`
- `integration-environment`
- `integration_spaces`
- `integrationspaces`
- `microsoft_integrationspaces`
- `rest_azureintegration_2023_11_14_preview`
- `azureintegration_2023_11_14_preview`

## Surface Summary

The reviewed `2023-11-14-preview` management surface centers on a nested hierarchy:

1. `Operations`
2. `Spaces`
3. `Applications`
4. `Application Resources`
5. `Business Processes`
6. `Business Process Versions`
7. `Infrastructure Resources`

The `Applications` group also carries several action-style methods for business-process development artifacts:

- `Save Business Process Development Artifact`
- `Get Business Process Development Artifact`
- `Delete Business Process Development Artifact`
- `List Business Process Development Artifacts`
- `Validate Business Process Development Artifact`

Broad route families identified from the reviewed pages:

- provider discovery:
  - `/providers/Microsoft.IntegrationSpaces/operations`
- top-level ARM resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}`
- nested application hierarchy:
  - `/spaces/{spaceName}/applications/{applicationName}`
  - `/spaces/{spaceName}/applications/{applicationName}/resources/{resourceName}`
  - `/spaces/{spaceName}/applications/{applicationName}/businessProcesses/{businessProcessName}`
  - `/spaces/{spaceName}/applications/{applicationName}/businessProcesses/{businessProcessName}/versions/{businessProcessVersion}`
- space-level infrastructure resources:
  - `/spaces/{spaceName}/infrastructureResources/{infrastructureResourceName}`
- application-scoped artifact actions:
  - `/applications/{applicationName}/saveBusinessProcessDevelopmentArtifact`
  - `/applications/{applicationName}/getBusinessProcessDevelopmentArtifact`
  - `/applications/{applicationName}/deleteBusinessProcessDevelopmentArtifact`
  - `/applications/{applicationName}/listBusinessProcessDevelopmentArtifacts`
  - `/applications/{applicationName}/validateBusinessProcessDevelopmentArtifact`

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2023-11-14-preview`
- all routes sit under the `Microsoft.IntegrationSpaces` provider namespace

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.IntegrationSpaces/operations?api-version=2023-11-14-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}?api-version=2023-11-14-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/applications/{applicationName}?api-version=2023-11-14-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/applications/{applicationName}/resources/{resourceName}?api-version=2023-11-14-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/applications/{applicationName}/businessProcesses/{businessProcessName}?api-version=2023-11-14-preview`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/applications/{applicationName}/businessProcesses/{businessProcessName}/versions/{businessProcessVersion}?api-version=2023-11-14-preview`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/infrastructureResources/{infrastructureResourceName}?api-version=2023-11-14-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/applications/{applicationName}/saveBusinessProcessDevelopmentArtifact?api-version=2023-11-14-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.IntegrationSpaces/spaces/{spaceName}/applications/{applicationName}/validateBusinessProcessDevelopmentArtifact?api-version=2023-11-14-preview`

Important routing implications from the reviewed pages:

- name patterns are consistently constrained to `[a-zA-Z0-9_.()-]{0,79}$` for `spaceName`, `applicationName`, `resourceName`, and `businessProcessName`
- `businessProcessVersion` is its own path segment and should be stored distinctly from the mutable parent business-process resource
- some routes are normal ARM resources while artifact routes are action-style `POST` methods returning domain payloads rather than ARM resource envelopes
- list-by-subscription and list-by-resource-group exist for `spaces`, while applications are listed by parent `space`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- ARM resources return standard fields such as:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.*`

### Operations

`Operations.List`
- provider route under `/providers/Microsoft.IntegrationSpaces/operations`
- returns provider operations including the main read/write/delete verbs for spaces and nested resources

Modeling implication:
- expose a deterministic `value` list covering each resource family implemented in the staged emulator

### Spaces

Reviewed method pages:
- `Create Or Update`
- `Patch`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Delete`

Key traits from the reviewed pages:
- `PUT /.../spaces/{spaceName}` creates a `Space`
- `PATCH` allows description/tag updates
- `Space` is the root parent for applications and infrastructure resources
- example payloads use a standard ARM envelope and `properties.provisioningState`

Modeling implications:
- persist one canonical `Space` store keyed by ARM ID
- implement `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-subscription, and list-by-resource-group
- synthesize stable `properties.provisioningState`

### Applications

Reviewed method pages:
- `Create Or Update`
- `Get`
- `Patch`
- `Delete`
- `List By Space`
- business-process artifact action methods

Key traits from the reviewed pages:
- applications are nested directly under a space
- the application resource returns an ARM envelope
- the same operation group also handles domain-specific artifact actions under the application path

Modeling implications:
- store applications canonically under a parent `Space`
- implement CRUD/list as regular ARM resources
- keep artifact actions separate from the ARM resource persistence path

### Application Resources

Reviewed method pages:
- `Create Or Update`
- `Get`
- `Patch`

Key traits from the reviewed pages:
- route:
  - `/spaces/{spaceName}/applications/{applicationName}/resources/{resourceName}`
- example properties include:
  - `resourceId`
  - `resourceType`
  - `resourceKind`

Modeling implications:
- treat these as straightforward nested ARM resources under an application
- validate that `properties.resourceId` and `properties.resourceType` are present on create

### Business Processes

Reviewed method pages:
- `Create Or Update`
- `Get`
- `Patch`

Key traits from the reviewed pages:
- route:
  - `/spaces/{spaceName}/applications/{applicationName}/businessProcesses/{businessProcessName}`
- example properties include:
  - `description`
  - `tableName`
  - `identifier`
  - `businessProcessStages`
  - `businessProcessMapping`

Modeling implications:
- business processes should be modeled as first-class nested ARM resources
- normalize the more complex maps into stable deterministic fixtures rather than trying to fully simulate workflow behavior

### Business Process Versions

Reviewed method pages:
- `Get`
- `List By Business Process`

Key traits from the reviewed pages:
- version resources are read-only children of a business process
- list route supports query parameters like `top`, `skip`, `filter`, `select`, `expand`, and `orderby`
- example payloads expose:
  - `properties.version`
  - `properties.tableName`
  - `properties.identifier`
  - `properties.businessProcessStages`
  - `properties.businessProcessMapping`

Modeling implications:
- seed at least one deterministic version per created business process
- support `GET` and list-by-parent with simple pagination/query tolerance

### Infrastructure Resources

Reviewed method pages:
- `Patch`
- inferred group routes from the SDK and operation-groups page

Key traits from the reviewed pages:
- route:
  - `/spaces/{spaceName}/infrastructureResources/{infrastructureResourceName}`
- these are nested under a space, not under an application

Modeling implications:
- start with `GET`, list-by-space, and `PATCH`
- treat them as read-mostly resources with seeded fixtures unless a reviewed create method is explicitly needed during implementation

### Business Process Development Artifact Actions

Reviewed method pages:
- `Save Business Process Development Artifact`
- `Get Business Process Development Artifact`
- `Validate Business Process Development Artifact`
- plus the operation-group listing for delete/list actions

Key traits from the reviewed pages:
- action routes are `POST`
- payloads are not simple ARM envelopes; they represent artifact documents and validation results
- these routes are application-scoped but operate on business-process development payloads

Modeling implications:
- maintain a small in-memory artifact store keyed by application ID plus artifact name/version
- implement deterministic success payloads for:
  - save
  - get
  - list
  - delete
  - validate

## Staged Implementation Plan

### Stage 0: Wiring and Coverage Discovery

Deliverables:
- add a new Azure router:
  - `internal/server/provider_azure_integration_environment.go`
- register the router in `internal/server/http.go`
- add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map the plan doc in the doc coverage script

Goal:
- the new service is discoverable as `azureintegration` and no longer falls through to generic `NotImplemented`

### Stage 1: Provider Operations and Root Spaces

Implement:
- `GET /providers/Microsoft.IntegrationSpaces/operations`
- `spaces`:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list by resource group
  - list by subscription

Goal:
- establish the top-level ARM store and root parent resource

### Stage 2: Applications and Application Resources

Implement:
- `applications`:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list by space
- `application resources`:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list by application

Goal:
- complete the core nested hierarchy used by most SDK examples

### Stage 3: Business Processes and Versioned Reads

Implement:
- `businessProcesses`:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - list by application
- `businessProcessVersions`:
  - `GET`
  - list by business process

Goal:
- model the central domain resources and their seeded read-only version history

### Stage 4: Infrastructure Resources

Implement:
- `infrastructureResources`:
  - list by space
  - `GET`
  - `HEAD`
  - `PATCH`

Goal:
- cover the space-scoped infrastructure branch with deterministic fixtures

### Stage 5: Application Artifact Actions

Implement:
- `saveBusinessProcessDevelopmentArtifact`
- `getBusinessProcessDevelopmentArtifact`
- `deleteBusinessProcessDevelopmentArtifact`
- `listBusinessProcessDevelopmentArtifacts`
- `validateBusinessProcessDevelopmentArtifact`

Goal:
- cover the non-ARM action routes that are likely to be hit by examples and docs-based probing

### Stage 6: Example and Validation

Add:
- `examples/azure/azureintegration-2023-11-14-preview/main.go`
- matching `Dockerfile`
- matching `docker-compose.yml`

Example flow should exercise:
- list operations
- create a space
- create an application
- create an application resource
- create a business process
- list/get a seeded business process version
- save/get/list/validate/delete a business-process development artifact
- patch an infrastructure resource fixture

Validation target:
- targeted provider tests
- example build
- service-specific Azure contract/io/doc/endpoint coverage runs

## Suggested In-Memory Model

Suggested stores:
- `spaces` keyed by ARM ID
- `applications` keyed by ARM ID
- `applicationResources` keyed by ARM ID
- `businessProcesses` keyed by ARM ID
- `businessProcessVersions` keyed by parent business-process ID plus version
- `infrastructureResources` keyed by ARM ID
- `artifacts` keyed by application ID plus artifact name

Shared helpers:
- ARM ID builders/parsers for each nesting level
- common ARM list response helper
- common `systemData` and `etag` fixture helpers
- common nested-resource delete cascading

## Testing Strategy

Add focused tests in:
- `internal/server/provider_azure_integration_environment_test.go`

Test buckets:
- operations discovery
- `spaces` CRUD/list/head
- nested `applications` CRUD/list/head
- nested `application resources` CRUD/list/head
- nested `business processes` CRUD/list/head
- seeded `business process versions` get/list
- `infrastructure resources` patch/get/list
- artifact action success paths
- validation failures:
  - invalid JSON
  - missing required path/body fields
  - missing parent resources
  - not-found get/patch/delete behavior

## Coverage Script Updates Required

Add `azureintegration` aliases to:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Map service -> doc plan in:
- `scripts/azure-doc-contract-coverage.py`

Recommended canonical coverage name:
- `azureintegration`

## Example Notes

Use the official Azure Go SDK package:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/integrationspaces/armintegrationspaces`

Recommended clients for the future example:
- `NewSpacesClient`
- `NewApplicationsClient`
- `NewApplicationResourcesClient`
- `NewBusinessProcessesClient`
- `NewBusinessProcessVersionsClient`
- `NewInfrastructureResourcesClient`

If any action route is not conveniently modeled by the generated client version pinned in the repo, use raw ARM requests only for those action methods while keeping the main example typed.

## Done Criteria

The Azure Integration Environment implementation should be considered complete when:

- the service no longer returns generic `NotImplemented` for the reviewed surface
- `spaces`, `applications`, `application resources`, `business processes`, and `business process versions` all return typed ARM payloads
- artifact actions return deterministic modeled success/error payloads
- the Azure Go SDK example builds and runs against Stackyard
- Azure contract/io/doc/endpoint coverage passes for `azureintegration`
