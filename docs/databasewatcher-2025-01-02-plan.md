# Azure Database Watcher (`databasewatcher_2025_01_02`) Staged Plan

## Objective

Emulate Azure Database Watcher REST API (`api-version=2025-01-02`) with deterministic local behavior for:

- watcher lifecycle and monitoring-state actions
- target registration and lifecycle flows
- alert rule proxy-resource lifecycle flows
- health validation start/get/list flows
- shared private link resource lifecycle flows
- provider operations discovery
- an Azure Go SDK example under `examples/azure/databasewatcher-2025-01-02`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/databasewatcher/operation-groups?view=rest-databasewatcher-2025-01-02`

Operation-group pages reviewed:

- `.../watchers?view=rest-databasewatcher-2025-01-02`
- `.../targets?view=rest-databasewatcher-2025-01-02`
- `.../alert-rule-resources?view=rest-databasewatcher-2025-01-02`
- `.../health-validations?view=rest-databasewatcher-2025-01-02`
- `.../shared-private-link-resources?view=rest-databasewatcher-2025-01-02`
- `.../operations/list?view=rest-databasewatcher-2025-01-02`

Method pages reviewed directly include:

- Watchers:
  - `watchers/create-or-update`
  - `watchers/delete`
  - `watchers/get`
  - `watchers/list-by-resource-group`
  - `watchers/list-by-subscription`
  - `watchers/start`
  - `watchers/stop`
  - `watchers/update`
- Targets:
  - `targets/create-or-update`
  - `targets/delete`
  - `targets/get`
  - `targets/list-by-watcher`
- Alert Rule Resources:
  - `alert-rule-resources/create-or-update`
  - `alert-rule-resources/delete`
  - `alert-rule-resources/get`
  - `alert-rule-resources/list-by-parent`
- Health Validations:
  - `health-validations/start-validation`
  - `health-validations/get`
  - `health-validations/list-by-parent`
- Shared Private Link Resources:
  - `shared-private-link-resources/create`
  - `shared-private-link-resources/delete`
  - `shared-private-link-resources/get`
  - `shared-private-link-resources/list-by-watcher`
- Operations:
  - `operations/list`

SDK cross-check used for operation-group parity:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databasewatcher/armdatabasewatcher`
  - `NewWatchersClient`
  - `NewTargetsClient`
  - `NewAlertRuleResourcesClient`
  - `NewHealthValidationsClient`
  - `NewSharedPrivateLinkResourcesClient`
  - `NewOperationsClient`

Notes on source behavior in this environment:

- several `learn.microsoft.com` pages are intermittently wrapped by authorization/cache shells
- where direct `en-us` pages were unstable, locale/fallback URLs were used to validate method shape and responses
- method-level pages provided consistent path, verb, request/response, and `ErrorResponse` details

## Surface Summary

Reviewed operation groups and methods:

- Watchers (8):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Start`, `Stop`, `Update`
- Targets (4):
  - `Create Or Update`, `Delete`, `Get`, `List By Watcher`
- Alert Rule Resources (4):
  - `Create Or Update`, `Delete`, `Get`, `List By Parent`
- Health Validations (3):
  - `Start Validation`, `Get`, `List By Parent`
- Shared Private Link Resources (4):
  - `Create`, `Delete`, `Get`, `List By Watcher`
- Operations (1):
  - `List`

Total reviewed methods: `24`.

## Service Naming

Canonical Stackyard naming:

- service key: `databasewatcher_2025_01_02`
- display name: `Azure Database Watcher`
- API version: `2025-01-02`

Recommended repo paths:

- implementation: `internal/server/provider_azure_databasewatcher_2025_01_02.go`
- tests: `internal/server/provider_azure_databasewatcher_2025_01_02_test.go`
- example: `examples/azure/databasewatcher-2025-01-02`
- plan doc: `docs/databasewatcher-2025-01-02-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DatabaseWatcher/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DatabaseWatcher/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DatabaseWatcher/...`

Representative routes:

- `GET /azure/providers/Microsoft.DatabaseWatcher/operations?api-version=2025-01-02`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}?api-version=2025-01-02`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}/start?api-version=2025-01-02`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}/stop?api-version=2025-01-02`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}/targets/{targetName}?api-version=2025-01-02`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}/alertRuleResources/{alertRuleResourceName}?api-version=2025-01-02`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}/healthValidations/{healthValidationName}/startValidation?api-version=2025-01-02`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DatabaseWatcher/watchers/{watcherName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}?api-version=2025-01-02`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- API paths follow standard ARM resource-id envelope patterns
- list methods return `value` and optional `nextLink`
- common error envelope is `ErrorResponse` (`error: ErrorDetail`, with nested `ErrorAdditionalInfo`)

Observed response patterns:

