# Azure App Testing Playwright Workspaces Data Plane (`playwright-dataplane-2025-09-01`) Staged Plan

## Objective

Emulate the Azure App Testing Playwright Workspaces data-plane API for `2025-09-01` with deterministic local behavior for:

- access-token lifecycle under `/playwrightworkspaces/{workspaceId}/access-tokens`
- test-run creation and listing under `/playwrightworkspaces/{workspaceId}/test-runs`
- remote-browser bootstrap via the workspace `/browsers` redirect route
- an Azure Go SDK-based example under `examples/azure/playwright-dataplane-2025-09-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-playwright-dataplane-2025-09-01` is not the ARM `Microsoft.LoadTestService` management plane already implemented in this repo.
- It is the Playwright workspace service endpoint rooted at the workspace `dataplaneUri`, using workspace IDs instead of ARM resource IDs.
- This plan intentionally treats it as a separate Stackyard service from ARM `playwright`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/operation-groups?view=rest-playwright-dataplane-2025-09-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/access-tokens?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/test-runs?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/workspaces?view=rest-playwright-dataplane-2025-09-01`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/access-tokens/create-or-replace?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/access-tokens/delete?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/access-tokens/get?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/access-tokens/list?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/test-runs/create-or-update?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/test-runs/list?view=rest-playwright-dataplane-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/playwright/dataplane/workspaces/get-browsers?view=rest-playwright-dataplane-2025-09-01`

SDK selection note:
- I did not find a service-specific typed Azure Go SDK package for this Playwright data-plane surface during review.
- Plan assumption: the example should use the official Azure Go SDK core runtime pipeline (`azcore/runtime` and `azcore/policy`) rather than inventing a non-existent typed client.

## Service Naming

Canonical Stackyard naming:
- service key: `playwright_data_plane`
- display name: `Azure App Testing Playwright Workspaces Data Plane`
- API version token: `2025-09-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_playwright_data_plane.go`
- tests: `internal/server/provider_azure_playwright_data_plane_test.go`
- example: `examples/azure/playwright-dataplane-2025-09-01`
- plan doc: `docs/playwright-dataplane-2025-09-01-plan.md`

Recommended coverage aliases:
- `playwright_data_plane`
- `playwright_dataplane`
- `playwright-data-plane`
- `playwright-dataplane`
- `azure_playwright_data_plane`
- `azure_playwright_dataplane`
- `app_testing_playwright_data_plane`
- `app_testing_playwright_workspaces_data_plane`
- `rest_playwright_dataplane_2025_09_01`
- `playwright_2025_09_01_dataplane`

## Surface Summary

The reviewed REST surface is split into 3 operation groups:

1. `Access Tokens`
2. `Test Runs`
3. `Workspaces`

Total linked method pages reviewed: `7`

Method inventory:

`Access Tokens`
- `Create Or Replace`
- `Delete`
- `Get`
- `List`

`Test Runs`
- `Create Or Update`
- `List`

`Workspaces`
- `Get Browsers`

## Route Envelope

Documented route family:
- `PUT {endpoint}/playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}?api-version=2025-09-01`
- `DELETE {endpoint}/playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}?api-version=2025-09-01`
- `GET {endpoint}/playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}?api-version=2025-09-01`
- `GET {endpoint}/playwrightworkspaces/{workspaceId}/access-tokens?api-version=2025-09-01`
- `PATCH {endpoint}/playwrightworkspaces/{workspaceId}/test-runs/{runId}?api-version=2025-09-01`
- `GET {endpoint}/playwrightworkspaces/{workspaceId}/test-runs?api-version=2025-09-01`
- `GET {endpoint}/playwrightworkspaces/{workspaceId}/browsers?api-version=2025-09-01&os={os}`
- `GET {endpoint}/playwrightworkspaces/{workspaceId}/browsers?api-version=2025-09-01&os={os}&runId={runId}`

Stackyard emulation prefix should remain distinct from ARM `playwright`.

