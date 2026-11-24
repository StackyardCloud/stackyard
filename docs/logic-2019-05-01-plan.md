# Azure Logic Apps (`logic-2019-05-01`) Staged Plan

## Objective

Emulate the Azure Logic Apps (Consumption) control plane (`Microsoft.Logic`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): workflows, and integration accounts with their schemas and maps. Workflows are stored but never run.

## References

Microsoft Learn pages for the `2019-05-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/logic/workflows/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/logic/workflows/enable`
- `https://learn.microsoft.com/en-us/rest/api/logic/workflows/disable`
- `https://learn.microsoft.com/en-us/rest/api/logic/integration-accounts/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/logic/integration-accounts/list-callback-url`
- `https://learn.microsoft.com/en-us/rest/api/logic/integration-account-schemas/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/logic/integration-account-maps/create-or-update`

## Service Naming

- service key: `logic`
- display name: `Azure Logic Apps`
- implementation: `internal/server/provider_azure_logic.go`
- tests: `internal/server/provider_azure_logic_test.go`
- example: `examples/azure/logic-2019-05-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `workflows` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `enable`, `disable` |
| `integrationAccounts` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listCallbackUrl` |
| `integrationAccounts/schemas` | proxy child | create/update, get, delete, list |
| `integrationAccounts/maps` | proxy child | create/update, get, delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- workflow `state` is `Enabled` (the default) or `Disabled`; `enable` and `disable` set it
- a workflow `definition`, when set, must be an object, and its `triggers`, `actions`, `parameters` and `outputs` must be objects; the definition is not otherwise checked
- `createdTime`, `changedTime`, `version` and `accessEndpoint` (https `prod-00.{location}.logic.azure.com:443/workflows/{id}`) are server-owned
- integration accounts need `sku.name` `Free`, `Basic` or `Standard`; `listCallbackUrl` returns a URL signed with a key derived from the account
- schemas need `schemaType` `Xml` and maps need `mapType` `Xslt`, `Xslt20`, `Xslt30` or `Liquid`; both take no location and report a `contentLink` with the size of any `content`

Out of scope for this stage:
- running workflows: triggers, runs, run actions, trigger histories and `listCallbackUrl` on a trigger, which need children derived from the definition
- hiding `SecureString` parameter values
- partners, agreements, certificates, sessions, assemblies, batch configurations, integration service environments and Standard (single-tenant) Logic Apps
- workflow `validate`, `move`, `regenerateAccessKey` and `listSwagger`

## Quality Gates

- `go test ./internal/server -run TestAzureLogic -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service logic`
- `python3 scripts/azure-io-contract-coverage.py --service logic`
- `python3 scripts/azure-doc-contract-coverage.py --service logic`
