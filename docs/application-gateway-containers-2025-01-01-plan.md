# Azure Application Gateway for Containers (`application-gateway-containers-2025-01-01`) Staged Plan

## Objective

Emulate Azure Application Gateway for Containers (`2025-01-01`) with deterministic local behavior for the ARM management-plane resource lifecycle under `Microsoft.ServiceNetworking`, including:

- provider `operations`
- top-level `trafficControllers`
- nested `frontends`
- nested `associations`
- nested `securityPolicies`
- a typed Azure Go SDK example using the Azure Service Networking SDK
- Azure coverage script wiring for the new service key

This service is a better fit for a staged implementation than classic Application Gateway because the surface is compact, hierarchical, and centered on one parent resource with three nested child collections.

## Primary References Reviewed

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/operation-groups?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/frontends-interface?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/associations-interface?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/security-policies-interface?view=rest-applicationgatewaycontainers-2025-01-01`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/operations/list?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface/create-or-update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface/delete?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface/get?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface/list-by-resource-group?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface/list-by-subscription?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/traffic-controller-interface/update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/frontends-interface/create-or-update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/frontends-interface/delete?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/frontends-interface/get?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/frontends-interface/list-by-traffic-controller?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/frontends-interface/update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/associations-interface/create-or-update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/associations-interface/delete?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/associations-interface/get?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/associations-interface/list-by-traffic-controller?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/associations-interface/update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/security-policies-interface/create-or-update?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/security-policies-interface/delete?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/security-policies-interface/get?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/security-policies-interface/list-by-traffic-controller?view=rest-applicationgatewaycontainers-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/applicationgatewaycontainers/security-policies-interface/update?view=rest-applicationgatewaycontainers-2025-01-01`

## Service Naming

Canonical Stackyard naming:
- service key: `application_gateway_containers`
- display name: `Azure Application Gateway for Containers`
- API version: `2025-01-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_application_gateway_containers.go`
- tests: `internal/server/provider_azure_application_gateway_containers_test.go`
- example: `examples/azure/application-gateway-containers-2025-01-01`
- plan doc: `docs/application-gateway-containers-2025-01-01-plan.md`

## Surface Summary

The reviewed REST surface is split into 5 operation groups:

1. `Operations`
2. `Traffic Controller Interface`
3. `Frontends Interface`
4. `Associations Interface`
5. `Security Policies Interface`

Total linked method pages reviewed: `22`

Method inventory:

`Operations`
- `List`

`Traffic Controller Interface`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`

`Frontends Interface`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Traffic Controller`
- `Update`

`Associations Interface`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Traffic Controller`
- `Update`

`Security Policies Interface`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Traffic Controller`
- `Update`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.ServiceNetworking/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ServiceNetworking/trafficControllers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}/frontends`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}/frontends/{frontendName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}/associations`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}/associations/{associationName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}/securityPolicies`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ServiceNetworking/trafficControllers/{trafficControllerName}/securityPolicies/{securityPolicyName}`

Target query:
- `api-version=2025-01-01`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:
- All routes are ARM OAuth2 routes in Learn; local Stackyard examples should continue using the repo’s Azure shared-key shim.
- All methods use `ErrorResponse` for unexpected failures.
- All names use the same ARM-style name regex:
  - `^[A-Za-z0-9]([A-Za-z0-9-_.]{0,62}[A-Za-z0-9])?$`
- Resource group names are required, min length `1`, max length `90`.

Create/update behavior from the reviewed pages:
- `PUT` create/update routes return `200` or `201`.
- `201` responses include `Retry-After`.
- `PATCH` update routes return `200`.
- Delete routes return:
  - `202 Accepted` with `Location` and `Retry-After`
  - or `204 No Content` when the resource does not exist

List behavior from the reviewed pages:
- list methods return typed `*ListResult` envelopes with `value`
- list result models can include `nextLink`

### Traffic Controller Shape

`TrafficController` request/response details from the reviewed pages:

Create:
- required: `location`
- optional: `tags`
- optional: `properties.securityPolicyConfigurations`

Update:
- optional: `tags`
- optional: `properties.securityPolicyConfigurations`

Get/list response fields:
- `properties.configurationEndpoints`
- `properties.frontends`
- `properties.associations`
- `properties.securityPolicies`
- `properties.securityPolicyConfigurations`
- `properties.provisioningState`

Important modeling implication:
- `TrafficController` is the parent resource and should maintain references to all nested child resource IDs.

