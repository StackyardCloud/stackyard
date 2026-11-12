# Azure Batch Management (`batchmanagement-2025-06-01`) Staged Plan

## Objective

Emulate the Azure Batch ARM management API for `2025-06-01` with deterministic local behavior for:

- provider `operations`
- location-scoped name availability, quotas, and supported VM SKU discovery
- top-level `batchAccounts` lifecycle and account-scoped utility actions
- account-scoped `applications`
- application-scoped `applicationPackages`
- account-scoped `pools`
- account-scoped `privateEndpointConnections`
- account-scoped `privateLinkResources`
- account-scoped `networkSecurityPerimeterConfigurations`
- an Azure Go SDK example under `examples/azure/batchmanagement-2025-06-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the ARM management plane for Azure Batch under `Microsoft.Batch`.
- This is not the Batch service data plane used for jobs, tasks, nodes, or runtime pool execution.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2025-06-01`.

Important documentation note:

- As of April 11, 2026, the requested operation-groups page for `rest-batchmanagement-2025-06-01` redirects to a `rest-batchmanagement-2024-07-01` fallback index.
- The individual operation-group and method pages for the main management surface do exist at `2025-06-01`, and those method pages were used for route and contract review.
- That redirected index also shows a `Certificate` group, but I did not find matching `2025-06-01` certificate method pages during review. To avoid planning against a mixed-version surface, this staged plan is scoped to the `2025-06-01` groups and methods I could verify directly: `Application`, `Application Package`, `Batch Account`, `Location`, `Network Security Perimeter`, `Operations`, `Pool`, `Private Endpoint Connection`, and `Private Link Resource`.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/operation-groups?view=rest-batchmanagement-2025-06-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application-package?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/location?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/network-security-perimeter?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/operations?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-endpoint-connection?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-link-resource?view=rest-batchmanagement-2025-06-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/operations/list?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/location/check-name-availability?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/location/get-quotas?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/location/list-supported-virtual-machine-skus?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/create?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/get?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/list?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/list-by-resource-group?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/get-keys?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/regenerate-key?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/synchronize-auto-storage-keys?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application/create?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application/get?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application/list?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application/update?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application-package/create?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application-package/activate?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application-package/get?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/application-package/list?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/create?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/get?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/list-by-batch-account?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/update?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/disable-auto-scale?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/stop-resize?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-endpoint-connection/get?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-endpoint-connection/list-by-batch-account?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-endpoint-connection/update?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-endpoint-connection/delete?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-link-resource/get?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-link-resource/list-by-batch-account?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/network-security-perimeter/get-configuration?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/network-security-perimeter/list-configurations?view=rest-batchmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchmanagement/network-security-perimeter/reconcile-configuration?view=rest-batchmanagement-2025-06-01`

Official Go SDK package reviewed:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch`

## SDK Selection Note

