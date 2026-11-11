# Azure Sphere (`azuresphere-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Sphere ARM management API for `2024-04-01` with deterministic local behavior for:

- provider `operations`
- top-level `catalogs` lifecycle and discovery routes
- nested `products`
- nested `deviceGroups`
- nested `devices`
- nested `deployments`
- catalog-scoped `images`
- catalog- and parent-scoped list/count/action routes such as:
  - `listDevices`
  - `listDeviceGroups`
  - `listDeployments`
  - `listDeviceInsights`
  - `countDevices`
  - `claimDevices`
  - `generateDefaultDeviceGroups`
  - `uploadImage`
- an Azure Go SDK example under `examples/azure/azuresphere-2024-04-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure Sphere under `Microsoft.AzureSphere`.
- This is not an IoT runtime/data plane and not a device-side OS/update transport protocol.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2024-04-01`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/operation-groups?view=rest-azuresphere-2024-04-01`

Operation-group and representative method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/operations/list?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/catalogs?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/catalogs/create-or-update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/catalogs/list-device-groups?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/catalogs/list-deployments?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/catalogs/list-device-insights?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/products/create-or-update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/products/list-by-catalog?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/products/count-devices?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/products/generate-default-device-groups?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/device-groups/create-or-update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/device-groups/get?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/device-groups/count-devices?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/device-groups/claim-devices?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/devices/create-or-update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/devices/update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/devices/list-by-device-group?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/devices/delete?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/deployments/create-or-update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/deployments/get?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/deployments/delete?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/images/create-or-update?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/images/get?view=rest-azuresphere-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azuresphere/images/delete?view=rest-azuresphere-2024-04-01`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sphere/armsphere`

SDK selection note:
- The official `armsphere` package exists and the Microsoft Learn Go samples on the reviewed pages use it directly.
- The package is suitable for the reviewed ARM management surface, including provider operations and the main Azure Sphere resource families.
- Plan assumption: prefer typed `armsphere` clients for the example and only fall back to raw ARM requests if a reviewed route is missing from the generated client set chosen at implementation time.

## Service Naming

Canonical Stackyard naming:
- service key: `azuresphere`
- display name: `Azure Sphere`
- provider namespace: `Microsoft.AzureSphere`
- API version: `2024-04-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_sphere.go`
- tests: `internal/server/provider_azure_sphere_test.go`
- example: `examples/azure/azuresphere-2024-04-01`
- plan doc: `docs/azuresphere-2024-04-01-plan.md`

Recommended coverage aliases:
- `azuresphere`
- `azure_sphere`
- `azure-sphere`
- `sphere`
- `microsoft_azuresphere`
- `rest_azuresphere_2024_04_01`
- `azuresphere_2024_04_01`

## Surface Summary

The reviewed `2024-04-01` management surface is strongly hierarchical:

1. `Operations`
2. `Catalogs`
3. `Products`
4. `Device Groups`
5. `Devices`
6. `Deployments`
7. `Images`

Several catalog- or parent-scoped action/list routes complement the tracked-resource hierarchy:

- `catalogs/listDevices`
- `catalogs/listDeviceGroups`
- `catalogs/listDeployments`
- `catalogs/listDeviceInsights`
- `catalogs/uploadImage`
- `products/countDevices`
- `products/generateDefaultDeviceGroups`
- `deviceGroups/countDevices`
- `deviceGroups/claimDevices`

This is broad enough that implementation should be staged by nesting depth and operator value:

- provider and catalog roots first
- product/device-group/device hierarchy second
- deployments and images third
- catalog-level report/list actions after the core resource lifecycle is stable

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2024-04-01`
- all routes sit under the `Microsoft.AzureSphere` provider namespace

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.AzureSphere/operations?api-version=2024-04-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}?api-version=2024-04-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/products/{productName}?api-version=2024-04-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/products/{productName}/deviceGroups/{deviceGroupName}?api-version=2024-04-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/products/{productName}/deviceGroups/{deviceGroupName}/devices/{deviceName}?api-version=2024-04-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/products/{productName}/deviceGroups/{deviceGroupName}/deployments/{deploymentName}?api-version=2024-04-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/images/{imageName}?api-version=2024-04-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/listDeviceInsights?api-version=2024-04-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/products/{productName}/generateDefaultDeviceGroups?api-version=2024-04-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureSphere/catalogs/{catalogName}/products/{productName}/deviceGroups/{deviceGroupName}/claimDevices?api-version=2024-04-01`

Important routing implications from the reviewed pages:

- nested parent existence checks matter for almost every route
- some names permit Azure Sphere system values such as:
  - `.default`
  - `.unassigned`
- several list/report routes are action-style `POST`s, not plain `GET` collection listings
- deletes for some resources explicitly allow `200`, `202`, or `204`
- image upload is represented both as normal image create and as a catalog-level action route, so implementation should keep one canonical image store behind both entry points

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- tracked resources use standard ARM fields:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.provisioningState`
- list routes return `value` and sometimes `nextLink`

