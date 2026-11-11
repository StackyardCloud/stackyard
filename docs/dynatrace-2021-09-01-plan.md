# Azure Dynatrace (`dynatrace_2021_09_01`) Staged Plan

## Objective

Emulate Azure Dynatrace Resource Manager REST API (`api-version=2021-09-01`) with deterministic local behavior for:

- monitor lifecycle and monitor action/list surfaces
- monitor SSO configuration lifecycle
- monitor tag-rule lifecycle
- provider operations discovery endpoint
- an Azure Go SDK example under `examples/azure/dynatrace-2021-09-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/dynatrace/operation-groups?view=rest-dynatrace-2021-09-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/operations?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/single-sign-on?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/tag-rules?view=rest-dynatrace-2021-09-01`

Method pages reviewed directly: all linked methods from all operation groups (`22` total).

Monitors methods (`13`):

- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/create-or-update?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/delete?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/get?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/get-account-credentials?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/get-sso-details?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/get-vm-host-payload?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/list-app-services?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/list-by-resource-group?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/list-by-subscription-id?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/list-hosts?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/list-linkable-environments?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/list-monitored-resources?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/monitors/update?view=rest-dynatrace-2021-09-01`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/dynatrace/operations/list?view=rest-dynatrace-2021-09-01`

Single Sign On methods (`3`):

- `https://learn.microsoft.com/en-us/rest/api/dynatrace/single-sign-on/create-or-update?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/single-sign-on/get?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/single-sign-on/list?view=rest-dynatrace-2021-09-01`

Tag Rules methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dynatrace/tag-rules/create-or-update?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/tag-rules/delete?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/tag-rules/get?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/tag-rules/list?view=rest-dynatrace-2021-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dynatrace/tag-rules/update?view=rest-dynatrace-2021-09-01`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dynatrace/armdynatrace`

## Surface Summary

Reviewed operation groups and methods:

- Monitors (13)
- Operations (1)
- Single Sign On (3)
- Tag Rules (5)

Total reviewed methods: `22`.

## Service Naming

Canonical Stackyard naming:

- service key: `dynatrace_2021_09_01`
- display name: `Azure Dynatrace`
- API version: `2021-09-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_dynatrace_2021_09_01.go`
- tests: `internal/server/provider_azure_dynatrace_2021_09_01_test.go`
- example: `examples/azure/dynatrace-2021-09-01`
- plan doc: `docs/dynatrace-2021-09-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Dynatrace.Observability/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Dynatrace.Observability/monitors`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules`

Representative routes:

- `GET /azure/providers/Dynatrace.Observability/operations?api-version=2021-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/getAccountCredentials?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/getSSODetails?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/getVMHostPayload?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listAppServices?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Dynatrace.Observability/monitors?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listHosts?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listLinkableEnvironments?api-version=2021-09-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listMonitoredResources?api-version=2021-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations/{configurationName}?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations/{configurationName}?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations?api-version=2021-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules?api-version=2021-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2021-09-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods document non-success paths via `Other Status Codes | ErrorResponse`

Observed response patterns:

- `200` appears in `22` methods
- `201` appears in `3` methods
- `202` appears in `2` methods
- `204` appears in `2` methods
- `401` appears in `1` method (`Monitors_GetSSODetails`)

Request-body coverage from reviewed pages:

- `7` methods include request bodies
- body-bearing methods:
  - `Monitors_CreateOrUpdate`
  - `Monitors_GetSSODetails`
  - `Monitors_ListLinkableEnvironments`
  - `Monitors_Update`
  - `SingleSignOn_CreateOrUpdate`
  - `TagRules_CreateOrUpdate`
  - `TagRules_Update`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add stage-0 provider skeleton at `internal/server/provider_azure_dynatrace_2021_09_01.go`
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key resolves as `dynatrace_2021_09_01`
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Dynatrace.Observability` route families
- enforce `api-version=2021-09-01` for handled routes
- preserve behavior for unrelated Azure providers

### Stage 2: Operations + Monitor Read/List Surface

- implement `Operations_List`
- implement monitor `Get`, `ListByResourceGroup`, and `ListBySubscriptionId`
- implement monitor POST action reads:
  - `GetAccountCredentials`
  - `GetVMHostPayload`
  - `ListAppServices`
  - `ListHosts`
  - `ListMonitoredResources`

### Stage 3: Monitor Mutations + SSO/Linkable Actions

- implement monitor `CreateOrUpdate`, `Update`, and `Delete`
- implement monitor actions with request payload validation:
  - `GetSSODetails`
  - `ListLinkableEnvironments`

### Stage 4: Single Sign On + Tag Rules Surfaces

- implement `SingleSignOn` create/get/list
- implement `TagRules` create/get/list/update/delete
- model deterministic child-resource IDs, names, and list paging envelopes

### Stage 5: Tests

- add provider tests `internal/server/provider_azure_dynatrace_2021_09_01_test.go` covering:
  - API-version validation and route ownership
  - monitor lifecycle and list scope behavior
  - SSO and tag-rule lifecycle flows
  - action payload validation (`userPrincipal`, `tenantId`, `region`)
  - delete status handling (`200`/`202`/`204`)

### Stage 6: Example and Validation

- add `examples/azure/dynatrace-2021-09-01` using Azure Go SDK core pipeline
- include one call per reviewed method (`22` calls)
- verify build and coverage script discoverability

Exit criteria:

- example compiles and is ready for end-to-end runs once handler logic is implemented
- coverage scripts include `dynatrace_2021_09_01` aliases and plan-doc mapping

