# Azure Datadog (`datadog_2025_06_11`) Staged Plan

## Objective

Emulate Azure Datadog REST API (`api-version=2025-06-11`) with deterministic local behavior for:

- Datadog monitor lifecycle, key-management, and utility action flows
- monitored-subscription lifecycle and update flows
- tag-rules and single-sign-on configuration surfaces
- marketplace agreement and billing information surfaces
- subscription-eligibility and organization resubscribe flows
- provider operations discovery
- an Azure Go SDK example under `examples/azure/datadog-2025-06-11`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/datadog/operation-groups?view=rest-datadog-2025-06-11`

Operation-group pages reviewed:

- `.../billing-info?view=rest-datadog-2025-06-11`
- `.../creation-supported?view=rest-datadog-2025-06-11`
- `.../marketplace-agreements?view=rest-datadog-2025-06-11`
- `.../monitored-subscriptions?view=rest-datadog-2025-06-11`
- `.../monitors?view=rest-datadog-2025-06-11`
- `.../operations?view=rest-datadog-2025-06-11`
- `.../organizations?view=rest-datadog-2025-06-11`
- `.../single-sign-on-configurations?view=rest-datadog-2025-06-11`
- `.../tag-rules?view=rest-datadog-2025-06-11`

Method pages reviewed directly include:

- Billing Info:
  - `billing-info/get`
- Creation Supported:
  - `creation-supported/get`
  - `creation-supported/list`
- Marketplace Agreements:
  - `marketplace-agreements/create-or-update`
  - `marketplace-agreements/list`
- Monitored Subscriptions:
  - `monitored-subscriptions/createor-update`
  - `monitored-subscriptions/delete`
  - `monitored-subscriptions/get`
  - `monitored-subscriptions/list`
  - `monitored-subscriptions/update`
- Monitors:
  - `monitors/create`
  - `monitors/delete`
  - `monitors/get`
  - `monitors/get-default-key`
  - `monitors/list`
  - `monitors/list-api-keys`
  - `monitors/list-by-resource-group`
  - `monitors/list-hosts`
  - `monitors/list-linked-resources`
  - `monitors/list-monitored-resources`
  - `monitors/refresh-set-password-link`
  - `monitors/set-default-key`
  - `monitors/update`
- Operations:
  - `operations/list`
- Organizations:
  - `organizations/resubscribe`
- Single Sign On Configurations:
  - `single-sign-on-configurations/create-or-update`
  - `single-sign-on-configurations/get`
  - `single-sign-on-configurations/list`
- Tag Rules:
  - `tag-rules/create-or-update`
  - `tag-rules/get`
  - `tag-rules/list`

SDK cross-check used for operation-group parity:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/datadog/armdatadog`
  - `NewBillingInfoClient`
  - `NewCreationSupportedClient`
  - `NewMarketplaceAgreementsClient`
  - `NewMonitoredSubscriptionsClient`
  - `NewMonitorsClient`
  - `NewOperationsClient`
  - `NewOrganizationsClient`
  - `NewSingleSignOnConfigurationsClient`
  - `NewTagRulesClient`

Notes on source behavior in this environment:

- operation-group and method pages intermittently show authorization/cache wrappers
- despite wrapper text, method route, request/response, and error details are readable and were verified
- operation-group listing was used to ensure every method page in scope was visited

## Surface Summary

Reviewed operation groups and methods:

- Billing Info (1):
  - `Get`
- Creation Supported (2):
  - `Get`, `List`
- Marketplace Agreements (2):
  - `Create Or Update`, `List`
- Monitored Subscriptions (5):
  - `Createor Update`, `Delete`, `Get`, `List`, `Update`
- Monitors (13):
  - `Create`, `Delete`, `Get`, `Get Default Key`, `List`, `List Api Keys`, `List By Resource Group`, `List Hosts`, `List Linked Resources`, `List Monitored Resources`, `Refresh Set Password Link`, `Set Default Key`, `Update`
- Operations (1):
  - `List`
- Organizations (1):
  - `Resubscribe`
- Single Sign On Configurations (3):
  - `Create Or Update`, `Get`, `List`
- Tag Rules (3):
  - `Create Or Update`, `Get`, `List`

Total reviewed methods: `31`.

## Service Naming

Canonical Stackyard naming:

- service key: `datadog_2025_06_11`
- display name: `Azure Datadog`
- API version: `2025-06-11`

Recommended repo paths:

- implementation: `internal/server/provider_azure_datadog_2025_06_11.go`
- tests: `internal/server/provider_azure_datadog_2025_06_11_test.go`
- example: `examples/azure/datadog-2025-06-11`
- plan doc: `docs/datadog-2025-06-11-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Datadog/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Datadog/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Datadog/...`

Representative routes:

- `GET /azure/providers/Microsoft.Datadog/operations?api-version=2025-06-11`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}?api-version=2025-06-11`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}?api-version=2025-06-11`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}/monitoredSubscriptions/{configurationName}?api-version=2025-06-11`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}/monitoredSubscriptions/{configurationName}?api-version=2025-06-11`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2025-06-11`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}/singleSignOnConfigurations/{configurationName}?api-version=2025-06-11`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}/resubscribe?api-version=2025-06-11`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Datadog/monitors/{monitorName}/getBillingInfo?api-version=2025-06-11`
- `PUT /azure/subscriptions/{sub}/providers/Microsoft.Datadog/agreements/default?api-version=2025-06-11`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.Datadog/subscriptionStatuses/default?api-version=2025-06-11&datadogOrganizationId={orgId}`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- API paths follow ARM resource-id and action-route conventions
- list responses use `value` and optional `nextLink`
- common error envelope is `ErrorResponse` with nested error detail/additional-info

Observed response patterns:

- resource create/update methods commonly return `200`/`201`, with some `202` async accepts
- delete methods are idempotent and commonly return `200`, `202`, or `204`
- action methods (`getBillingInfo`, `getDefaultKey`, `setDefaultKey`, `listHosts`, etc.) return `200`
- documented async headers include `Location`, `Azure-AsyncOperation`, and `Retry-After` depending on method

Notable request/flow specifics:

- monitored-subscription config and tag-rule resource names commonly use a `default` convention
- creation-supported queries require `datadogOrganizationId`
- organizations resubscribe requires request-body selection of subscription/resource-group/sku
- single-sign-on and tag-rules are monitor-child resources with dedicated configuration payloads

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `datadog_2025_06_11` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Datadog routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Microsoft.Datadog/` path families
- strict `api-version=2025-06-11` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Datadog route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Monitors and Provider Operations

Implement:

- Monitors: all 13 methods
- Operations: `List`

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic list ordering and paging behavior
- async-friendly create/update/delete handling where documented

Exit criteria:

- core monitor lifecycle and monitor utility actions no longer return `NotImplemented`

### Stage 3: Monitored Subscriptions, Tag Rules, and SSO

Implement:

- Monitored Subscriptions: all 5 methods
- Tag Rules: all 3 methods
- Single Sign On Configurations: all 3 methods

Behavior:

- deterministic child-resource addressing under monitor scope
- stable readback for subscription, tag-rule, and SSO configuration payloads
- idempotent delete semantics and documented async handling

Exit criteria:

- monitor child-resource configuration surfaces are modeled end-to-end

### Stage 4: Billing, Marketplace, Creation-Supported, and Organizations

Implement:

- Billing Info: `Get`
- Marketplace Agreements: both methods
- Creation Supported: both methods
- Organizations: `Resubscribe`

Behavior:

- deterministic organization/marketplace state projection
- request validation for org-id query and resubscribe payload fields
- stable response payloads for downstream portal/workflow automation

Exit criteria:

- all 9 Datadog operation groups are represented

### Stage 5: Testing and Contract Hardening

Add tests for:

- happy-path lifecycle across all 31 methods
- API-version/method enforcement and route ownership
- request-body/query validation failures
- deterministic pagination and ordering
- async status/header behavior and delete idempotency

Exit criteria:

- `provider_azure_datadog_2025_06_11_test.go` covers stage 1-4 behavior
- service passes strict contract/IO/doc coverage checks

## Coverage Script Integration

Add aliases for `datadog_2025_06_11` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:

- `datadog_2025_06_11`
- `datadog-2025-06-11`
- `datadog`
- `data_dog`
- `data-dog`
- `azure_datadog`
- `microsoft_datadog`
- `rest_datadog_2025_06_11`

Map plan doc:

- `SERVICE_PLAN_DOCS["datadog_2025_06_11"] = ["datadog-2025-06-11-plan.md"]`

## Example Plan

Add `examples/azure/datadog-2025-06-11`:

- `main.go`: Azure Go SDK (`azcore/runtime`) authenticated client that exercises representative operations from each Datadog operation group
- `Dockerfile`: builds and runs the example binary
- `docker-compose.yml`: local stack wiring with Datadog env vars

Example coverage target:

- one representative request per reviewed method family
- create/read/list/update/delete coverage for monitor + child resources
- action coverage for billing, keys, host/resource listing, and resubscribe

## Validation Commands

- `go test ./internal/server -run 'TestAzureDatadog20250611' -count=1`
- `go build ./examples/azure/datadog-2025-06-11`
- `python3 scripts/azure-contract-coverage.py --service datadog --require-service datadog_2025_06_11 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datadog --require-service datadog_2025_06_11 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datadog --require-service datadog_2025_06_11 --fail-on any`

Expected stage-0 result:

- router and docs/coverage wiring pass service discovery checks
- strict coverage checks fail until implementation/tests are added in later stages