### Operations

`Operations.List`
- provider route under `/providers/Microsoft.AzureSphere/operations`
- returns provider operation metadata

Modeling implication:
- expose a deterministic `value` list that includes at least the implemented catalog/product/device-group/device/deployment/image operations and the major action routes

### Catalogs

Reviewed pages:
- `Create Or Update`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`
- `List Devices`
- `List Device Groups`
- `List Deployments`
- `List Device Insights`
- `Upload Image`

Key traits from the reviewed pages:
- `PUT` requires `location`
- catalog response includes:
  - `properties.provisioningState`
  - `properties.tenantId`
- catalog-level action routes return report-like `value` payloads rather than tracked resources

Modeling implications:
- persist one canonical catalog store keyed by ARM ID
- implement normal ARM CRUD/list routes first
- back catalog-level report routes from the same nested product/device-group/device/deployment stores
- synthesize a stable Azure Sphere `tenantId`

### Products

Reviewed pages:
- `Create Or Update`
- `Get`
- `List By Catalog`
- `Delete`
- `Update`
- `Count Devices`
- `Generate Default Device Groups`

Key traits from the reviewed pages:
- product names have a distinct pattern and can include the reserved system names on some routes
- `generateDefaultDeviceGroups` is an action-style `POST`
- `countDevices` returns a small scalar response wrapper

Modeling implications:
- persist products as tracked child resources under catalogs
- model the default/unassigned device-group generation as actual child-resource creation in the canonical store
- keep count routes derived from real child state, not static numbers

### Device Groups

Reviewed pages:
- `Create Or Update`
- `Get`
- `Delete`
- `List By Product`
- `Update`
- `Count Devices`
- `Claim Devices`

Key traits from the reviewed pages:
- response includes:
  - `properties.allowCrashDumpsCollection`
  - `properties.hasDeployment`
  - `properties.osFeedType`
  - `properties.regionalDataBoundary`
- `claimDevices` is a bulk action

Modeling implications:
- persist device groups as tracked resources under products
- derive `hasDeployment` from whether a deployment exists in the canonical child store
- for `claimDevices`, support a deterministic bulk assignment flow that either creates placeholder devices or moves known devices into the requested group

### Devices

Reviewed pages:
- `Create Or Update`
- `Get`
- `List By Device Group`
- `Update`
- `Delete`

Key traits from the reviewed pages:
- device name pattern is long and strict
- update semantics include moving a device by targeting a different product/device-group path
- catalog-level claiming uses `.default` and `.unassigned` names

Modeling implications:
- persist devices canonically with parent references that can be updated
- keep patch/update behavior centered on parent reassignment and simple metadata updates

### Deployments

Reviewed pages:
- `Create Or Update`
- `Get`
- `Delete`
- `List By Device Group`

Key traits from the reviewed pages:
- deployment request body includes:
  - `properties.deployedImages`
  - `properties.deploymentId`
- delete may return `200`, `202`, or `204`

Modeling implications:
- persist deployments as tracked resources under device groups
- validate referenced images shallowly by ID when possible
- return deterministic long-running-friendly delete responses if needed by SDK examples

### Images

Reviewed pages:
- `Create Or Update`
- `Get`
- `Delete`
- `List By Catalog`

Key traits from the reviewed pages:
- create accepts `properties.image` as base64-encoded content
- reads expose image metadata such as:
  - `properties.imageId`
  - `properties.imageName`
  - `properties.imageType`
  - `properties.uri`
  - `properties.regionalDataBoundary`
- delete may return `200`, `202`, or `204`

Modeling implications:
- store images canonically under catalogs
- normalize uploaded content into deterministic metadata instead of preserving full binary payloads in responses
- use one shared image store for both regular image CRUD and catalog upload-image action routes

## Recommended Implementation Stages

### Stage 0: Router and Coverage Wiring

Add the service skeleton before filling in resources:

- add a new provider file:
  - `internal/server/provider_azure_sphere.go`
- route `Microsoft.AzureSphere` traffic in:
  - `internal/server/http.go`
- add service state maps to `Server`
- update Azure coverage aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- reserve service key:
  - `azuresphere`

Deliverable:
- Azure Sphere routes are intercepted by a dedicated provider instead of falling through to foundation success

### Stage 1: Provider and Catalog Foundation

Implement:

- `GET /providers/Microsoft.AzureSphere/operations`
- catalog `PUT/GET/HEAD/PATCH/DELETE`
- catalog list-by-resource-group and list-by-subscription

Data model:
- one canonical catalog store keyed by ARM ID

Tests:
- operations payload
- catalog CRUD/list
- invalid catalog name and missing location validation

### Stage 2: Products and Device Groups

Implement:

- product `PUT/GET/HEAD/PATCH/DELETE`
- product list-by-catalog
- product `countDevices`
- product `generateDefaultDeviceGroups`
- device-group `PUT/GET/HEAD/PATCH/DELETE`
- device-group list-by-product
- device-group `countDevices`

Data model:
- one product store keyed by ARM ID
- one device-group store keyed by ARM ID

Tests:
- parent existence enforcement
- generated `.default` and `.unassigned` groups
- count routes driven by real nested state

### Stage 3: Devices

Implement:

- device `PUT/GET/HEAD/PATCH/DELETE`
- device list-by-device-group
- device-group `claimDevices`
- catalog `listDevices`

Data model:
- one device store keyed by ARM ID plus catalog/product/group references

Tests:
- direct device lifecycle
- bulk claim flow
- movement across groups using update semantics

### Stage 4: Deployments and Images

Implement:

- deployment `PUT/GET/HEAD/DELETE`
- deployment list-by-device-group
- catalog `listDeployments`
- image `PUT/GET/HEAD/DELETE`
- image list-by-catalog
- catalog `uploadImage`

Data model:
- one deployment store keyed by ARM ID
- one image store keyed by ARM ID

Tests:
- deployment/image lifecycle
- deployment references to images
- upload-image action and normal image create share the same backing state

### Stage 5: Catalog Reports and Insights

Implement:

- catalog `listDeviceGroups`
- catalog `listDeviceInsights`

Recommended modeling:
- `listDeviceGroups` should aggregate the real canonical child store
- `listDeviceInsights` can return deterministic synthetic events derived from known devices rather than trying to model real telemetry

### Stage 6: Example and Coverage Completion

Add the example:

- `examples/azure/azuresphere-2024-04-01`

Recommended example flow:

- create a catalog
- create a product
- generate default device groups
- create or claim a device
- create an image
- create a deployment that references that image
- call one count or list action route

Update coverage scripts:

- add service aliases listed above
- map the service to this plan doc in doc coverage
- ensure endpoint coverage points the service name at `Microsoft.AzureSphere`

## Example Plan

Recommended future example files:

- `examples/azure/azuresphere-2024-04-01/main.go`
- `examples/azure/azuresphere-2024-04-01/Dockerfile`
- `examples/azure/azuresphere-2024-04-01/docker-compose.yml`

Recommended SDK usage:

- use `armsphere`
- prefer typed clients such as:
  - `NewOperationsClient`
  - `NewCatalogsClient`
  - `NewProductsClient`
  - `NewDeviceGroupsClient`
  - `NewDevicesClient`
  - `NewDeploymentsClient`
  - `NewImagesClient`

Example should demonstrate:

- ARM endpoint override to Stackyard
- one nested resource creation chain
- one action/list route outside simple CRUD

Avoid in the first example:

- full coverage of every report/list action
- deep paging behavior
- binary image realism beyond a simple deterministic metadata flow

## Coverage Script Wiring

During implementation, update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended mappings:

- canonical service: `azuresphere`
- aliases:
  - `azure_sphere`
  - `azure-sphere`
  - `sphere`

Documentation mapping:

- `azuresphere` -> `azuresphere-2024-04-01-plan.md`

## Validation Plan

When implementation starts, validate in this order:

1. `go test ./internal/server -run 'TestAzureSphere' -count=1`
2. `go build ./examples/azure/azuresphere-2024-04-01`
3. `python3 scripts/azure-contract-coverage.py --service azuresphere --require-service azuresphere --fail-on any`
4. `python3 scripts/azure-io-contract-coverage.py --service azuresphere --require-service azuresphere --fail-on any`
5. `python3 scripts/azure-doc-contract-coverage.py --service azuresphere --require-service azuresphere --fail-on any`
6. `python3 scripts/azure-endpoint-coverage.py --service azuresphere --require-service azuresphere --fail-on any`

Success criteria:

- every Azure Sphere endpoint covered by the selected stage returns success or the expected modeled validation error
- no reviewed endpoint falls through to `NotImplemented`
- the example builds cleanly against the chosen SDK package

## Risks and Open Questions

- Some Microsoft Learn pages in this surface partially hide full examples or definitions behind authorization, but the route signatures, parameters, status codes, and enough schema detail remain visible to plan implementation correctly.
- The nested hierarchy is strict, so the provider should centralize parent-ID parsing and existence checks early to avoid repeated route bugs.
- The service mixes tracked resources with parent-scoped `POST` list/report routes, so implementation should not assume every collection uses standard ARM `GET list` semantics.
