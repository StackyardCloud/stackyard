# Azure App Testing Playwright Workspaces (`playwright-resourcemanager-2025-09-01`) Staged Plan

## Objective

Emulate the Azure App Testing Playwright Workspaces ARM management API for `2025-09-01` with deterministic local behavior for:

- provider `operations`
- top-level `playwrightWorkspaces` lifecycle and discovery routes
- global `checkNameAvailability`
- subscription-location `playwrightQuotas`
- resource-scoped `playwrightWorkspaces/{name}/quotas`
- an Azure Go SDK example under `examples/azure/playwright-resourcemanager-2025-09-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Playwright workspaces under `Microsoft.LoadTestService`.
- This is not the Playwright browser/data plane itself.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2025-09-01`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/operation-groups?view=rest-playwright-resourcemanager-2025-09-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-quotas?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspace-quotas?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/operations/list?view=rest-playwright-resourcemanager-2025-09-01`

Representative linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/operations/list?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/check-name-availability?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/create-or-update?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/get?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/list-by-resource-group?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/list-by-subscription?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/update?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspaces/delete?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-quotas/get?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-quotas/list-by-subscription?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspace-quotas/get?view=rest-playwright-resourcemanager-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/resourcemanager/playwright-workspace-quotas/list-by-playwright-workspace?view=rest-playwright-resourcemanager-2025-09-01`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/playwright/armplaywright`

## Service Naming

Canonical Stackyard naming:
- service key: `playwright`
- display name: `Azure App Testing Playwright Workspaces`
- provider namespace: `Microsoft.LoadTestService`
- API version: `2025-09-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_playwright.go`
- tests: `internal/server/provider_azure_playwright_test.go`
- example: `examples/azure/playwright-resourcemanager-2025-09-01`
- plan doc: `docs/playwright-resourcemanager-2025-09-01-plan.md`

Recommended coverage aliases:
- `playwright`
- `playwright_workspaces`
- `playwright-workspaces`
- `azure_playwright`
- `azure_playwright_workspaces`
- `app_testing_playwright`
- `app_testing_playwright_workspaces`
- `microsoft_loadtestservice_playwrightworkspaces`
- `playwright_resourcemanager_2025_09_01`
- `rest_playwright_resourcemanager_2025_09_01`

## Surface Summary

The reviewed `2025-09-01` management surface exposes 4 operation groups:

1. `Operations`
2. `Playwright Workspaces`
3. `Playwright Quotas`
4. `Playwright Workspace Quotas`

Broad route families identified from the reviewed pages:

- provider discovery:
  - `/providers/Microsoft.LoadTestService/operations`
- top-level ARM resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}`
- subscription and resource-group listing:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/playwrightWorkspaces`
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces`
- global name check:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/checkNameAvailability`
- subscription-level location quotas:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas/{playwrightQuotaName}`
- workspace-scoped quotas:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}/quotas`
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}/quotas/{quotaName}`

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2025-09-01`
- top-level resources use `Microsoft.LoadTestService`

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.LoadTestService/operations?api-version=2025-09-01`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/checkNameAvailability?api-version=2025-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}?api-version=2025-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}?api-version=2025-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}?api-version=2025-09-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/playwrightWorkspaces?api-version=2025-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces?api-version=2025-09-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas?api-version=2025-09-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas/{playwrightQuotaName}?api-version=2025-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}/quotas?api-version=2025-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces/{playwrightWorkspaceName}/quotas/{quotaName}?api-version=2025-09-01`

Important routing implications from the reviewed pages:

- `playwrightWorkspaceName` is constrained to `^[a-zA-Z0-9-]{3,24}$`
- quota resources are read-only child routes and should not be modeled as writable ARM resources
- deletes are explicitly asynchronous on the method page and should return ARM-style accepted semantics

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- top-level `playwrightWorkspaces` returns standard ARM resource fields:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.dataplaneUri`
  - `properties.localAuth`
  - `properties.regionalAffinity`
  - `properties.workspaceId`
  - `properties.provisioningState`

### Operations

`Operations.List`
- provider route under `/providers/Microsoft.LoadTestService/operations`
- returns provider operations including `PlaywrightWorkspaces/Read` and `PlaywrightWorkspaces/Write`

Modeling implication:
- expose a deterministic `value` list including at least:
  - `Microsoft.LoadTestService/PlaywrightWorkspaces/Read`
  - `Microsoft.LoadTestService/PlaywrightWorkspaces/Write`
  - `Microsoft.LoadTestService/loadTests/Read`
  - `Microsoft.LoadTestService/loadTests/Write`
  - `Microsoft.LoadTestService/loadTests/Delete`

The extra `loadTests` operations appear in the official example payload, so they should be preserved rather than trimmed out.

### Playwright Workspaces

