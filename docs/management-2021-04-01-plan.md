# Azure Management Groups (`management-2021-04-01`) Staged Plan

## Objective

Emulate Azure management groups (`Microsoft.Management/managementGroups`): a tenant hierarchy under the tenant root group, with subscriptions placed in it. Management groups have their own handler (`internal/server/provider_azure_management.go`) rather than the shared ARM handler, because they need the hierarchy rules below.

## References

Microsoft Learn pages for the `2021-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/managementgroups/management-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/managementgroups/management-groups/get`
- `https://learn.microsoft.com/en-us/rest/api/managementgroups/management-groups/update`
- `https://learn.microsoft.com/en-us/rest/api/managementgroups/management-groups/get-descendants`
- `https://learn.microsoft.com/en-us/rest/api/managementgroups/management-group-subscriptions/create`
- `https://learn.microsoft.com/en-us/rest/api/managementgroups/check-name-availability/check-name-availability`

## Service Naming

- service key: `management`
- display name: `Azure Management Groups`
- implementation: `internal/server/provider_azure_management.go`
- tests: `internal/server/provider_azure_management_test.go`
- example: `examples/azure/management-2021-04-01/`

## Operations In Scope

All paths are under `/providers/Microsoft.Management`.

| Resource | Operations |
|---|---|
| `managementGroups` | list |
| `managementGroups/{groupId}` | create/update (PUT), update (PATCH), get (with `$expand=children` and `$recurse=true`), delete |
| `managementGroups/{groupId}/descendants` | list |
| `managementGroups/{groupId}/subscriptions` | list |
| `managementGroups/{groupId}/subscriptions/{subscriptionId}` | create (place), get, delete (move to root) |
| `checkNameAvailability` | `Invalid` / `AlreadyExists` / available |

## Behavior

- the tenant root group (ID `00000000-0000-0000-0000-000000000000`, display name `Tenant Root Group`) exists from the start; it cannot be moved or deleted
- group IDs are 1-90 letters, digits, hyphens, underscores, periods or parentheses and cannot end with a period (400 `InvalidResourceName`); display names default to the ID and are at most 90 characters
- PUT takes `properties.displayName` and `properties.details.parent.id`; PATCH takes `displayName` and `parentGroupId` (an ID or a group name); a new group's parent defaults to the root, and a GUID parent that names no group, such as the client's own tenant ID, means the root
- the parent must exist, and a group cannot move under itself or a descendant (400 `InvalidRequestContent`)
- create and update return 200 with the group; `details` reports the parent, a version that each update increments, `updatedTime` and `updatedBy`
- `$expand=children` adds the direct child groups and subscriptions, groups first; `$recurse=true` nests their children; `descendants` lists every group and subscription below a group
- a group with children cannot be deleted (400); deleting a missing group returns 204
- subscription IDs must be GUIDs (400 `InvalidSubscriptionId`); placing a subscription in a group moves it out of its previous group, and removing it moves it back to the root
- every subscription ID exists in Stackyard, so the hierarchy holds only subscriptions a client has placed or moved back to the root; the others are not listed under the root
- a missing group or a subscription not in the group returns 404 `NotFound`

Out of scope for this stage:
- hierarchy settings, entities, tenant-level checks and the `getEntities` and `tenantBackfillStatus` operations
- `$expand=path` and `ancestors`, and `$filter` on lists
- access control: any caller can change any group
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureManagementGroup -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service management`
- `python3 scripts/azure-io-contract-coverage.py --service management`
- `python3 scripts/azure-doc-contract-coverage.py --service management`
