# Azure Front Door Service - Web Application Firewall (`frontdoorservice_webapplicationfirewall_2025_11_01`) Staged Plan

## Objective

Emulate Azure Front Door Service Web Application Firewall REST API (`api-version=2025-11-01`) with deterministic local behavior for:

- provider and subscription-scoped Front Door name availability checks
- classic Front Door resource lifecycle and validation surfaces
- frontend endpoint HTTPS enable/disable and read/list surfaces
- rules engine lifecycle surfaces
- WAF policy lifecycle and patch surfaces
- network experiment profile and experiment lifecycle surfaces
- reporting, managed rule set listing, and preconfigured endpoint listing surfaces
- an Azure Go SDK example under `examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/operation-groups?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/endpoints?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/experiments?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-door-name-availability?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-door-name-availability-with-subscription?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/frontend-endpoints?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/managed-rule-sets?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/preconfigured-endpoints?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/reports?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/rules-engines?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

Method pages reviewed directly: all linked methods from all operation groups (`38` total).

- endpoints methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/endpoints/purge-content?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- experiments methods (5):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/experiments/create-or-update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/experiments/delete?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/experiments/get?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/experiments/list-by-profile?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/experiments/update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- front-door-name-availability methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-door-name-availability/check?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- front-door-name-availability-with-subscription methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-door-name-availability-with-subscription/check?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- front-doors methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors/create-or-update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors/delete?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors/get?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors/list?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors/list-by-resource-group?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/front-doors/validate-custom-domain?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- frontend-endpoints methods (4):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/frontend-endpoints/disable-https?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/frontend-endpoints/enable-https?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/frontend-endpoints/get?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/frontend-endpoints/list-by-front-door?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- managed-rule-sets methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/managed-rule-sets/list?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- network-experiment-profiles methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles/create-or-update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles/delete?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles/get?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles/list?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles/list-by-resource-group?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/network-experiment-profiles/update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- policies methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies/create-or-update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies/delete?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies/get?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies/list?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies/list-by-subscription?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/policies/update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- preconfigured-endpoints methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/preconfigured-endpoints/list?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- reports methods (2):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/reports/get-latency-scorecards?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/reports/get-timeseries?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

- rules-engines methods (4):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/rules-engines/create-or-update?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/rules-engines/delete?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/rules-engines/get?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/webapplicationfirewall/rules-engines/list-by-front-door?view=rest-frontdoorservice-webapplicationfirewall-2025-11-01`

## Surface Summary

Reviewed operation groups and methods:

- Endpoints (1)
- Experiments (5)
- Front Door Name Availability (1)
- Front Door Name Availability With Subscription (1)
- Front Doors (6)
- Frontend Endpoints (4)
- Managed Rule Sets (1)
- Network Experiment Profiles (6)
- Policies (6)
- Preconfigured Endpoints (1)
- Reports (2)
- Rules Engines (4)

Total reviewed methods: `38`.

## Service Naming

Canonical Stackyard naming:

