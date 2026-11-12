# Azure AD B2C (`activedirectory-2021-04-01`) Staged Plan

## Objective

Emulate Azure AD B2C (`2021-04-01`) with deterministic local behavior for the ARM management-plane surface under `Microsoft.AzureActiveDirectory`, including:

- provider `Operations`
- `B2C Tenants`
- `Guest Usages`
- a typed Azure Go SDK example using the Azure Active Directory External Identities SDK
- Azure coverage script wiring for the new service key

Although the user request described this as "Azure AD B2C", the Microsoft Learn REST surface at `rest-activedirectory-2021-04-01` is the Azure AD B2C / External Identities ARM provider under `Microsoft.AzureActiveDirectory`, so the implementation plan is scoped to that actual API surface.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/operation-groups?view=rest-activedirectory-2021-04-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/operations?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages?view=rest-activedirectory-2021-04-01`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/operations/list?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/check-name-availability?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/create?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/delete?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/get?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/list-by-resource-group?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/list-by-subscription?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/b2c-tenants/update?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages/create?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages/delete?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages/get?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages/list-by-resource-group?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages/list-by-subscription?view=rest-activedirectory-2021-04-01`
- `https://learn.microsoft.com/en-us/rest/api/activedirectory/guest-usages/update?view=rest-activedirectory-2021-04-01`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/azureadexternalidentities/armazureadexternalidentities`

SDK package note:
- the older track-1 package page for `github.com/AZURE/azure-sdk-for-go/services/azureadexternalidentities/mgmt/2021-04-01/azureadexternalidentities` explicitly marks itself deprecated and points to the track-2 replacement above, so the example should use the newer `armazureadexternalidentities` package.

## Service Naming

Canonical Stackyard naming:
- service key: `activedirectory`
- display name: `Azure AD B2C`
- API version: `2021-04-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_activedirectory.go`
- tests: `internal/server/provider_azure_activedirectory_test.go`
- example: `examples/azure/activedirectory-2021-04-01`
- plan doc: `docs/activedirectory-2021-04-01-plan.md`

## Surface Summary

The reviewed surface is compact and split into 3 operation groups:

1. `Operations`
2. `B2C Tenants`
3. `Guest Usages`

Total linked method pages reviewed: `14`

Method inventory:

`Operations`
- `List`

`B2C Tenants`
- `Check Name Availability`
- `Create`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`

`Guest Usages`
- `Create`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.AzureActiveDirectory/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AzureActiveDirectory/checkNameAvailability`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/b2cDirectories`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/b2cDirectories/{resourceName}`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AzureActiveDirectory/b2cDirectories`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/guestUsages`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/guestUsages/{resourceName}`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.AzureActiveDirectory/guestUsages`

Target query:
- `api-version=2021-04-01`

Important routing implications:
- the provider mixes a subscription-level action route (`checkNameAvailability`) with normal ARM resource CRUD routes
- `B2C Tenants` and `Guest Usages` use the same resource naming convention:
  - `resourceName` is the initial tenant domain name such as `contoso.onmicrosoft.com`

## Contract Notes From Reviewed Pages

Shared management-plane traits:
- all reviewed methods use Azure ARM OAuth2 routes
- `api-version` is required on every reviewed method page
- non-success responses use `Cloud Error`
- list routes return `value` envelopes
- `systemData` appears on resource models for both `B2C Tenants` and `Guest Usages`

### Operations

`Operations - List`
- `GET /providers/Microsoft.AzureActiveDirectory/operations`
- response: `Available Operations`
- sample operation names from the reviewed page include:
  - `Microsoft.AzureActiveDirectory/b2cDirectories/write`
  - `Microsoft.AzureActiveDirectory/b2cDirectories/read`
  - `Microsoft.AzureActiveDirectory/b2cDirectories/delete`
  - `Microsoft.AzureActiveDirectory/b2ctenants/read`
  - `Microsoft.AzureActiveDirectory/guestUsages/write`
  - `Microsoft.AzureActiveDirectory/guestUsages/read`
  - `Microsoft.AzureActiveDirectory/guestUsages/delete`

Modeling implication:
- this is a fixed deterministic list endpoint and should be implemented first to make coverage and examples discover the provider cleanly.

### B2C Tenants

`B2C Tenants - Check Name Availability`
- `POST /subscriptions/{subscriptionId}/providers/Microsoft.AzureActiveDirectory/checkNameAvailability`
- reviewed request fields:
  - `name`
  - `countryCode`
- reviewed response fields:
  - `nameAvailable`
  - `message`
  - `reason`

Modeling implication:
- deterministic name-validation is enough initially:
  - invalid or duplicate names should return a stable `nameAvailable=false`
  - otherwise return `nameAvailable=true`

`B2C Tenants - Create`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/b2cDirectories/{resourceName}`
- reviewed description:
  - initiates an async request to create both the B2C tenant and the linked Azure resource
