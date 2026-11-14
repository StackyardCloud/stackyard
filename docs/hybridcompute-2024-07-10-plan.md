# Azure Arc-enabled Servers (`hybridcompute-2024-07-10`) Staged Plan

## Objective

Emulate the Azure Arc-enabled servers control plane (`Microsoft.HybridCompute`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): machines and their VM extensions. Machines report `Connected` at once; no Connected Machine agent runs and extensions install nothing.

## References

Microsoft Learn pages for the `2024-07-10` API version:

- `https://learn.microsoft.com/en-us/rest/api/hybridcompute/machines/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridcompute/machines/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridcompute/machine-extensions/create-or-update`

## Service Naming

- service key: `hybridcompute`
- display name: `Azure Arc-enabled Servers`
- implementation: `internal/server/provider_azure_hybridcompute.go`
- tests: `internal/server/provider_azure_hybridcompute_test.go`
- example: `examples/azure/hybridcompute-2024-07-10/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `machines` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `machines/extensions` | tracked child | create/update, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- `kind`, when set, must be `AVS`, `HCI`, `SCVMM`, `VMware`, `EPS`, `GCP` or `AWS`; `osType`, when set, must be `Windows` or `Linux` (any case)
- `status` (`Connected`), `vmId` and `lastStatusChange` are server-owned
- extensions need `publisher` and `type`; `protectedSettings` is accepted but never returned

Out of scope for this stage:
- the Connected Machine agent: no heartbeat, instance metadata or agent upgrade
- patch assessment and installation, run commands, licenses, license profiles, gateways, private link scopes and network profiles
- extension upgrades and the extension instance view
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureHybridCompute -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service hybridcompute`
- `python3 scripts/azure-io-contract-coverage.py --service hybridcompute`
- `python3 scripts/azure-doc-contract-coverage.py --service hybridcompute`