- create/update methods generally return `200` / `201`
- watcher action methods (`start`, `stop`) and health-validation start support sync or async responses (`200` / `202`)
- delete behavior is idempotent and method-specific:
  - `watchers/delete`: `202` / `204`
  - `shared-private-link-resources/delete`: `202` / `204`
  - `targets/delete`: `200` / `204`
  - `alert-rule-resources/delete`: `200` / `204`
- provider operations list returns `200` with `OperationListResult`

Notable payload constraints from reviewed pages:

- watcher names and child resource names use regex constraints
- target payloads are discriminator-based (`SqlDb`, `SqlEp`, `SqlMi` property variants)
- alert-rule proxy resources require template/version/creation metadata
- shared private link resources require `privateLinkResourceId`, `groupId`, and `requestMessage`
- health validation start uses explicit action route: `/startValidation`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `databasewatcher_2025_01_02` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Database Watcher routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Microsoft.DatabaseWatcher/` families
- strict `api-version=2025-01-02` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Database Watcher route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Watchers and Provider Operations

Implement:

- Watchers: `CreateOrUpdate`, `Delete`, `Get`, `ListByResourceGroup`, `ListBySubscription`, `Start`, `Stop`, `Update`
- Operations: `List`

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic watcher status transitions (`Starting`, `Running`, `Stopping`, `Stopped`)
- async-friendly action/delete responses where documented (`202` with `Location`/`Retry-After`)

Exit criteria:

- core watcher lifecycle and monitoring actions no longer return `NotImplemented`

### Stage 3: Targets

Implement:

- Targets: `CreateOrUpdate`, `Delete`, `Get`, `ListByWatcher`

Behavior:

- discriminator-aware payload normalization for target property variants
- deterministic list ordering and paging behavior
- idempotent delete semantics (`200`/`204`)

Exit criteria:

- watcher-to-target management flows are covered end-to-end

### Stage 4: Alert Rule Resources

Implement:

- Alert Rule Resources: `CreateOrUpdate`, `Delete`, `Get`, `ListByParent`

Behavior:

- required creation metadata validation (`alertRuleTemplateId`, `alertRuleTemplateVersion`, etc.)
- deterministic readback and list behavior
- idempotent delete semantics (`200`/`204`)

Exit criteria:

- alert-rule proxy resource lifecycle is modeled and testable

### Stage 5: Shared Private Link Resources

Implement:

- Shared Private Link Resources: `Create`, `Delete`, `Get`, `ListByWatcher`

Behavior:

- required request validation for private link identifiers/group metadata
- deterministic status modeling (`Pending`, `Approved`, `Rejected`, `Disconnected`)
- async delete semantics (`202`/`204`)

Exit criteria:

- shared private link setup and teardown flows are represented

### Stage 6: Health Validations and Contract Hardening

Implement:

- Health Validations: `StartValidation`, `Get`, `ListByParent`

Behavior:

- deterministic validation-run state transitions (`NotStarted`, `Running`, `Succeeded`, `Failed`, `TimedOut`, `Canceled`)
- issues payload modeling for validation findings
- async `startValidation` behavior support (`200`/`202`)

Testing scope:

- happy-path lifecycle across all six operation groups
- API-version/method enforcement and route ownership
- validation failures for malformed/unsupported payload variants
- deterministic paging, ordering, and status transitions

Exit criteria:

- no `NotImplemented` responses for reviewed 24-method surface
- service has dedicated tests and passes strict coverage checks

## Coverage Script Integration

Add aliases for `databasewatcher_2025_01_02` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:

- `databasewatcher_2025_01_02`
- `databasewatcher-2025-01-02`
- `databasewatcher`
- `database_watcher`
- `database-watcher`
- `azure_databasewatcher`
- `microsoft_databasewatcher`
- `rest_databasewatcher_2025_01_02`

Map plan doc:

- `SERVICE_PLAN_DOCS["databasewatcher_2025_01_02"] = ["databasewatcher-2025-01-02-plan.md"]`

## Example Plan

Add `examples/azure/databasewatcher-2025-01-02`:

- `main.go`: Azure Go SDK (`azcore/runtime`) authenticated client that exercises representative operations from every reviewed operation group
- `Dockerfile`: builds and runs the example binary
- `docker-compose.yml`: local stack wiring with Database Watcher env vars

Example coverage target:

- request coverage for all six operation groups
- lifecycle and action coverage for watcher/target/alert/shared-link surfaces
- provider operations and health-validation flows

## Validation Commands

- `go test ./internal/server -run 'TestAzureDatabaseWatcher20250102' -count=1`
- `go build ./examples/azure/databasewatcher-2025-01-02`
- `python3 scripts/azure-contract-coverage.py --service databasewatcher --require-service databasewatcher_2025_01_02 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service databasewatcher --require-service databasewatcher_2025_01_02 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service databasewatcher --require-service databasewatcher_2025_01_02 --fail-on any`

Expected stage-0 result:

- router and docs/coverage wiring pass service discovery checks
- strict coverage checks fail until implementation/tests are added in later stages