- reviewed required request fields:
  - `location`
  - `sku`
- reviewed request properties:
  - `properties.createTenantProperties.countryCode`
  - `properties.createTenantProperties.displayName`
  - `tags`
- reviewed responses:
  - success returns `B2CTenant Resource`
  - failures return `Cloud Error`

`B2C Tenants - Get`
- `GET .../b2cDirectories/{resourceName}`
- response: `B2CTenant Resource`

`B2C Tenants - List By Resource Group`
- `GET .../resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/b2cDirectories`
- response: `B2CTenant Resource List`

`B2C Tenants - List By Subscription`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.AzureActiveDirectory/b2cDirectories`
- response: `B2CTenant Resource List`

`B2C Tenants - Update`
- `PATCH .../b2cDirectories/{resourceName}`
- reviewed mutable request fields:
  - `sku`
  - `properties.billingConfig`
  - `properties.tenantId`
  - `tags`
- reviewed sample update body shows:
  - `properties.billingConfig.billingType`
  - `sku.name`

`B2C Tenants - Delete`
- `DELETE .../b2cDirectories/{resourceName}`
- reviewed responses:
  - `200 OK`
  - `202 Accepted`
  - `204 No Content`
- the reviewed page explicitly documents `Location` and `Retry-After` headers for `202 Accepted`

Important modeled `B2CTenant Resource` fields from the reviewed pages:
- `id`, `name`, `type`, `location`, `tags`
- `properties.tenantId`
- `properties.billingConfig`
- `sku`
- `systemData`

Modeling implications:
- the emulator can store a canonical `B2CTenant Resource` keyed by full ARM ID
- create/delete can still behave synchronously in Stackyard while returning stable success payloads
- for compatibility with the reviewed delete page, a lightweight deterministic async facade is useful:
  - accept immediate `200/204` for normal paths
  - optionally support a stable `202` + `Location` poll route later if coverage/examples need it

### Guest Usages

`Guest Usages - Create`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/guestUsages/{resourceName}`
- reviewed request fields:
  - `location`
  - `properties.tenantId`
  - `tags`
- response: `Guest Usages Resource`

`Guest Usages - Get`
- `GET .../guestUsages/{resourceName}`
- response: `Guest Usages Resource`

`Guest Usages - List By Resource Group`
- `GET .../resourceGroups/{resourceGroupName}/providers/Microsoft.AzureActiveDirectory/guestUsages`
- response: `Guest Usages Resource List`

