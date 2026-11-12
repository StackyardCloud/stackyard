# Azure Blueprints (`blueprints-2018-11-01-preview`) Staged Plan

## Objective

Emulate the Azure Blueprints ARM management API for `2018-11-01-preview` with deterministic local behavior for:

- blueprint definitions
- blueprint artifacts
- published blueprint versions
- published blueprint artifacts
- blueprint assignments
- assignment operations
- an Azure Go SDK example under `examples/azure/blueprints-2018-11-01-preview`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the Azure Blueprints ARM management plane under `Microsoft.Blueprint`.
- This is not a data plane and it is not a normal resource-group-scoped ARM provider.
- Routes are rooted at either:
  - management-group scope: `/providers/Microsoft.Management/managementGroups/{managementGroup}/providers/Microsoft.Blueprint/...`
  - subscription scope: `/subscriptions/{subscriptionId}/providers/Microsoft.Blueprint/...`

Important modeling note:

- The service is structurally staged:
  - draft blueprint definitions
  - draft artifacts under a definition
  - immutable published versions
  - assignments of a published version
  - operation-history records under an assignment
- The emulator should preserve those relationships instead of flattening everything into one generic resource bucket.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/blueprints/operation-groups?view=rest-blueprints-2018-11-01-preview`

Operation-group and group-level pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/blueprints/blueprints?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/artifacts?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-blueprints?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-artifacts?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignments?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignment-operations?view=rest-blueprints-2018-11-01-preview`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/blueprints/blueprints/create-or-update?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/blueprints/get?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/blueprints/list?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/blueprints/delete?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/artifacts/create-or-update?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/artifacts/get?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/artifacts/list?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/artifacts/delete?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-blueprints/create?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-blueprints/get?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-blueprints/list?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-blueprints/delete?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-artifacts/get?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/published-artifacts/list?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignments/create-or-update?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignments/get?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignments/list?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignments/delete?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignments/who-is-blueprint?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignment-operations/get?view=rest-blueprints-2018-11-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/blueprints/assignment-operations/list?view=rest-blueprints-2018-11-01-preview`

Official Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/blueprint/armblueprint`

SDK selection note:

- The official Azure Go SDK package for this surface is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/blueprint/armblueprint`.
- The older track-1 `github.com/AZURE/azure-sdk-for-go/services/preview/blueprint/mgmt/2018-11-01-preview/blueprint` package still exists, but its package page explicitly points to `armblueprint` as the replacement.
- The plan assumption is therefore:
  - prefer `armblueprint` for the example
  - keep the example scoped to clients clearly exposed there: definitions, artifacts, published versions, assignments, and assignment operations

## Service Naming

Canonical Stackyard naming:

- service key: `blueprints`
- display name: `Azure Blueprints`
- API version token: `2018-11-01-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_blueprints.go`
- tests: `internal/server/provider_azure_blueprints_test.go`
- example: `examples/azure/blueprints-2018-11-01-preview`
- plan doc: `docs/blueprints-2018-11-01-preview-plan.md`

Recommended coverage aliases:

- `blueprints`
- `blueprint`
- `azure_blueprints`
- `azure_blueprint`
- `microsoft_blueprint`
- `rest_blueprints_2018_11_01_preview`
- `blueprints_2018_11_01_preview`

## API Surface Summary

The reviewed pages expose six main route families:

- blueprint definitions:
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}`
- draft artifacts:
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}/artifacts`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}/artifacts/{artifactName}`
- published versions:
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}/versions`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}/versions/{versionId}`
- published artifacts:
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}/versions/{versionId}/artifacts`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprints/{blueprintName}/versions/{versionId}/artifacts/{artifactName}`
- assignments:
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprintAssignments`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprintAssignments/{assignmentName}`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprintAssignments/{assignmentName}/whoIsBlueprint`
- assignment operations:
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprintAssignments/{assignmentName}/assignmentOperations`
  - `/{resourceScope}/providers/Microsoft.Blueprint/blueprintAssignments/{assignmentName}/assignmentOperations/{assignmentOperationName}`

Modeling implications:

- `resourceScope` must support both management-group and subscription forms
- draft blueprints and published versions should be stored separately
- publishing a version should snapshot the current draft blueprint and artifact state into an immutable published-version record
- assignment create should target a published blueprint version, not an unpublished draft definition
- assignment operations should be generated from assignment lifecycle events and exposed as read-only history

## Contract Notes From Reviewed Method Pages

Shared traits:

- failures use ARM-style `CloudError`
- list routes return `value` and often `nextLink`
- resource responses use stable ARM fields:
  - `id`
  - `name`
  - `type`
  - `properties.*`

Important data-model traits visible in the reviewed pages:

- blueprint definitions include:
  - `displayName`
  - `description`
  - `targetScope`
  - `parameters`
  - `resourceGroups`
  - readonly `status`
- artifacts are discriminated unions with at least:
  - policy assignment artifacts
  - role assignment artifacts
  - ARM template artifacts
- assignments include:
  - `identity`
  - `location`
  - `properties.blueprintId`
  - `properties.scope`
  - `properties.parameters`
  - `properties.resourceGroups`
  - `properties.locks`
  - `properties.provisioningState`
  - readonly `properties.status`
