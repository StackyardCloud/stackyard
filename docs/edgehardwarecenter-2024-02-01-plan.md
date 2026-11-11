# Azure Edge Hardware Center (`edgehardwarecenter_2024_02_01`) Staged Plan

## Objective

Emulate Azure Edge Hardware Center Resource Manager REST API (`api-version=2024-02-01`) with deterministic local behavior for:

- address lifecycle and list surfaces
- order item lifecycle plus cancel/return actions
- order read/list surfaces
- product and configuration discovery/list surfaces
- provider operations discovery endpoint
- an Azure Go SDK example under `examples/azure/edgehardwarecenter-2024-02-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/operation-groups?view=rest-edgehardwarecenter-2024-02-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/operations?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/orders?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/products-and-configurations?view=rest-edgehardwarecenter-2024-02-01`

Method pages reviewed directly: all linked methods from all operation groups (`21` total).

Addresses methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses/create?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses/delete?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses/get?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses/list-by-resource-group?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses/list-by-subscription?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/addresses/update?view=rest-edgehardwarecenter-2024-02-01`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/operations/list?view=rest-edgehardwarecenter-2024-02-01`

Order Items methods (`8`):

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/cancel?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/create?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/delete?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/get?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/list-by-resource-group?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/list-by-subscription?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/return?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/order-items/update?view=rest-edgehardwarecenter-2024-02-01`

Orders methods (`3`):

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/orders/get?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/orders/list-by-resource-group?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/orders/list-by-subscription?view=rest-edgehardwarecenter-2024-02-01`

Products And Configurations methods (`3`):

- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/products-and-configurations/list-configurations?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/products-and-configurations/list-product-families?view=rest-edgehardwarecenter-2024-02-01`
- `https://learn.microsoft.com/en-us/rest/api/edgehardwarecenter/products-and-configurations/list-product-families-metadata?view=rest-edgehardwarecenter-2024-02-01`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/edgeorder/armedgeorder/v2`

## Surface Summary

Reviewed operation groups and methods:

- Addresses (6)
- Operations (1)
- Order Items (8)
- Orders (3)
- Products And Configurations (3)

Total reviewed methods: `21`.

## Service Naming

Canonical Stackyard naming:

- service key: `edgehardwarecenter_2024_02_01`
- display name: `Azure Edge Hardware Center`
- API version: `2024-02-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_edgehardwarecenter_2024_02_01.go`
- tests: `internal/server/provider_azure_edgehardwarecenter_2024_02_01_test.go`
- example: `examples/azure/edgehardwarecenter-2024-02-01`
- plan doc: `docs/edgehardwarecenter-2024-02-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.EdgeOrder/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/addresses`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/orderItems`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/orders`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orders`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/locations/{location}/orders/{orderName}`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/listConfigurations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/listProductFamilies`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/productFamiliesMetadata`

Representative routes:

- `GET /azure/providers/Microsoft.EdgeOrder/operations?api-version=2024-02-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}?api-version=2024-02-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}?api-version=2024-02-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}/cancel?api-version=2024-02-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}/return?api-version=2024-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/locations/{location}/orders/{orderName}?api-version=2024-02-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/listConfigurations?api-version=2024-02-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/listProductFamilies?api-version=2024-02-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/productFamiliesMetadata?api-version=2024-02-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2024-02-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods document non-success paths as `Other Status Codes | ErrorResponse`
- create/update/delete/action methods use LRO-style `202` on multiple endpoints

Observed response patterns:

- `200` appears in `19` methods
- `202` appears in `7` methods
- `204` appears in `3` methods

Request-body coverage from reviewed pages:

- `8` methods include request bodies
- body-bearing methods:
  - `Addresses_Create`
  - `Addresses_Update`
  - `OrderItems_Cancel`
  - `OrderItems_Create`
  - `OrderItems_Return`
  - `OrderItems_Update`
  - `ProductsAndConfigurations_ListConfigurations`
  - `ProductsAndConfigurations_ListProductFamilies`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add stage-0 provider skeleton at `internal/server/provider_azure_edgehardwarecenter_2024_02_01.go`
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key resolves as `edgehardwarecenter_2024_02_01`
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.EdgeOrder` route families for this API version
- enforce `api-version=2024-02-01` for handled routes
- preserve behavior for unrelated Azure providers

### Stage 2: Provider Ops + Read/List Surfaces

- implement `Operations_List`
- implement address read/list routes
- implement order item read/list routes
- implement order read/list routes
- implement product/configuration list routes:
  - `ListConfigurations`
  - `ListProductFamilies`
  - `ListProductFamiliesMetadata`

### Stage 3: Address + Order Item Mutations

- implement `Addresses_Create`, `Addresses_Update`, `Addresses_Delete`
- implement `OrderItems_Create`, `OrderItems_Update`, `OrderItems_Delete`
- implement `OrderItems_Cancel` and `OrderItems_Return`
- support deterministic `200`/`202`/`204` status behavior for lifecycle/action methods

### Stage 4: Payload and Query Validation

- validate request payload essentials for:
  - address contact/shipping details
  - order item order references and product hierarchy
  - cancellation/return reasons
  - product/configuration filter payloads
- parse optional paging/expansion parameters:
  - `$skipToken`
  - `$expand`

### Stage 5: Tests

- add provider tests `internal/server/provider_azure_edgehardwarecenter_2024_02_01_test.go` covering:
  - API-version validation and route ownership
  - address lifecycle/read/list flows
  - order item lifecycle + cancel/return actions
  - order read/list routes
  - product/configuration listing routes and paging parameters
  - deterministic error envelopes

### Stage 6: Example and Validation

- add `examples/azure/edgehardwarecenter-2024-02-01` using Azure Go SDK core pipeline
- include one call per reviewed method (`21` calls)
- verify build and coverage script discoverability

Exit criteria:

- example compiles and is ready for end-to-end runs once handlers are implemented
- coverage scripts include `edgehardwarecenter_2024_02_01` aliases and plan-doc mapping

