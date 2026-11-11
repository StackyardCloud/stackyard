# Azure Application Gateway (`application-gateway-2025-05-01`) Staged Plan

## Objective

Emulate Azure Application Gateway (`2025-05-01`) with deterministic local behavior for the ARM management-plane resource lifecycle, backend health actions, subscription-level capability discovery endpoints, and a typed Azure Go SDK example that runs against Stackyard without live Azure infrastructure.

The Microsoft Learn surface for this API version is concentrated in a single operation group, `Application Gateways`, with 17 linked method pages. That makes this service substantially more focused than large `Microsoft.Web` or `Microsoft.CognitiveServices` surfaces, and suitable for a more complete staged implementation in fewer slices.

Primary references reviewed:
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/operation-groups?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/backend-health?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/backend-health-on-demand?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/create-or-update?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/delete?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/get?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/get-ssl-predefined-policy?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-all?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-available-request-headers?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-available-response-headers?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-available-server-variables?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-available-ssl-options?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-available-ssl-predefined-policies?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/list-available-waf-rule-sets?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/start?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/stop?view=rest-application-gateway-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/update-tags?view=rest-application-gateway-2025-05-01`

## Service Naming

The request called this “Azure Account Management”, but the supplied documentation is for Azure Application Gateway under `Microsoft.Network`. To keep implementation, example, and coverage naming aligned with the referenced API, this plan targets:

- service key: `application_gateway`
- display name: `Azure Application Gateway`
- API version: `2025-05-01`

Recommended repo naming:
- implementation file: `internal/server/provider_azure_application_gateway.go`
- tests: `internal/server/provider_azure_application_gateway_test.go`
- strict-signal tests if needed: `internal/server/provider_azure_application_gateway_strict_signals_test.go`
- example path: `examples/azure/application-gateway-2025-05-01`
- plan doc: `docs/application-gateway-2025-05-01-plan.md`

## Route Envelope

Stackyard route envelope for this service:
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailable*`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGateways`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}/*`

Target API version:
- `api-version=2025-05-01`

Representative paths from the reviewed method pages:
- list all gateways: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGateways`
- list available request headers: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailableRequestHeaders`
- list available response headers: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailableResponseHeaders`
- list available server variables: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailableServerVariables`
- list available SSL options: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default`
- list available SSL predefined policies: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default/predefinedPolicies`
- get one predefined policy: `/subscriptions/{subscriptionId}/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default/predefinedPolicies/{predefinedPolicyName}`
- application gateway resource: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}`
- backend health: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}/backendhealth`
- backend health on demand: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}/getBackendHealthOnDemand`
- start: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}/start`
- stop: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/applicationGateways/{applicationGatewayName}/stop`

## API Surface And Contract Notes

The `2025-05-01` Learn surface reviewed here exposes 17 linked method pages, all within the `Application Gateways` group:
- `Backend Health`
- `Backend Health On Demand`
- `Create Or Update`
- `Delete`
- `Get`
- `Get Ssl Predefined Policy`
- `List`
- `List All`
- `List Available Request Headers`
- `List Available Response Headers`
- `List Available Server Variables`
- `List Available Ssl Options`
- `List Available Ssl Predefined Policies`
- `List Available Waf Rule Sets`
- `Start`
- `Stop`
- `Update Tags`

Shared contract characteristics across the reviewed method pages:
- Authentication is Azure AD OAuth2 on the Learn pages, while Stackyard local examples continue to use the repo’s shared-key Azure shim transport.
- Read/list methods return typed ARM or collection payloads such as `ApplicationGateway`, `ApplicationGatewayListResult`, `ApplicationGatewayAvailableSslOptions`, `ApplicationGatewayAvailableSslPredefinedPolicies`, `ApplicationGatewayAvailableWafRuleSetsResult`, or `string[]`.
- Action methods use `POST` and include:
  - `backendhealth`
  - `getBackendHealthOnDemand`
  - `start`
  - `stop`
- Backend health methods support optional `$expand` query parameters.
- Mutation responses commonly include asynchronous variants:
  - `Create Or Update`: `200` or `201`, and may be long-running
  - `Delete`: `200`, `202`, or `204`
  - `Start` / `Stop`: `200` or `202`
  - `Update Tags`: `200`
- Error payloads are modeled as `CloudError` or `Error` depending on the endpoint page.

## Non-Goals

- Do not emulate the full Azure network control-plane graph behind Application Gateway resources.
- Do not validate every nested child collection inside `ApplicationGatewayPropertiesFormat` semantically in the first slice.
- Do not simulate actual packet forwarding, WAF inspection, TLS termination, or backend probe execution.
- Do not implement unrelated `Microsoft.Network` resources outside the application-gateway surface.

## Stage 0: Contract Skeleton And Coverage Naming

- Add a dedicated Azure Application Gateway router for `Microsoft.Network/applicationGateways` and the subscription-level `applicationGatewayAvailable*` endpoints.
- Add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Recommended aliases:
  - `application_gateway`
  - `applicationgateway`
  - `application_gateway_2025_05_01`
  - `applicationgateway_2025_05_01`
  - `azure_application_gateway`
  - `azure_application_gateway_2025_05_01`
  - `rest_application_gateway_2025_05_01`
- Map this plan doc in `azure-doc-contract-coverage.py`.
- Add route-recognition tests for:
  - subscription-level availability endpoints
  - resource-group scoped application-gateway resource routes
  - action routes (`backendhealth`, `getBackendHealthOnDemand`, `start`, `stop`)

Exit criteria:
- Coverage scripts discover the new canonical service key.
- The Azure router claims Application Gateway paths deterministically.

## Stage 1: Validation Foundation

- Validate `api-version` the same way as other Azure management-plane handlers.
- Enforce supported methods:
  - `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
