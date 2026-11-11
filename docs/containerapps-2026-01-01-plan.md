# Azure Container Apps (`containerapps-2026-01-01`) Staged Plan

## Objective

Emulate Azure Container Apps for API version `2026-01-01` with deterministic ARM-style behavior under `Microsoft.App`, including:

- provider `operations`
- core resource lifecycles for `managedEnvironments`, `containerApps`, and `jobs`
- representative child and adjacent families such as revisions, auth configs, diagnostics, certificates, dapr components, storages, source controls, and session pools
- subscription-level and location-level discovery endpoints such as `usages`, `billingMeters`, `availableWorkloadProfiles`, and name-availability checks
- an Azure Go SDK example under `examples/azure/containerapps-2026-01-01`
- Azure coverage-script wiring for the new service key

The reviewed surface is large. It should be implemented in staged slices around route families and shared models, not as one monolithic provider.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/operation-groups?view=rest-resource-manager-containerapps-2026-01-01`

Representative operation-group and method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/get?view=rest-resource-manager-containerapps-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/create-or-update?view=rest-resource-manager-containerapps-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/operations/list?view=rest-resource-manager-containerapps-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/list-by-resource-group?view=rest-resource-manager-containerapps-2025-07-01`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers`

## Service Naming

Canonical Stackyard naming:
- service key: `containerapps`
- display name: `Azure Container Apps`
- provider namespace: `Microsoft.App`
- API version: `2026-01-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_containerapps.go`
- tests: `internal/server/provider_azure_containerapps_test.go`
- example: `examples/azure/containerapps-2026-01-01`
- plan doc: `docs/containerapps-2026-01-01-plan.md`

## Surface Summary

The reviewed `2026-01-01` operation-groups page exposes these families:

1. `Available Workload Profiles`
2. `Billing Meters`
3. `Certificates`
4. `Connected Environments`
5. `Connected Environments Certificates`
6. `Connected Environments Dapr Components`
7. `Connected Environments Storages`
8. `Container Apps`
9. `Container Apps Auth Configs`
10. `Container Apps Diagnostics`
11. `Container Apps Revision Replicas`
12. `Container Apps Revisions`
13. `Container Apps Session Pools`
14. `Container Apps Source Controls`
15. `Dapr Components`
16. `Get Custom Domain Verification Id`
17. `Http Route Config`
18. `Java Components`
19. `Job Execution`
20. `Jobs`
21. `Jobs Executions`
22. `Logic Apps`
23. `Maintenance Configurations`
24. `Managed Certificates`
25. `Managed Environment Diagnostics`
26. `Managed Environment Private Endpoint Connections`
27. `Managed Environment Private Link Resources`
28. `Managed Environment Usages`
29. `Managed Environments`
30. `Managed Environments Diagnostics`
31. `Managed Environments Storages`
32. `Namespaces`
33. `Operations`
34. `Usages`

Important implication:
- this service is materially broader than the recent Azure ARM providers added in this repo
- a realistic staged plan must prioritize the resource families most likely to appear in examples and coverage first, then layer in the adjacent children and discovery endpoints

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.App/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.App/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.App/*`
- `/azure/subscriptions/{subscriptionId}/locations/{location}/providers/Microsoft.App/*`

Primary top-level resource families to cover:

- `/managedEnvironments`
- `/containerApps`
- `/jobs`
- `/daprComponents`
- `/certificates`
- `/managedCertificates`
- `/connectedEnvironments`
- `/javaComponents`
- `/logicApps`
- `/maintenanceConfigurations`
- `/namespaces`

Representative child resource and action routes from the reviewed surface:

- `/containerApps/{name}/authConfigs/{authName}`
- `/containerApps/{name}/diagnostics`
- `/containerApps/{name}/revisions`
- `/containerApps/{name}/revisions/{revisionName}`
- `/containerApps/{name}/revisions/{revisionName}/replicas`
- `/containerApps/{name}/sessionPools`
- `/containerApps/{name}/sourcecontrols/{sourceControlName}`
- `/jobs/{name}/executions`
- `/managedEnvironments/{name}/storages`
- `/managedEnvironments/{name}/privateEndpointConnections`
- `/managedEnvironments/{name}/privateLinkResources`

Target query:
- `api-version=2026-01-01`

## Contract Notes From Reviewed Pages

Shared management-plane traits:

- every reviewed method requires `api-version`
- failures use ARM-style `ErrorResponse` or service error envelopes
- create/update/delete operations commonly behave like long-running operations
- local Stackyard examples should continue to use the repo’s Azure shim transport rather than Azure AD

### Operations

`Operations.List`
- `GET /providers/Microsoft.App/operations`
- returns available RP operations

