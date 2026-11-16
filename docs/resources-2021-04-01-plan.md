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
- extension types (`Extension: true`) at `{scope}/providers/{ns}/{type}/{name}` on a subscription, resource group or any resource; lists return only resources attached directly to the scope, the scope resource itself is not required to exist (it may belong to a service with its own store), and they are deleted with a shared-handler resource they attach to; a type can restrict the scopes it attaches to (`ValidateScope`, 400 `InvalidScope`), and its `Actions` and `Transitions` are served at `POST {scope}/providers/{ns}/{type}/{name}/{action}`
- `POST {resourceId}/{action}` through per-type `Actions`, or through `Transitions`, which merge server-owned properties into the stored resource (such as `state` on start/stop) and return 200 with no body
- provider actions (`registerAzureARMProviderAction`) at `POST /subscriptions/{sub}/providers/{ns}/{action}` and `POST /providers/{ns}/{action}`, used for each namespace's own name availability check; actions registered as `locations/{action}` are served at `POST /subscriptions/{sub}/providers/{ns}/locations/{location}/{action}`
- resource group actions (`registerAzureARMGroupAction`) at `POST /subscriptions/{sub}/resourceGroups/{rg}/providers/{ns}/locations/{location}/{action}`, for name checks scoped to one resource group; a missing group returns 404 `ResourceGroupNotFound`
- read-only views (`registerAzureARMView`) computed from emulator state, served for GET at `{scope}/providers/{ns}/{collection}` and `.../{collection}/{name}`, where the scope is a tenant, subscription, resource group or resource; other methods return 405 `MethodNotAllowed`, unless an extension type shares the collection (such as quota limits, listed with their defaults but written by PUT), in which case they go to it, a missing resource group returns 404 `ResourceGroupNotFound`, and a name with no item directly under the scope returns 404 `ResourceNotFound`
- nested views (`registerAzureARMView(ns, "{type}/{collection}", ...)`) under a stored resource of that type, at `{resourceId}/{collection}` and `.../{collection}/{name}`, such as a standby pool's runtime views; a missing resource returns 404 `ParentResourceNotFound`
- PATCH body hooks (`PatchBody`) for APIs whose update model differs from the resource, such as support tickets' top-level fields or tags-only updates; they reshape and check the body (400 `InvalidRequestContent`) and may set `StateProperties` values, which PATCH otherwise keeps
- subscription types (`Subscription: true`) at `/subscriptions/{id}/providers/{ns}/{type}/{name}` outside any resource group, with child types, such as support tickets; one may also be `Tenant`, served at both scopes, and tenant lists show only tenant-scope resources
- server actions (`ServerActions`), like `Actions` but given the server, for steps that change other records, such as moving a collection's resources; nested views check that their parent exists only when it is a stored type
- linked resources (`Linked`): a PUT hook, run under the store lock, that returns server-owned properties and resources ARM creates outside the record's ID, such as a private endpoint's network interface; they are stored with it, dropped when a later PUT no longer returns them, and deleted with it, and the hook can reject the PUT (400 `InvalidRequestContent`); PATCH on such a type returns 405 `MethodNotAllowed`, since it would leave the linked resources stale, as it does for types marked `NoPatch` whose API has none
- `KeepLocation` keeps the optional `location` a client sends to a proxy or extension type, and `CreateStatus` replaces 201 for a PUT that creates a resource, for APIs that never answer 201
- inline children (`InlineChildren`), such as a virtual network's `subnets`: a parent PUT that sets the property validates each item like a child PUT and replaces the child resources with them (deleting any not listed); a PUT without it (or with null) keeps them, rejecting the PUT if `ValidateInParent` fails for any of them against the new payload; item names cannot contain `/`, `?` or `#`; PATCH never changes them; and parent responses always list the current children, with `id`, `name`, `type` and `properties`, after any child PUT, PATCH, transition or DELETE
- parent-aware validation (`ValidateInParent`), given the parent's properties (including the inline sibling list) and the child's name, for rules such as a subnet staying inside its network's address space; inline items are checked against the parent payload the same way
- tenant types (`Tenant: true`) at `/providers/{ns}/{type}/{name}` outside any subscription, such as subscription aliases; they are proxy resources and are not served under a subscription
- scope actions (`registerAzureARMScopeAction`) at `POST {scope}/providers/{ns}/{path}`, where the scope is a subscription, resource group or resource and the path may have several segments, such as `policyStates/latest/queryResults`; other methods return 405 `MethodNotAllowed` and a missing resource group returns 404 `ResourceGroupNotFound`
- name existence checks (`registerAzureARMNameCheck`) at `HEAD /providers/{ns}/{collection}/{name}`: 200 when a resource of the registered type has that name in any subscription, 404 otherwise
- optional per-type `ReadOnlyProperties` (given the client properties and the final ID, creation and update times), `ValidateName` (400 `InvalidResourceName`), `Validate` and `DefaultChildren` hooks
- optional per-type `WriteOnlyProperties`: dotted property paths, such as passwords, accepted on PUT and PATCH but never stored or returned; a segment ending in `[]` applies to every element of an array, as in `roles[].password`
- optional per-type `StateProperties` with their initial values, such as `state` `Ready`, or `nil` for a property a new resource lacks: only `Transitions` change them, so client values on PUT and PATCH are ignored and the stored value is kept

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
