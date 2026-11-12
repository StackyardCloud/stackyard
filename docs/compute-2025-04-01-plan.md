# Azure Compute (`compute-2025-04-01`) Staged Plan

## Objective

Emulate the Azure Compute ARM management API for `2025-04-01` with deterministic local behavior for:

- provider `operations`
- subscription and location inventory utilities such as `usage`, `resourceSkus`, VM sizes, and image catalogs
- placement and access primitives such as `availabilitySets`, `proximityPlacementGroups`, and `sshPublicKeys`
- capacity and host-placement resources such as `capacityReservationGroups`, `capacityReservations`, `dedicatedHostGroups`, and `dedicatedHosts`
- managed storage and image resources such as `disks`, `snapshots`, `images`, `galleries`, `galleryImages`, and `galleryImageVersions`
- compute-instance resources such as `virtualMachines`, `virtualMachineExtensions`, and `virtualMachineRunCommands`
- scale-set resources such as `virtualMachineScaleSets`, `virtualMachineScaleSetVMs`, and related child/action routes
- `cloudServices` and their role/instance tails
- restore-point families such as `restorePointCollections` and `restorePoints`
- an Azure Go SDK example under `examples/azure/compute-2025-04-01`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed Learn surface is the ARM management plane under `Microsoft.Compute`.
- This is not the guest OS, VM serial console, or instance metadata data plane.
- The `rest-compute-2025-04-01` index is broad and not perfectly uniform:
  - many operation groups resolve directly to `2025-04-01`
  - some linked groups or methods still point at adjacent Compute API versions such as `2025-01-02`, `2024-11-04`, `2024-03-03`, or `2021-07-01`
- The staged plan below treats those version splits as an implementation constraint, not a reason to invent a fake uniform surface.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/compute/operation-groups?view=rest-compute-2025-04-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/compute/operations?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/availability-sets?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/dedicated-host-groups?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/cloud-services?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/gallery-application-versions?view=rest-compute-2025-04-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/compute/operations/list?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/usage/list?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/resource-skus/list?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/availability-sets/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/proximity-placement-groups/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/ssh-public-keys/create?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservation-groups/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservations/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/dedicated-host-groups/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/dedicated-hosts/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/disks/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/snapshots/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/images/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/galleries/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/gallery-images/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/gallery-image-versions/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/community-galleries/get?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/shared-gallery-images/get?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/cloud-services/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-extensions/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-run-commands/list?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-set-vms/get?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-set-vm-extensions/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/restore-point-collections/create-or-update?view=rest-compute-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/compute/restore-points/create?view=rest-compute-2025-04-01`

Official Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute`

## SDK Selection Note

- The future example should prefer the official `armcompute` typed clients for core tracked resources and list routes.
- Because the Compute surface is generated in several package versions and some linked method pages still map to adjacent API versions, the example should allow a narrow `azcore/runtime` fallback only where the local `armcompute` clients do not line up cleanly with the reviewed route.
- The plan should not assume one generated client package covers every reviewed operation family at `2025-04-01`.

## Service Naming

Canonical Stackyard naming:

- service key: `compute`
- display name: `Azure Compute`
- API version token: `2025-04-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_compute.go`
- tests: `internal/server/provider_azure_compute_test.go`
- example: `examples/azure/compute-2025-04-01`
- plan doc: `docs/compute-2025-04-01-plan.md`

Recommended coverage aliases:

- `compute`
- `azure_compute`
- `microsoft_compute`
- `compute_2025_04_01`
- `rest_compute_2025_04_01`

## API Surface Summary

The reviewed pages expose several distinct route families rather than one narrow resource tree:

- provider and regional inventory:
  - `/providers/Microsoft.Compute/operations`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Compute/locations/{location}/usages`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Compute/skus`
  - location-scoped VM-size and image-catalog routes
- foundational placement and access resources:
  - `availabilitySets`
  - `proximityPlacementGroups`
  - `sshPublicKeys`