Modeling implication:
- implement early as a deterministic `value` list that covers every route family Stackyard exposes for this service

### Container Apps

Reviewed `Get` and `Create Or Update` pages show the `ContainerApp` shape includes:
- `id`, `name`, `type`, `location`, `kind`, `identity`, `tags`
- `properties.environmentId`
- `properties.configuration`
- `properties.template`
- `properties.workloadProfileName`
- `properties.provisioningState`
- ingress, registries, secrets, dapr, and revision settings under nested configuration/template objects

Modeling implication:
- `containerApps` should be one of the first implemented resource families
- the first pass should preserve the caller’s nested `configuration` and `template` objects without trying to validate every subfield

### Managed Environments

This family anchors much of the broader service.

Modeling implication:
- `managedEnvironments` should be implemented before or alongside `containerApps`
- child families such as storages, diagnostics, private endpoints, usages, and certificates can share environment-keyed storage and shallow fixtures

### Jobs

Jobs and job executions are a parallel execution surface to container apps.

Modeling implication:
- `jobs` should be part of the first or second major slice because they are a top-level resource family with a straightforward lifecycle
- `job executions` can be deterministic child list/action fixtures derived from stored jobs

### Certificates And Managed Certificates

These are important but shallow for an emulator.

Modeling implication:
- preserve key fields like domain names, subject names, expiration markers, and provisioning state
- defer real validation of certificate material

### Revisions, Replicas, Diagnostics, And Source Controls

These are natural children of stored container app state.

Modeling implication:
- revision and replica lists can be synthetic projections of a stored app template
- diagnostics and source controls can be implemented as thin typed children with deterministic status payloads

### Subscription And Location Discovery

The service includes multiple read-only discovery families:
- `Usages`
- `Managed Environment Usages`
- `Billing Meters`
- `Available Workload Profiles`
- custom domain verification
- namespace and name availability helpers

Modeling implication:
- these should be implemented as deterministic typed lists and singleton payloads, not as generic placeholders

## Non-Goals

- do not emulate real container scheduling, ingress routing, KEDA scaling, Dapr execution, or revision rollout logic
- do not emulate actual container image pulls or job execution runtimes
- do not implement every preview-only or deeply nested schema field on the first pass
- do not implement full LRO polling semantics for every mutation unless required by the selected SDK example path

## Stage 0: Coverage Naming And Router Claiming

- Add a dedicated Azure Container Apps router for `Microsoft.App`
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Recommended aliases:
  - `containerapps`
  - `container_apps`
  - `container-apps`
  - `azure_container_apps`
  - `azure_containerapps`
  - `containerapps_2026_01_01`
  - `rest_resource_manager_containerapps_2026_01_01`
  - `microsoft_app`
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route-recognition tests for:
  - RP operations
  - subscription discovery routes
  - `managedEnvironments`
  - `containerApps`
  - `jobs`

Exit criteria:
- coverage scripts discover the new canonical service key
- Azure routing claims Container Apps management paths deterministically

## Stage 1: Shared Validation And ARM Envelope

- Validate `api-version=2026-01-01`
- Enforce supported methods per route family:
  - `GET`, `PUT`, `PATCH`, `POST`, `DELETE`, `HEAD`
- Return deterministic ARM-style invalid-request and not-found envelopes
- Add helpers for:
  - subscription/resource-group route parsing
  - location-scoped discovery parsing
  - environment/app/job child route parsing
  - shared `systemData`, `etag`, and `provisioningState` stamping

Exit criteria:
- the service no longer falls through to generic `NotImplemented` for claimed paths
- negative tests cover invalid version, malformed JSON, missing required fields, and not-found

## Stage 2: RP Operations And Read-Only Discovery

- Implement:
  - `Operations`
  - `Usages`
  - `Managed Environment Usages`
  - `Billing Meters`
  - `Available Workload Profiles`
  - `Get Custom Domain Verification Id`
  - `Namespaces` and any name-availability helper routes exposed in the selected SDK/example flow

Use deterministic typed payloads:
- collection envelopes with `value`
- small static lists of operations, meters, profiles, and quota entries
- stable verification IDs and namespace availability responses

Exit criteria:
- all discovery families return typed payloads and no longer rely on fallback handlers

## Stage 3: Managed Environments Foundation

- Implement persistent state for `managedEnvironments`:
  - create/get/list by resource group/list by subscription/patch/delete
- Preserve key request fields:
  - `location`
  - `tags`
  - `properties.appLogsConfiguration`
  - `properties.vnetConfiguration`
  - `properties.workloadProfiles`
  - `properties.zoneRedundant`
- Stamp deterministic server fields:
  - `id`, `name`, `type`, `systemData`, `etag`, `properties.provisioningState`

