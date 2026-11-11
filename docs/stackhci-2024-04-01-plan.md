# Azure Stack HCI (`stackhci-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Stack HCI ARM management API exposed by the reviewed `rest-stackhci-2024-04-01` pages with deterministic local behavior for:

- provider `operations`
- top-level `clusters` lifecycle and cluster-scoped actions
- cluster-scoped `arcSettings`
- nested `extensions` under `arcSettings`
- cluster-scoped `deploymentSettings`
- cluster-scoped `updates`
- cluster-scoped `updateSummaries`
- nested `updateRuns` under `updates`
- `{resourceUri}`-scoped `edgeDevices`
- an Azure Go SDK example under `examples/azure/stackhci-2024-04-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn pages are the Azure Stack HCI ARM management plane under `Microsoft.AzureStackHCI`.
- This is not the guest, Arc-enabled server, or data-plane surface.
- The primary implementation target should therefore be `management.azure.com` routes with `api-version=2024-04-01`.

Important surface-shape note:

- The provider `operations` page for `2024-04-01` also advertises adjacent resource families such as `VirtualMachines`, `VirtualNetworks`, `VirtualHardDisks`, `NetworkInterfaces`, `GalleryImages`, `VirtualMachines/Extensions`, and `RegisteredSubscriptions`.
- I did not rely on those operation-list-only families as the core implementation target for this plan because the operation-group pages I reviewed center on `clusters`, `arcSettings`, `extensions`, `deploymentSettings`, `updates`, `updateRuns`, `updateSummaries`, and `edgeDevices`.
- The plan includes a final reconciliation stage for those additional families after the reviewed operation-group surface is green.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/stackhci/operation-groups?view=rest-stackhci-2024-04-01`

Operation-group and representative linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/stackhci/operations/list?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/create?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/get?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/list-by-resource-group?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/update?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/configure-remote-support?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/create-identity?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/clusters/upload-certificate?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings/create?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings/get?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings/create-identity?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings/generate-password?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings/initialize-disable-process?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/arc-settings/update?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/extensions?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/extensions/create?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/extensions/upgrade?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/deployment-settings?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/deployment-settings/create-or-update?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/deployment-settings/get?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/updates?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/updates/post?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/update-summaries?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/update-summaries/get?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/update-runs?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/update-runs/get?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/edge-devices?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/edge-devices/create-or-update?view=rest-stackhci-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/stackhci/edge-devices/validate?view=rest-stackhci-2024-04-01`

Azure Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/azurestackhci/armazurestackhci/v2`

SDK selection note:

- The official Go package clearly covers `operations`, `clusters`, `arcSettings`, and `extensions`.
- The reviewed Learn surface for `2024-04-01` also includes `deploymentSettings`, `updates`, `updateSummaries`, `updateRuns`, and `edgeDevices`.
- Plan assumption: use typed SDK clients for the parts the package exposes cleanly, and fall back to `azcore/runtime` requests in the example only if a reviewed `2024-04-01` family is not exposed by the generated package version available in the repo at implementation time.

## Recommended Repo Layout

- provider implementation:
  - `internal/server/provider_azure_stackhci.go`
- focused tests:
  - `internal/server/provider_azure_stackhci_test.go`
- example:
  - `examples/azure/stackhci-2024-04-01`

Coverage script updates:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Canonical service key recommendation:

- `stackhci`

## API Surface Summary

The reviewed pages show three main route shapes:

- subscription/provider routes:
  - `GET /providers/Microsoft.AzureStackHCI/operations`
- standard ARM tracked resources nested under resource groups and clusters:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureStackHCI/clusters/{clusterName}`
  - child collections beneath clusters such as `arcSettings`, `deploymentSettings`, `updates`, and `updateSummaries`
- `{resourceUri}`-based child routes:
  - `/{resourceUri}/providers/Microsoft.AzureStackHCI/edgeDevices/{edgeDeviceName}`

Modeling implications:

- the emulator should maintain a canonical cluster store keyed by ARM ID
- most child resources should validate parent cluster existence before mutating state
- `edgeDevices` need a generic parent-resource lookup keyed by normalized `resourceUri`
- several routes are action-style `POST` operations that can be answered synchronously with stable state transitions unless the example or SDK path requires explicit async polling

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use `ErrorResponse`
- list routes return `value`
- tracked resources return standard ARM fields such as:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.*`

### Operations

`Operations.List`

- route:
  - `/providers/Microsoft.AzureStackHCI/operations`
- returns `OperationListResult`
- the example payload includes operations not only for the reviewed cluster families, but also for other HCI-adjacent resource types

Modeling implications:

- expose a deterministic operation list broad enough to match the families implemented in the emulator
- include the reviewed cluster, arc, extension, deployment, update, update-summary, update-run, and edge-device verbs first

### Clusters

Reviewed method pages:

- `Create`
- `Get`
- `List By Resource Group`
- `Update`
- `Configure Remote Support`
- `Create Identity`
- `Upload Certificate`

