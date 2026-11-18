# Azure Lab Services (`labservices-2023-06-07`) Staged Plan

## Objective

Emulate Azure Lab Services (`Microsoft.LabServices`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): lab plans with their defaults and images, and labs with their users, schedules and virtual machines. Publishing a lab creates a virtual machine per seat. Nothing runs: machines only change state, and invitations are never emailed.

## References

Microsoft Learn pages for the `2023-06-07` API version:

- `https://learn.microsoft.com/en-us/rest/api/labservices/lab-plans/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/lab-plans/update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/lab-plans/delete`
- `https://learn.microsoft.com/en-us/rest/api/labservices/lab-plans/save-image`
- `https://learn.microsoft.com/en-us/rest/api/labservices/images/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/images/update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/images/list-by-lab-plan`
- `https://learn.microsoft.com/en-us/rest/api/labservices/labs/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/labs/update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/labs/publish`
- `https://learn.microsoft.com/en-us/rest/api/labservices/labs/sync-group`
- `https://learn.microsoft.com/en-us/rest/api/labservices/users/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/users/update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/users/invite`
- `https://learn.microsoft.com/en-us/rest/api/labservices/schedules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/schedules/update`
- `https://learn.microsoft.com/en-us/rest/api/labservices/virtual-machines/get`
- `https://learn.microsoft.com/en-us/rest/api/labservices/virtual-machines/start`
- `https://learn.microsoft.com/en-us/rest/api/labservices/virtual-machines/stop`
- `https://learn.microsoft.com/en-us/rest/api/labservices/virtual-machines/reimage`
- `https://learn.microsoft.com/en-us/rest/api/labservices/virtual-machines/redeploy`
- `https://learn.microsoft.com/en-us/rest/api/labservices/virtual-machines/reset-password`
- `https://learn.microsoft.com/en-us/rest/api/labservices/skus/list`
- `https://learn.microsoft.com/en-us/rest/api/labservices/usages/list-by-location`

## Service Naming

- service key: `labservices`
- display name: `Azure Lab Services`
- implementation: `internal/server/provider_azure_labservices.go`, `internal/server/provider_azure_labservices_validate.go`
- tests: `internal/server/provider_azure_labservices_test.go`
- example: `examples/azure/labservices-2023-06-07/`

## Operations In Scope

All paths are under `/subscriptions/{id}` unless noted.

| Resource | Operations |
|---|---|
| `resourceGroups/{rg}/providers/Microsoft.LabServices/labPlans` | create, get, list by resource group and subscription, update (PATCH), delete; POST `saveImage` |
| `.../labPlans/{plan}/images` | get, list, update (PUT and PATCH) |
| `resourceGroups/{rg}/providers/Microsoft.LabServices/labs` | create, get, list by resource group and subscription, update (PATCH), delete; POST `publish`, `syncGroup` |
| `.../labs/{lab}/users` | create, get, list, update (PATCH), delete; POST `invite` |
| `.../labs/{lab}/schedules` | create, get, list, update (PATCH), delete |
| `.../labs/{lab}/virtualMachines` | get, list; POST `start`, `stop`, `reimage`, `redeploy`, `resetPassword` |
| `providers/Microsoft.LabServices/skus` | list |
| `providers/Microsoft.LabServices/locations/{location}/usages` | list |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 1 to 100 letters, digits, hyphens, underscores or periods
- lab plans check their default connection and shutdown profiles, allowed regions, ID lengths and a `SystemAssigned` identity; PATCH takes the properties Lab Plans - Update lists
- every lab plan offers two marketplace images, disabled; PUT and PATCH only change an existing image's `enabledState` and `availableRegions`, and images can't be deleted
- `saveImage` needs a new image `name`, a virtual machine of a lab in the plan and a shared gallery on the plan, and adds a specialized image to the plan; a name already in use is a 409
- labs require their shutdown, connection, security and virtual machine profiles; the virtual machine profile needs a create option, usage quota (at most 9999 hours), image, SKU name (capacity 0 to 400) and admin user, with a password for `TemplateVM` labs; passwords are never returned
- a lab's `labPlanId` must name a stored lab plan (a PATCH naming a missing one is a 409); a new lab takes the plan's default subnet as its network profile, which can't change or be added after creation
- labs start in `Draft` with a registration code and an OS type taken from the image; PATCH takes the properties Labs - Update lists, except the create option and image
- a `TemplateVM` lab has a `template` virtual machine; `publish` sets the lab `Published` and adds a user machine per seat, named `0`, `1` and so on, dropping seats above the capacity; machines keep their state, and their connection details follow the lab on PUT and publish
- `syncGroup` needs a roster Azure AD group
- machines start `Stopped`; `start` runs them, `stop` and `reimage` stop them, `redeploy` keeps their state, and `resetPassword` needs the admin or non-admin username and a password
- users need an email address; `invite` marks the invitation sent; PATCH takes only `additionalUsageQuota`
- schedules need `stopAt` and an IANA `timeZoneId`, with `startAt` before `stopAt` and a daily or weekly recurrence (weekly needs week days)
- `skus` lists the sizes from the Skus - List sample; `usages` counts the seats of the subscription's labs in a location per SKU family

Out of scope for this stage:
- LMS roster sync, registration and claiming machines
- a regional size catalog and real quotas
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureLabServices -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service labservices`
- `python3 scripts/azure-io-contract-coverage.py --service labservices`
- `python3 scripts/azure-doc-contract-coverage.py --service labservices`
