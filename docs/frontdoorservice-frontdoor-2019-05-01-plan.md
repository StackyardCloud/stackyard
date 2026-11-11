# Azure Front Door Service - Front Door (`frontdoorservice_frontdoor_2019_05_01`) Staged Plan

## Objective

Emulate Azure Front Door Service (classic Front Door) REST API (`api-version=2019-05-01`) with deterministic local behavior for:

- provider and subscription-scoped Front Door name availability checks
- front door resource lifecycle (create/get/list/delete)
- front door custom-domain validation
- endpoint content purge action
- frontend endpoint read/list and custom HTTPS enable/disable actions
- an Azure Go SDK example under `examples/azure/frontdoorservice-frontdoor-2019-05-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/operation-groups?view=rest-frontdoorservice-frontdoor-2019-05-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/check-front-door-name-availability?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/check-front-door-name-availability-with-subscription?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/endpoints?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/frontend-endpoints?view=rest-frontdoorservice-frontdoor-2019-05-01`

Method pages reviewed directly: all linked methods from all operation groups (`13` total).

- check-front-door-name-availability-with-subscription methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/check-front-door-name-availability-with-subscription/check-front-door-name-availability-with-subscription?view=rest-frontdoorservice-frontdoor-2019-05-01`

- check-front-door-name-availability methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/check-front-door-name-availability/check-front-door-name-availability?view=rest-frontdoorservice-frontdoor-2019-05-01`

- endpoints methods (1):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/endpoints/purge-content?view=rest-frontdoorservice-frontdoor-2019-05-01`

- front-doors methods (6):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors/create-or-update?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors/delete?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors/get?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors/list?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors/list-by-resource-group?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/front-doors/validate-custom-domain?view=rest-frontdoorservice-frontdoor-2019-05-01`

- frontend-endpoints methods (4):
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/frontend-endpoints/disable-https?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/frontend-endpoints/enable-https?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/frontend-endpoints/get?view=rest-frontdoorservice-frontdoor-2019-05-01`
- `https://learn.microsoft.com/en-us/rest/api/frontdoorservice/frontdoor/frontend-endpoints/list-by-front-door?view=rest-frontdoorservice-frontdoor-2019-05-01`

## Surface Summary

Reviewed operation groups and methods:

- Check Front Door Name Availability (1)
- Check Front Door Name Availability With Subscription (1)
- Endpoints (1)
- Front Doors (6)
- Frontend Endpoints (4)

Total reviewed methods: `13`.

## Service Naming

Canonical Stackyard naming:

- service key: `frontdoorservice_frontdoor_2019_05_01`
- display name: `Azure Front Door Service - Front Door`
- API version: `2019-05-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_frontdoorservice_frontdoor_2019_05_01.go`
- tests: `internal/server/provider_azure_frontdoorservice_frontdoor_2019_05_01_test.go`
- example: `examples/azure/frontdoorservice-frontdoor-2019-05-01`
- plan doc: `docs/frontdoorservice-frontdoor-2019-05-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Network/checkFrontDoorNameAvailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/checkFrontDoorNameAvailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/frontDoors`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/purge`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/validateCustomDomain`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/frontendEndpoints/{frontendEndpointName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/frontendEndpoints/{frontendEndpointName}/enableHttps`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/frontDoors/{frontDoorName}/frontendEndpoints/{frontendEndpointName}/disableHttps`

Representative routes:

- `POST /azure/providers/Microsoft.Network/checkFrontDoorNameAvailability?api-version=2019-05-01`
- `POST /azure/subscriptions/{sub}/providers/Microsoft.Network/checkFrontDoorNameAvailability?api-version=2019-05-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/frontDoors/{frontDoorName}?api-version=2019-05-01`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.Network/frontDoors?api-version=2019-05-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/frontDoors/{frontDoorName}/purge?api-version=2019-05-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/frontDoors/{frontDoorName}/validateCustomDomain?api-version=2019-05-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/frontDoors/{frontDoorName}/frontendEndpoints/{frontendEndpointName}/enableHttps?api-version=2019-05-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2019-05-01`
- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods include `Other Status Codes`
- mutating actions document async variants returning `202 Accepted`

Observed response-code coverage across all reviewed methods:

- `200 OK`: `10`
- `201 Created`: `1`
- `202 Accepted`: `5`
- `204 No Content`: `1`
- `Other Status Codes`: `13`

Observed verb coverage:

- `GET`: `5`
- `POST`: `6`
- `PUT`: `1`
- `DELETE`: `1`

Request-body coverage from reviewed pages:

- `6` methods include request bodies:
  - `check-front-door-name-availability/check-front-door-name-availability`
  - `check-front-door-name-availability-with-subscription/check-front-door-name-availability-with-subscription`
  - `endpoints/purge-content`
  - `front-doors/create-or-update`
  - `front-doors/validate-custom-domain`
  - `frontend-endpoints/enable-https`

SDK cross-check:

- Front Door classic management routes are under `Microsoft.Network` and align with the Azure Go SDK network management family (`armnetwork`)
- stage-0 example uses Azure Go SDK `azcore/runtime` calls so all reviewed routes can be exercised without typed-client API-version drift

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `frontdoorservice_frontdoor_2019_05_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- docs coverage maps service key to this plan
- router dispatch includes a dedicated Front Door (classic) stage-0 hook

### Stage 1: Route Ownership and API-Version Gate

Implement:

- route ownership for the Front Door classic route families under `Microsoft.Network/frontDoors`
- strict `api-version=2019-05-01` validation on owned routes
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Front Door classic route families are deterministically owned
- version/method failures are stable and testable

### Stage 2: Name Availability + Front Door CRUD

Implement:

- both name-availability methods
- `front-doors` create/get/list/list-by-resource-group/delete
- `front-doors/validate-custom-domain`

Behavior:

- deterministic resource-name availability outputs
- canonical ARM envelope for front door resources (`id`, `name`, `type`, `location`, `tags`, `properties`)
- deterministic LRO placeholders for `202` paths

Exit criteria:

- core front door lifecycle and validation surfaces are represented

### Stage 3: Endpoint Actions and Frontend HTTPS

Implement:

- `endpoints/purge-content`
- `frontend-endpoints/get`
- `frontend-endpoints/list-by-front-door`
- `frontend-endpoints/enable-https`
- `frontend-endpoints/disable-https`

Behavior:

- deterministic custom HTTPS state transitions on frontend endpoints
- deterministic purge action response behavior for sync/async responses

Exit criteria:

- endpoint and frontend endpoint action surfaces are represented

### Stage 4: Testing and Contract Hardening

Add tests for:

- API-version/method enforcement and route ownership
- name-availability request validation and deterministic responses
- front door create/get/list/delete lifecycle behavior
- frontend HTTPS enable/disable and idempotency semantics
- purge and validate custom domain action behavior

Exit criteria:

- provider tests cover Stage 1-3 behavior
- service passes strict contract/IO/doc coverage checks

## Coverage Script Integration

Add aliases for `frontdoorservice_frontdoor_2019_05_01` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:

- `frontdoorservice_frontdoor_2019_05_01`
- `frontdoorservice-frontdoor-2019-05-01`
- `frontdoorservice_frontdoor`
- `frontdoorservice-frontdoor`
- `azure_frontdoorservice_frontdoor`
- `microsoft_network_frontdoors`
- `rest_frontdoorservice_frontdoor_2019_05_01`

Map plan doc:

- `SERVICE_PLAN_DOCS["frontdoorservice_frontdoor_2019_05_01"] = ["frontdoorservice-frontdoor-2019-05-01-plan.md"]`

## Example Plan

Add `examples/azure/frontdoorservice-frontdoor-2019-05-01`:

- `main.go`: Azure Go SDK (`azcore/runtime`) client that exercises representative methods from each reviewed operation group
- `Dockerfile`: builds and runs the example binary
- `docker-compose.yml`: local stack wiring with Front Door classic env vars

Example coverage target:

- one representative request per reviewed method family
- create/get/list/delete front door lifecycle requests
- endpoint purge and custom-domain validation requests
- frontend endpoint get/list and HTTPS enable/disable requests

## Validation Commands

- `go build ./examples/azure/frontdoorservice-frontdoor-2019-05-01`
- `python3 scripts/azure-contract-coverage.py --service frontdoorservice-frontdoor-2019-05-01 --require-service frontdoorservice_frontdoor_2019_05_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service frontdoorservice-frontdoor-2019-05-01 --require-service frontdoorservice_frontdoor_2019_05_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service frontdoorservice-frontdoor-2019-05-01 --require-service frontdoorservice_frontdoor_2019_05_01 --fail-on any`

Expected stage-0 result:

- docs and coverage wiring pass service discovery checks
- strict contract checks remain partial until later-stage provider implementation/tests are added
