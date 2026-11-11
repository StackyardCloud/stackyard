# Azure Device Update Data Plane (`deviceupdate_dataplane_2022_10_01`) Staged Plan

## Objective

Emulate Azure Device Update Data Plane REST API (`api-version=2022-10-01`) with deterministic local behavior for:

- update catalog import/query/delete workflows
- device/group/class/deployment management workflows
- operation-status polling and compliance surfaces
- diagnostics log-collection workflows
- an Azure Go SDK example under `examples/azure/deviceupdate-dataplane-2022-10-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/operation-groups?view=rest-deviceupdate-dataplane-2022-10-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management?view=rest-deviceupdate-dataplane-2022-10-01`

Method pages reviewed directly: all linked methods from both operation groups (`50` total).

Device Update operation-group method pages (`11`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/delete-update?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/get-file?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/get-operation-status?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/get-update?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/import-update?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/list-files?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/list-names?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/list-operation-statuses?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/list-providers?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/list-updates?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/list-versions?view=rest-deviceupdate-dataplane-2022-10-01`

Device Management operation-group method pages (`39`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/create-or-update-deployment?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/delete-deployment?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/delete-deployment-for-device-class-subgroup?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/delete-device-class?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/delete-device-class-subgroup?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/delete-group?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-best-updates-for-device-class-subgroup?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-deployment?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-deployment-for-device-class-subgroup?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-deployment-status?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-device?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-device-class?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-device-class-subgroup?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-device-class-subgroup-deployment-status?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-device-class-subgroup-update-compliance?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-device-module?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-group?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-log-collection?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-log-collection-detailed-status?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-operation-status?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-update-compliance?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/get-update-compliance-for-group?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/import-devices?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-best-updates-for-group?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-deployments-for-device-class-subgroup?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-deployments-for-group?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-device-classes?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-device-class-subgroups-for-group?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-devices?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-device-states-for-device-class-subgroup-deployment?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-groups?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-health-of-devices?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-installable-updates-for-device-class?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-log-collections?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/list-operation-statuses?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/retry-deployment?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/start-log-collection?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/stop-deployment?view=rest-deviceupdate-dataplane-2022-10-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/update-device-class?view=rest-deviceupdate-dataplane-2022-10-01`

SDK selection note:

- The current Azure SDK for Go repository contains `armdeviceupdate` (management plane), but no dedicated typed Track 2 package for this 2022-10-01 Device Update data-plane surface.
- Plan assumption: examples and emulation contract wiring should use Azure Go SDK core runtime pipeline (`azcore/runtime`, `azcore/policy`), which is still official Azure Go SDK usage.

## Surface Summary

Reviewed operation groups and methods:

- Device Update (11):
  - delete/import/get/list methods for updates, providers, names, versions, files, and operation statuses
- Device Management (39):
  - deployment, group, device class, device, compliance, diagnostics, and operation-status methods

Total reviewed methods: `50`.

## Service Naming

Canonical Stackyard naming:

- service key: `deviceupdate_dataplane_2022_10_01`
- display name: `Azure Device Update Data Plane`
- API version: `2022-10-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_deviceupdate_dataplane_2022_10_01.go`
- tests: `internal/server/provider_azure_deviceupdate_dataplane_2022_10_01_test.go`
- example: `examples/azure/deviceupdate-dataplane-2022-10-01`
- plan doc: `docs/deviceupdate-dataplane-2022-10-01-plan.md`

## Route Envelope

Upstream docs model `{endpoint}` as a service endpoint (not ARM subscription/provider envelope), with this common root:

- `/{endpoint}/deviceUpdate/{instanceId}/...`

Representative upstream routes:

- `GET {endpoint}/deviceUpdate/{instanceId}/updates?api-version=2022-10-01`
- `POST {endpoint}/deviceUpdate/{instanceId}/updates:import?api-version=2022-10-01`
- `PUT {endpoint}/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments/{deploymentId}?api-version=2022-10-01`
- `GET {endpoint}/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments/{deploymentId}/status?api-version=2022-10-01`
- `PUT {endpoint}/deviceUpdate/{instanceId}/management/deviceDiagnostics/logCollections/{operationId}?api-version=2022-10-01`

Recommended Stackyard data-plane route family:

- `/azure/deviceupdate-dataplane/2022-10-01/deviceUpdate/{instanceId}/...`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:

- all reviewed methods include `api-version=2022-10-01`
- security scope is OAuth2 (`api.adu.microsoft.com/.default`)
- method pages consistently document `Other Status Codes | ErrorResponse`
- list-style routes use paged envelopes (`value`, optional `nextLink`)

Observed response patterns:

- dominant success code is `200 OK` for get/list/read/action queries
- most management delete routes return `204 No Content`
- long-running methods include:
  - `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/import-update?view=rest-deviceupdate-dataplane-2022-10-01`: `200` or `202` with `Operation-Location`
  - `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-update/delete-update?view=rest-deviceupdate-dataplane-2022-10-01`: `202` with `Operation-Location`
  - `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/import-devices?view=rest-deviceupdate-dataplane-2022-10-01`: `202` with `Operation-Location`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/dataplane/device-management/start-log-collection?view=rest-deviceupdate-dataplane-2022-10-01` returns `201 Created`

Selected request-body requirements from reviewed pages:

- `create-or-update-deployment` requires `deploymentId`, `groupId`, `startDateTime`, and `update`
- `update-device-class` (`application/merge-patch+json`) requires `friendlyName`
- `start-log-collection` requires `deviceList`
- `import-update` body is an array of import items with `importManifest` and `files` (manifest schema reference is documented on the method page)
- `list-health-of-devices` requires `filter` query parameter

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `deviceupdate_dataplane_2022_10_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Device Update Data Plane routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- ownership of `/azure/deviceupdate-dataplane/2022-10-01/` route family
- strict `api-version=2022-10-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- data-plane routes are deterministically owned
- version and method failures are stable and testable

### Stage 2: Device Update Operation Group

Implement all `11` methods in `device-update/*`:

- `Delete Update`
- `Get File`
- `Get Operation Status`
- `Get Update`
- `Import Update`
- `List Files`
- `List Names`
- `List Operation Statuses`
- `List Providers`
- `List Updates`
- `List Versions`

Behavior:

- deterministic update catalog store keyed by `{provider,name,version}`
- deterministic file metadata and operation-status projections
- LRO-style response headers and operation-status records for import/delete flows

Exit criteria:

- all `device-update/*` routes return modeled responses
- no `NotImplemented` on Device Update operation-group methods

### Stage 3: Device Management Deployment/Class/Group Core

Implement deployment and hierarchy surfaces:

- create/get/list/delete/retry/stop deployment methods
- device class get/list/update/delete methods
- group get/list/delete methods
- subgroup get/list/delete methods

Behavior:

- deterministic relationship modeling between groups, subgroups, classes, and deployments
- idempotent deletion semantics with documented status codes
- deployment status fixtures with stable counters and timestamps

Exit criteria:

- deployment lifecycle and core hierarchy routes are covered end-to-end

### Stage 4: Device/Compliance/Diagnostics/Operations

Implement remaining Device Management surfaces:

- device and module lookup/list methods
- update compliance methods (global/group/subgroup)
- best-update and installable-update query methods
- diagnostics log-collection methods
- operation-status list/get methods
- import-devices LRO method

Behavior:

- deterministic compliance projections and rollups
- log collection state lifecycle (`queued`, `running`, `succeeded`, `failed`)
- operation history model for background operations

Exit criteria:

- all `device-management/*` routes return modeled responses
- no `NotImplemented` on Device Management methods

### Stage 5: Tests

Add provider tests for:

- route ownership and `api-version` gating
- device-update import/delete operation lifecycles
- deployment/class/group CRUD and status flows
- compliance and diagnostics query semantics
- operation-status polling and idempotent deletes

Exit criteria:

- `go test ./internal/server -run 'TestAzureDeviceUpdateDataPlane20221001' -count=1` passes

### Stage 6: Example and Validation

Add `examples/azure/deviceupdate-dataplane-2022-10-01`:

- `main.go` using Azure Go SDK `azcore/runtime` pipeline
- `Dockerfile`
- `docker-compose.yml`
- cover representative calls across both operation groups (including write, list, LRO, and cleanup paths)

Validation:

- build the example binary
- run targeted coverage scripts for alias + plan mapping correctness

Exit criteria:

- example compiles and exercises the new route families
- coverage scripts recognize service key and aliases