Recommended Stackyard route family:
- `/azure/playwright-data-plane/2025-09-01/playwrightworkspaces/{workspaceId}/access-tokens`
- `/azure/playwright-data-plane/2025-09-01/playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}`
- `/azure/playwright-data-plane/2025-09-01/playwrightworkspaces/{workspaceId}/test-runs`
- `/azure/playwright-data-plane/2025-09-01/playwrightworkspaces/{workspaceId}/test-runs/{runId}`
- `/azure/playwright-data-plane/2025-09-01/playwrightworkspaces/{workspaceId}/browsers`

Important routing notes from the reviewed pages:
- This API does not use the Azure ARM `subscriptions/.../providers/...` envelope.
- The upstream `endpoint` is the workspace `dataplaneUri`, documented as a regional Playwright service base URL of the form `{region}.api.playwright.microsoft.com`.
- `workspaceId`, `accessTokenId`, and `runId` are all documented as GUID-format strings using the same pattern family.
- `Get Browsers` is not a JSON-returning resource lookup; the success path is an HTTP redirect with a `location` header.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- security is OAuth2 with the Playwright data-plane scope `playwright.microsoft.com/.default`
- errors use `Azure.Core.Foundations.ErrorResponse`
- every reviewed route supports `x-ms-client-request-id`
- many reviewed routes also accept `x-ms-useragent`
- list routes return paged payloads with `value` and `nextLink`

### Access Tokens

Reviewed method pages:
- `Create Or Replace`
- `Delete`
- `Get`
- `List`

Key traits from the reviewed pages:

`Create Or Replace`
- `PUT /playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}`
- request body requires:
  - `name`
  - `expiryAt`
- `accessTokenId`
  - min length `3`
  - max length `36`
  - pattern `[A-Za-z0-9]+(-[A-Za-z0-9]+)+`
- `name`
  - min length `3`
  - max length `24`
  - pattern `^[a-zA-Z0-9-]{3,24}$`
- returns `200` on update or `201` on create
- response model includes:
  - `id`
  - `name`
  - `jwtToken`
  - `createdAt`
  - `expiryAt`
  - `state`

`Get`
- `GET /playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}`
- returns `200` with the access-token resource

`Delete`
- `DELETE /playwrightworkspaces/{workspaceId}/access-tokens/{accessTokenId}`
- returns `204 No Content`

`List`
- `GET /playwrightworkspaces/{workspaceId}/access-tokens`
- supports OData parameters:
  - `$select`
  - `$filter`
  - `$orderby`
  - `$top`
  - `$skip`
- default page size is `10`
- response shape is paged:
  - `value[]`
  - `nextLink`

Modeling implications:
- store tokens keyed by `(workspaceId, accessTokenId)`
- preserve both the user-facing `name` and the opaque `jwtToken`
- make list pagination deterministic and simple, even if OData filtering is only shallow on the first pass
- mark tokens as `Active` or `Expired` from `expiryAt` relative to emulator time
- keep delete idempotent and aligned with `204`

### Test Runs

Reviewed method pages:
- `Create Or Update`
- `List`

Key traits from the reviewed pages:

`Create Or Update`
- `PATCH /playwrightworkspaces/{workspaceId}/test-runs/{runId}`
- request content type is `application/merge-patch+json`
- request body fields include:
  - `displayName`
  - `config`
  - `ciConfig`
- `displayName`
  - min length `1`
  - max length `200`
- response is `200` or `201`
- response model includes:
  - `id`
  - `displayName`
  - `creatorId`
  - `creatorName`
  - `config.framework.name`
  - `config.framework.version`
  - `config.framework.runnerName`
  - `config.sdkLanguage`
  - `config.maxWorkers`
  - `ciConfig.providerName`
  - `ciConfig.branch`
  - `ciConfig.author`
  - `ciConfig.commitId`
  - `ciConfig.revisionUrl`
  - `summary.status`
  - `summary.billableTime`
  - `summary.numBrowserSessions`
  - `summary.maxConcurrentBrowserSessions`
  - `summary.startTime`
  - `summary.endTime`
  - `summary.duration`
  - `summary.errorMessages`