- assignment delete accepts optional `deleteBehavior=all`
- `whoIsBlueprint` returns the Blueprints service principal object ID for an assignment scope
- assignment operations expose deployment history under each assignment update

Scope constraints visible across the reviewed pages:

- valid scopes are:
  - management groups: `/providers/Microsoft.Management/managementGroups/{managementGroup}`
  - subscriptions: `/subscriptions/{subscriptionId}`

## Recommended State Model

Keep canonical stores for:

- blueprint definitions keyed by `{resourceScope}/{blueprintName}`
- draft artifacts keyed by `{resourceScope}/{blueprintName}/{artifactName}`
- published blueprints keyed by `{resourceScope}/{blueprintName}/{versionId}`
- published artifacts keyed by `{resourceScope}/{blueprintName}/{versionId}/{artifactName}`
- blueprint assignments keyed by `{resourceScope}/{assignmentName}`
- assignment operations keyed by `{resourceScope}/{assignmentName}/{operationName}`

Recommended seeded fixtures:

- one subscription-scoped blueprint definition
- one management-group-scoped blueprint definition
- one draft artifact of each main artifact kind under at least one blueprint
- one published version for a subscription-scoped blueprint
- one assignment tied to the published version
- one assignment-operation history row with deterministic deployment details

Recommended workflow behavior:

- draft `Create Or Update` mutates the definition or draft artifact in place
- published-version `Create` snapshots the current draft definition plus current draft artifacts into immutable version storage
- assignment `Create Or Update` validates the referenced published blueprint ID, writes the assignment, and appends a deterministic assignment operation history row

## Staged Implementation Plan

### Stage 1: Route Skeleton, Scope Parsing, and Coverage Wiring

Create the initial provider and route registration for:

- blueprint definitions
- draft artifacts
- scope parsing for both management-group and subscription roots

Include:

- a shared `resourceScope` parser and normalizer
- basic ARM error helpers for blueprint-specific invalid-scope and not-found paths
- initial `blueprints` coverage aliases in the Azure coverage scripts

### Stage 2: Blueprint Definitions and Draft Artifacts

Implement:

- `Blueprints` create, get, list, delete
- `Artifacts` create, get, list, delete

Recommended outcome:

- definition and artifact stores share the same normalized scope and blueprint name
- artifact payloads preserve the discriminated artifact kind in returned fixtures
- list routes are non-empty for seeded fixtures and coherent after mutations

### Stage 3: Published Versions and Published Artifacts

Implement:

- `Published Blueprints` create, get, list, delete
- `Published Artifacts` get and list

Recommended outcome:

- publishing creates an immutable snapshot of the current draft definition plus draft artifacts
- deleting a published version removes only that immutable version, not the draft definition
- published artifact reads are backed by snapshot data, not live draft artifacts

### Stage 4: Assignments

Implement:

- assignment create, get, list, delete
- `whoIsBlueprint`

Recommended outcome:

- assignments must reference a published blueprint version ID
- assignment `properties.scope` and outer `resourceScope` should remain coherent
- delete should honor the presence or absence of `deleteBehavior=all` in a deterministic but lightweight way
- `whoIsBlueprint` can return a stable seeded service principal object ID

### Stage 5: Assignment Operations

Implement:

- assignment operation list
- assignment operation get

Recommended outcome:

- assignment create and update append synthetic operation-history records
- assignment delete may append a terminal operation or preserve existing history, but must return stable data for get/list
- operation payloads should expose deployment/job-history structure similar to the reviewed examples

### Stage 6: Example and Coverage Finalization

Add the example under `examples/azure/blueprints-2018-11-01-preview` using the official `armblueprint` package.

Recommended example flow:

1. create a subscription-scoped blueprint definition
2. add one draft artifact
3. publish version `v1`
4. create an assignment for that published version
5. get `whoIsBlueprint`
6. list assignment operations

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Testing Strategy

Add focused provider tests for:

- blueprint definition lifecycle at subscription scope
- blueprint definition lifecycle at management-group scope
- artifact lifecycle and artifact kind preservation
- published version creation and immutable published artifact reads
- assignment lifecycle including `whoIsBlueprint`
- assignment operations list/get after assignment create or update
- negative paths for invalid scope strings, missing blueprints, missing published versions, and missing assignments

Coverage-oriented assertions should verify:

- every documented route family resolves to a concrete handler
- scope parsing works for both management-group and subscription forms
- no reviewed route falls through to `NotImplemented`

## Coverage Script Updates To Make During Implementation

Add a new Azure service entry for:

- service key: `blueprints`
- plan doc: `docs/blueprints-2018-11-01-preview-plan.md`
- example directory: `examples/azure/blueprints-2018-11-01-preview`

Include alias mapping for:

- `blueprints`
- `blueprint`
- `azure_blueprints`
- `microsoft_blueprint`
- `rest_blueprints_2018_11_01_preview`

## Validation Targets For The Future Implementation

At implementation time, validate with:

- `go test ./internal/server -run 'TestAzureBlueprints' -count=1`
- `go build ./examples/azure/blueprints-2018-11-01-preview`
- `python3 scripts/azure-contract-coverage.py --service blueprints --require-service blueprints --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service blueprints --require-service blueprints --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service blueprints --require-service blueprints --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service blueprints --require-service blueprints --fail-on any`
