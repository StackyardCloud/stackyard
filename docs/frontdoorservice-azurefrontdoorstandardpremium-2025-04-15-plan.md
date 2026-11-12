# Azure Front Door Service - Standard Premium (`frontdoorservice_azurefrontdoorstandardpremium_2025_04_15`) Staged Plan

## Objective

Emulate Azure Front Door Service Standard/Premium REST API (`api-version=2025-04-15`) with deterministic local behavior for:

- AFD custom domain lifecycle surfaces
- AFD endpoint lifecycle and action surfaces
- AFD origin-group and origin lifecycle surfaces
- profile-scoped validation/upgrade surfaces
- endpoint-name availability validation
- log analytics query surfaces
- route, rule-set, and rule lifecycle surfaces
- secret and security-policy lifecycle surfaces
- an Azure Go SDK example under `examples/azure/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/operation-groups?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origins?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-profiles?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/check-endpoint-name-availability?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/routes?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rule-sets?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rules?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/secrets?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/security-policies?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

Method pages reviewed directly: all linked methods from all operation groups (`61` total).

- afd-custom-domains methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains/list-by-profile?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains/refresh-validation-token?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-custom-domains/update?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- afd-endpoints methods (8):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/list-by-profile?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/list-resource-usage?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/purge-content?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/update?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-endpoints/validate-custom-domain?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- afd-origin-groups methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups/list-by-profile?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups/list-resource-usage?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origin-groups/update?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- afd-origins methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origins/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origins/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origins/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origins/list-by-origin-group?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-origins/update?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- afd-profiles methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-profiles/check-endpoint-name-availability?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-profiles/check-host-name-availability?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-profiles/list-resource-usage?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-profiles/upgrade?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/afd-profiles/validate-secret?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- check-endpoint-name-availability methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/check-endpoint-name-availability/check-endpoint-name-availability?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- log-analytics methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics/get-log-analytics-locations?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics/get-log-analytics-metrics?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics/get-log-analytics-rankings?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics/get-log-analytics-resources?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics/get-waf-log-analytics-metrics?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/log-analytics/get-waf-log-analytics-rankings?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- routes methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/routes/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/routes/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/routes/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/routes/list-by-endpoint?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/routes/update?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- rule-sets methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rule-sets/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rule-sets/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rule-sets/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rule-sets/list-by-profile?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rule-sets/list-resource-usage?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- rules methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rules/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rules/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rules/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rules/list-by-rule-set?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/rules/update?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- secrets methods (4):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/secrets/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/secrets/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/secrets/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/secrets/list-by-profile?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

- security-policies methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/security-policies/create?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/security-policies/delete?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/security-policies/get?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/security-policies/list-by-profile?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/azurefrontdoorstandardpremium/security-policies/patch?view=rest-frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`

## Surface Summary

Reviewed operation groups and methods:

- AFD Custom Domains (6)
- AFD Endpoints (8)
- AFD Origin Groups (6)
- AFD Origins (5)
- AFD Profiles (5)
- Check Endpoint Name Availability (1)
- Log Analytics (6)
- Routes (5)
- Rule Sets (5)
- Rules (5)
- Secrets (4)
- Security Policies (5)

Total reviewed methods: `61`.

## Service Naming

Canonical Stackyard naming:

- service key: `frontdoorservice_azurefrontdoorstandardpremium_2025_04_15`
- display name: `Azure Front Door Service - Standard Premium`
- API version: `2025-04-15`

Recommended repo paths:

- implementation: `internal/server/provider_azure_frontdoorservice_azurefrontdoorstandardpremium_2025_04_15.go`
- tests: `internal/server/provider_azure_frontdoorservice_azurefrontdoorstandardpremium_2025_04_15_test.go`
- example: `examples/azure/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- plan doc: `docs/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/checkEndpointNameAvailability`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/customDomains/{customDomainName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/afdEndpoints/{endpointName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/originGroups/{originGroupName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/originGroups/{originGroupName}/origins/{originName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/afdEndpoints/{endpointName}/routes/{routeName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/ruleSets/{ruleSetName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/ruleSets/{ruleSetName}/rules/{ruleName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/secrets/{secretName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/profiles/{profileName}/securityPolicies/{securityPolicyName}`
- profile utility/action routes such as `checkEndpointNameAvailability`, `checkHostNameAvailability`, `validateSecret`, `upgrade`, `getLogAnalytics*`

Representative routes:

- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/checkEndpointNameAvailability?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/afdEndpoints/{endpoint}?api-version=2025-04-15`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/afdEndpoints/{endpoint}/purge?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/originGroups/{originGroup}?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/originGroups/{originGroup}/origins/{origin}?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/afdEndpoints/{endpoint}/routes/{route}?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/ruleSets/{ruleSet}/rules/{rule}?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/secrets/{secret}?api-version=2025-04-15`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/profiles/{profile}/securityPolicies/{securityPolicy}?api-version=2025-04-15`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-04-15`
- all reviewed methods are ARM OAuth2-protected (`azure_auth`, `user_impersonation`)
- all reviewed methods document `Other Status Codes` with `ErrorResponse`
- most mutating operations include LRO variants and may return `202 Accepted` with `azure-asyncoperation` and/or `location` headers

Observed response-code coverage across all reviewed methods:

- `200 OK`: `61`
- `201 Created`: `9`
- `202 Accepted`: `27`
- `204 No Content`: `9`

Observed verb coverage across all reviewed methods:

- `GET`: `24`
- `POST`: `12`
- `PUT`: `9`
- `PATCH`: `7`
- `DELETE`: `9`

Request-body coverage from reviewed pages:

- `22` methods include request bodies:
  - `afd-custom-domains/create`
  - `afd-custom-domains/update`
  - `afd-endpoints/create`
  - `afd-endpoints/purge-content`
  - `afd-endpoints/update`
  - `afd-endpoints/validate-custom-domain`
  - `afd-origin-groups/create`
  - `afd-origin-groups/update`
  - `afd-origins/create`
  - `afd-origins/update`
  - `afd-profiles/check-endpoint-name-availability`
  - `afd-profiles/check-host-name-availability`
  - `afd-profiles/upgrade`
  - `afd-profiles/validate-secret`
  - `check-endpoint-name-availability/check-endpoint-name-availability`
  - `routes/create`
  - `routes/update`
  - `rules/create`
  - `rules/update`
  - `secrets/create`
  - `security-policies/create`
  - `security-policies/patch`

SDK cross-check:

- the management-plane provider namespace is `Microsoft.Cdn`, and AFD Standard/Premium operations align with the Azure Go SDK `armcdn` family for typed clients
- stage-0 example uses Azure Go SDK `azcore/runtime` directly so all reviewed routes can be demonstrated without typed-client version skew

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `frontdoorservice_azurefrontdoorstandardpremium_2025_04_15` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- keep stage-0 router non-owning to avoid changing current `cdn` route behavior until dedicated Front Door route ownership is implemented

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- router dispatch includes a dedicated Front Door stage-0 hook without behavior regressions

### Stage 1: Route Ownership and API-Version Gate

Implement:

- route ownership for Front Door Standard/Premium families under `Microsoft.Cdn/profiles/*`
- strict `api-version=2025-04-15` validation on owned routes
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Front Door route families are no longer shadowed by generic CDN handling
- version/method failures are stable and testable

### Stage 2: Core Resource CRUD Surfaces

Implement:

- AFD Custom Domains (6)
- AFD Endpoints (8)
- AFD Origin Groups (6)
- AFD Origins (5)
- Routes (5)
- Rule Sets (5)
- Rules (5)
- Secrets (4)
- Security Policies (5)

Behavior:

- deterministic ARM resource envelopes (`id`, `name`, `type`, `properties`, optional `tags`, `systemData`)
- parent-child validation across profile, endpoint, origin-group, route, and rule-set scopes
- stable list ordering and deterministic LRO placeholders for `202` flows

Exit criteria:

- CRUD and list/get surfaces from all core resource groups return deterministic, schema-consistent payloads

### Stage 3: Validation, Profile Actions, and Analytics Utilities

Implement:

- AFD Profiles methods (5)
- Check Endpoint Name Availability method (1)
- Log Analytics methods (6)

Behavior:

- deterministic endpoint/host-name availability responses (`nameAvailable`, `reason`, `message`, `availableHostname`)
- deterministic profile-upgrade and secret-validation outputs
- deterministic analytics fixtures for locations/resources/metrics/rankings methods

Exit criteria:

- all non-CRUD operation groups are functionally represented

### Stage 4: Testing and Contract Hardening

Add tests for:

- route ownership and API-version/method enforcement
- parent-resource existence and validation failures
- happy-path lifecycle flows across all resource groups
- list ordering/paging behavior (where applicable)
- async response shape/header consistency on LRO-style methods

Exit criteria:

- provider test suite covers Stage 1-3 behavior
- service passes strict contract/IO/doc coverage checks

## Coverage Script Integration

Add aliases for `frontdoorservice_azurefrontdoorstandardpremium_2025_04_15` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:

- `frontdoorservice_azurefrontdoorstandardpremium_2025_04_15`
- `frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `frontdoorservice_azurefrontdoorstandardpremium`
- `frontdoorservice-azurefrontdoorstandardpremium`
- `azure_frontdoorservice_azurefrontdoorstandardpremium`
- `azure_frontdoor_service_standard_premium`
- `rest_frontdoorservice_azurefrontdoorstandardpremium_2025_04_15`

Map plan doc:

- `SERVICE_PLAN_DOCS["frontdoorservice_azurefrontdoorstandardpremium_2025_04_15"] = ["frontdoorservice-azurefrontdoorstandardpremium-2025-04-15-plan.md"]`

## Example Plan

Add `examples/azure/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`:

- `main.go`: Azure Go SDK (`azcore/runtime`) client that exercises representative methods from each reviewed operation group
- `Dockerfile`: builds and runs the example binary
- `docker-compose.yml`: local stack wiring with Front Door env vars

Example coverage target:

- one or more representative calls per operation group
- core create/get/list/update/delete coverage for AFD resource families
- profile validation and endpoint-availability actions
- secrets/security-policy and rules/routing examples

## Validation Commands

- `go build ./examples/azure/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15`
- `python3 scripts/azure-contract-coverage.py --service frontdoorservice-azurefrontdoorstandardpremium-2025-04-15 --require-service frontdoorservice_azurefrontdoorstandardpremium_2025_04_15 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service frontdoorservice-azurefrontdoorstandardpremium-2025-04-15 --require-service frontdoorservice_azurefrontdoorstandardpremium_2025_04_15 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service frontdoorservice-azurefrontdoorstandardpremium-2025-04-15 --require-service frontdoorservice_azurefrontdoorstandardpremium_2025_04_15 --fail-on any`

Expected stage-0 result:

- docs and coverage wiring pass service discovery checks
- strict contract checks remain partial until later-stage provider implementation/tests are completed