- The official Azure Go SDK package for this surface is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4`.
- The package’s current package page is rooted at `armbatch`, but it explicitly indicates that the highest tagged major version is `v4`.
- The package index shows typed clients covering the exact families needed for this staged plan:
  - `AccountClient`
  - `ApplicationClient`
  - `ApplicationPackageClient`
  - `LocationClient`
  - `OperationsClient`
  - `PoolClient`
  - `PrivateEndpointConnectionClient`
  - `PrivateLinkResourceClient`
- I did not find a clearly exposed `NetworkSecurityPerimeter` typed client on the reviewed package index, so the future example should prefer typed `armbatch/v4` clients where available and use `azcore/runtime` only as a small targeted fallback for NSP routes if needed.

## Service Naming

Canonical Stackyard naming:

- service key: `batch_management`
- display name: `Azure Batch Management`
- API version token: `2025-06-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_batch_management.go`
- tests: `internal/server/provider_azure_batch_management_test.go`
- example: `examples/azure/batchmanagement-2025-06-01`
- plan doc: `docs/batchmanagement-2025-06-01-plan.md`

Recommended coverage aliases:

- `batch_management`
- `batchmanagement`
- `batch-management`
- `azure_batch_management`
- `azure_batchmanagement`
- `microsoft_batch`
- `rest_batchmanagement_2025_06_01`
- `batchmanagement_2025_06_01`

## API Surface Summary

The reviewed pages expose four major route shapes:

- provider operations:
  - `/providers/Microsoft.Batch/operations`
- subscription and location utilities:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Batch/locations/{locationName}/checkNameAvailability`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Batch/locations/{locationName}/quotas`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Batch/locations/{locationName}/supportedVirtualMachineSkus`
- standard ARM tracked resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Batch/batchAccounts/{accountName}`
- nested child resources and actions under a Batch account:
  - `applications`
  - `applications/{applicationName}/versions`
  - `pools`
  - `privateEndpointConnections`
  - `privateLinkResources`
  - `networkSecurityPerimeterConfigurations`
  - account actions like `listKeys`, `regenerateKeys`, and `syncAutoStorageKeys`

Modeling implications:

- the emulator should keep one canonical Batch account store keyed by ARM ID
- most child-resource routes should validate parent account existence before mutation
- `applications`, `applicationPackages`, and `pools` behave as first-class ARM child resources with their own list/get/create/update/delete semantics
- private-link resources are read-only inventory, while private-endpoint connections are mutable child resources
- location utilities and provider operations can be modeled as seeded read-only inventory
- several Batch account methods are action routes rather than standard ARM CRUD and need stable, purpose-built fixtures

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `CloudError`
- list routes return `value`
- resource responses use standard ARM fields such as:
  - `id`
  - `name`
  - `type`
  - `etag`
  - `tags`
  - `systemData`
  - `properties.*`

Common parameter constraints visible across the reviewed pages:

- `accountName`:
  - min length `3`
  - max length `24`
  - pattern `^[a-zA-Z0-9]+$`
- `applicationName`:
  - min length `1`
  - max length `64`
  - pattern `^[a-zA-Z0-9_-]+$`
- `versionName`:
  - min length `1`
  - max length `64`
  - pattern `^[a-zA-Z0-9_-][a-zA-Z0-9_.-]*$`
- `poolName`:
  - min length `1`
  - max length `64`
  - pattern `^[a-zA-Z0-9_-]+$`

### Operations

Reviewed method page:

- `List`

Key traits from the reviewed page:

- route:
  - `/providers/Microsoft.Batch/operations`
- returns `OperationListResult`

Modeling implications:

- expose a deterministic operation list covering the families implemented in the emulator
- include read/write/delete/action verbs for accounts, applications, packages, pools, private endpoints, private links, location utilities, and NSP routes

### Location

Reviewed method pages:

- `Check Name Availability`
- `Get Quotas`
- `List Supported Virtual Machine Skus`

Key traits from the reviewed pages:

- routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Batch/locations/{locationName}/checkNameAvailability`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Batch/locations/{locationName}/quotas`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Batch/locations/{locationName}/supportedVirtualMachineSkus`
- name-availability uses `POST` with `{name, type}` and returns `nameAvailable`, optional `reason`, and optional `message`
- quotas and supported-SKU routes are read-only lists

Modeling implications:

- seed deterministic per-location quota and SKU inventory
- make name availability depend on current account names in the emulator plus basic validation

### Batch Accounts

Reviewed method pages:

- `Create`
- `Delete`
- `Get`
- `List`
- `List By Resource Group`
- `Get Keys`
- `Regenerate Key`
- `Synchronize Auto Storage Keys`
- `Update`

Key traits from the reviewed pages:

- root resource path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Batch/batchAccounts/{accountName}`
- create is a true `PUT`
- update is `PATCH`
- utility action routes include:
  - `POST .../listKeys`
  - `POST .../regenerateKeys`
  - `POST .../syncAutoStorageKeys`
- key routes only apply when `allowedAuthenticationModes` contains `SharedKey`
- list routes exist at both subscription and resource-group scope

Modeling implications:

- store `allowedAuthenticationModes`, keys, network profile, identity, and auto-storage references on the account
- key-related routes should validate shared-key eligibility rather than blindly succeeding
- list/get/update fixtures should preserve realistic ARM shape and account properties

### Applications

Reviewed method pages:

- `Create`
- `Delete`
- `Get`
- `List`
- `Update`

Key traits from the reviewed pages:

- child resource path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Batch/batchAccounts/{accountName}/applications/{applicationName}`
- update is `PATCH`
- list route supports optional `maxresults`

Modeling implications:

- applications should be first-class child resources stored separately from account properties
- pagination behavior can be deterministic and simple while still honoring `maxresults`

### Application Packages

Reviewed method pages:

- `Create`
- `Activate`
- `Delete`
- `Get`
- `List`

Key traits from the reviewed pages:

- child resource path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Batch/batchAccounts/{accountName}/applications/{applicationName}/versions/{versionName}`
- create returns an upload-oriented package record and activation is a separate action
- list route supports optional `maxresults`

Modeling implications:

- package records should track a version-level lifecycle such as pending upload, active, and deleted
- create should return stable upload metadata rather than pretending the binary was already ingested

### Pools

Reviewed method pages:

- `Create`
- `Delete`
- `Disable Auto Scale`
- `Get`
- `List By Batch Account`
- `Stop Resize`
- `Update`

Key traits from the reviewed pages:

- child resource path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Batch/batchAccounts/{accountName}/pools/{poolName}`
- action routes:
  - `POST .../disableAutoScale`
  - `POST .../stopResize`
- pool payloads are materially richer than simple ARM wrappers and include VM config, scale settings, allocation state, identity, and network details

Modeling implications:

- pools should have a dedicated typed store rather than a generic child-resource bag
- action routes should update stable pool state fields such as autoscale/resize/allocation status
- staged implementation should prioritize deterministic success fixtures over full pool behavior emulation

### Private Endpoint Connections and Private Link Resources

Reviewed method pages:

- `Private Endpoint Connection`: `Get`, `List By Batch Account`, `Update`, `Delete`
- `Private Link Resource`: `Get`, `List By Batch Account`

Key traits from the reviewed pages:

- both are account-scoped child routes
- private endpoint connections are mutable and expose provisioning plus connection-state details
- private link resources are read-only inventory

Modeling implications:

- model private endpoint connections as mutable child resources with stable `Approved`/`Pending` style state
- model private link resources as seeded read-only descriptors

### Network Security Perimeter Configurations

Reviewed method pages:

- `Get Configuration`
- `List Configurations`
- `Reconcile Configuration`

Key traits from the reviewed pages:

- route family:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Batch/batchAccounts/{accountName}/networkSecurityPerimeterConfigurations`
- `reconcile` is an action route on a named configuration

Modeling implications:

- store NSP configurations as account-scoped child resources
- `reconcile` should mutate a small stable state field, timestamp, or reconciliation result instead of attempting real NSP orchestration

## Staged Implementation Plan

### Stage 0: Router Skeleton and Coverage Wiring

- Add a dedicated Batch Management router surface under `Microsoft.Batch`.
- Add the service key and aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Map this plan doc in doc coverage for `batch_management`.

Exit criteria:

- Coverage scripts resolve `batch_management` and versioned aliases consistently.

### Stage 1: Provider Operations and Location Utilities

- Implement:
  - `GET /providers/Microsoft.Batch/operations`
  - location `checkNameAvailability`
  - location `quotas`
  - location `supportedVirtualMachineSkus`
- Add deterministic validation for `subscriptionId`, `locationName`, and `api-version`.

Exit criteria:

- Provider and location routes return stable typed fixtures and validation errors.

### Stage 2: Batch Account Core Lifecycle

- Implement account:
  - `PUT`
  - `GET`
  - `PATCH`
  - `DELETE`
  - subscription list
  - resource-group list
- Add account action routes:
  - `listKeys`
  - `regenerateKeys`
  - `syncAutoStorageKeys`

Exit criteria:

- Accounts are stored canonically by ARM ID and all reviewed account routes are contract-tested.

### Stage 3: Applications and Application Packages

- Implement `applications` CRUD/list.
- Implement `applicationPackages` create/activate/get/list/delete.
- Track package activation state separately from package creation.

Exit criteria:

- Application and package routes return stable, realistic account-child resource fixtures.

### Stage 4: Pools

- Implement `pools` create/get/list/update/delete.
- Implement pool action routes:
  - `disableAutoScale`
  - `stopResize`
- Keep pool state explicit enough to show autoscale and resize transitions.

Exit criteria:

- Pool lifecycle and action routes are covered with deterministic state transitions.

### Stage 5: Private Networking and NSP

- Implement mutable `privateEndpointConnections`.
- Implement read-only `privateLinkResources`.
- Implement `networkSecurityPerimeterConfigurations` get/list and `reconcile`.

Exit criteria:

- Account-scoped private networking and NSP routes no longer fall through to `NotImplemented`.

### Stage 6: Example and Validation

- Add `examples/azure/batchmanagement-2025-06-01` using `armbatch/v4`.
- Prefer typed clients for:
  - operations
  - location utilities
  - accounts
  - applications
  - application packages
  - pools
  - private endpoint connections
  - private link resources
- Use a small raw-pipeline fallback only if NSP routes are not cleanly exposed by `armbatch/v4`.
- Validate with:
  - focused `go test`
  - `go build ./examples/azure/batchmanagement-2025-06-01`
  - the four Azure coverage scripts for `batch_management`

Exit criteria:

- The new service example builds, focused tests pass, and the service-specific coverage scripts show no `NotImplemented` fallthroughs.
