# Azure Migrate Assessment (`migrate-assessment-2024-01-15`) Staged Plan

## Objective

Emulate the Azure Migrate Assessment ARM management API for `2024-01-15` with deterministic local behavior for:

- provider `operations`
- top-level `assessmentProjects` lifecycle and discovery routes
- core nested resources such as `groups`, `machines`, `assessments`, `assessedMachines`, `assessmentOptions`, and `projectSummary`
- collector and connectivity resources such as `vmwarecollectors`, `hypervcollectors`, `servercollectors`, `importcollectors`, `privateEndpointConnections`, and `privateLinkResources`
- workload-specific families such as AKS, AVS, SQL v2, and Web App assessment resources
- an Azure Go SDK example under `examples/azure/migrate-assessment-2024-01-15`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure Migrate Assessment under `Microsoft.Migrate`.
- This is not the Azure Migrate appliance data plane or agent endpoint surface.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2024-01-15`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/operation-groups?view=rest-migrate-assessment-2024-01-15`

Representative operation-group and method pages reviewed across the surface:
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/operations/list?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-projects-operations/create?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-projects-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-projects-operations/list-by-resource-group?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-projects-operations/list-by-subscription?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-projects-operations/delete?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/groups-operations/create?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/groups-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/groups-operations?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-options-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-project-summary-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessment-project-summary-operations?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/machines-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessments-operations/create?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessments-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessments-operations/list-by-group?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessed-machines-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/assessed-machines-operations/list-by-assessment?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/vmware-collectors-operations/create?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/server-collectors-operations/create?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/import-collectors-operations/create?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/private-endpoint-connection-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/private-endpoint-connection-operations/list-by-assessment-project?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/private-endpoint-connection-operations/update?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/private-link-resource-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/private-link-resource-operations/list-by-assessment-project?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/aks-options-operations/get?view=rest-migrate-assessment-2024-01-15`
- `https://learn.microsoft.com/en-us/rest/api/migrate/assessment/aks-summary-operations/get?view=rest-migrate-assessment-2024-01-15`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/migrationassessment/armmigrationassessment`

SDK selection note:
- The published Go SDK package uses the newer `migrationassessment` module name rather than `migrate`.
- The package appears to expose a broad subset of this API family, but version alignment may not be exact for every `2024-01-15` route.
- Plan assumption: use typed `armmigrationassessment` clients where available, and raw `azcore/arm` requests only for uncovered gaps.

## Service Naming

Canonical Stackyard naming:
- service key: `migrate_assessment`
- display name: `Azure Migrate Assessment`
- provider namespace: `Microsoft.Migrate`
- API version: `2024-01-15`

Recommended repo paths:
- implementation: `internal/server/provider_azure_migrate_assessment.go`
- tests: `internal/server/provider_azure_migrate_assessment_test.go`
- example: `examples/azure/migrate-assessment-2024-01-15`
- plan doc: `docs/migrate-assessment-2024-01-15-plan.md`

Recommended coverage aliases:
- `migrate_assessment`
- `migrate-assessment`
- `migration_assessment`
- `azure_migrate_assessment`
- `azure_migrationassessment`
- `microsoft_migrate`
- `microsoft_migrate_assessmentprojects`
- `rest_migrate_assessment_2024_01_15`
- `migrate_assessment_2024_01_15`

## Surface Summary

The reviewed `2024-01-15` management surface exposes a large set of operation groups, including:

1. `Operations`
2. `Assessment Projects Operations`
3. `Assessment Project Summary Operations`
4. `Assessment Options Operations`
5. `Groups Operations`
6. `Machines Operations`
7. `Assessments Operations`
8. `Assessed Machines Operations`
9. `Vmware Collectors Operations`
10. `Hyperv Collectors Operations`
11. `Server Collectors Operations`
12. `Import Collectors Operations`
13. `Sql Collector Operations`
14. `Web App Collector Operations`
15. `Private Endpoint Connection Operations`
16. `Private Link Resource Operations`
17. `Aks Assessment Operations`
18. `Aks Cluster Operations`
19. `Aks Cost Detail Operations`
20. `Aks Options Operations`
21. `Aks Summary Operations`
22. `Avs Assessments Operations`
23. `Avs Assessed Machines Operations`
24. `Avs Assessment Options Operations`
25. `Sql Assessment V 2 Operations`
26. `Sql Assessment V 2 Summary Operations`
27. `Assessed Sql Database V 2 Operations`
28. `Assessed Sql Instance V 2 Operations`
29. `Assessed Sql Machines Operations`
30. `Assessed Sql Recommended Entity Operations`
31. `Web App Assessment V 2 Operations`
32. `Web App Assessment V 2 Summary Operations`
33. `Assessed Web App V 2 Operations`
34. `Assessed Web Application Operations`

