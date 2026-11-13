# Azure Site Recovery (`recoveryservicessiterecovery-2024-10-01`) Staged Plan

## Objective

Emulate the Azure Site Recovery control plane inside a Recovery Services vault (`Microsoft.RecoveryServices/vaults/replication*`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). The vault itself is covered by `docs/recoveryservices-2024-04-01-plan.md`.

## References

Microsoft Learn pages for the `2024-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/site-recovery/replication-fabrics/create`
- `https://learn.microsoft.com/en-us/rest/api/site-recovery/replication-protection-containers/create`
- `https://learn.microsoft.com/en-us/rest/api/site-recovery/replication-protection-container-mappings/create`
- `https://learn.microsoft.com/en-us/rest/api/site-recovery/replication-policies/create`

## Service Naming

- service key: `recoveryservicessiterecovery`
- display name: `Azure Site Recovery`
- implementation: `internal/server/provider_azure_recoveryservicessiterecovery.go`
- tests: `internal/server/provider_azure_recoveryservicessiterecovery_test.go`
- example: `examples/azure/recoveryservicessiterecovery-2024-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `vaults/replicationFabrics` | proxy child | create, get, delete, list |
| `.../replicationFabrics/{fabric}/replicationProtectionContainers` | proxy child | create, get, delete, list |
| `.../replicationProtectionContainers/{container}/replicationProtectionContainerMappings` | proxy child | create, get, update (PATCH), delete, list |
| `vaults/replicationPolicies` | proxy child | create, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- fabrics need `customDetails.instanceType` (`Azure`, `HyperVSite`, `VMM`, `VMware`, `VMwareV2`, `InMageRcm` or `AzureStackHCI`), and Azure fabrics also need `customDetails.location`; fabrics report `fabricType`, `health` `Normal`, `friendlyName` and an `internalIdentifier`
- policies need `providerSpecificInput.instanceType`; mappings need a `policyId` and `providerSpecificInput.instanceType`
- `providerSpecificInput` is never returned: policies and mappings echo it as `providerSpecificDetails`, and a PATCH without input keeps the stored details
- containers report `role` `Primary`; mappings report `state` `Paired` and `health` `Normal`
- deleting a fabric removes its containers and mappings

Out of scope for this stage:
- replication protected items, failover, recovery plans, networks and network mappings, storage classifications, vCenters, jobs, events, alert and vault settings
- the vault-wide lists (`replicationProtectionContainers`, `replicationProtectionContainerMappings` and so on)
- checking that `policyId` and `targetProtectionContainerId` name existing resources
- the fabric `remove` (graceful delete) and other POST operations, and long-running operation polling

## Quality Gates

- `go test ./internal/server -run TestAzureSiteRecovery -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service recoveryservicessiterecovery`
- `python3 scripts/azure-io-contract-coverage.py --service recoveryservicessiterecovery`
- `python3 scripts/azure-doc-contract-coverage.py --service recoveryservicessiterecovery`