Reviewed method pages:
- `Check Name Availability`
- `Create Or Update`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`
- `Delete`

Key traits from the reviewed pages:

- `Check Name Availability`
  - `POST /subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/checkNameAvailability`
  - request body includes:
    - `name`
    - `type`
  - checks availability globally

- `Create Or Update`
  - `PUT /.../playwrightWorkspaces/{playwrightWorkspaceName}`
  - required request field:
    - `location`
  - optional resource properties include:
    - `properties.localAuth`
    - `properties.regionalAffinity`
    - `tags`
  - can return `200` or `201`
  - sample `201` includes `Azure-AsyncOperation`
  - response properties include:
    - `dataplaneUri`
    - `regionalAffinity`
    - `localAuth`
    - `workspaceId`
    - `provisioningState`

- `Update`
  - `PATCH /.../playwrightWorkspaces/{playwrightWorkspaceName}`
  - synchronous per method page
  - request body allows:
    - `properties.localAuth`
    - `properties.regionalAffinity`
    - `tags`

- `Delete`
  - `DELETE /.../playwrightWorkspaces/{playwrightWorkspaceName}`
  - asynchronous per method page
  - documented responses:
    - `202 Accepted` with `Location` and `Retry-After`
    - `204 No Content` when resource does not exist

Modeling implications:
- persist one canonical `PlaywrightWorkspace` store keyed by ARM ID
- preserve caller-provided `tags`, `localAuth`, and `regionalAffinity`
- synthesize stable values for:
  - `properties.dataplaneUri`
  - `properties.workspaceId`
  - `properties.provisioningState`
- implement `HEAD` on the resource path even if the doc page only highlights `GET`, because repo coverage often expects ARM resource existence probes
- delete should be idempotent and should prefer ARM-style accepted/no-content semantics over generic `200`

### Subscription-Level Playwright Quotas

Reviewed method pages:
- `Get`
- `List By Subscription`

Key traits from the reviewed pages:

- routes are location-scoped under subscription:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas/{playwrightQuotaName}`
- sample resource name is `ExecutionMinutes`
- response shape includes:
  - `id`
  - `name`
  - `type`
  - `systemData`
  - `properties.freeTrial.workspaceId`
  - `properties.freeTrial.state`
  - `properties.provisioningState`

Modeling implications:
- quotas can be deterministic read-only fixtures keyed by subscription and location
- at minimum expose `ExecutionMinutes`
- `freeTrial.state` should use stable enum-like values such as `Active` or `NotApplicable`

### Workspace Quotas

Reviewed method pages:
- `Get`
- `List By Playwright Workspace`

Key traits from the reviewed pages:

- routes are child resources under a workspace:
  - `/playwrightWorkspaces/{playwrightWorkspaceName}/quotas`
  - `/playwrightWorkspaces/{playwrightWorkspaceName}/quotas/{quotaName}`
- sample quota name is also `ExecutionMinutes`
- these pages are read-only and intended to report effective quotas for the workspace

Modeling implications:
- workspace quotas should be derived from the stored workspace plus a stable fixture table
- they should not be independently writable
- the first pass can keep the shape shallow but must return typed resources rather than generic placeholders

## Non-Goals

- do not emulate actual Playwright browser sessions, test runs, traces, or browser pools
- do not emulate the Playwright workspace data plane behind `dataplaneUri`
- do not build full Azure long-running-operation polling infrastructure beyond the minimum headers and stable accepted responses needed by examples and coverage
- do not over-validate enum values in the first pass unless coverage or the SDK example requires it

## Stage 0: Coverage Naming And Router Claiming

- Add a dedicated Azure Playwright ARM router under `Microsoft.LoadTestService`
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Map the plan doc for `playwright` in the doc coverage script
- Ensure the router claims:
  - `/azure/providers/Microsoft.LoadTestService/operations`
  - `/azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/checkNameAvailability`
  - `/azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/locations/{location}/playwrightQuotas*`
  - `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/playwrightWorkspaces*`

Deliverables:
- route skeleton
- alias wiring
- plan doc mapping

## Stage 1: Shared ARM Validation And Canonical Resource Model

- Implement shared path parsing and validation for:
  - `subscriptionId`
  - `resourceGroupName`
  - `location`
  - `playwrightWorkspaceName`
  - quota name path segments
- Define canonical in-memory models for:
  - `PlaywrightWorkspace`
  - `PlaywrightQuota`
  - `PlaywrightWorkspaceQuota`
- Implement common ARM helpers for:
  - `id`, `name`, `type`
  - `systemData`
  - `tags`
  - `provisioningState`

Recommended canonical fields for workspace storage:
- ARM metadata:
  - `subscriptionId`
  - `resourceGroupName`
  - `location`
  - `name`
  - `id`
  - `type`
  - `tags`
  - `systemData`