- Return deterministic `InvalidRequest` responses for malformed `api-version` or structurally invalid requests.
- Add negative tests for invalid query state and unsupported method routing.

Exit criteria:
- Request validation signals and negative tests satisfy the Azure contract and IO coverage scripts.

## Stage 2: Subscription-Level Capability Endpoints

- Implement deterministic typed responses for:
  - `List Available Request Headers`
  - `List Available Response Headers`
  - `List Available Server Variables`
  - `List Available Ssl Options`
  - `List Available Ssl Predefined Policies`
  - `Get Ssl Predefined Policy`
  - `List Available Waf Rule Sets`
- Use realistic but stable payloads:
  - string arrays for header/server-variable endpoints
  - `value` arrays plus `nextLink`-omittable collection envelopes for SSL and WAF endpoints

Exit criteria:
- All subscription-level discovery endpoints return typed deterministic payloads and no longer fall back to generic staged responses.

## Stage 3: Core Application Gateway Resource Lifecycle

- Implement deterministic state for `ApplicationGateway` resources:
  - `Create Or Update`
  - `Get`
  - `List`
  - `List All`
  - `Update Tags`
  - `Delete`
- Model the minimum useful fields from reviewed examples:
  - `id`, `name`, `type`, `location`, `tags`
  - `properties.provisioningState`
  - `properties.operationalState`
  - `properties.sku`
  - representative nested collections like `gatewayIPConfigurations`, `frontendPorts`, `frontendIPConfigurations`, `backendAddressPools`, and `backendHttpSettingsCollection`
- Keep nested child collections shallow and deterministic; prefer references and one sample element over deep semantic validation.
- Return synchronous success payloads unless the Azure Go SDK example requires a long-running-operation shape.

Exit criteria:
- Core resource CRUD/list operations return typed `ApplicationGateway` payloads or collection envelopes.
- No core resource route returns `NotImplemented`.

## Stage 4: Action Endpoints

- Implement deterministic responses for:
  - `Backend Health`
  - `Backend Health On Demand`
  - `Start`
  - `Stop`
- Backend health payloads should include enough structure for the SDK and tests to assert:
  - backend pool identity
  - backend HTTP setting identity
  - backend server entries
  - health state markers
- `Start` and `Stop` should toggle a stored operational-state field and return either `200` success or a minimal accepted/poller-compatible response if the SDK requires it.

Exit criteria:
- Application Gateway action endpoints produce typed non-`NotImplemented` responses.
- Repeated start/stop calls remain deterministic.

## Stage 5: Example Coverage

- Add `examples/azure/application-gateway-2025-05-01` using the Azure Go SDK package family for ARM Network management:
  - expected package family: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork`
  - exact module major version should be chosen during implementation based on current SDK availability during `go mod tidy`
- Follow the repo’s existing typed Azure example pattern:
  - `azsdkshim.StaticTokenCredential{}`
  - `arm.ClientOptions`
  - `cloud.ResourceManager` endpoint override to Stackyard
  - `azsdkshim.NewTransport("/azure", account, subscriptionKey)`
- Example should exercise a representative, compile-stable subset:
  - list all gateways
  - create or update gateway
  - get gateway
  - list by resource group
  - update tags
  - backend health
  - start
  - stop
  - list available request/response headers
  - list available SSL predefined policies

Exit criteria:
- Example compiles and runs against Stackyard without live Azure credentials.
- Example does not report `NotImplemented` for the selected calls.

## Stage 6: Coverage Script Completion

- Update the Azure coverage scripts so the new service key resolves consistently.
- Ensure the implementation file and tests expose the validation/output/negative-signal patterns expected by:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add doc-source coverage through this plan file.

Exit criteria:
- The three Azure coverage scripts report the new service as discovered and strict-complete for the implemented slice.

## Recommended First Mergeable Slice

The first mergeable implementation should include:
- Stage 0
- Stage 1
- Stage 2
- Stage 3
- Stage 5 with a narrowed SDK example that covers:
  - `List All`
  - `Create Or Update`
  - `Get`
  - `Update Tags`
  - `List Available Request Headers`
  - `List Available Response Headers`
  - `List Available Ssl Predefined Policies`
- Stage 6 coverage alias/doc wiring

That slice gives Stackyard a meaningful `Microsoft.Network/applicationGateways` presence quickly, keeps the first example small, and avoids stalling on the more structured backend-health payloads until the core resource state is already stable.

## Files Expected To Change During Implementation

- `internal/server/provider_azure_application_gateway.go`
- `internal/server/provider_azure_application_gateway_test.go`
- `internal/server/http.go`
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`
- `examples/azure/application-gateway-2025-05-01/main.go`
- `examples/azure/application-gateway-2025-05-01/Dockerfile`
- `examples/azure/application-gateway-2025-05-01/docker-compose.yml`

## Risks And Notes

- The `ApplicationGateway` model is wide; deep fidelity is unnecessary for the first staged implementation as long as the SDK and coverage checks get typed, deterministic shapes.
- The backend-health methods are the most likely place where a generic foundation payload is insufficient, so they should get explicit typed fixtures early.
- Subscription-level capability endpoints are cheap wins and should be implemented before resource actions.
- Because this surface is entirely under `Microsoft.Network`, route matching must avoid accidentally claiming unrelated network-service paths that also live under the same provider.
