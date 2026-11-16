# Azure Standby Pools (`standbypool-2026-04-01`) Staged Plan

## Objective

Emulate the Azure Standby Pools control plane (`Microsoft.StandbyPool`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): standby virtual machine pools and standby container group pools, with their runtime views and standby machines. Pools are always full; no machine or container group is created.

## References

Microsoft Learn pages for the `2026-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/standbypool/standby-virtual-machine-pools/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/standbypool/standby-container-group-pools/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/standbypool/standby-virtual-machine-pool-runtime-views/get`
- `https://learn.microsoft.com/en-us/rest/api/standbypool/standby-virtual-machines/list-by-standby-virtual-machine-pool-resource`

## Service Naming

- service key: `standbypool`
- display name: `Azure Standby Pools`
- implementation: `internal/server/provider_azure_standbypool.go`
- tests: `internal/server/provider_azure_standbypool_test.go`
- example: `examples/azure/standbypool-2026-04-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `standbyVirtualMachinePools` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `standbyContainerGroupPools` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `{pool}/runtimeViews` | read-only nested view | list, get (`latest`) |
| `standbyVirtualMachinePools/standbyVirtualMachines` | read-only nested view | list, get |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- pool names are 3-24 letters, digits or hyphens
- `elasticityProfile.maxReadyCapacity` is required, from 0 to 2000; `minReadyCapacity` is at most `maxReadyCapacity`; `refillPolicy` is `always`
- virtual machine pools need `virtualMachineState` `Running`, `Deallocated`, `Hibernated` or `Mix`; `Mix` needs `vmStateDistribution` percentages adding up to 100; `attachedVirtualMachineScaleSetId` must be a scale set ID
- container group pools need a container group profile ID; `subnetIds` must be subnet IDs
- the `latest` runtime view reports `maxReadyCapacity` instances in the pool's state (split by percentage for `Mix`; container groups are `Running`) and a healthy status
- a virtual machine pool with a scale set lists `maxReadyCapacity` standby machines named `{scaleSet}_{n}`, as virtual machines in the scale set's resource group
- views under a missing pool return 404 `ParentResourceNotFound`

Out of scope for this stage:
- per-zone counts, predictions (`prediction`) and dynamic sizing
- creating, refilling or handing out machines and container groups
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureStandbyPool -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service standbypool`
- `python3 scripts/azure-io-contract-coverage.py --service standbypool`
- `python3 scripts/azure-doc-contract-coverage.py --service standbypool`