- properties:
  - `dataplaneUri`
  - `localAuth`
  - `regionalAffinity`
  - `workspaceId`
  - `provisioningState`

Deliverables:
- internal model types
- helper constructors
- validation error fixtures

## Stage 2: Provider Operations And Check Name Availability

- Implement `Operations.List`
- Implement `Check Name Availability`

Behavior notes:
- `checkNameAvailability` should reject missing `name` and `type`
- return deterministic availability behavior:
  - available for unused valid names
  - unavailable when a stored workspace already uses the name
  - optionally unavailable for a small fixed reserved-name set

Test coverage:
- operations list contains Playwright-specific entries
- name availability succeeds for unused names
- name availability reports collisions for existing names
- invalid request bodies return typed ARM errors

## Stage 3: Core Playwright Workspace Lifecycle

- Implement:
  - `Create Or Update`
  - `Get`
  - `HEAD`
  - `List By Resource Group`
  - `List By Subscription`
  - `Update`
  - `Delete`

Behavior notes:
- `PUT` should create or replace the resource
- `PATCH` should merge `tags`, `localAuth`, and `regionalAffinity`
- `GET` and list routes should return full typed workspace payloads
- `HEAD` should reflect existence cleanly
- `DELETE` should:
  - return `202 Accepted` with `Location` and `Retry-After` for existing resources
  - return `204 No Content` when the workspace is already absent

Suggested deterministic defaults:
- `properties.localAuth`: `Enabled`
- `properties.regionalAffinity`: `Enabled`
- `properties.provisioningState`: `Succeeded`
- `properties.workspaceId`: stable UUID-like synthetic value derived from ARM ID
- `properties.dataplaneUri`: stable URI derived from workspace identity

Test coverage:
- full create/get/list/update/delete lifecycle
- idempotent delete behavior
- `HEAD` resource existence
- invalid workspace names are rejected

## Stage 4: Subscription-Level Quotas

- Implement:
  - `Playwright Quotas - List By Subscription`
  - `Playwright Quotas - Get`

Behavior notes:
- quotas are keyed by `(subscriptionId, location, quotaName)`
- start with a deterministic fixture set containing at least:
  - `ExecutionMinutes`
- response should include:
  - `properties.freeTrial.workspaceId`
  - `properties.freeTrial.state`
  - `properties.provisioningState`

Test coverage:
- list returns deterministic values
- get returns the matching named quota
- unknown quota names return ARM `404`

## Stage 5: Workspace-Scoped Quotas

- Implement:
  - `Playwright Workspace Quotas - List By Playwright Workspace`
  - `Playwright Workspace Quotas - Get`

Behavior notes:
- these routes should require the parent workspace to exist
- derive quota values from a stable fixture table instead of introducing a second mutable store
- start with at least:
  - `ExecutionMinutes`

Test coverage:
- list/get succeed for an existing workspace
- parent workspace missing returns `404`
- unknown quota name returns `404`

## Stage 6: Azure Go SDK Example

- Add a new example under:
  - `examples/azure/playwright-resourcemanager-2025-09-01`
- Use:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/playwright/armplaywright`

Recommended example flow:
- create `ClientFactory`
- call `OperationsClient.NewListPager`
- call `WorkspacesClient.BeginCreateOrUpdate` or `CreateOrUpdate`
- call `WorkspacesClient.Get`
- call `WorkspacesClient.NewListByResourceGroupPager`
- call `WorkspacesClient.Update`
- call `PlaywrightQuotasClient.NewListBySubscriptionPager`
- call `PlaywrightWorkspaceQuotasClient.NewListByPlaywrightWorkspacePager`

If the generated SDK shape uses pagers plus pollers:
- keep the example typed and explicit
- prefer the smallest successful sequence that demonstrates the managed surface without needing full async wait complexity in Stackyard

Deliverables:
- `main.go`
- `Dockerfile`
- `docker-compose.yml`

## Stage 7: Coverage Integration

- Update service alias maps in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add the plan-doc mapping:
  - `playwright -> playwright-resourcemanager-2025-09-01-plan.md`
- Ensure service-specific coverage can resolve:
  - `playwright`
  - `playwright_workspaces`
  - `app_testing_playwright`

Validation target for a future implementation:
- `python3 scripts/azure-contract-coverage.py --service playwright --require-service playwright --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service playwright --require-service playwright --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service playwright --require-service playwright --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service playwright --require-service playwright --fail-on any`

## Stage Ordering Recommendation

Recommended merge sequence:

1. Stage 0 plus Stage 1
2. Stage 2
3. Stage 3
4. Stage 4 plus Stage 5
5. Stage 6 plus Stage 7

That order keeps the first merge focused on route ownership and canonical models, then lands the core workspace lifecycle before the read-only quota tails and the SDK example.