- capacity and host placement:
  - `capacityReservationGroups`
  - `capacityReservations`
  - `hostGroups`
  - `hosts`
- storage and image resources:
  - `disks`
  - `snapshots`
  - `images`
  - `galleries`
  - `galleryImages`
  - `galleryImageVersions`
  - gallery-application tails
  - community/shared gallery read routes
- compute execution resources:
  - `virtualMachines`
  - `virtualMachineExtensions`
  - `virtualMachineRunCommands`
- scale-set resources:
  - `virtualMachineScaleSets`
  - `virtualMachineScaleSetVMs`
  - scale-set extension and child action routes
- additional managed compute families:
  - `cloudServices`
  - role and role-instance inspection/action routes
  - `restorePointCollections`
  - `restorePoints`

Modeling implications:

- this provider needs several parent-child stores rather than one flat ARM map
- list and action routes are a major part of the contract, especially for VM, VMSS, disk, and gallery families
- many mutating routes are documented as long-running operations with `202 Accepted`, so the eventual emulator should reuse the repo’s ARM LRO conventions
- several catalog routes are effectively read-only fixtures and should be staged separately from tracked-resource CRUD

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style error envelopes
- list routes return `value`
- tracked resources carry standard ARM fields such as `id`, `name`, `type`, `location`, `tags`, `systemData`, and `properties.*`
- many create, update, delete, and action routes document LRO behavior via `azure-asyncoperation`, `location`, or similar headers

Key family-specific notes from reviewed pages:

- `operations`, `usage`, and `resourceSkus` are read-only and fit deterministic fixture responses
- placement resources such as `availabilitySets` and `proximityPlacementGroups` are straightforward tracked ARM resources and are good early implementation targets
- `sshPublicKeys` adds action-style behavior beyond CRUD, so it should not be merged blindly into generic ARM resource helpers
- capacity reservations and dedicated hosts require parent existence and placement validation, making them a natural second-stage hierarchy
- disks, snapshots, and images are top-level tracked resources but have richer `creationData`, encryption, and source-link fields than simple ARM stubs
- galleries form a deep tree with strong parent validation requirements across gallery, image, and version levels
- VM and VMSS contracts include many action routes beyond CRUD, so the first implementation slice should prioritize lifecycle and inspection routes before edge actions
- `cloudServices` remains a distinct Compute family with its own nested role and instance routes; it should be staged after the main VM and VMSS paths, not mixed into the first provider drop
- restore-point families are tightly tied to VM and disk lineage and should be staged after the core compute instance model exists

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the new service shell and alias wiring without claiming endpoint completeness:

- reserve service key `compute`
- add provider registration in `internal/server/http.go`
- add empty or minimal provider scaffolding in `internal/server/provider_azure_compute.go`
- add plan-doc mapping and aliases in the three Azure coverage scripts
- create the example directory scaffold under `examples/azure/compute-2025-04-01`

Exit criteria:

- routing compiles
- coverage scripts can resolve the new service key
- no endpoints are yet claimed beyond stage-ready work

### Stage 1: Provider Operations and Regional Catalogs

Implement the low-risk read-only routes first:

- `Operations.List`
- location `Usage.List`
- `ResourceSkus.List`
- core VM size and VM image catalog routes reviewed from the Compute index

Why first:

- these routes establish provider identity and regional fixture strategy
- they are read-only and unblock the example without committing to deep write behavior

### Stage 2: Foundational Placement and Access Resources

Implement the simplest tracked ARM resources:

- `availabilitySets`
- `proximityPlacementGroups`
- `sshPublicKeys`

Why here:

- these resources are independent of VM and VMSS execution state
- they establish common ARM CRUD, list, tag update, and HEAD behavior for the provider

### Stage 3: Capacity Reservation and Dedicated Host Hierarchies

Implement placement hierarchies that add parent-child validation:

- `capacityReservationGroups`
- `capacityReservations`
- `hostGroups`
- `hosts`

