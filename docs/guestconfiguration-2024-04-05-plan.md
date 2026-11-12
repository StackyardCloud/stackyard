# Azure Guest Configuration (`guestconfiguration_2024_04_05`) Staged Plan

## Objective

Emulate Azure Guest Configuration REST API (`api-version=2024-04-05`) with deterministic local behavior for:

- guest configuration assignment lifecycle for VM, VMSS, Arc-enabled server (HCRP), and Connected VMware vSphere VM targets
- assignment report get/list surfaces across supported target types
- provider operations discovery surface
- an Azure Go SDK example under `examples/azure/guestconfiguration-2024-04-05`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/operation-groups?view=rest-guestconfiguration-2024-04-05`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignment-reports?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignment-reports-vm-ss?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments-vm-ss?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments-reports?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignment-reports?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignments?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/operations?view=rest-guestconfiguration-2024-04-05`

Method pages reviewed directly: all linked methods from all operation groups (`27` total).

- guest-configuration-assignment-reports methods (2):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignment-reports/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignment-reports/list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-assignment-reports-vm-ss methods (2):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignment-reports-vm-ss/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignment-reports-vm-ss/list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-assignments methods (6):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments/create-or-update?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments/delete?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments/list?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments/rg-list?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments/subscription-list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-assignments-vm-ss methods (4):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments-vm-ss/create-or-update?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments-vm-ss/delete?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments-vm-ss/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-assignments-vm-ss/list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-connected-vm-warev-sphere-assignments methods (4):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments/create-or-update?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments/delete?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments/list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-connected-vm-warev-sphere-assignments-reports methods (2):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments-reports/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-connected-vm-warev-sphere-assignments-reports/list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-hcrp-assignment-reports methods (2):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignment-reports/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignment-reports/list?view=rest-guestconfiguration-2024-04-05`

- guest-configuration-hcrp-assignments methods (4):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignments/create-or-update?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignments/delete?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignments/get?view=rest-guestconfiguration-2024-04-05`
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/guest-configuration-hcrp-assignments/list?view=rest-guestconfiguration-2024-04-05`

- operations methods (1):
- `https://learn.microsoft.com/en-us/rest/api/guestconfiguration/operations/list?view=rest-guestconfiguration-2024-04-05`

## Surface Summary

Reviewed operation groups and methods:

- Guest Configuration Assignment Reports (2)
- Guest Configuration Assignment Reports VM SS (2)
- Guest Configuration Assignments (6)
- Guest Configuration Assignments VM SS (4)
- Guest Configuration Connected VM warev Sphere Assignments (4)
- Guest Configuration Connected VM warev Sphere Assignments Reports (2)
- Guest Configuration HCRP Assignment Reports (2)
- Guest Configuration HCRP Assignments (4)
- Operations (1)

Total reviewed methods: `27`.

## Service Naming

Canonical Stackyard naming:

- service key: `guestconfiguration_2024_04_05`
- display name: `Azure Guest Configuration`
- API version: `2024-04-05`

Recommended repo paths:

- implementation: `internal/server/provider_azure_guestconfiguration_2024_04_05.go`
- tests: `internal/server/provider_azure_guestconfiguration_2024_04_05_test.go`
- example: `examples/azure/guestconfiguration-2024-04-05`
- plan doc: `docs/guestconfiguration-2024-04-05-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.GuestConfiguration/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{guestConfigurationAssignmentName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachineScaleSets/{vmssName}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{name}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/virtualmachines/{vmName}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{guestConfigurationAssignmentName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.HybridCompute/machines/{machineName}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{guestConfigurationAssignmentName}`
- the corresponding `/reports` and `/reports/{reportId}` sub-resources for each target family

Representative routes:

- `GET /azure/providers/Microsoft.GuestConfiguration/operations?api-version=2024-04-05`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{vm}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{assignment}?api-version=2024-04-05`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{vm}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{assignment}/reports?api-version=2024-04-05`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachineScaleSets/{vmss}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{assignment}?api-version=2024-04-05`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ConnectedVMwarevSphere/virtualmachines/{vm}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{assignment}?api-version=2024-04-05`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.HybridCompute/machines/{machine}/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments/{assignment}?api-version=2024-04-05`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2024-04-05`
- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods include `Other Status Codes`

