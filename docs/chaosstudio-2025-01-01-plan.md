# Azure Chaos Studio (`chaosstudio-2025-01-01`) Staged Plan

## Objective

Emulate the Azure Chaos Studio ARM management API for `2025-01-01` with deterministic local behavior for:

- provider `operations`
- location-scoped `targetTypes`
- location-scoped `capabilityTypes`
- extension-resource `targets`
- nested extension-resource `capabilities`
- top-level `experiments`
- experiment execution inspection routes
- subscription-location `operationStatuses`
- an Azure Go SDK example under `examples/azure/chaosstudio-2025-01-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the ARM management plane under `Microsoft.Chaos`.
- This is not the runtime behavior inside a guest agent or a real fault-injection engine.
- The reviewed `2025-01-01` surface splits into four distinct route families:
  - provider discovery: `operations`
  - location inventory: `targetTypes` and `capabilityTypes`
  - extension resources attached to tracked regional resources: `targets` and `capabilities`
  - tracked top-level resources plus actions: `experiments`, executions, and async `operationStatuses`

Important modeling note:

- `targets` and `capabilities` are not standalone ARM resources under a Chaos Studio resource group tree.
- They extend an existing parent resource path such as a VM or VMSS resource and therefore should be stored keyed by the normalized parent ARM path plus the Chaos suffix.
- The emulator should accept documented parent path shapes generically rather than hard-coding only one parent provider like `Microsoft.Compute`.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/operation-groups?view=rest-chaosstudio-2025-01-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/operations?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/target-types?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capability-types?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/targets?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capabilities?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/operation-statuses?view=rest-chaosstudio-2025-01-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/operations/list-all?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/target-types/list?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/target-types/get?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capability-types/list?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capability-types/get?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/targets/create-or-update?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/targets/get?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/targets/list?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/targets/delete?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capabilities/create-or-update?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capabilities/get?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capabilities/list?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/capabilities/delete?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/create-or-update?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/get?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/list?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/list-all?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/update?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/delete?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/start?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/cancel?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/get-execution?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/list-all-executions?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/experiments/execution-details?view=rest-chaosstudio-2025-01-01`
- `https://learn.microsoft.com/en-us/rest/api/chaosstudio/operation-statuses/get?view=rest-chaosstudio-2025-01-01`

Official Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/chaos/armchaos`

## SDK Selection Note

- The official Azure Go SDK package for this service is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/chaos/armchaos`.
- The current public package examples are generated from `2024-01-01`, but the client family lines up closely with the reviewed `2025-01-01` surface:
  - `OperationsClient`
  - `TargetTypesClient`
  - `CapabilityTypesClient`
  - `TargetsClient`
  - `CapabilitiesClient`
  - `ExperimentsClient`
  - `OperationStatusesClient`
- The future example should therefore:
  - prefer typed `armchaos` clients for the main flow
  - use `azcore/runtime` only as a targeted fallback if a reviewed `2025-01-01` method is not exposed cleanly by the local package revision

## Service Naming

Canonical Stackyard naming:

- service key: `chaosstudio`
- display name: `Azure Chaos Studio`
- API version token: `2025-01-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_chaosstudio.go`
- tests: `internal/server/provider_azure_chaosstudio_test.go`
- example: `examples/azure/chaosstudio-2025-01-01`
- plan doc: `docs/chaosstudio-2025-01-01-plan.md`

Recommended coverage aliases:

- `chaosstudio`
- `chaos_studio`
- `chaos`
- `microsoft_chaos`
- `chaosstudio_2025_01_01`
- `rest_chaosstudio_2025_01_01`

## API Surface Summary

The reviewed pages expose five major route families:

- provider operations:
  - `/providers/Microsoft.Chaos/operations`
- subscription-location discovery:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Chaos/locations/{location}/targetTypes`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Chaos/locations/{location}/targetTypes/{targetTypeName}`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Chaos/locations/{location}/targetTypes/{targetTypeName}/capabilityTypes`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Chaos/locations/{location}/targetTypes/{targetTypeName}/capabilityTypes/{capabilityTypeName}`
- extension resources on tracked parent resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{parentProviderNamespace}/{parentResourceType}/{parentResourceName}/providers/Microsoft.Chaos/targets/{targetName}`
  - `.../targets/{targetName}/capabilities/{capabilityName}`
- top-level tracked resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Chaos/experiments/{experimentName}`
- action and inspection routes:
  - `.../experiments/{experimentName}/start`
  - `.../experiments/{experimentName}/cancel`
  - `.../experiments/{experimentName}/executions`
  - `.../experiments/{experimentName}/executions/{executionId}`
  - `.../experiments/{experimentName}/executions/{executionId}/getExecutionDetails`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Chaos/locations/{location}/operationStatuses/{operationId}`

Modeling implications:

- target and capability type routes are read-only seeded inventory by location
- target and capability resources must validate and retain parent path metadata
- experiment execution routes should be backed by deterministic synthetic executions rather than a real scheduler
- async `start` and `cancel` routes should surface coherent polling through `operationStatuses`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value` and several support `continuationToken`
- top-level and extension resources carry standard ARM fields such as:
  - `id`
  - `name`
  - `type`
  - `location`
  - `systemData`
  - `properties.*`
