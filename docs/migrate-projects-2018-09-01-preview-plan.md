# Azure Migrate Projects (`migrate-projects-2018-09-01-preview`) Staged Plan

## Objective

Emulate the Azure Migrate Projects ARM management API for `2018-09-01-preview` with deterministic local behavior for:

- provider `operations`
- top-level `migrateProjects` lifecycle and discovery routes
- project actions such as `registerTool` and `refreshSummary`
- the broader `Microsoft.Migrate` resource families advertised by the provider operations list, including `solutions`, `machines`, `migrateEvents`, `databases`, and `databaseInstances`
- an Azure Go SDK example under `examples/azure/migrate-projects-2018-09-01-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure Migrate Projects under `Microsoft.Migrate`.
- This is distinct from the newer Migrate Assessment surface and distinct from the Discovery / appliance APIs.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2018-09-01-preview`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/operation-groups?view=rest-migrate-projects-2018-09-01-preview`

Representative operation-group and method pages reviewed across the surface:
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/operations/list?view=rest-migrate-projects-2018-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/migrate-projects/put-migrate-project?view=rest-migrate-projects-2018-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/migrate-projects/get-migrate-project?view=rest-migrate-projects-2018-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/migrate-projects/patch-migrate-project?view=rest-migrate-projects-2018-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/migrate-projects/delete-migrate-project?view=rest-migrate-projects-2018-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/migrate-projects/refresh-migrate-project-summary?view=rest-migrate-projects-2018-09-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/migrate/projects/migrate-projects/register-tool?view=rest-migrate-projects-2018-09-01-preview`

Azure Go SDK package expected for the future example:
- `github.com/AZURE/azure-sdk-for-go/services/preview/migrate/mgmt/2018-09-01-preview/migrate`

SDK selection note:
- The closest official Azure Go SDK package for this API version is the older track-1 preview `migrate` management client.
- The generated package surface appears broader than just `ProjectsClient`; it also exposes clients for solutions, events, machines, databases, and database instances in the same API family.
- Plan assumption: prefer the generated track-1 clients where practical, and only fall back to raw request construction if the generated package shape makes a documented route awkward to demonstrate.

## Service Naming

Canonical Stackyard naming:
- service key: `migrate_projects`
- display name: `Azure Migrate Projects`
- provider namespace: `Microsoft.Migrate`
- API version: `2018-09-01-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_migrate_projects.go`
- tests: `internal/server/provider_azure_migrate_projects_test.go`
- example: `examples/azure/migrate-projects-2018-09-01-preview`
- plan doc: `docs/migrate-projects-2018-09-01-preview-plan.md`

Recommended coverage aliases:
- `migrate_projects`
- `migrate-projects`
- `azure_migrate_projects`
- `microsoft_migrate_projects`
- `rest_migrate_projects_2018_09_01_preview`
- `migrate_projects_2018_09_01_preview`

## Surface Summary

The reviewed `2018-09-01-preview` management surface is centered on `migrateProjects`, but the provider operation list also advertises related families that belong to the same service version:

1. `Operations`
2. `Migrate Projects`
3. `Solutions`
4. `Machines`
5. `Migrate Events`
6. `Databases`
7. `Database Instances`

The linked project pages are enough to define the core vertical slice:

- create or update a project
- read and patch a project
- delete a project
- refresh a project summary
- register a tool against a project

The provider operations list makes it clear that this service version spans more than just the top-level project resource. The implementation plan should therefore stage the broader child and sibling families after the core project routes are stable.

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- reviewed project routes require `api-version=2018-09-01-preview`
- reviewed routes sit under the `Microsoft.Migrate` provider namespace

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.Migrate/operations?api-version=2018-09-01-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}?api-version=2018-09-01-preview`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}?api-version=2018-09-01-preview`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}?api-version=2018-09-01-preview`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}?api-version=2018-09-01-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}/refreshSummary?api-version=2018-09-01-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}/registerTool?api-version=2018-09-01-preview`