### Frontend Shape

`Frontend` request/response details from the reviewed pages:

Create:
- required: `location`
- optional: `tags`
- sample request uses empty `properties`

Update:
- patch body is tags-only

Get/list response fields:
- `properties.fqdn`
- `properties.provisioningState`

Important modeling implication:
- `Frontend` is a very light child resource; deterministic `fqdn` generation is sufficient for the first implementation slice.

### Association Shape

`Association` request/response details from the reviewed pages:

Create:
- required: `location`
- required: `properties.associationType`
- optional/expected: `properties.subnet.id`
- optional: `tags`

Update:
- patch body may include `properties.associationType`
- patch body may include `properties.subnet.id`
- optional: `tags`

Get/list response fields:
- `properties.associationType`
- `properties.subnet.id`
- `properties.provisioningState`

Important modeling implication:
- the useful deterministic interpretation for Stackyard is `associationType=subnets` with a stored subnet ARM ID string.

### Security Policy Shape

`SecurityPolicy` request/response details from the reviewed pages:

Create:
- required: `location`
- expected: `properties.wafPolicy.id`
- optional: `tags`

Update:
- patch body may include `properties.wafPolicy.id`
- optional: `tags`

Get/list response fields:
- `properties.policyType`
- `properties.wafPolicy.id`
- `properties.provisioningState`

Important modeling implication:
- for the first slice, only `waf` policy type needs to be modeled.

### Provider Operations Shape

`Operations - List` exposes:
- `name`
- `isDataAction`
- `display.provider`
- `display.resource`
- `display.operation`
- `display.description`
- `origin`

Important modeling implication:
- the provider operations fixture should enumerate the CRUD/read actions for:
  - `trafficControllers`
  - `frontends`
  - `associations`
  - `securityPolicies`

## Azure Go SDK Example Plan

