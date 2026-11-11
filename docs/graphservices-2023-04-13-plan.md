# Azure Graph Services (`graphservices_2023_04_13`) Staged Plan

## Objective

Emulate Azure Graph Services REST API (`api-version=2023-04-13`) with deterministic local behavior for:

- Graph Services account resource lifecycle surfaces
- provider-scoped operations discovery surface
- an Azure Go SDK example under `examples/azure/graphservices-2023-04-13`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/graphservices/operation-groups?view=rest-graphservices-2023-04-13`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts?view=rest-graphservices-2023-04-13`
- `https://learn.microsoft.com/en-us/rest/api/graphservices/operations?view=rest-graphservices-2023-04-13`

Method pages reviewed directly: all linked methods from all operation groups (`7` total).

- accounts methods (6):
- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts/create-and-update?view=rest-graphservices-2023-04-13`
- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts/delete?view=rest-graphservices-2023-04-13`
- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts/get?view=rest-graphservices-2023-04-13`
- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts/list-by-resource-group?view=rest-graphservices-2023-04-13`
- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts/list-by-subscription?view=rest-graphservices-2023-04-13`
- `https://learn.microsoft.com/en-us/rest/api/graphservices/accounts/update?view=rest-graphservices-2023-04-13`

- operations methods (1):
- `https://learn.microsoft.com/en-us/rest/api/graphservices/operations/list?view=rest-graphservices-2023-04-13`

## Surface Summary

Reviewed operation groups and methods:

- Accounts (6)
- Operations (1)

Total reviewed methods: `7`.

## Service Naming

Canonical Stackyard naming:

- service key: `graphservices_2023_04_13`
- display name: `Azure Graph Services`
- API version: `2023-04-13`

Recommended repo paths:

- implementation: `internal/server/provider_azure_graphservices_2023_04_13.go`
- tests: `internal/server/provider_azure_graphservices_2023_04_13_test.go`
- example: `examples/azure/graphservices-2023-04-13`
- plan doc: `docs/graphservices-2023-04-13-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.GraphServices/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.GraphServices/accounts`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.GraphServices/accounts/{resourceName}`

Representative routes:

- `GET /azure/providers/Microsoft.GraphServices/operations?api-version=2023-04-13`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.GraphServices/accounts/{resourceName}?api-version=2023-04-13`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.GraphServices/accounts?api-version=2023-04-13`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.GraphServices/accounts/{resourceName}?api-version=2023-04-13`
- `DELETE /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.GraphServices/accounts/{resourceName}?api-version=2023-04-13`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2023-04-13`
- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods include `Other Status Codes`

Observed response-code coverage across all reviewed methods:

- `200 OK`: `7`
- `201 Created`: `1`
- `204 No Content`: `1`
- `Other Status Codes`: `7`

Observed verb coverage:

- `GET`: `4`
- `PUT`: `1`
- `PATCH`: `1`
- `DELETE`: `1`

Request-body coverage from reviewed pages:

- `2` methods include request bodies:
- `accounts/create-and-update`
- `accounts/update`

SDK cross-check:

- operation-page samples reference Azure Go SDK typed clients in `armgraphservices`
- stage-0 example uses Azure Go SDK `azcore/runtime` directly so all reviewed routes can be exercised without typed-client version skew

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `graphservices_2023_04_13` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- keep stage-0 router non-owning to avoid changing existing routing behavior until ownership is intentionally implemented

Exit criteria:

- coverage scripts resolve service key and aliases
- docs coverage maps service key to this plan
- router dispatch includes a dedicated Graph Services stage-0 hook

### Stage 1: Route Ownership and API-Version Gate

Implement:

- route ownership for `Microsoft.GraphServices` families
- strict `api-version=2023-04-13` validation on owned routes
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Graph Services route families are deterministically owned
- version/method failures are stable and testable

### Stage 2: Accounts Lifecycle and Operations Discovery

Implement:

- accounts create-and-update/get/list-by-resource-group/list-by-subscription/update/delete
- operations list

Behavior:

- deterministic ARM resource envelopes (`id`, `name`, `type`, `location`, `tags`, `properties`)
- deterministic operation-list payload with Graph Services account operation entries
- deterministic delete semantics for `200` and `204` paths

Exit criteria:

- all reviewed methods are represented with deterministic behavior

### Stage 3: Testing and Contract Hardening

Add tests for:

- route ownership and API-version enforcement
- account lifecycle and patch-tag behavior
- list scoping semantics (resource group vs subscription)
- operations list response shape and error-envelope consistency

Coverage alignment:

- satisfy contract, I/O, and documentation coverage expectations for `graphservices_2023_04_13`

## Example and Coverage Script Tasks

Example package:

- add `examples/azure/graphservices-2023-04-13/main.go`
- add container assets:
  - `examples/azure/graphservices-2023-04-13/Dockerfile`
  - `examples/azure/graphservices-2023-04-13/docker-compose.yml`

Coverage script wiring:

- add aliases in all three Azure coverage scripts for canonical and user-friendly spellings
- add docs plan mapping in `azure-doc-contract-coverage.py`

Recommended aliases:

- `graphservices_2023_04_13`
- `graphservices-2023-04-13`
- `graphservices`
- `graph-services`
- `azure_graphservices`
- `azure_graph_services`
- `microsoft_graphservices`
- `microsoft_graph_services_accounts`
- `rest_graphservices_2023_04_13`

## Validation Commands

- `gofmt -w internal/server/provider_azure_graphservices_2023_04_13.go`
- `gofmt -w examples/azure/graphservices-2023-04-13/main.go`
- `go build ./examples/azure/graphservices-2023-04-13`
- `go test ./internal/server -run '^$'`
- `python3 scripts/azure-contract-coverage.py --service graphservices-2023-04-13 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service graphservices-2023-04-13 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service graphservices-2023-04-13 --fail-on any`
