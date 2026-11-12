# Azure Front Door Service - Edge Actions (`edge_actions_2025_12_01_preview`) Staged Plan

## Objective

Emulate Azure Front Door Service Edge Actions REST API (`api-version=2025-12-01-preview`) with deterministic local behavior for:

- edge action resource lifecycle surfaces
- edge action version lifecycle and version-code action surfaces
- edge action execution filter lifecycle surfaces
- an Azure Go SDK example under `examples/azure/edge-actions-2025-12-01-preview`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/edge-actions/operation-groups?view=rest-edge-actions-2025-12-01-preview`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-execution-filters?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions?view=rest-edge-actions-2025-12-01-preview`

Method pages reviewed directly: all linked methods from all operation groups (`19` total).

- edge-action-execution-filters methods (5):
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-execution-filters/create?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-execution-filters/delete?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-execution-filters/get?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-execution-filters/list-by-edge-action?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-execution-filters/update?view=rest-edge-actions-2025-12-01-preview`

- edge-action-versions methods (8):
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/create?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/delete?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/deploy-version-code?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/get?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/get-version-code?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/list-by-edge-action?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/swap-default?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-action-versions/update?view=rest-edge-actions-2025-12-01-preview`

- edge-actions methods (6):
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions/create?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions/delete?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions/get?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions/list-by-resource-group?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions/list-by-subscription?view=rest-edge-actions-2025-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/edge-actions/edge-actions/update?view=rest-edge-actions-2025-12-01-preview`

## Surface Summary

Reviewed operation groups and methods:

- Edge Action Execution Filters (5)
- Edge Action Versions (8)
- Edge Actions (6)

Total reviewed methods: `19`.

## Service Naming

Canonical Stackyard naming:

- service key: `edge_actions_2025_12_01_preview`
- display name: `Azure Front Door Service - Edge Actions`
- API version: `2025-12-01-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_edge_actions_2025_12_01_preview.go`
- tests: `internal/server/provider_azure_edge_actions_2025_12_01_preview_test.go`
- example: `examples/azure/edge-actions-2025-12-01-preview`
- plan doc: `docs/edge-actions-2025-12-01-preview-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Cdn/edgeActions`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}/deployVersionCode`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}/getVersionCode`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}/swapDefault`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/executionFilters/{executionFilter}`

Representative routes:

- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}?api-version=2025-12-01-preview`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.Cdn/edgeActions?api-version=2025-12-01-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}?api-version=2025-12-01-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}/deployVersionCode?api-version=2025-12-01-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/versions/{version}/swapDefault?api-version=2025-12-01-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Cdn/edgeActions/{edgeActionName}/executionFilters/{executionFilter}?api-version=2025-12-01-preview`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-12-01-preview`
- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods include `Other Status Codes`
- mutating operations document async variants returning `202 Accepted`

Observed response-code coverage across all reviewed methods:

- `200 OK`: `18`
- `201 Created`: `3`
- `202 Accepted`: `9`
- `204 No Content`: `4`
- `Other Status Codes`: `19`

Observed verb coverage:

- `GET`: `7`
- `POST`: `3`
- `PUT`: `3`
- `PATCH`: `3`
- `DELETE`: `3`

Request-body coverage from reviewed pages:

- `7` methods include request bodies:
- `edge-actions/create`
- `edge-actions/update`
- `edge-action-versions/create`
- `edge-action-versions/deploy-version-code`
- `edge-action-versions/update`
- `edge-action-execution-filters/create`
- `edge-action-execution-filters/update`

SDK cross-check:

- the provider namespace is `Microsoft.Cdn`
- operation-page samples show Azure Go SDK typed clients in `armedgeactions`
- stage-0 example uses Azure Go SDK `azcore/runtime` directly so all reviewed methods can be exercised without typed-client version skew

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `edge_actions_2025_12_01_preview` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- keep stage-0 router non-owning to avoid changing existing CDN routing behavior until ownership is implemented

Exit criteria:

- coverage scripts resolve service key and aliases
- docs coverage maps service key to this plan
- router dispatch includes a dedicated Edge Actions stage-0 hook

### Stage 1: Route Ownership and API-Version Gate

Implement:

- route ownership for `Microsoft.Cdn/edgeActions` families
- strict `api-version=2025-12-01-preview` validation on owned routes
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- all Edge Actions route families are deterministically owned
- version/method failures are stable and testable

### Stage 2: Edge Action and Version Core Lifecycle

Implement:

- edge action create/get/list-by-resource-group/list-by-subscription/update/delete
- edge action version create/get/list-by-edge-action/update/delete

Behavior:

- deterministic ARM resource envelopes (`id`, `name`, `type`, `location`, `tags`, `properties`)
- deterministic LRO placeholders for methods documenting `202`

Exit criteria:

- core edge action and version lifecycles are represented

### Stage 3: Version Code Actions and Execution Filters

Implement:

- version actions: `deployVersionCode`, `getVersionCode`, `swapDefault`
- execution filter create/get/list-by-edge-action/update/delete

Behavior:

- deterministic version-code retrieval/deploy payload handling
- deterministic default-version swap behavior
- deterministic execution filter header/value and version-link semantics

Exit criteria:

- action and execution-filter surfaces are represented

### Stage 4: Testing and Contract Hardening

Add tests for:

- API-version and method enforcement across all route families
- lifecycle behavior and idempotency semantics
- action behaviors for deploy/get/swap version code
- consistent error envelope behavior

Coverage alignment:

- satisfy contract, I/O, and documentation coverage expectations for `edge_actions_2025_12_01_preview`

## Example and Coverage Script Tasks

Example package:

- add `examples/azure/edge-actions-2025-12-01-preview/main.go`
- add container assets:
  - `examples/azure/edge-actions-2025-12-01-preview/Dockerfile`
  - `examples/azure/edge-actions-2025-12-01-preview/docker-compose.yml`

Coverage script wiring:

- add aliases in all three Azure coverage scripts for canonical and user-friendly spellings
- add docs plan mapping in `azure-doc-contract-coverage.py`

Recommended aliases:

- `edge_actions_2025_12_01_preview`
- `edge-actions-2025-12-01-preview`
- `edge_actions`
- `edge-actions`
- `edgeactions_2025_12_01_preview`
- `azure_edge_actions`
- `azure_edgeactions`
- `microsoft_cdn_edgeactions`
- `rest_edge_actions_2025_12_01_preview`

## Validation Commands

- `gofmt -w internal/server/provider_azure_edge_actions_2025_12_01_preview.go`
- `gofmt -w examples/azure/edge-actions-2025-12-01-preview/main.go`
- `go build ./examples/azure/edge-actions-2025-12-01-preview`
- `go test ./internal/server -run '^$'`
- `python3 scripts/azure-contract-coverage.py --service edge-actions-2025-12-01-preview --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service edge-actions-2025-12-01-preview --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service edge-actions-2025-12-01-preview --fail-on any`