This is too large to implement as one flat slice. The implementation should be staged by resource family, with core ARM project/group/assessment resources first and workload-specific tails later.

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2024-01-15`
- all routes sit under the `Microsoft.Migrate` provider namespace

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.Migrate/operations?api-version=2024-01-15`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}?api-version=2024-01-15`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Migrate/assessmentProjects?api-version=2024-01-15`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/groups/{groupName}?api-version=2024-01-15`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/groups/{groupName}/updateMachines?api-version=2024-01-15`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/machines/{machineName}?api-version=2024-01-15`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/groups/{groupName}/assessments/{assessmentName}?api-version=2024-01-15`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/groups/{groupName}/assessments/{assessmentName}/assessedMachines?api-version=2024-01-15`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/assessmentOptions/default?api-version=2024-01-15`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/vmwarecollectors/{collectorName}?api-version=2024-01-15`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/privateEndpointConnections?api-version=2024-01-15`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}/privateLinkResources?api-version=2024-01-15`

Important routing implications from the reviewed pages:
- core resource nesting is consistent and deep:
  - `assessmentProjects/{projectName}`
  - `groups/{groupName}`
  - `assessments/{assessmentName}`
  - `assessedMachines/{assessedMachineName}`
- many workload-specific families sit directly under the project root rather than under groups
- several collector names and machine IDs are effectively opaque ARM child names with permissive patterns
- some create routes in the family are modeled as long-running ARM operations and should tolerate accepted-style responses in later stages

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- top-level and child resources return standard ARM resource fields:
  - `id`
  - `name`
  - `type`
  - `systemData`
  - `properties.*`
- resource names generally follow the pattern `^[^<>&:\?/#]{1,260}$`
- subscription IDs are UUIDs

### Operations

`Operations.List`
- provider route under `/providers/Microsoft.Migrate/operations`
- returns `OperationListResult`
- sample payload includes:
  - `name`
  - `isDataAction`
  - `display.provider`
  - `display.resource`
  - `display.operation`
  - `display.description`
  - `origin`
  - `actionType`
  - `nextLink`

Modeling implication:
- expose a deterministic `value` list covering the core resource families implemented in each stage
- include at least `assessmentProjects`, `groups`, `machines`, `assessments`, collector resources, and private link resources once those slices are implemented

### Assessment Projects

Reviewed method pages:
- `Create`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Delete`

Key traits from the reviewed pages:
- top-level project route:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Migrate/assessmentProjects/{projectName}`
- project listing routes exist at both resource-group and subscription scope
- delete is idempotent with `200` or `204`

Modeling implications:
- persist a canonical project store keyed by ARM ID
- support `PUT`, `GET`, `HEAD`, list-by-RG, list-by-subscription, and delete
- synthesize a stable project status payload and leave deeper per-workload details to later stages

### Groups

Reviewed method pages:
- `Create`
- `Get`
- `List By Assessment Project`
- `Delete`
- `Update Machines`

Key traits from the reviewed pages:
- group route nests under a project:
  - `/assessmentProjects/{projectName}/groups/{groupName}`
- `Update Machines` is an action route for membership updates
- create body includes at least:
  - `properties.groupType`

Modeling implications:
- store groups under their parent project
- maintain a simple membership list of referenced machine IDs
- implement `Update Machines` as a deterministic membership replace/merge operation

### Machines

Reviewed method page:
- `Get`

Key traits from the reviewed page:
- machine route nests directly under project:
  - `/assessmentProjects/{projectName}/machines/{machineName}`
- sample payload includes detailed machine inventory:
  - `properties.disks`
  - `properties.networkAdapters`
  - `properties.productSupportStatus`
  - `properties.groups`
  - `properties.sqlInstances`
  - `properties.webApplications`
  - `properties.workloadSummary`

Modeling implications:
- treat machines as mostly read-only discovered inventory in the first pass
- generate deterministic machine fixtures so group membership and assessments have something to point at

### Assessments And Assessed Machines

Reviewed method pages:
- `Assessments Create`
- `Assessments Get`
- `Assessments List By Group`
- `Assessments Delete`
- `Assessed Machines Get`
- `Assessed Machines List By Assessment`

Key traits from the reviewed pages:
- assessment route nests under:
  - `/assessmentProjects/{projectName}/groups/{groupName}/assessments/{assessmentName}`
- assessed-machine route nests under:
  - `/assessmentProjects/{projectName}/groups/{groupName}/assessments/{assessmentName}/assessedMachines/{assessedMachineName}`
- assessment payload is large and includes:
  - sizing and cost settings
  - location, pricing tier, storage redundancy, VM families
  - utilization percentile and time-range controls
  - suitability summary and error summary

Modeling implications:
- first-pass assessment creation should accept a shallow subset of user-configurable fields and synthesize the rest deterministically
- assessed machines should be derived from group membership and exposed as read-only children of an assessment

### Assessment Options And Project Summary

Reviewed method pages:
- `Assessment Options Get`
- `Assessment Project Summary Get`
- `Assessment Project Summary List By Assessment Project`

Modeling implications:
- these are good early read-only routes for coverage because they have stable, low-mutation payloads
- `assessmentOptions/default` should be a deterministic singleton per project
- `projectSummary` should aggregate counts from the emulator’s current project, group, and assessment state

### Collectors

Reviewed method pages:
- `Vmware Collectors Create`
- `Server Collectors Create`
- `Import Collectors Create`

Additional collector families listed on the operation-groups page:
- `Hyperv Collectors Operations`
- `Sql Collector Operations`
- `Web App Collector Operations`

Modeling implications:
- treat collectors as top-level child resources under a project
- collector families share similar CRUD/list structure and should use a common generic implementation with family-specific `type` values
- create/get/list/delete is a strong first-pass target; family-specific action routes can follow later if coverage requires them

### Private Link And Private Endpoint Connections

Reviewed method pages:
- `Private Endpoint Connection Get`
- `Private Endpoint Connection List By Assessment Project`
- `Private Endpoint Connection Update`
- `Private Link Resource Get`
- `Private Link Resource List By Assessment Project`

Modeling implications:
- implement these as read-mostly child resources under a project
- private endpoint connection update should accept and persist the connection-state subdocument
- private link resources can be deterministic fixtures derived from the project

### Workload-Specific Families

Reviewed method pages:
- `Aks Options Operations Get`
- `Aks Summary Operations Get`

Operation-groups page also lists:
- `Aks Assessment Operations`
- `Aks Cluster Operations`
- `Aks Cost Detail Operations`
- `Avs Assessed Machines Operations`
- `Avs Assessment Options Operations`
- `Avs Assessments Operations`
- `Sql Assessment Options Operations`
- `Sql Assessment V 2 Operations`
- `Sql Assessment V 2 Summary Operations`
- `Assessed Sql Database V 2 Operations`
- `Assessed Sql Instance V 2 Operations`
- `Assessed Sql Machines Operations`
- `Assessed Sql Recommended Entity Operations`
- `Web App Assessment Options Operations`
- `Web App Assessment V 2 Operations`
- `Web App Assessment V 2 Summary Operations`
- `Assessed Web App V 2 Operations`
- `Assessed Web Application Operations`

Modeling implications:
- these should be staged after the generic project/group/machine/assessment foundation is in place
- the generic approach is:
  - create the parent assessment/option resource
  - expose workload-specific read/list children
  - return deterministic summary/detail payloads derived from the parent and current machine membership

## State Model Recommendation

Recommended in-memory state:
- `assessmentProjects`
- `assessmentGroups`
- `assessmentMachines`
- `assessmentResources`
- `assessedMachines`
- `assessmentOptions`
- `projectSummaries`
- `collectorResources`
- `privateEndpointConnections`
- `privateLinkResources`
- workload-specific read models for AKS, AVS, SQL v2, and Web App v2

Suggested type families:
- `azureMigrateAssessmentProject`
- `azureMigrateAssessmentGroup`
- `azureMigrateAssessmentMachine`
- `azureMigrateAssessment`
- `azureMigrateAssessmentCollector`
- `azureMigrateAssessmentPrivateEndpointConnection`
- `azureMigrateAssessmentPrivateLinkResource`

## Staged Implementation Plan

### Stage 0: Route And Coverage Wiring

Goals:
- reserve a distinct Azure ARM router entry for `Microsoft.Migrate`
- add service aliases to Azure coverage scripts
- map the plan doc for doc coverage

Work:
- add route entry in `internal/server/http.go`
- add aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:
- coverage scripts resolve `migrate_assessment`
- unmatched Migrate routes no longer fall through to unrelated handlers

### Stage 1: Core Provider Skeleton And Operations

Goals:
- establish shared ARM validation, `api-version` handling, and provider operations

Work:
- create `internal/server/provider_azure_migrate_assessment.go`
- implement:
  - `GET /providers/Microsoft.Migrate/operations`
- add shared helpers for:
  - subscription/resource-group/project route parsing
  - ARM error responses
  - `systemData`
  - paged `value` responses

Exit criteria:
- provider discovery is implemented
- invalid routes fail with modeled ARM errors rather than `NotImplemented`

### Stage 2: Assessment Projects

Goals:
- implement the top-level persisted resource first

Work:
- implement:
  - `PUT`, `GET`, `HEAD`, `DELETE`
  - list by resource group
  - list by subscription
- persist tags and a minimal deterministic properties block

Exit criteria:
- `assessmentProjects` routes are complete and coverage-safe
- repeated deletes are idempotent

### Stage 3: Groups, Machines, Assessment Options, Project Summary

Goals:
- build the project-scoped foundation used by later assessment routes

Work:
- implement:
  - groups create/get/list/delete
  - groups update-machines action
  - machines get and list-by-project if required by coverage
  - `assessmentOptions/default`
  - project summary get/list

Exit criteria:
- a project can contain groups and discovered machine fixtures
- summary and options endpoints return typed success payloads

### Stage 4: Assessments And Assessed Machines

Goals:
- implement the main evaluation resource family

Work:
- implement:
  - assessments create/get/list/delete
  - assessed-machines get/list
- derive assessed machines from current group membership
- synthesize cost/suitability/status data deterministically

Exit criteria:
- end-to-end project -> group -> assessment -> assessed machine flow is complete

### Stage 5: Generic Collectors And Private Networking

Goals:
- cover the shared project child families with minimal duplicated code

Work:
- implement collector CRUD/list for:
  - vmware
  - hyperv
  - server
  - import
  - sql
  - webapp
- implement private endpoint connections get/list/update
- implement private link resources get/list

Exit criteria:
- collector and private networking families no longer return `NotImplemented`

### Stage 6: Workload-Specific Assessment Tails

Goals:
- add the specialized resource families after the shared foundation is stable

Work:
- implement staged slices for:
  - AKS options, assessments, clusters, cost details, summaries
  - AVS options, assessments, assessed machines
  - SQL assessment options, SQL v2 assessments, summaries, assessed SQL children
  - Web App assessment options, Web App v2 assessments, summaries, assessed web app children

Exit criteria:
- workload-specific coverage for the chosen slice is green before moving to the next slice

### Stage 7: Example And Coverage Validation

Goals:
- add a typed Azure Go SDK example and prove the service is wired across repo validation

Work:
- add:
  - `examples/azure/migrate-assessment-2024-01-15/main.go`
  - matching `Dockerfile`
  - matching `docker-compose.yml`
- example should use `armmigrationassessment` clients where available
- run:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
  - `scripts/azure-endpoint-coverage.py`

Exit criteria:
- the new service is discoverable
- no implemented endpoint falls through to `NotImplemented`

## Testing Strategy

Focused tests should cover:
- provider operations list
- project create/get/list/delete
- group membership update and derived machine references
- assessment create/list/get/delete
- assessed-machines list derived from group membership
- collector create/get/list/delete
- private endpoint connection update
- invalid `api-version`, invalid subscription IDs, and malformed ARM names

Because the surface is large, tests should be added in the same staged order as implementation so each slice remains reviewable.

## Implementation Risks

Main risks from the reviewed surface:
- the service is broad enough that a single-provider file can become unwieldy
- workload-specific resource families have deeply nested paths and slightly different shape contracts
- some method pages are authorization-gated, so route discovery is easy but full payload fidelity may need iteration during implementation
- SDK coverage may lag specific `2024-01-15` routes even though `armmigrationassessment` exists

Recommended mitigation:
- split implementation by family once the provider grows beyond a manageable single file
- start with shared ARM and generic child-resource helpers
- keep examples typed where possible and fall back to raw pipeline requests only for exact gaps

## Definition Of Done

The service is done when:
- `migrate_assessment` is routed separately and resolved by coverage aliases
- the core project/group/assessment families are implemented with deterministic ARM payloads
- collector and private networking families no longer return `NotImplemented`
- at least one representative workload-specific family is fully covered, then the remaining families follow in staged slices
- an example exists under `examples/azure/migrate-assessment-2024-01-15`
- focused tests pass
- Azure coverage scripts confirm the service is wired and does not fall through to `NotImplemented`
