# Azure Arc Site Manager (`sitemanager-2023-07-01-preview`) Staged Plan

## Objective

Emulate Azure Arc Site Manager (`2023-07-01-preview`) with deterministic local behavior for the ARM management-plane surface under `Microsoft.Edge`, including:

- provider `Operations`
- `Sites`
- a Go example built with the Azure Go SDK runtime pipeline
- Azure coverage script wiring for the new service key

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/operation-groups?view=rest-sitemanager-2023-07-01-preview`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/operations?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites?view=rest-sitemanager-2023-07-01-preview`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/operations/list?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites/list-by-subscription?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites/list-by-resource-group?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites/get?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites/create-or-update?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites/update?view=rest-sitemanager-2023-07-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/sitemanager/sites/delete?view=rest-sitemanager-2023-07-01-preview`

Azure Go SDK approach selected for implementation:
- use the Azure Go SDK core runtime pipeline (`github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`) for the example

SDK package note:
- I did not find a stable, service-specific track-2 Go management client package for this preview `Microsoft.Edge` Site Manager surface while reviewing the available references, so the example should use the Azure Go SDK runtime pipeline directly rather than inventing a non-existent typed client.

## Service Naming

Canonical Stackyard naming:
- service key: `sitemanager`
- display name: `Azure Arc Site Manager`
- API version: `2023-07-01-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_sitemanager.go`
- tests: `internal/server/provider_azure_sitemanager_test.go`
- example: `examples/azure/sitemanager-2023-07-01-preview`
- plan doc: `docs/sitemanager-2023-07-01-preview-plan.md`

## Surface Summary

The reviewed surface is compact and split into 2 operation groups:

1. `Operations`
2. `Sites`

Total linked method pages reviewed: `7`

Method inventory:

`Operations`
- `List`

`Sites`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.Edge/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Edge/sites`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Edge/sites`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Edge/sites/{siteName}`

Target query:
- `api-version=2023-07-01-preview`

Important routing implications:
- `Sites` only exposes create/get/update/delete at resource-group scope
- subscription scope is list-only
- the provider namespace is `Microsoft.Edge`

## Contract Notes From Reviewed Pages

Shared management-plane traits:
- all reviewed methods require `api-version`
- non-success responses use Azure ARM `Error Response` / cloud error shapes
- list routes return `value` envelopes
- the resource model follows normal ARM conventions with `id`, `name`, `type`, `location`, `tags`, and `properties`

### Operations

`Operations - List`
- `GET /providers/Microsoft.Edge/operations`
- response returns the provider operation catalog

Modeling implication:
- implement as a fixed deterministic list advertising `read`, `write`, and `delete` operations for `Microsoft.Edge/sites`

### Sites

`Sites - Create Or Update`
- `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Edge/sites/{siteName}`
- reviewed request body includes the standard ARM resource envelope
- reviewed responses return the site resource model

`Sites - Update`
- `PATCH .../sites/{siteName}`
- reviewed request body is a partial ARM resource update
- reviewed responses return the site resource model

`Sites - Get`
- `GET .../sites/{siteName}`
- response returns the site resource model

`Sites - Delete`
- `DELETE .../sites/{siteName}`
- success responses include normal ARM delete semantics

`Sites - List By Resource Group`
- `GET .../resourceGroups/{resourceGroupName}/providers/Microsoft.Edge/sites`
- response returns a `value` list

`Sites - List By Subscription`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.Edge/sites`
- response returns a `value` list

Important modeled site fields from the reviewed pages:
- `id`, `name`, `type`, `location`, `tags`
- `kind`
- `identity`
- `extendedLocation`
- `properties`

Modeling implications:
- a single in-memory site store keyed by full ARM ID is sufficient
- create can require only `location` and accept pass-through `properties`
- patch can allow partial updates for `tags`, `kind`, `identity`, `extendedLocation`, and `properties`
- the emulator should synthesize `properties.provisioningState = Succeeded`

## Staged Implementation Plan

### Stage 0: Coverage And Planning Wiring

Add repo wiring before server code:
- add `docs/sitemanager-2023-07-01-preview-plan.md`
- add `sitemanager` aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map the plan doc in `SERVICE_PLAN_DOCS`

Exit criteria:
- coverage scripts discover `sitemanager`
- doc coverage sees the official Learn references

### Stage 1: Provider Skeleton And Operations

Implement:
- `handleAzureSiteManagerRouter`
- path detection for `/providers/Microsoft.Edge/operations`
- `Operations - List`
- API-version validation
- shared request-body parsing helpers

Exit criteria:
- operations endpoint returns deterministic provider metadata
- invalid `api-version` returns `400`

### Stage 2: Core Sites CRUD And Listing

Implement:
- `List By Subscription`
- `List By Resource Group`
- `Get`
- `Create Or Update`
- `Update`
- `Delete`

State model:
- store canonical site resources by full ARM ID
- preserve `location`, `kind`, `tags`, `identity`, `extendedLocation`, and arbitrary `properties`

Validation baseline:
- `location` required on `PUT`
- malformed JSON rejected with `400`
- missing site returns `404`

Exit criteria:
- all six `Sites` methods return non-`NotImplemented`
- resource lifecycle is deterministic and list routes reflect stored state

### Stage 3: Tests

Add focused provider tests covering:
- operations list
- site create/get/head/list by RG/list by subscription/patch/delete
- validation failures
- not-found behavior
- invalid `api-version`

Exit criteria:
- `go test ./internal/server -run 'TestAzureSiteManager' -count=1` passes

### Stage 4: Example

Add `examples/azure/sitemanager-2023-07-01-preview`:
- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example flow:
- list operations
- create site
- get site
- list by resource group
- patch site
- delete site

SDK approach:
- use `azcore/policy` + `azcore/runtime` for a small management-plane client

Exit criteria:
- `go build ./examples/azure/sitemanager-2023-07-01-preview` passes

### Stage 5: Validation

Run:
- `go test ./internal/server -run 'TestAzureSiteManager' -count=1`
- `go build ./examples/azure/sitemanager-2023-07-01-preview`
- `python3 scripts/azure-contract-coverage.py --service sitemanager --require-service sitemanager --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service sitemanager --require-service sitemanager --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service sitemanager --require-service sitemanager --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service sitemanager --require-service sitemanager --fail-on any`
- `make ci`

Success target:
- all `sitemanager` endpoints return successful or modeled contract-valid responses
- no `sitemanager` endpoint returns `NotImplemented`
- if `make ci` fails, remaining failures should be unrelated existing repo issues