- mutating experiment actions document async behavior with `202 Accepted`

### Operations

Reviewed method page:

- `List All`

Key traits from the reviewed page:

- route:
  - `/providers/Microsoft.Chaos/operations`
- returns `OperationListResult`
- sample results are paged and include standard ARM provider-operation metadata

Modeling implications:

- expose a deterministic operation list that matches the implemented emulator surface
- include `nextLink` only if the service needs pagination for contract coverage; otherwise a single page is sufficient

### Target Types and Capability Types

Reviewed method pages:

- `Target Types.List`
- `Target Types.Get`
- `Capability Types.List`
- `Capability Types.Get`

Key traits from the reviewed pages:

- all four routes are subscription-location scoped
- `targetTypeName` pattern:
  - `^[a-zA-Z0-9_\\-\\.]+$`
- `capabilityTypeName` pattern:
  - `^[a-zA-Z0-9\\-\\.]+-\\d\\.\\d$`
- list routes support `continuationToken`

Modeling implications:

- seed a small location-indexed catalog of target types such as VM or VMSS agent targets
- capability types should hang off seeded target types and expose stable versioned names
- do not overbuild this as mutable state; these routes are better modeled as static inventory

### Targets and Capabilities

Reviewed method pages:

- `Targets.Create Or Update`
- `Targets.Get`
- `Targets.List`
- `Targets.Delete`
- `Capabilities.Create Or Update`
- `Capabilities.Get`
- `Capabilities.List`
- `Capabilities.Delete`

Key traits from the reviewed pages:

- both families extend an arbitrary tracked parent resource path
- parent path segments are validated separately:
  - `parentProviderNamespace`
  - `parentResourceType`
  - `parentResourceName`
- `targetName` pattern:
  - `^[a-zA-Z0-9_\\-\\.]+$`
- `capabilityName` pattern:
  - `^[a-zA-Z0-9\\-\\.]+-\\d\\.\\d$`
- list routes support `continuationToken`

Modeling implications:

- keep canonical target state keyed by normalized parent ARM path plus target name
- keep capability state keyed by target ID plus capability name
- capability create should validate the parent target exists
- target create should capture enough parent metadata to return stable ARM IDs without needing to emulate the parent provider itself

### Experiments

Reviewed method pages:

- `Create Or Update`
- `Get`
- `List`
- `List All`
- `Update`
- `Delete`
- `Start`
- `Cancel`
- `Get Execution`
- `List All Executions`
- `Execution Details`

Key traits from the reviewed pages:

- experiment path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Chaos/experiments/{experimentName}`
- `experimentName` pattern:
  - `^[^<>%&:?#/\\\\]+$`
- create requires:
  - `location`
  - `properties.selectors`
  - `properties.steps`
- list supports:
  - `running`
  - `continuationToken`
- `start` and `cancel` are action routes returning async semantics
- executions are separate subresources keyed by GUID execution IDs
- `getExecutionDetails` is a `POST` action under an execution ID and returns step, branch, and action status detail

Modeling implications:

- the emulator should preserve the essential experiment structure:
  - selectors
  - steps
  - branches
  - actions
- execution records can be synthesized from the stored experiment shape at `start` time
- `cancel` should transition an in-flight execution into a terminal canceled state rather than deleting it
- `running=true` filtering should be supported on experiment list routes because it is explicitly documented

### Operation Statuses

Reviewed method page:

- `Get`

Key traits from the reviewed page:

- route:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Chaos/locations/{location}/operationStatuses/{operationId}`
- response includes:
  - `id`
  - `name`
  - `startTime`
  - `endTime`
  - `status`
  - optional `percentComplete`
  - optional nested `operations`
  - optional `resourceId`
  - optional `error`

Modeling implications:

- `operationStatuses` should be a first-class state store for experiment `start` and `cancel`
- one stable polling progression is enough for coverage:
  - `Creating` or `Running`
  - then terminal `Succeeded` or `Canceled`

## Recommended State Model

Keep canonical stores for:

- experiments keyed by `{subscriptionId}/{resourceGroup}/{experimentName}`
- targets keyed by `{subscriptionId}/{resourceGroup}/{parentProviderNamespace}/{parentResourceType}/{parentResourceName}/{targetName}`
- capabilities keyed by target key plus capability name
- experiment executions keyed by experiment ID plus execution GUID
- async operation statuses keyed by `{subscriptionId}/{location}/{operationId}`

Keep seeded read-only fixture stores for:

- provider operations
- location-scoped target types
- target-type-scoped capability types

Recommended derived behavior:

- starting an experiment creates one execution record and one async operation record
- execution detail payloads derive deterministically from the stored experiment steps and branches
- deleting an experiment removes only the experiment resource, not historical execution records, unless the implementation chooses a simpler full-delete model consistently
- targets and capabilities should default to a succeeded provisioning state and enabled runtime state unless the request explicitly drives another supported state

## Staged Implementation Plan

### Stage 1: Route Skeleton and Coverage Wiring

Create the initial provider and route registration for:

- provider operations
- target-type and capability-type discovery routes
- target and capability extension-resource routes
- experiment CRUD and execution routes
- async `operationStatuses`

Include:

- initial `chaosstudio` coverage aliases in the Azure coverage scripts
- server wiring in `internal/server/http.go`
- shared helpers for generic parent-resource ID normalization

### Stage 2: Provider Operations and Location Discovery

Implement:

- `Operations.List All`
- `TargetTypes.List`
- `TargetTypes.Get`
- `CapabilityTypes.List`
- `CapabilityTypes.Get`

Recommended outcome:

- the service is discoverable through the provider operation list
- location inventory routes return stable fixtures and correct parent-child relationships

### Stage 3: Targets and Capabilities

Implement:

- `Targets.Create Or Update`
- `Targets.Get`
- `Targets.List`
- `Targets.Delete`
- `Capabilities.Create Or Update`
- `Capabilities.Get`
- `Capabilities.List`
- `Capabilities.Delete`

Recommended outcome:

- extension resources round-trip correctly under arbitrary valid parent provider paths
- capability lifecycle correctly depends on target existence
- list routes support deterministic paging if coverage requires `continuationToken`

### Stage 4: Experiment Lifecycle

Implement:

- `Experiments.Create Or Update`
- `Experiments.Get`
- `Experiments.List`
- `Experiments.List All`
- `Experiments.Update`
- `Experiments.Delete`

Recommended outcome:

- experiments preserve the requested selector and step structure
- list routes support `running` filtering coherently
- patch updates avoid mutating immutable identity fields accidentally

### Stage 5: Executions, Async Actions, Example, and Final Coverage Validation

Implement:

- `Experiments.Start`
- `Experiments.Cancel`
- `Experiments.Get Execution`
- `Experiments.List All Executions`
- `Experiments.Execution Details`
- `OperationStatuses.Get`
- the Azure Go SDK example in `examples/azure/chaosstudio-2025-01-01`
- final coverage-script doc mapping

Recommended example flow:

- create the client factory
- list provider operations
- list one location’s target types and one target type’s capability types
- create a target on a sample parent resource path
- create one capability under that target
- create an experiment with one selector and one simple step
- start the experiment and fetch its execution status or details

## Testing Strategy

Target focused lifecycle and validation tests rather than a giant all-in-one suite.

Minimum test coverage should include:

- provider operations and discovery routes
- target-type and capability-type get/list behavior
- target create/get/list/delete under a realistic parent resource path
- capability create/get/list/delete under an existing target
- experiment create/get/list/update/delete
- experiment start plus execution list/get/detail inspection
- async operation-status polling for `start` and `cancel`

Recommended negative cases:

- malformed `targetName`, `capabilityName`, or `experimentName`
- capability create under a missing target
- target create with invalid parent path components
- experiment create without required selectors or steps
- `start` or `cancel` against a missing experiment
- execution lookup with a non-GUID execution ID

## Coverage Script Updates To Make During Implementation

Update the following scripts:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for:

- `chaosstudio`
- `chaos_studio`
- `chaos`
- `microsoft_chaos`
- `chaosstudio_2025_01_01`
- `rest_chaosstudio_2025_01_01`

Add the plan doc mapping:

- `chaosstudio`: `chaosstudio-2025-01-01-plan.md`

## Validation Targets For The Future Implementation

After implementation, validate with:

- `go test ./internal/server -run 'TestAzureChaosStudio' -count=1`
- `go build ./examples/azure/chaosstudio-2025-01-01`
- `python3 scripts/azure-contract-coverage.py --service chaosstudio --require-service chaosstudio --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service chaosstudio --require-service chaosstudio --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service chaosstudio --require-service chaosstudio --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service chaosstudio --require-service chaosstudio --fail-on any`

Expected quality bar:

- no Chaos Studio endpoint falls through to `NotImplemented`
- the example builds cleanly against the chosen Azure Go SDK package
- the coverage scripts recognize `chaosstudio` and map it to this plan doc