`Guest Usages - List By Subscription`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.AzureActiveDirectory/guestUsages`
- response: `Guest Usages Resource List`

`Guest Usages - Update`
- `PATCH .../guestUsages/{resourceName}`
- reviewed mutable request field:
  - `tags`

`Guest Usages - Delete`
- `DELETE .../guestUsages/{resourceName}`
- reviewed responses:
  - `200 OK`
  - `204 No Content`

Important modeled `Guest Usages Resource` fields from the reviewed pages:
- `id`, `name`, `type`, `location`, `tags`
- `properties.tenantId`
- `systemData`

Modeling implication:
- this resource family is a straightforward ARM CRUD slice and can be implemented with ordinary in-memory resource storage.

## Suggested In-Memory Model

Add Azure Active Directory state to `internal/server/http.go` `Server` storage:

- `azureActiveDirectoryB2CTenants map[string]map[string]any`
- `azureActiveDirectoryGuestUsages map[string]map[string]any`

Recommended helper patterns:
- canonical ID builder for:
  - B2C tenant IDs
  - guest usage IDs
- query validation helper for `api-version`
- shared `Cloud Error` response helper
- helper for deterministic `systemData`
- helper for `checkNameAvailability` validation

Recommended seed defaults:
- one stable `operations` list fixture
- generated `tenantId` values when create request omits or cannot supply one
- deterministic default `billingConfig.billingType`, such as `MAU`

## Implementation Stages

### Stage 0: Routing And Coverage Wiring

Deliverables:
- add Azure router hook for `Microsoft.AzureActiveDirectory`
- reserve service key `activedirectory`
- add coverage aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map the service plan doc in doc coverage

Acceptance target:
- coverage scripts discover the new service key
- doc coverage resolves the plan file

### Stage 1: Provider Operations And Name Availability

Deliverables:
- implement:
  - `Operations - List`
  - `B2C Tenants - Check Name Availability`
- add focused tests for:
  - required `api-version`
  - deterministic operation list shape
  - duplicate-name and invalid-name checks

Acceptance target:
- read-only discovery endpoints are green
- examples can probe the provider before resource creation

### Stage 2: B2C Tenants Core Lifecycle

Deliverables:
- implement:
  - `Create`
  - `Get`
  - `List By Resource Group`
  - `List By Subscription`
  - `Update`
  - `Delete`
- validate required create payload:
  - `location`
  - `sku`
  - `properties.createTenantProperties.countryCode`
  - `properties.createTenantProperties.displayName`
- persist:
  - `tenantId`
  - `billingConfig`
  - `sku`
  - `tags`
  - `systemData`

Acceptance target:
- full B2C tenant CRUD coverage passes
- no B2C tenant routes return `NotImplemented`

### Stage 3: Guest Usages Lifecycle

Deliverables:
- implement:
  - `Create`
  - `Get`
  - `List By Resource Group`
  - `List By Subscription`
  - `Update`
  - `Delete`
- validate create payload:
  - `location`
  - `properties.tenantId`

Acceptance target:
- full guest-usage coverage passes
- no guest-usage routes return `NotImplemented`

### Stage 4: Azure Go SDK Example

Deliverables:
- add `examples/azure/activedirectory-2021-04-01`
- use `armazureadexternalidentities`
- exercise:
  - operations list
  - check name availability
  - B2C tenant create/get/update/list/delete
  - guest usage create/get/update/list/delete
- follow existing repo example pattern:
  - shared Azure shim transport
  - small end-to-end lifecycle

Acceptance target:
- `go build ./examples/azure/activedirectory-2021-04-01`
- example is suitable for future smoke coverage

### Stage 5: Validation, CI, And Coverage Closure

Deliverables:
- add focused server tests for success, validation, and not-found flows
- run:
  - `go test ./internal/server -run 'TestAzureActiveDirectory' -count=1`
  - `python3 scripts/azure-contract-coverage.py --service activedirectory --require-service activedirectory --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service activedirectory --require-service activedirectory --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service activedirectory --require-service activedirectory --fail-on any`
  - `python3 scripts/azure-endpoint-coverage.py --service activedirectory --require-service activedirectory --fail-on any`

Acceptance target:
- all new service endpoints are covered successfully
- no `NotImplemented` responses remain for `activedirectory`

## Coverage Script Updates To Plan For

Add service aliases for:
- `activedirectory`
- `active_directory`
- `activedirectory_2021_04_01`
- `rest_activedirectory_2021_04_01`
- `azure_activedirectory`
- `azure_active_directory`
- `azure_ad_b2c`
- `azureadexternalidentities`
- `microsoft_azureactivedirectory`

Doc coverage mapping:
- `activedirectory -> activedirectory-2021-04-01-plan.md`

## Testing Strategy

Server tests should cover:
- provider operations list shape
- name availability success and duplicate-name rejection
- B2C tenant create/get/list/update/delete
- guest usage create/get/list/update/delete
- `404` for missing resources
- `400` for missing `api-version`
- `400` for malformed create/update payloads
- deterministic `systemData` and list envelope shape

SDK example validation target:
- buildable example with typed SDK clients and repo-standard Azure shim transport

## Risks And Decisions

### Async B2C Tenant Create/Delete

The reviewed create/delete pages describe asynchronous behavior, especially delete with `202`, `Location`, and `Retry-After`.

Decision for initial implementation:
- keep first implementation synchronous and deterministic unless coverage/examples require polling semantics
- preserve a clean extension point for a later lightweight async facade

This keeps the first slice smaller while still matching the modeled resource shape.

### Track-2 SDK Surface

The reviewed Learn pages are for `2021-04-01`, but the currently recommended Go SDK package is `armazureadexternalidentities`.

Decision:
- prefer the track-2 package for the example
- if generated client naming differs from the method-group names in Learn, keep the example focused on the same REST semantics rather than mirroring doc titles exactly

## Definition Of Done

The Azure AD B2C service emulation is complete when:

- `Microsoft.AzureActiveDirectory` routes are implemented under `/azure`
- `Operations`, `B2C Tenants`, and `Guest Usages` all return deterministic modeled responses
- the Azure Go SDK example builds successfully
- Azure contract, IO, doc, and aggregate endpoint coverage all pass for `activedirectory`
- no endpoint for the new service returns `NotImplemented`