Observed response-code coverage across all reviewed methods:

- `200 OK`: `27`
- `201 Created`: `4`
- `204 No Content`: `4`
- `Other Status Codes`: `27`

Observed verb coverage:

- `GET`: `19`
- `PUT`: `4`
- `DELETE`: `4`

Request-body coverage from reviewed pages:

- `4` methods include request bodies:
- `guest-configuration-assignments/create-or-update`
- `guest-configuration-assignments-vm-ss/create-or-update`
- `guest-configuration-connected-vm-warev-sphere-assignments/create-or-update`
- `guest-configuration-hcrp-assignments/create-or-update`

SDK cross-check:

- this management-plane surface targets `Microsoft.GuestConfiguration` across host providers (`Microsoft.Compute`, `Microsoft.HybridCompute`, `Microsoft.ConnectedVMwarevSphere`)
- stage-0 examples use Azure Go SDK `azcore/runtime` so all reviewed routes can be exercised without typed-client version skew

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `guestconfiguration_2024_04_05` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- keep stage-0 router non-owning to avoid changing existing routing behavior until ownership is intentionally implemented

Exit criteria:

- coverage scripts resolve service key and aliases
- docs coverage maps service key to this plan
- router dispatch includes a dedicated Guest Configuration stage-0 hook

### Stage 1: Route Ownership and API-Version Gate

Implement:

- route ownership for `Microsoft.GuestConfiguration` route families under all reviewed host provider path variants
- strict `api-version=2024-04-05` validation on owned routes
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Guest Configuration route families are deterministically owned
- version/method failures are stable and testable

### Stage 2: Assignment Lifecycle Surfaces

Implement:

- VM assignment create/get/list/delete plus RG/subscription assignment list
- VMSS assignment create/get/list/delete
- Connected VMware vSphere assignment create/get/list/delete
- HCRP assignment create/get/list/delete

Behavior:

- deterministic ARM resource envelopes (`id`, `name`, `type`, `location`, `properties`)
- deterministic compliance/provisioning fields
- deterministic deletion responses for routes that document `200` and/or `204`

Exit criteria:

- all assignment lifecycle methods are represented

### Stage 3: Assignment Report and Operations Surfaces

Implement:

- report list/get for VM, VMSS, Connected VMware vSphere, and HCRP assignment families
- provider operations list

Behavior:

- deterministic report list ordering and report shape
- deterministic operation list payload entries for Guest Configuration provider operations

Exit criteria:

- all report and operations methods are represented

### Stage 4: Testing and Contract Hardening

Add tests for:

- route ownership and API-version enforcement
- assignment lifecycle semantics by host provider family
- report list/get behavior and scoping
- operation list shape and error-envelope consistency

Coverage alignment:

- satisfy contract, I/O, and documentation coverage expectations for `guestconfiguration_2024_04_05`

## Example and Coverage Script Tasks

Example package:

- add `examples/azure/guestconfiguration-2024-04-05/main.go`
- add container assets:
  - `examples/azure/guestconfiguration-2024-04-05/Dockerfile`
  - `examples/azure/guestconfiguration-2024-04-05/docker-compose.yml`

Coverage script wiring:

- add aliases in all three Azure coverage scripts for canonical and user-friendly spellings
- add docs plan mapping in `azure-doc-contract-coverage.py`

Recommended aliases:

- `guestconfiguration_2024_04_05`
- `guestconfiguration-2024-04-05`
- `guestconfiguration`
- `guest_configuration`
- `guest-configuration`
- `azure_guestconfiguration`
- `azure_guest_configuration`
- `microsoft_guestconfiguration`
- `rest_guestconfiguration_2024_04_05`

## Validation Commands

- `gofmt -w internal/server/provider_azure_guestconfiguration_2024_04_05.go`
- `gofmt -w examples/azure/guestconfiguration-2024-04-05/main.go`
- `go build ./examples/azure/guestconfiguration-2024-04-05`
- `go test ./internal/server -run '^$'`
- `python3 scripts/azure-contract-coverage.py --service guestconfiguration-2024-04-05 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service guestconfiguration-2024-04-05 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service guestconfiguration-2024-04-05 --fail-on any`