The official Go SDK package for this surface is:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicenetworking/armservicenetworking`

The reviewed package exposes the exact clients needed:
- `OperationsClient`
- `TrafficControllerInterfaceClient`
- `FrontendsInterfaceClient`
- `AssociationsInterfaceClient`
- `SecurityPoliciesInterfaceClient`
- `ClientFactory`

Recommended example path:
- `examples/azure/application-gateway-containers-2025-01-01`

Recommended example coverage:
- provider operations list
- traffic controller create/get/list-by-rg/list-by-subscription/update/delete
- frontend create/get/list/update/delete
- association create/get/list/update/delete
- security policy create/get/list/update/delete

Follow the existing Azure example pattern:
- `azsdkshim.StaticTokenCredential{}`
- `arm.ClientOptions`
- `cloud.ResourceManager` endpoint override
- `azsdkshim.NewTransport("/azure", account, subscriptionKey)`

## Coverage Script Plan

Add aliases to:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:
- `application_gateway_containers`
- `applicationgatewaycontainers`
- `application_gateway_containers_2025_01_01`
- `applicationgatewaycontainers2025_01_01`
- `application_gateway_for_containers`
- `application_gateway_for_containers_2025_01_01`
- `azure_application_gateway_containers`
- `azure_application_gateway_for_containers`
- `microsoft_servicenetworking_trafficcontrollers`
- `rest_applicationgatewaycontainers_2025_01_01`

Doc coverage mapping:
- `application_gateway_containers -> application-gateway-containers-2025-01-01-plan.md`

## Staged Implementation Plan

## Stage 0: Service Skeleton And Naming

- Add the new plan doc.
- Add the canonical service alias set to the three Azure coverage scripts.
- Introduce a new Azure router entry point for `Microsoft.ServiceNetworking`.
- Reserve the canonical implementation/test file names.

Exit criteria:
- Coverage discovery can resolve `application_gateway_containers`.
- Router recognition is deterministic for all parent and child route prefixes.

## Stage 1: Provider Operations And Validation Foundation

- Implement `/azure/providers/Microsoft.ServiceNetworking/operations`.
- Add common ARM validation behavior:
  - invalid or missing `api-version`
  - malformed JSON
  - required path segment validation
  - method gating by route
- Return deterministic Azure-style `InvalidRequest` / `NotFound` payloads.

Exit criteria:
- Contract and IO coverage scripts see request-validation and typed success signals.
- No claimed route falls through to generic `NotImplemented`.

## Stage 2: Traffic Controller Core Resource

- Implement deterministic stateful CRUD for `trafficControllers`.
- Implement:
  - `Create Or Update`
  - `Get`
  - `List By Resource Group`
  - `List By Subscription`
  - `Update`
  - `Delete`
- Persist:
  - `subscriptionId`
  - `resourceGroup`
  - `name`
  - `location`
  - `tags`
  - `properties.configurationEndpoints`
  - `properties.securityPolicyConfigurations`
  - reference lists for `frontends`, `associations`, `securityPolicies`
  - `properties.provisioningState`

Recommended first-pass behavior:
- create returns `201` on first write, `200` on overwrite
- delete returns `202` with `Location` and `Retry-After`, while the actual state transition can remain immediate internally
- missing resource delete returns `204`

Exit criteria:
- Parent traffic controller lifecycle is fully typed and stateful.
- ARM list/get routes return modeled `TrafficController` / `TrafficControllerListResult` payloads.

## Stage 3: Frontends Child Resource

- Implement typed child CRUD/list under a traffic controller:
  - `Create Or Update`
  - `Get`
  - `List By Traffic Controller`
  - `Update`
  - `Delete`
- Persist:
  - `location`
  - `tags`
  - deterministic `properties.fqdn`
  - `properties.provisioningState`
- Update parent `trafficController.properties.frontends` references on create/delete.

Exit criteria:
- Every frontend route returns typed `Frontend` or `FrontendListResult`.
- Parent references stay in sync.

## Stage 4: Associations Child Resource

- Implement typed child CRUD/list under a traffic controller:
  - `Create Or Update`
  - `Get`
  - `List By Traffic Controller`
  - `Update`
  - `Delete`
- Persist:
  - `location`
  - `tags`
  - `properties.associationType`
  - `properties.subnet.id`
  - `properties.provisioningState`
- Update parent `trafficController.properties.associations` references on create/delete.

Exit criteria:
- Association routes return typed `Association` / `AssociationListResult`.
- Parent references stay in sync.

## Stage 5: Security Policies Child Resource

- Implement typed child CRUD/list under a traffic controller:
  - `Create Or Update`
  - `Get`
  - `List By Traffic Controller`
  - `Update`
  - `Delete`
- Persist:
  - `location`
  - `tags`
  - `properties.policyType`
  - `properties.wafPolicy.id`
  - `properties.provisioningState`
- Update parent `trafficController.properties.securityPolicies` references on create/delete.
- Reflect an active WAF policy in `trafficController.properties.securityPolicyConfigurations`.

Exit criteria:
- Security policy routes return typed `SecurityPolicy` / `SecurityPolicyListResult`.
- Parent `securityPolicyConfigurations` behavior is consistent and deterministic.

## Stage 6: Typed Azure Go SDK Example

- Add `examples/azure/application-gateway-containers-2025-01-01`.
- Use `armservicenetworking.NewClientFactory(...)` or the explicit per-client constructors.
- Cover a representative lifecycle across all 5 clients.
- Reuse the repo’s standard Stackyard Azure ARM transport shim.

Exit criteria:
- Example compiles.
- Example runs locally against Stackyard without live Azure credentials.
- Selected SDK calls do not return `NotImplemented`.

## Stage 7: Coverage Completion

- Update doc coverage mapping.
- Ensure implementation and tests expose the expected signals for:
  - request validation
  - output fixtures
  - negative tests
  - output shape tests
- Run targeted Azure coverage commands for the new service key.

Exit criteria:
- `azure-contract-coverage.py` passes for `application_gateway_containers`
- `azure-io-contract-coverage.py` passes for `application_gateway_containers`
- `azure-doc-contract-coverage.py` passes for `application_gateway_containers`

## Recommended First Mergeable Slice

The first mergeable implementation should include:

- Stage 0
- Stage 1
- Stage 2
- Stage 3
- Stage 7 alias/doc wiring

That slice gives Stackyard a coherent parent resource plus the simplest child resource, which is enough to validate:
- provider operations
- ARM parent CRUD/list/update/delete
- nested resource routing
- Azure SDK example scaffolding

The next slice should add:
- associations
- security policies
- full typed SDK example

## Non-Goals

- Do not implement actual Kubernetes traffic management or packet routing behavior.
- Do not validate referenced subnet IDs or WAF policy IDs against other Azure providers.
- Do not emulate unrelated `Microsoft.ServiceNetworking` services outside this documented surface.
- Do not attempt full ARM LRO semantics beyond the headers/statuses needed for SDK compatibility in the first pass.
