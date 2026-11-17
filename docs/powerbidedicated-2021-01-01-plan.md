# Azure Power BI Embedded (`powerbidedicated-2021-01-01`) Staged Plan

## Objective

Emulate Power BI Embedded (`Microsoft.PowerBIDedicated`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): dedicated capacities that scale, suspend and resume, and the auto scale v-cores that add capacity to them. No reports run; a capacity is only its SKU, administrators, mode and state.

## References

Microsoft Learn pages for the `2021-01-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/create`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/update`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/suspend`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/resume`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/list-skus`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/list-skus-for-capacity`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/capacities/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/auto-scale-v-cores/create`
- `https://learn.microsoft.com/en-us/rest/api/power-bi-embedded/auto-scale-v-cores/update`

## Service Naming

- service key: `powerbidedicated`
- display name: `Power BI Embedded`
- implementation: `internal/server/provider_azure_powerbidedicated.go`
- tests: `internal/server/provider_azure_powerbidedicated_test.go`
- example: `examples/azure/powerbidedicated-2021-01-01/`

## Operations In Scope

All paths are under `/subscriptions/{id}` unless noted.

| Resource | Operations |
|---|---|
| `resourceGroups/{rg}/providers/Microsoft.PowerBIDedicated/capacities` | create, get, list by resource group and subscription, update (PATCH), delete; POST `suspend`, `resume`; GET `skus` |
| `resourceGroups/{rg}/providers/Microsoft.PowerBIDedicated/autoScaleVCores` | create, get, list by resource group and subscription, update (PATCH), delete |
| `providers/Microsoft.PowerBIDedicated/skus` | list |
| `providers/Microsoft.PowerBIDedicated/locations/{location}/checkNameAvailability` | POST |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- capacity names are 3 to 63 lowercase letters and digits starting with a letter; v-core names are 3 to 63 letters, digits, hyphens or underscores
- capacities take an A SKU (`A1` to `A6`, tier `PBIE_Azure`, filled in when omitted), optional administrators (at least one identity when given) and a mode (`Gen1` or `Gen2`, default `Gen2`); PATCH takes the SKU, tags, administrators and mode
- `suspend` pauses a capacity (`state` and `provisioningState` `Paused`) and `resume` restarts it (`Succeeded`); PUT and PATCH keep the state, and clients can't set it
- a capacity's `skus` lists the A SKUs other than its own; `checkNameAvailability` reports invalid names and names used by any capacity
- v-cores take the `AutoScale` SKU (tier `AutoScale`, filled in when omitted, capacity at least 0), a capacity limit of at least 0 and the capacity's object ID (a GUID); PATCH takes the SKU, tags and capacity limit

Out of scope for this stage:
- EM and P SKUs, workloads and capacity usage
- linking a v-core to a stored capacity: the object ID is checked only for its shape
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzurePowerBIDedicated -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service powerbidedicated`
- `python3 scripts/azure-io-contract-coverage.py --service powerbidedicated`
- `python3 scripts/azure-doc-contract-coverage.py --service powerbidedicated`