Key traits from the reviewed pages:

- cluster resource path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AzureStackHCI/clusters/{clusterName}`
- create requires `location` and `identity.type`
- cluster properties include AAD identifiers, cloud management endpoint, desired properties, log collection, remote support, and software assurance data
- cluster action routes are `POST` under the cluster resource

Modeling implications:

- persist canonical clusters keyed by ARM ID
- support `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-resource-group, and list-by-subscription
- stage the following action endpoints as property mutations on the stored cluster:
  - `configureRemoteSupport`
  - `createIdentity`
  - `extendSoftwareAssuranceBenefit`
  - `triggerLogCollection`
  - `uploadCertificate`
- keep action results deterministic and lightweight even if the real service is long-running

### Arc Settings

Reviewed method pages:

- `Create`
- `Get`
- `Create Identity`
- `Generate Password`
- `Initialize Disable Process`
- `Update`
- operation-group index also lists:
  - `Consent And Install Default Extensions`
  - `Delete`
  - `List By Cluster`

Key traits from the reviewed pages:

- parented beneath a cluster:
  - `/clusters/{clusterName}/arcSettings/{arcSettingName}`
- action-style routes emit specialized payloads rather than generic ARM resources

Modeling implications:

- keep one canonical arc-setting store keyed by cluster child ARM ID
- support create/get/list/update/delete for normal lifecycle coverage
- model action routes separately:
  - identity creation returns a deterministic identity payload
  - password generation returns a deterministic secret-shaped payload
  - disable-process and consent/install actions mutate flags on the same stored arc-setting record

### Extensions

Reviewed method pages:

- `Create`
- `Get`
- `Upgrade`
- operation-group index also lists:
  - `Delete`
  - `List By Arc Setting`
  - `Update`

Key traits from the reviewed pages:

- nested beneath:
  - `/clusters/{clusterName}/arcSettings/{arcSettingName}/extensions/{extensionName}`
- lifecycle is conventional ARM-style CRUD with an additional `upgrade` action

Modeling implications:

- store extensions keyed by full child ARM ID
- allow pragmatic patch/update semantics for mutable properties such as auto-upgrade and settings blobs
- `upgrade` should transition version/provisioning metadata in a stable, obvious way

### Deployment Settings

Reviewed method pages:

- `Create Or Update`
- `Get`
- operation-group index also lists:
  - `Delete`
  - `List By Clusters`

Key traits from the reviewed pages:

- route:
  - `/clusters/{clusterName}/deploymentSettings/{deploymentSettingsName}`
- the request body is large and infrastructure-oriented, including intent/network/storage wiring and deployment observability details

Modeling implications:

- avoid full deep validation of every nested field on the first pass
- validate parent cluster existence and a small set of high-signal structural requirements
- persist the full request body shape so read-after-write and example flows are stable

### Updates, Update Summaries, and Update Runs

Reviewed method pages:

- `Updates.Post`
- `Update Summaries.Get`
- `Update Runs.Get`
- operation-group indexes also list:
  - `Updates`: `Delete`, `Get`, `List`, `Put`
  - `Update Summaries`: `Delete`, `Get`, `List`, `Put`
  - `Update Runs`: `Delete`, `Get`, `List`, `Put`

Key traits from the reviewed pages:

- all of these are cluster-scoped resource families
- `Updates.Post` is an action-oriented "apply update" flow
- `UpdateRuns` are nested under a specific update

Modeling implications:

- treat `updates` as canonical stored resources
- derive `updateSummaries` and `updateRuns` from update state where possible rather than storing three disconnected models
- seed one available update and allow `POST` to move it into an applied/in-progress state with a corresponding run record

### Edge Devices

Reviewed method pages:

- `Create Or Update`
- `Validate`
- search result pages also reviewed for:
  - `Get`
  - `List`

Key traits from the reviewed pages:

- routes are anchored by `resourceUri`, not by subscription/resource-group path decomposition
- create requires:
  - `kind: HCI`
  - `properties` payload
- `validate` is a long-running action route

Modeling implications:

- build a generic helper for `{resourceUri}` parsing and normalization
- key edge-device state by `{resourceUri,edgeDeviceName}`
- answer `validate` with a deterministic success payload and optionally a synthetic operation status field if coverage expects action semantics

## Staged Implementation

### Stage 0: Router and Coverage Wiring

Add the service shell and discovery:

- add `internal/server/provider_azure_stackhci.go`
- register the new router in `internal/server/http.go`
- add `Server` storage maps for:
  - clusters
  - arc settings
  - extensions
  - deployment settings
  - updates
  - update summaries
  - update runs
  - edge devices
- add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Acceptance target:

- service is discoverable by the coverage scripts
- no routed Stack HCI path falls through to `NotImplemented`

### Stage 1: Provider and Core Clusters

Implement:

- `GET /providers/Microsoft.AzureStackHCI/operations`
- cluster `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`
- cluster list-by-resource-group
- cluster list-by-subscription

Modeling notes:

- seed one cluster with stable ARM metadata and representative `properties`
- keep response shapes broad enough for future child-resource references

Acceptance target:

- base resource lifecycle works
- tests cover bad `api-version`, not-found, list, and read-after-write

### Stage 2: Cluster Actions

Implement pragmatic stateful versions of:

- `configureRemoteSupport`
- `createIdentity`
- `extendSoftwareAssuranceBenefit`
- `triggerLogCollection`
- `uploadCertificate`

Modeling notes:

- action responses should mutate the parent cluster record where it makes sense
- avoid over-modeling certificates or support sessions; use stable synthetic payloads

Acceptance target:

- action routes stop returning `NotImplemented`
- action-triggered state is visible on subsequent `GET cluster`

### Stage 3: Arc Settings and Extensions

Implement:

- `arcSettings` create/get/list/update/delete
- arc-setting actions:
  - `createIdentity`
  - `generatePassword`
  - `initializeDisableProcess`
  - `consentAndInstallDefaultExtensions`
- `extensions` create/get/list/update/delete
- `extensions/upgrade`

Modeling notes:

- default one arc-setting per cluster is sufficient for coverage-first emulation
- extension upgrade should clearly alter extension version or status fields

Acceptance target:

- nested cluster child resources are fully wired
- tests cover parent validation, nested list/get, and action flows

### Stage 4: Deployment and Update Families

Implement:

- `deploymentSettings` create/get/list/delete
- `updates` get/list/put/delete/post
- `updateSummaries` get/list/put/delete
- `updateRuns` get/list/put/delete

Modeling notes:

- persist the deployment-settings body largely as-submitted
- derive summary/run state from the update store where practical
- treat `Updates.Post` as the main action endpoint that drives update-run creation

Acceptance target:

- deployment and update coverage no longer falls through to `NotImplemented`
- update runs are visible after an update action

### Stage 5: `{resourceUri}` Children

Implement:

- `edgeDevices` create/get/list/delete
- `edgeDevices/validate`

Modeling notes:

- normalize `resourceUri` so equivalent parent IDs do not create duplicate stores
- support at least one seeded parent path for deterministic tests and example output

Acceptance target:

- `{resourceUri}`-based routes resolve cleanly
- validate action has stable success behavior

### Stage 6: Example

Add `examples/azure/stackhci-2024-04-01`:

- prefer `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/azurestackhci/armazurestackhci/v2`
- if the package available in the repo does not expose every reviewed `2024-04-01` family, use the typed SDK for clusters/arc/extensions and use raw `azcore/runtime` requests for deployment/update/edge-device routes

Suggested example flow:

1. list provider `operations`
2. create a cluster
3. get and patch the cluster
4. configure remote support
5. create an arc setting
6. create and upgrade an extension
7. create deployment settings
8. list and apply updates
9. get update summaries and update runs
10. create and validate an edge device

Acceptance target:

- example builds cleanly
- example exercises the core reviewed surface without private hooks

### Stage 7: Provider-Operations Reconciliation Tail

After the reviewed operation-group pages are green, reconcile the additional families advertised by `Operations.List`, including:

- `VirtualMachines`
- `VirtualNetworks`
- `VirtualHardDisks`
- `NetworkInterfaces`
- `GalleryImages`
- `VirtualMachines/Extensions`
- `RegisteredSubscriptions`

Rationale:

- these appear in the provider operation list for `2024-04-01`
- they may require additional direct Learn-page verification during implementation because the operation-group pages reviewed for this plan focused on the cluster-centered families above

Acceptance target:

- either implement those tails in a follow-up slice or document precisely why they belong to a separate HCI sub-surface

## Test Plan

Focused provider tests should cover:

- provider route detection and invalid `api-version`
- cluster create/get/list/update/delete lifecycle
- cluster action routes mutating visible state
- arc-setting and extension nested-resource lifecycles
- deployment-settings read-after-write
- update application producing summaries and runs
- `{resourceUri}`-based edge-device create/list/get/validate flows
- not-found behavior for unknown cluster, child resource, and parent `resourceUri`

## Validation Plan After Implementation

- `go test ./internal/server -run 'TestAzureStackHCI' -count=1`
- `go build ./examples/azure/stackhci-2024-04-01`
- `python3 scripts/azure-contract-coverage.py --service stackhci --require-service stackhci --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service stackhci --require-service stackhci --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service stackhci --require-service stackhci --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service stackhci --require-service stackhci --fail-on any`

## Risks and Follow-Ups

- The `2024-04-01` Stack HCI surface is broad, and `Operations.List` advertises more resource types than were cleanly surfaced by the reviewed operation-group pages. The implementation should therefore land in slices rather than as one large provider.
- `deploymentSettings` has a very large nested schema. Coverage-first emulation should preserve shape and a few high-signal validations rather than fully reproducing every infrastructure rule.
- The official Go SDK package clearly covers clusters/arc/extensions, but may lag some of the newer `2024-04-01` child families. The example may need a mixed typed-plus-raw approach.