`Run Status` enum values from the reviewed definition:
- `RUNNING`
- `CLIENT_COMPLETE`
- `SERVER_COMPLETE`

`SDKLanguage` enum values from the reviewed definition:
- `JAVASCRIPT`
- `TYPESCRIPT`
- `CSHARP`

`List`
- `GET /playwrightworkspaces/{workspaceId}/test-runs`
- supports OData query parameters such as:
  - `$filter`
  - `$top`
- default page size is `10`
- response shape is paged:
  - `value[]`
  - `nextLink`

Modeling implications:
- persist test runs keyed by `(workspaceId, runId)`
- support create-via-patch semantics and return `201` on first write
- keep request merging shallow but deterministic
- synthesize `creatorId`, `creatorName`, and summary timestamps if not supplied
- expose stable list ordering and simple `nextLink` pagination

### Workspaces

Reviewed method page:
- `Get Browsers`

Key traits from the reviewed page:

`Get Browsers`
- `GET /playwrightworkspaces/{workspaceId}/browsers?os={os}`
- optional query parameter:
  - `runId`
- required query parameter:
  - `os`
- success response is a redirection, not JSON
- success headers include:
  - `location`
  - `x-ms-client-request-id`

Modeling implications:
- implement this route as a deterministic redirect to a synthetic browser endpoint URL
- preserve `runId` if supplied so browser bootstrap can be correlated with a prior test run
- validate `os` as a required enum-like value and reject missing values with modeled errors
- keep the redirect payload-free to match the reviewed contract

## State Model Recommendation

Recommended in-memory state:
- `playwrightWorkspaces`
  - keyed by `workspaceId`
  - lightweight metadata only, enough to accept data-plane traffic
- `playwrightAccessTokens`
  - keyed by `workspaceId` then `accessTokenId`
- `playwrightTestRuns`
  - keyed by `workspaceId` then `runId`

Suggested resource structs:
- `azurePlaywrightDataPlaneToken`
- `azurePlaywrightDataPlaneRun`
- `azurePlaywrightDataPlaneWorkspaceSession`

Recommended cross-linking:
- tokens remain scoped to one workspace
- runs remain scoped to one workspace
- `Get Browsers` can optionally verify that `runId` exists in the same workspace before returning the redirect

## Validation And Error Shape

Shared validation to implement early:
- require `api-version=2025-09-01`
- require non-empty `workspaceId`
- enforce documented GUID-like path constraints for:
  - `workspaceId`
  - `accessTokenId`
  - `runId`
- enforce token-name pattern and length on create/update
- require `expiryAt` and `name` for token create-or-replace
- require `os` on browser bootstrap

Error-shape guidance from the reviewed pages:
- use Azure-style:
  - `{"error":{"code":"...","message":"..."}}`
- return `x-ms-error-code` on failure responses where practical

Suggested emulator error codes:
- `InvalidRequest`
- `InvalidApiVersion`
- `WorkspaceNotFound`
- `AccessTokenNotFound`
- `TestRunNotFound`
- `InvalidWorkspaceId`
- `InvalidAccessTokenId`
- `InvalidRunId`
- `InvalidTokenName`
- `MissingRequiredParameter`

## Staged Implementation Plan

### Stage 0: Route And Coverage Wiring

Goals:
- reserve a distinct data-plane route family in the Azure router
- add service aliases to the Azure coverage scripts
- add the plan doc mapping for future doc coverage

Work:
- add router entrypoints in `internal/server/http.go`
- map service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- keep the service separate from ARM `playwright`

Exit criteria:
- coverage scripts can resolve `playwright_data_plane`
- unmatched routes no longer fall through to unrelated Azure handlers

### Stage 1: Shared Data-Plane Skeleton

Goals:
- implement the common handler, api-version validation, error writer, and local state containers

Work:
- create `internal/server/provider_azure_playwright_data_plane.go`
- add helpers for:
  - workspace path extraction
  - GUID-like ID validation
  - OData-style paging
  - deterministic timestamps
  - redirect response writing

