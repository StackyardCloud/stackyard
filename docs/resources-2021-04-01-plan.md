# Azure Resource Manager (`resources-2021-04-01`) Staged Plan

## Objective

Emulate the core `Microsoft.Resources` management surface that every other ARM service depends on:

- subscriptions (get, list)
- resource groups (create or update, get, check existence, update, delete, list)
- resources in a resource group (list)
- resource providers (list, get, register)

This also introduces the shared ARM tracked-resource handler (`internal/server/azure_resource_manager.go`), so later Azure services can register a resource type instead of hand-writing routing, CRUD, and error envelopes.

## References

Microsoft Learn pages cited (each URL was confirmed to resolve; request samples show `api-version=2021-04-01` for resource groups/providers and `2022-12-01` for subscriptions):

- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups`
- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups/get`
- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups/check-existence`
- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups/update`
- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups/delete`
- `https://learn.microsoft.com/en-us/rest/api/resources/resource-groups/list`
- `https://learn.microsoft.com/en-us/rest/api/resources/resources/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/resources/providers/list`
- `https://learn.microsoft.com/en-us/rest/api/resources/providers/get`
- `https://learn.microsoft.com/en-us/rest/api/resources/providers/register`
- `https://learn.microsoft.com/en-us/rest/api/resources/subscriptions/list`
- `https://learn.microsoft.com/en-us/rest/api/resources/subscriptions/get`

## Service Naming

- service key: `resources`
- display name: `Azure Resource Manager`
- implementation: `internal/server/provider_azure_resources.go` (+ shared `internal/server/azure_resource_manager.go`)
- tests: `internal/server/provider_azure_resources_test.go`

## Behavior

| Route (under `/azure`) | Methods | Notes |
|---|---|---|
| `/subscriptions` | GET | Returns the default local subscription |
| `/subscriptions/{sub}` | GET | Any ID is reported `Enabled` |
| `/subscriptions/{sub}/resourcegroups` | GET | Sorted by name |
| `/subscriptions/{sub}/resourcegroups/{rg}` | PUT, GET, HEAD, PATCH, DELETE | Names are case-insensitive; delete cascades to resources in the group |
| `/subscriptions/{sub}/resourcegroups/{rg}/resources` | GET | Resources registered through the shared ARM handler |
| `/subscriptions/{sub}/providers` | GET | Namespaces of registered resource types plus `Microsoft.Resources` |
| `/subscriptions/{sub}/providers/{ns}` | GET | Every well-formed namespace reports `Registered` |
| `/subscriptions/{sub}/providers/{ns}/register` | POST | Same as GET |

Errors use the ARM envelope `{"error":{"code","message"}}` plus the `x-ms-error-code` header:

| Condition | Status | Code |
|---|---|---|
| Missing `location` on PUT | 400 | `LocationRequired` |
| Body is not a JSON object | 400 | `InvalidRequestContent` |
| Empty `api-version` | 400 | `InvalidApiVersionParameter` |
| Changing an existing group's location | 409 | `InvalidResourceGroupLocation` |
| Group missing (GET, PATCH, DELETE) | 404 | `ResourceGroupNotFound` |
| Malformed provider namespace | 404 | `InvalidResourceNamespace` |
| Unsupported method | 405 | `MethodNotAllowed` |

Every namespace reports `Registered` because tools such as Terraform's `azurerm` provider register a fixed list of namespaces at startup and fail if any registration fails.

## Shared ARM Handler

`azure_resource_manager.go` serves any type registered with `registerAzureARMResourceType` (from an `init()` in the service's own file, so no `http.go` edit is needed per service):

- PUT (201 create / 200 update), GET, HEAD (204/404), PATCH (tags, identity, sku, merged properties), DELETE (200 / 204 when absent)
- list by resource group and by subscription
- `ResourceGroupNotFound` when the parent group does not exist, `ResourceNotFound` when the resource does not
- nested child types (`Type: "registries/webhooks"`): routed under the parent resource, `ParentResourceNotFound` when the parent is missing, deleted with the parent, and not listed at resource group scope
- proxy types (`Proxy: true`) that take no location
- extension types (`Extension: true`) at `{scope}/providers/{ns}/{type}/{name}` on a subscription, resource group or any resource; lists return only resources attached directly to the scope, the scope resource itself is not required to exist (it may belong to a service with its own store), and they are deleted with a shared-handler resource they attach to
- `POST {resourceId}/{action}` through per-type `Actions`
- provider actions (`registerAzureARMProviderAction`) at `POST /subscriptions/{sub}/providers/{ns}/{action}` and `POST /providers/{ns}/{action}`, used for each namespace's own name availability check
- optional per-type `ReadOnlyProperties` (given the client properties), `ValidateName` (400 `InvalidResourceName`), `Validate` and `DefaultChildren` hooks

Out of scope for this stage:
- deployments (needs an ARM template evaluator)
- the tags API and resource moves
- long-running-operation polling (all operations complete synchronously)
- services with their own stores (everything implemented before this handler): their resources do not require an existing resource group, are not removed by resource group deletion, and are not listed by `.../resourcegroups/{rg}/resources`. Migrating a service onto the shared handler brings it into all three.

## Quality Gates

- `go test ./internal/server -run 'TestAzureResourceGroup|TestAzureSubscriptionsAndProviders' -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service resources`
- `python3 scripts/azure-io-contract-coverage.py --service resources`
- `python3 scripts/azure-doc-contract-coverage.py --service resources`