Exit criteria:
- `managedEnvironments` lifecycle is stable and typed
- later child families can anchor on environment state

## Stage 4: Container Apps Core Lifecycle

- Implement persistent state for `containerApps`:
  - create/get/list by resource group/list by subscription/patch/delete/head
- Preserve key request fields:
  - `location`
  - `kind`
  - `identity`
  - `tags`
  - `properties.environmentId`
  - `properties.configuration`
  - `properties.template`
  - `properties.workloadProfileName`
- Stamp deterministic server fields:
  - `properties.provisioningState`
  - `properties.latestRevisionName`
  - `properties.latestReadyRevisionName`
  - `properties.outboundIPAddresses`

Exit criteria:
- `containerApps` is fully CRUD-capable with typed payloads
- no core app route returns `NotImplemented`

## Stage 5: Jobs Core Lifecycle

- Implement persistent state for `jobs`:
  - create/get/list by resource group/list by subscription/patch/delete
- Preserve the caller’s configuration and template objects with shallow validation
- Add deterministic child execution projections:
  - `job executions`
  - single execution detail fixtures when needed

Exit criteria:
- `jobs` and one execution-view family are covered

## Stage 6: App And Environment Children

- Implement representative child families derived from stored parent state:
  - `Container Apps Auth Configs`
  - `Container Apps Diagnostics`
  - `Container Apps Revisions`
  - `Container Apps Revision Replicas`
  - `Container Apps Session Pools`
  - `Container Apps Source Controls`
  - `Managed Environments Storages`
  - `Managed Environment Diagnostics`
  - `Managed Environment Private Endpoint Connections`
  - `Managed Environment Private Link Resources`

Suggested first-pass behavior:
- revisions and replicas are synthetic read models derived from stored app template/configuration
- auth configs, diagnostics, session pools, source controls, and storages are thin child resources with deterministic typed fields
- private endpoint and private link resources are read-only or CRUD only where necessary for coverage

Exit criteria:
- the most common child routes for apps and environments are present and typed

## Stage 7: Adjacent Resource Families

- Implement the next layer of top-level families with shallow but typed fixtures:
  - `Dapr Components`
  - `Certificates`
  - `Managed Certificates`
  - `Connected Environments`
  - `Connected Environments Certificates`
  - `Connected Environments Dapr Components`
  - `Connected Environments Storages`
  - `Java Components`
  - `Logic Apps`
  - `Maintenance Configurations`
  - `Http Route Config`

Pragmatic implementation note:
- these can often reuse the same generic ARM-resource storage helper with per-collection validation and fixture shaping

Exit criteria:
- the broader `Microsoft.App` surface is present enough that coverage no longer reports route gaps across the main operation groups

## Stage 8: Azure Go SDK Example

- Add `examples/azure/containerapps-2026-01-01`
- Use `armappcontainers`
- Follow the repo’s Azure example pattern:
  - `azsdkshim.StaticTokenCredential{}`
  - `arm.ClientOptions`
  - `cloud.ResourceManager` endpoint override to Stackyard
  - `azsdkshim.NewTransport("/azure", account, subscriptionKey)`

Recommended example flow:
- list RP operations
- create one managed environment
- create one container app in that environment
- get the app
- list app revisions
- create one job
- list jobs
- patch app tags or configuration
- delete the job and app

Exit criteria:
- the example compiles cleanly
- the example exercises the primary route families without special-case hacks

## Stage 9: Coverage And Verification

- Update service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Map this plan doc in `SERVICE_PLAN_DOCS`
- Add focused tests for:
  - operations and discovery endpoints
  - managed environment lifecycle
  - container app lifecycle
  - jobs lifecycle
  - selected child route families
  - invalid body and missing required fields
  - invalid `api-version`

Expected verification commands after implementation:
- `go test ./internal/server -run 'TestAzureContainerApps' -count=1`
- `go build ./examples/azure/containerapps-2026-01-01`
- `python3 scripts/azure-contract-coverage.py --service containerapps --require-service containerapps --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service containerapps --require-service containerapps --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service containerapps --require-service containerapps --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service containerapps --require-service containerapps --fail-on any`

Exit criteria:
- all implemented Container Apps endpoints are discovered by the coverage scripts
- no claimed Container Apps route returns `NotImplemented`
- targeted validation commands succeed

## Implementation Order Recommendation

Recommended merge order:

1. Stage 0 and Stage 1
2. Stage 2 and Stage 3
3. Stage 4 and Stage 5
4. Stage 6
5. Stage 7
6. Stage 8 and Stage 9

This order lands the highest-signal `Microsoft.App` resource families first, keeps each slice mergeable, and avoids blocking the example on the long tail of adjacent operation groups.