Exit criteria:
- invalid routes fail with consistent Azure-style error payloads
- empty happy-path fixtures can be returned without `NotImplemented`

### Stage 2: Access Tokens

Goals:
- complete all token CRUD/list paths first because they are compact and strongly typed

Work:
- implement:
  - `PUT access-tokens/{accessTokenId}`
  - `GET access-tokens/{accessTokenId}`
  - `DELETE access-tokens/{accessTokenId}`
  - `GET access-tokens`
- persist `jwtToken`, `name`, `createdAt`, `expiryAt`, `state`
- add OData-lite handling for `$top`, `$skip`, and stable ordering

Exit criteria:
- token routes pass strict endpoint coverage
- list returns `value` plus `nextLink`
- no token route returns `NotImplemented`

### Stage 3: Test Runs

Goals:
- implement the core run-tracking surface with deterministic merge-patch behavior

Work:
- implement:
  - `PATCH test-runs/{runId}`
  - `GET test-runs`
- preserve or synthesize:
  - `config`
  - `ciConfig`
  - `summary`
  - creator metadata
- make first write return `201`, subsequent write return `200`

Exit criteria:
- test-run list/create coverage passes
- enum-bearing response fixtures include valid `status` and `sdkLanguage`
- no test-run route returns `NotImplemented`

### Stage 4: Browser Bootstrap Redirect

Goals:
- implement the only non-JSON success route in the reviewed surface

Work:
- implement:
  - `GET browsers?os={os}[&runId={runId}]`
- validate `os`
- optionally validate `runId` if present
- return a deterministic redirect target in the `location` header

Exit criteria:
- route returns redirect semantics instead of JSON
- redirect is stable and coverage-friendly

### Stage 5: Example And Tests

Goals:
- add an example and focused tests that exercise each reviewed method family

Work:
- add:
  - `internal/server/provider_azure_playwright_data_plane_test.go`
  - `examples/azure/playwright-dataplane-2025-09-01/main.go`
  - matching `Dockerfile`
  - matching `docker-compose.yml`
- example should use Azure SDK core pipeline requests against Stackyard:
  - create/list/get/delete token
  - create/list run
  - get browser redirect

Exit criteria:
- focused Go tests pass
- example builds cleanly

### Stage 6: Full Coverage Validation

Goals:
- prove the new service is discoverable and not falling through to default handlers

Work:
- run the new service through:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
  - `scripts/azure-endpoint-coverage.py`

Exit criteria:
- all endpoints for the new service return success or modeled contract/service errors rather than `NotImplemented`
- service aliases resolve cleanly in all Azure coverage scripts

## Testing Strategy

Focused server tests should cover:
- token create/get/list/delete
- token update via repeated `PUT`
- token expiration-state derivation
- token paging behavior
- test-run create then list
- repeated run patch preserving prior fields
- browser redirect success with and without `runId`
- validation failures for missing `os`, bad IDs, and bad token names
- `api-version` rejection

Example expectations:
- use the workspace `dataplaneUri` and `workspaceId` produced by the ARM Playwright workspace emulator when possible
- otherwise allow fixed local test fixtures for a purely data-plane example

## Implementation Risks

Main risks from the reviewed surface:
- `Get Browsers` is redirect-based and easy to mishandle as a JSON endpoint
- list routes advertise OData query semantics; implementing too little could cause coverage mismatches if examples rely on paging
- the data plane depends conceptually on workspace IDs and endpoints emitted by the ARM plane, so the emulator should keep those conventions aligned

Recommended mitigation:
- keep first-pass OData support intentionally small but deterministic
- align workspace ID format and endpoint shape with the already-implemented ARM Playwright workspace provider
- add focused tests for redirect headers and paged list payloads early

## Definition Of Done

The service is done when:
- `playwright_data_plane` is routed separately from ARM `playwright`
- all 7 reviewed method pages have concrete emulator behavior
- an example exists under `examples/azure/playwright-dataplane-2025-09-01`
- the Azure coverage scripts know the new service aliases
- focused tests pass
- the Azure coverage scripts confirm the service no longer returns `NotImplemented`