Why here:

- these families are more relational than Stage 2 but still narrower than VM and disk contracts
- they force correct ARM ID normalization and parent existence checks early

### Stage 4: Managed Storage and Image Resources

Implement the top-level managed artifact families:

- `disks`
- `snapshots`
- `images`

Then add the gallery tree:

- `galleries`
- `galleryImages`
- `galleryImageVersions`

Late within the stage:

- `galleryApplications`
- `galleryApplicationVersions`
- community/shared gallery read-only routes

Why here:

- these resources are important dependencies for VM and VMSS creation flows
- the gallery tree is deep enough that it deserves its own contained pass before compute-instance orchestration

### Stage 5: Virtual Machines and Immediate Child Resources

Implement the core VM contract:

- `virtualMachines` CRUD and list routes
- key action routes reviewed from the VM pages such as start, power off, restart, deallocate, redeploy, and instance view where documented in the index
- `virtualMachineExtensions`
- `virtualMachineRunCommands`

Why here:

- VM lifecycle is one of the main reasons users will exercise this provider
- it depends on earlier placement and image/storage stages but does not require VMSS modeling yet

### Stage 6: Virtual Machine Scale Sets and VMSS Child Trees

Implement the scale-set surface separately from standalone VMs:

- `virtualMachineScaleSets`
- `virtualMachineScaleSetVMs`
- `virtualMachineScaleSetVMExtensions`
- selected reviewed VMSS action routes and inspection routes

Why here:

- VMSS contracts are large and deserve isolation from standalone VM lifecycle work
- parent-child-child validation is stricter than earlier stages

### Stage 7: Cloud Services and Restore-Point Families

Implement the remaining major Compute-managed families:

- `cloudServices`
- cloud-service roles and role-instance inspection or action routes reviewed from the index
- `restorePointCollections`
- `restorePoints`

Why here:

- both families are important but less foundational than VM, disk, and gallery work
- restore-point behavior is easier to model once VM and disk resource shapes already exist

### Stage 8: Long-Tail Reconciliation and Example Finalization

Reconcile remaining reviewed families or action routes not yet covered:

- edge read-only catalog tails
- community/shared gallery gaps
- narrower action routes on VMs, VMSS, disks, galleries, or cloud services that were deferred from earlier stages
- any linked method pages from the operation-groups index that surfaced version mismatches and were intentionally postponed

Finalize:

- the example under `examples/azure/compute-2025-04-01`
- coverage aliases and doc mappings
- focused provider tests for validation and representative parent-child flows

## Example Scope

The future example should exercise a representative but not absurdly broad management flow:

- create or update an availability set or proximity placement group
- create or update a gallery or disk resource
- create or update a VM or VMSS resource
- list provider operations or regional usage to confirm read routes

Recommended example files:

- `examples/azure/compute-2025-04-01/main.go`
- `examples/azure/compute-2025-04-01/Dockerfile`
- `examples/azure/compute-2025-04-01/docker-compose.yml`

## Coverage Script Updates

When implementation begins, update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add:

- alias normalization for `compute`
- plan-doc mapping to `compute-2025-04-01-plan.md`
- any example-path or service-name hints needed by the existing coverage tooling

## Validation Plan

After implementation, validate with:

- focused provider tests for `compute`
- example build for `examples/azure/compute-2025-04-01`
- Azure contract coverage for `--service compute`
- Azure IO coverage for `--service compute`
- Azure doc coverage for `--service compute`
- Azure endpoint coverage for `--service compute`
- full `make ci`, expecting unrelated repo failures to be reported separately if they persist

## Recommended Implementation Order

If implemented incrementally, merge in this order:

1. Stage 0 and Stage 1
2. Stage 2 and Stage 3
3. Stage 4
4. Stage 5
5. Stage 6
6. Stage 7 and Stage 8

This ordering keeps the early slices small enough to verify while still front-loading the resource families that VM and VMSS creation depend on.