- service key: `frontdoorservice_webapplicationfirewall_2025_11_01`
- display name: `Azure Front Door Service - Web Application Firewall`
- API version: `2025-11-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_frontdoorservice_webapplicationfirewall_2025_11_01.go`
- tests: `internal/server/provider_azure_frontdoorservice_webapplicationfirewall_2025_11_01_test.go`
- example: `examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01`
- plan doc: `docs/frontdoorservice-webapplicationfirewall-2025-11-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Network/checkFrontDoorNameAvailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/checkFrontDoorNameAvailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/frontDoors`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/frontDoorWebApplicationFirewallManagedRuleSets`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/NetworkExperimentProfiles`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/frontendEndpoints/{frontendEndpointName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/rulesEngines/{rulesEngineName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies/{policyName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/NetworkExperimentProfiles/{profileName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/NetworkExperimentProfiles/{profileName}/Experiments/{experimentName}`

Representative routes:

- `POST /azure/providers/Microsoft.Network/checkFrontDoorNameAvailability?api-version=2025-11-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/frontDoors/{frontDoorName}?api-version=2025-11-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/frontDoors/{frontDoorName}/rulesEngines/{rulesEngineName}?api-version=2025-11-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/FrontDoorWebApplicationFirewallPolicies/{policyName}?api-version=2025-11-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/NetworkExperimentProfiles/{profileName}?api-version=2025-11-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/NetworkExperimentProfiles/{profileName}/Experiments/{experimentName}?api-version=2025-11-01`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/NetworkExperimentProfiles/{profileName}/Experiments/{experimentName}/timeseries?api-version=2025-11-01&startDateTimeUTC={start}&endDateTimeUTC={end}&aggregationInterval={interval}&timeseriesType={type}`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-11-01`
- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods include `Other Status Codes`
- mutating actions document async variants returning `202 Accepted`

Observed response-code coverage across all reviewed methods:

- `200 OK`: `36`
- `201 Created`: `5`
- `202 Accepted`: `15`
- `204 No Content`: `5`
- `Other Status Codes`: `38`

Observed verb coverage:

- `GET`: `19`
- `POST`: `6`
- `PUT`: `5`
- `PATCH`: `3`
- `DELETE`: `5`

Request-body coverage from reviewed pages:

- `13` methods include request bodies:
- `endpoints/purge-content`
- `experiments/create-or-update`
- `experiments/update`
- `front-door-name-availability/check`
- `front-door-name-availability-with-subscription/check`
- `front-doors/create-or-update`
- `front-doors/validate-custom-domain`
- `frontend-endpoints/enable-https`
- `network-experiment-profiles/create-or-update`
- `network-experiment-profiles/update`
- `policies/create-or-update`
- `policies/update`
- `rules-engines/create-or-update`

SDK cross-check:

- this surface maps to Azure Front Door classic + WAF management resources under `Microsoft.Network`
- the Azure Go SDK has typed clients in `armfrontdoor`, but stage-0 examples use Azure Go SDK `azcore/runtime` calls to avoid typed-client API-version skew and to exercise every reviewed route directly

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `frontdoorservice_webapplicationfirewall_2025_11_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- keep stage-0 router non-owning to avoid changing current `network`/`frontdoor` behavior until ownership is intentionally implemented

Exit criteria:

- coverage scripts resolve service key and aliases
- docs coverage maps service key to this plan
- router dispatch includes a dedicated Front Door WAF stage-0 hook

### Stage 1: Route Ownership and API-Version Gate

Implement:

- route ownership for reviewed `Microsoft.Network` families associated with this API surface
- strict `api-version=2025-11-01` validation on owned routes
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- all reviewed route families are deterministically owned
- version/method failures are stable and testable

### Stage 2: Core Front Door, Policy, and Rules Engine Surfaces

Implement:

- both name availability operations
- front door create/get/list/list-by-resource-group/delete/validate-custom-domain
- frontend endpoint get/list-by-front-door and enable/disable HTTPS actions
- rules engine create/get/list/delete
- policies create/get/list/list-by-subscription/update/delete
- managed rule set list

Behavior:

- deterministic ARM resource envelopes (`id`, `name`, `type`, `location`, `tags`, `properties`)
- deterministic LRO placeholders for operations documenting `202`
- deterministic custom HTTPS and policy state transitions

Exit criteria:

- core Front Door classic + WAF policy surfaces are represented

### Stage 3: Network Experiment and Reporting Surfaces

Implement:

- network experiment profile create/get/list/list-by-resource-group/update/delete
- experiment create/get/list-by-profile/update/delete
- preconfigured endpoint list
- reports `get-latency-scorecards` and `get-timeseries`

Behavior:

- deterministic experiment/profile resource lifecycle
- deterministic report payloads keyed by requested interval/time windows

Exit criteria:

- experiment/profile/report surfaces are represented with deterministic payloads

### Stage 4: Testing and Contract Hardening

Add tests for:

- route ownership and API-version enforcement for all method families
- method-level success codes and minimal input validation paths
- resource lifecycle and idempotency semantics
- deterministic error envelope consistency

Coverage alignment:

- satisfy contract, I/O, and documentation coverage expectations for `frontdoorservice_webapplicationfirewall_2025_11_01`

## Example and Coverage Script Tasks

Example package:

- add `examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01/main.go`
- add container assets:
  - `examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01/Dockerfile`
  - `examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01/docker-compose.yml`

Coverage script wiring:

- add aliases in all three Azure coverage scripts for canonical and user-friendly spellings
- add docs plan mapping in `azure-doc-contract-coverage.py`

Recommended aliases:

- `frontdoorservice_webapplicationfirewall_2025_11_01`
- `frontdoorservice-webapplicationfirewall-2025-11-01`
- `frontdoorservice_webapplicationfirewall`
- `frontdoorservice-webapplicationfirewall`
- `azure_frontdoorservice_webapplicationfirewall`
- `frontdoor_waf_2025_11_01`
- `frontdoor_web_application_firewall`
- `microsoft_network_frontdoorwebapplicationfirewallpolicies`
- `rest_frontdoorservice_webapplicationfirewall_2025_11_01`

## Validation Commands

- `gofmt -w internal/server/provider_azure_frontdoorservice_webapplicationfirewall_2025_11_01.go`
- `gofmt -w examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01/main.go`
- `go build ./examples/azure/frontdoorservice-webapplicationfirewall-2025-11-01`
- `go test ./internal/server -run '^$'`
- `python3 scripts/azure-contract-coverage.py --service frontdoorservice-webapplicationfirewall-2025-11-01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service frontdoorservice-webapplicationfirewall-2025-11-01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service frontdoorservice-webapplicationfirewall-2025-11-01 --fail-on any`
