# Azure Container Registry (`containerregistry-2023-07-01`) Staged Plan

## Objective

Emulate the Azure Container Registry control plane (`Microsoft.ContainerRegistry`) on the shared ARM handler (`internal/server/azure_resource_manager.go`), the first service to use its nested child types, proxy types and actions.

## References

Microsoft Learn pages for the `2023-07-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/containerregistry/registries`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/registries/create`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/registries/list-credentials`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/registries/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/replications`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/webhooks`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/scope-maps`
- `https://learn.microsoft.com/en-us/rest/api/containerregistry/tokens`

## Service Naming

- service key: `containerregistry`
- display name: `Azure Container Registry`
- implementation: `internal/server/provider_azure_containerregistry.go`
- tests: `internal/server/provider_azure_containerregistry_test.go`
- example: `examples/azure/containerregistry-2023-07-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `registries` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listCredentials` |
| `registries/replications` | tracked child | create/update, get, update, delete, list |
| `registries/webhooks` | tracked child | create/update, get, update, delete, list |
| `registries/scopeMaps` | proxy child | create/update, get, update, delete, list |
| `registries/tokens` | proxy child | create/update, get, update, delete, list |
| `providers/Microsoft.ContainerRegistry/checkNameAvailability` | action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- registry names are 5-50 alphanumeric characters (400 `InvalidResourceName`), and `checkNameAvailability` compares them case-insensitively across all subscriptions
- `sku.name` must be `Basic`, `Standard` or `Premium` (400 `InvalidRequestContent`)
- `loginServer` is server-owned: `{lower(name)}.azurecr.io`
- `listCredentials` returns the registry name as username and two passwords derived from the resource ID, only when `properties.adminUserEnabled` is true (400 `AdminUserDisabled` otherwise; the code name is an emulator choice)
- webhooks require `properties.serviceUri` and a non-empty `properties.actions`; scope maps require a non-empty `properties.actions`
- deleting a registry deletes its replications, webhooks, scope maps and tokens

Out of scope for this stage:
- the registry data plane (`/v2/` Docker distribution API), so `docker push` does not work yet
- `regenerateCredential`, `listUsages`, `importImage`, `generateCredentials`, webhook `ping`/`getCallbackConfig`/`listEvents`
- SKU-dependent limits (for example, replications require `Premium`)
- connected registries, agent pools, task runs, cache rules, credential sets and private endpoint connections

## Quality Gates

- `go test ./internal/server -run TestAzureContainerRegistry -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service containerregistry`
- `python3 scripts/azure-io-contract-coverage.py --service containerregistry`
- `python3 scripts/azure-doc-contract-coverage.py --service containerregistry`