Important routing implications from the reviewed pages:
- the top-level resource collection is `migrateProjects`
- the action routes are project-scoped POSTs rather than standalone ARM action resources
- `operations` sits at the provider root and uses the older `GET /providers/Microsoft.Migrate/operations` shape without a subscription segment in the sample
- the project payload uses older ARM resource conventions, including `eTag`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style error responses
- top-level resources return standard fields:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `eTag`
  - `properties.*`

### Operations

`Operations.List`
- route: `/providers/Microsoft.Migrate/operations`
- returns `OperationResultList`
- sample payload includes operations for:
  - `MigrateProjects`
  - `MigrateProjects/Solutions`
  - `MigrateProjects/Machines`
  - `MigrateProjects/MigrateEvents`
  - `MigrateProjects/Databases`
  - `MigrateProjects/DatabaseInstances`

Modeling implication:
- even an initial implementation focused on projects should publish a deterministic operations list that reflects the staged scope and reserves the tail resource families for later additions

### Migrate Projects

Reviewed method pages:
- `Put Migrate Project`
- `Get Migrate Project`
- `Patch Migrate Project`
- `Delete Migrate Project`

Key traits from the reviewed pages:
- top-level route:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/migrateProjects/{migrateProjectName}`
- `PUT` creates or updates the resource
- `PATCH` supports partial updates and can update only tags
- `DELETE` treats deleting a non-existent project as a no-op
- response model includes:
  - `eTag`
  - `properties.provisioningState`
  - `properties.registeredTools`
  - `properties.summary`
  - `properties.refreshSummaryState`
  - `properties.lastSummaryRefreshedTime`

Modeling implications:
- persist canonical project state keyed by ARM ID
- support `PUT`, `GET`, `HEAD`, `PATCH`, and `DELETE`
- keep `summary` deterministic and small while still matching the documented `ServersProjectSummary` / `DatabaseProjectSummary` style
- model `DELETE` as idempotent success

### Refresh Summary

Reviewed method page:
- `Refresh Migrate Project Summary`

Key traits from the reviewed page:
- route:
  - `/migrateProjects/{migrateProjectName}/refreshSummary`
- request body includes `goal`
  - `Servers`
  - `Databases`
- response shape represents refresh state

Modeling implications:
- implement this as a project state transition that updates:
  - `properties.refreshSummaryState`
  - `properties.lastSummaryRefreshedTime`
  - `properties.summary`
- allow deterministic refresh results for both goals

### Register Tool

Reviewed method page:
- `Register Tool`

Key traits from the reviewed page:
- route:
  - `/migrateProjects/{migrateProjectName}/registerTool`
- request body contains `tool`
- documented tool enum includes:
  - `ServerDiscovery`
  - `ServerAssessment`
  - `ServerMigration`
  - `Cloudamize`
  - `Turbonomic`
  - `Zerto`
  - `CorentTech`
  - `ServerAssessmentV1`
  - `ServerMigration_Replication`
  - `Carbonite`
  - `DataMigrationAssistant`
  - `DatabaseMigrationService`
- response contains:
  - `isRegistered`

Modeling implications:
- treat registration as a mutation on the project’s `registeredTools`
- keep registration idempotent
- return a simple `RegistrationResult`

## Recommended Stages

### Stage 0: Wiring and Coverage Discovery

Goal:
- make the new service discoverable in coverage and reserve the route family

Work:
- add `migrate_projects` aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- add the provider entry point in `internal/server/http.go`
- add the plan doc itself
- add a focused test scaffold

Exit criteria:
- coverage scripts recognize `migrate_projects`
- the provider router claims `Microsoft.Migrate/migrateProjects` routes

### Stage 1: Operations + Core Project Lifecycle

Goal:
- cover the essential project resource behavior

Work:
- implement `GET /providers/Microsoft.Migrate/operations`
- implement project:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
- seed deterministic project properties:
  - `provisioningState`
  - `registeredTools`
  - `summary`
  - `refreshSummaryState`
  - `lastSummaryRefreshedTime`

Exit criteria:
- operations and core project CRUD are green

### Stage 2: Project Actions

Goal:
- complete the documented linked action routes on the project itself

Work:
- implement `POST .../registerTool`
- implement `POST .../refreshSummary`
- validate documented enums for:
  - `tool`
  - `goal`
- update project state based on action execution

Exit criteria:
- the core action routes return modeled success and validation failures

### Stage 3: Project Summary and Tail Resource Skeleton

Goal:
- align behavior with the broader provider surface advertised by `operations`

Work:
- harden project summary payloads for both server and database goals
- add minimal deterministic read/list scaffolding for:
  - `solutions`
  - `machines`
  - `migrateEvents`
  - `databases`
  - `databaseInstances`
- decide whether these should be fully implemented in this service or intentionally deferred to separate follow-up plans

Exit criteria:
- the provider no longer advertises clearly missing tail families without at least deterministic read/list placeholders

### Stage 4: Example and Validation

Goal:
- make the new emulator easy to verify and consume

Work:
- add `examples/azure/migrate-projects-2018-09-01-preview`
- use the official Azure Go SDK package:
  - `github.com/AZURE/azure-sdk-for-go/services/preview/migrate/mgmt/2018-09-01-preview/migrate`
- demonstrate:
  - operations list
  - project create/get
  - tool registration
  - summary refresh
- finalize coverage aliases and doc mappings
- run targeted service coverage and repo validation expected for new Azure services

Exit criteria:
- example builds cleanly
- all targeted service endpoints pass the contract, IO, doc, and endpoint coverage scripts

## Implementation Notes

Suggested in-memory layout:

- top-level projects map keyed by ARM ID
- child maps keyed by project ARM ID plus child name for future tails:
  - solutions
  - machines
  - migrate events
  - databases
  - database instances

Suggested helper structure:

- shared ARM helpers:
  - validate `api-version`
  - parse subscription, resource group, and project names
  - emit ARM errors
- project helpers:
  - canonical ARM ID and type construction
  - `eTag` generation
  - summary refresh mutation
  - tool registration mutation

Response-shape strategy:

- prioritize the documented project fields and action outputs over exhaustive reproduction of every optional summary subfield
- keep summary payloads deterministic:
  - `instanceType`
  - `refreshSummaryState`
  - small integer counters
  - `lastSummaryRefreshedTime`

## Testing Strategy

Add focused tests in `internal/server/provider_azure_migrate_projects_test.go` for:

- operations list
- project create/get/head/patch/delete
- idempotent delete on missing project
- `registerTool` success and invalid tool validation
- `refreshSummary` success for both `Servers` and `Databases`
- invalid API version handling
- missing project handling
- missing location handling

Coverage validation target after implementation:

- `python3 scripts/azure-contract-coverage.py --service migrate_projects --require-service migrate_projects --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service migrate_projects --require-service migrate_projects --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service migrate_projects --require-service migrate_projects --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service migrate_projects --require-service migrate_projects --fail-on any`

## Risks and Gaps

- Several Microsoft Learn pages in this older preview family are partially gated, so implementation may need to infer some tail resource shapes from the generated SDK and operation list when the public linked docs are thin.
- The official Go SDK package is track-1 preview code, which is acceptable for the example but may require more boilerplate than newer ARM clients.
- The provider operation list clearly advertises a broader surface than the immediately linked project methods. The implementation should be explicit about which tails are included in the first pass and which are deferred.

## Definition of Done

The staged implementation should be considered complete when:

- `migrate_projects` is recognized by the Azure coverage scripts
- project routes under `Microsoft.Migrate/migrateProjects` return modeled responses instead of `NotImplemented`
- `registerTool` and `refreshSummary` are deterministic and test-covered
- the example builds and exercises the local emulator
- the targeted Azure coverage scripts for `migrate_projects` pass cleanly
