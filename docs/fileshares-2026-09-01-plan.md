# Azure File Shares (`fileshares-2026-09-01`) Staged Plan

## Objective

Emulate Azure Files top-level file shares (`Microsoft.FileShares`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): NFS file shares on the provisioned v2 billing model, with snapshots, private endpoint connections and redundancy conversion, and the per-region limit, recommendation, usage and name checks. No data is stored: a share is its provisioning.

## References

Microsoft Learn pages for the `2026-09-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/update`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/get`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/delete`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/list-by-parent`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/start-redundancy-conversion`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-shares/get-redundancy-conversion-status`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-share/get-limits`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-share/get-provisioning-recommendation`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-share/get-usage-data`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-share-snapshot/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-share-snapshot/update`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/file-share-snapshot/list`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/private-endpoint-connections/create`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/private-endpoint-connections/list-by-file-share`
- `https://learn.microsoft.com/en-us/rest/api/fileshares/private-link-resources/list`

Provisioning rules come from:

- `https://learn.microsoft.com/en-us/azure/storage/files/understanding-billing` (provisioned v2 limits, recommendation formulas, guardrails, bursting, the 24-hour decrease wait, and SSD NFS as the only `Microsoft.FileShares` offer)
- `https://learn.microsoft.com/en-us/azure/storage/files/storage-files-scale-targets` (10,000 shares per subscription per region, 200 snapshots per share)

## Service Naming

- service key: `fileshares`
- display name: `File Shares`
- implementation: `internal/server/provider_azure_fileshares.go`, `internal/server/provider_azure_fileshares_validate.go`
- tests: `internal/server/provider_azure_fileshares_test.go`
- example: `examples/azure/fileshares-2026-09-01/`

## Operations In Scope

All paths are under `/subscriptions/{id}`.

| Resource | Operations |
|---|---|
| `resourceGroups/{rg}/providers/Microsoft.FileShares/fileShares` | create, get, list by resource group and subscription, update (PATCH), delete; POST `startRedundancyConversion`, `getRedundancyConversionStatus` |
| `.../fileShares/{name}/fileShareSnapshots` | create, get, list, update (PATCH), delete |
| `.../fileShares/{name}/privateEndpointConnections` | create, get, list, delete |
| `.../fileShares/{name}/privateLinkResources` | get, list |
| `providers/Microsoft.FileShares/locations/{location}` | POST `checkNameAvailability`, `getLimits`, `getProvisioningRecommendation`, `getUsageData` |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- share names are 3 to 63 lowercase letters, digits and single hyphens, starting and ending with a letter or digit; `mountName` defaults to the name and follows the same rules
- shares need `provisionedStorageGiB` (32 to 262,144) and `redundancy` (`Local` or `Zone`); `mediaTier` is `SSD` and `protocol` is `NFS` (the defaults), and `scaleMode` defaults to `Compatibility`
- IOPS and throughput default to the recommendations `MIN(MAX(3000 + storage, 3000), 102400)` and `MIN(MAX(100 + CEILING(0.1 * storage), 100), 10340)`, and can be set from 3,000 IOPS and 100 MiB/s up to 5 times the recommendation (capped at the maximums)
- `includedBurstIOPerSec` is `MIN(MAX(3 * IOPS, 10000), 102400)` and `maxBurstIOPerSecCredits` is `(burst - IOPS) * 3600`
- raising a provisioned quantity sets its `...NextAllowedDowngrade` 24 hours ahead, and lowering it before then is rejected
- `mediaTier`, `protocol`, `scaleMode`, `mountName` and `redundancy` can't change; PATCH takes only the update properties (provisioning, NFS settings, public access, network access, encryption)
- a subscription can have 10,000 shares per region, and a share 200 snapshots; share names are unique per subscription and region
- snapshots record `snapshotTime` at creation and take an initiator and string metadata; PATCH changes metadata only
- private endpoint connections need a connection state (`Pending`, `Approved` or `Rejected`) and are listed on the share; the private link resource is the `fileshare` group
- `startRedundancyConversion` changes the share's redundancy at once (409 if it already has it), and `getRedundancyConversionStatus` reports it `Complete` (404 before any conversion)
- `getLimits` reports the limits and provisioning constants above, `getProvisioningRecommendation` the recommendations for a storage size, `getUsageData` the shares in the subscription and region, and `checkNameAvailability` invalid names and names used in the subscription and region

Out of scope for this stage:
- file data, mounting and the NFS data plane
- creating private endpoint connections from a Microsoft.Network private endpoint
- long-running operation polling and gradual conversions: every operation completes synchronously
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzureFileShares -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service fileshares`
- `python3 scripts/azure-io-contract-coverage.py --service fileshares`
- `python3 scripts/azure-doc-contract-coverage.py --service fileshares`
