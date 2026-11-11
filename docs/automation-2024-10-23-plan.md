# Azure Automation (`automation-2024-10-23`) Staged Plan

## Objective

Emulate Azure Automation (`2024-10-23`) with deterministic local behavior for the ARM management-plane surface under `Microsoft.Automation`, including:

- account lifecycle and provider operations
- asset/resource lifecycle for core automation objects
- runbook execution and stream inspection flows
- runtime-environment and package management
- DSC, source-control, hybrid worker, watcher, and software-update tails
- a typed Azure Go SDK example using the Azure Automation SDK
- Azure coverage script wiring for the new service key

This surface is broad enough that it should be implemented in staged slices, with each slice grouped by shared route patterns and response models rather than by one operation group at a time.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/automation/operation-groups?view=rest-automation-2024-10-23`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/automation/activity?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/automation-account?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/certificate?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/connection?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/connection-type?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/credential?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/dsc-configuration?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/dsc-node?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/dsc-node-configuration?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/hybrid-runbook-worker-group?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/hybrid-runbook-workers?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/job?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/job-schedule?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/job-stream?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/module?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/operations?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/package?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/private-endpoint-connections?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/private-link-resources?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/python3package?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/runbook?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/runbook-draft?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/runtime-environments?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/schedule?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/software-update-configuration-machine-runs?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/software-update-configurations?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/source-control?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/source-control-sync-job?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/source-control-sync-job-streams?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/statistics?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/test-job?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/test-job-streams?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/variable?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/watcher?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/webhook?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/convert-graph-runbook-content?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/deleted-automation-accounts?view=rest-automation-2024-10-23`

Representative linked method pages reviewed for staging and model/route details:
- `https://learn.microsoft.com/en-us/rest/api/automation/automation-account/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/automation-account/update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/job/create?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/job/resume?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/runbook/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/runbook/get-content?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/schedule/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/variable/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/webhook/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/webhook/list-by-automation-account?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/runtime-environments/create?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/package/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/private-endpoint-connections/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/software-update-configurations/create?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/source-control/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/credential/create-or-update?view=rest-automation-2024-10-23`
- `https://learn.microsoft.com/en-us/rest/api/automation/certificate/create-or-update?view=rest-automation-2024-10-23`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/automation/armautomation`

## Service Naming

Canonical Stackyard naming:
- service key: `automation`
- display name: `Azure Automation`
- API version: `2024-10-23`

Recommended repo paths:
- implementation: `internal/server/provider_azure_automation.go`
- tests: `internal/server/provider_azure_automation_test.go`
- example: `examples/azure/automation-2024-10-23`
- plan doc: `docs/automation-2024-10-23-plan.md`

## Surface Summary

The reviewed surface is large and spans several clusters:

1. Provider/account and account-adjacent operations
2. Asset lifecycle resources
3. Runbook/job execution resources
4. Runtime environment and package resources
5. DSC and hybrid worker resources
6. Source control and sync resources
7. Software update resources
8. Miscellaneous read-only/utility resources

High-value operation groups for an initial useful emulator:
- `Operations`
- `Automation Account`
- `Credential`
- `Certificate`
- `Connection`
- `Connection Type`
- `Variable`
- `Module`
- `Runbook`
- `Runbook Draft`
- `Schedule`
- `Job Schedule`
- `Job`
- `Job Stream`
- `Webhook`
- `Runtime Environments`
- `Package`
- `Python3Package`
- `Private Endpoint Connections`
- `Private Link Resources`
- `Source Control`
- `Source Control Sync Job`
- `Source Control Sync Job Streams`
- `Statistics`
- `Usages`

Second-tier tails that should be planned but can ship later:
- `Activity`
- `Agent Registration Information`
- `Fields`
- `Object Data Types`
- `Dsc Configuration`
- `Dsc Node`
- `Dsc Node Configuration`
- `Node Count Information`
- `Node Reports`
- `Hybrid Runbook Worker Group`
- `Hybrid Runbook Workers`
- `Test Job`
- `Test Job Streams`
- `Watcher`
- `Software Update Configurations`
- `Software Update Configuration Runs`
- `Software Update Configuration Machine Runs`
- `convert Graph Runbook Content`
- `deleted Automation Accounts`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/providers/Microsoft.Automation/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Automation/automationAccounts`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Automation/automationAccounts/{automationAccountName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Automation/automationAccounts/{automationAccountName}/{childCollection}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Automation/automationAccounts/{automationAccountName}/{childCollection}/{name}`

Examples of key child collections:
- `credentials`
- `certificates`
- `connections`
- `connectionTypes`
- `variables`
- `modules`
- `runbooks`
- `runbooks/{runbookName}/draft`
- `schedules`
- `jobSchedules`
- `jobs`
- `jobs/{jobName}/streams`
- `webhooks`
- `sourceControls`
- `sourceControlSyncJobs`
- `runtimeEnvironments`
- `runtimeEnvironments/{runtimeEnvironmentName}/packages`
- `runtimeEnvironments/{runtimeEnvironmentName}/python3Packages`
- `privateEndpointConnections`
- software-update and DSC collections

Target query:
- `api-version=2024-10-23`

## Contract Notes From Reviewed Pages

Shared management-plane traits:
- ARM OAuth2 routes in Learn; Stackyard should continue to use the repo’s Azure shared-key shim for local examples/tests.
- `resourceGroupName` is consistently required with `minLength: 1`, `maxLength: 90`.
- create/update routes typically return `200` or `201` with typed resource payloads.
- most list routes return `value` envelopes and may expose `nextLink`.

### Automation Account

From the reviewed create/update pages:
- `PUT /automationAccounts/{automationAccountName}`
- request body includes:
  - `location`
  - `tags`
  - `identity`
  - `properties.disableLocalAuth`
  - `properties.publicNetworkAccess`
  - `properties.encryption`
  - `properties.sku`
- response body includes:
  - `id`, `name`, `type`, `location`, `tags`
  - `properties.sku`
  - `properties.state`
  - `properties.creationTime`
  - `properties.lastModifiedBy`
  - `properties.lastModifiedTime`

Modeling implication:
- Automation Account should be the parent resource store for most nested collections.
- An initial deterministic fixture can keep `sku.name`, `state`, and timestamps stable while ignoring deep encryption/private-link internals until the network stage.

### Jobs and Streams

From the reviewed job pages:
- `Job.Create` creates a runbook job under an account.
- `Job.Resume`, `Job.Stop`, and `Job.Suspend` are action-style POSTs.
- `Job.Get Output` and `Job Stream`/`Test Job Streams` imply separate output and stream fixtures.

Modeling implication:
- keep a canonical job record with:
  - `runbook.name`
  - `parameters`
  - `status`
  - `creationTime`, `startTime`, `lastModifiedTime`
  - synthetic `output`
  - synthetic `streams[]`
- action endpoints only need deterministic state transitions for coverage:
  - `New` -> `Running`
  - `Suspend` -> `Suspended`
  - `Resume` -> `Running`
  - `Stop` -> `Stopped`

### Runbooks and Drafts

From the reviewed runbook pages:
- `Runbook.Create Or Update` supports published and draft workflows.
- `Runbook.Get Content` returns textual content, not just JSON metadata.
- `Runbook Draft` has a separate sub-resource.

Modeling implication:
- runbook metadata and textual content should be stored separately.
- first implementation should support:
  - metadata get/list/create/update/delete
  - content get for both published and draft variants
  - a deterministic default `publish` flow if the SDK example requires it

### Credentials, Certificates, Connections, Connection Types, Variables

These groups all follow the same basic pattern:
- create/update
- get
- list by automation account
- delete
- sometimes patch/update

Modeling implication:
- implement these together as the “asset registry” stage.
- they can share:
  - common child-resource routing
  - typed name/id helpers
  - common list envelopes
  - generic not-found/error helpers

### Runtime Environments and Packages

From the reviewed runtime/package pages:
- `Runtime Environments` are parent resources.
- `Package` and `Python3Package` hang off runtime environments.
- request bodies carry package source/import details and lifecycle state.

Modeling implication:
- represent runtime environments as nested parents under automation accounts.
- packages should be stored under a `(account, runtimeEnvironment, packageName)` key.
- one generic package fixture builder can cover both `Package` and `Python3Package`, with a separate Python2 stage only if coverage actually targets it.

### Private Endpoint Connections and Private Link Resources

From the reviewed pages:
- `Private Endpoint Connections` support approve/reject style `Create Or Update`, `Get`, `List`, and `Delete`.
- `Private Link Resources` is read-only discovery.

Modeling implication:
- keep private endpoint connections as account-scoped child resources.
- a deterministic connection-state model is enough:
  - `Pending`
  - `Approved`
  - `Rejected`
- private link resources can be fixture-only.

### Source Control and Sync Jobs

From the reviewed source-control pages:
- `Source Control` is CRUD-like account-scoped state.
- `Source Control Sync Job` and `Source Control Sync Job Streams` are execution/readback tails.

Modeling implication:
- source control should be modeled before sync jobs.
- sync jobs can be synthetic execution envelopes generated off source control records.

### Software Update Configurations

From the reviewed create page:
- `Software Update Configurations` has a comparatively large nested payload with schedule, targets, and update configuration fields.

Modeling implication:
- treat software-update configuration modeling as its own late stage.
- do not block initial Automation service launch on complete update-management semantics.

## Azure Go SDK Example Plan

The official Go SDK package for this surface is:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/automation/armautomation`

The reviewed SDK surface includes clients for the main staged slices, including:
- `AccountClient`
- `CertificateClient`
- `ConnectionClient`
- `ConnectionTypeClient`
- `CredentialClient`
- `JobClient`
- `JobStreamClient`
- `ModuleClient`
- `PackageClient`
- `PrivateEndpointConnectionsClient`
- `PrivateLinkResourcesClient`
- `RunbookClient`
- `RunbookDraftClient`
- `RuntimeEnvironmentsClient`
- `ScheduleClient`
- `SourceControlClient`
- `SourceControlSyncJobClient`
- `SourceControlSyncJobStreamsClient`
- `StatisticsClient`
- `TestJobClient`
- `TestJobStreamsClient`
- `UsagesClient`
- `VariableClient`
- `WatcherClient`
- `WebhookClient`
- and a `ClientFactory`

Recommended example path:
- `examples/azure/automation-2024-10-23`

Example goals:
- construct `ClientFactory` with the repo’s `azsdkshim`
- create/get/list/update/delete an automation account
- create/get/list/update/delete one asset from each core family:
  - credential
  - variable
  - runbook
  - schedule
  - webhook
- create a job and inspect job/stream state
- create a runtime environment and a package
- call one read-only discovery route:
  - operations
  - private link resources
  - statistics or usages

## Coverage Script Wiring

Update:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:
- `automation`
- `azure_automation`
- `automation_2024_10_23`
- `rest_automation_2024_10_23`
- `microsoft_automation`

Recommended doc mapping:
- `automation -> ["automation-2024-10-23-plan.md"]`

## Staged Implementation Plan

### Stage 0: Service Key, Router, and Skeleton

Goal:
- establish the service key, route matcher, test file, example directory, and coverage aliases before deeper modeling

Deliverables:
- `internal/server/provider_azure_automation.go`
- `internal/server/provider_azure_automation_test.go`
- Azure router insertion in `internal/server/http.go`
- coverage alias/doc updates
- example directory scaffold

Behavior:
- recognize `Microsoft.Automation` ARM routes
- reject invalid empty `api-version`
- return typed provider `operations` fixture
- return typed not-found/invalid-request errors for recognized but unimplemented nested resources

### Stage 1: Automation Accounts and Read-Only Discovery

Implement:
- `Operations.List`
- `Automation Account.CreateOrUpdate`
- `Automation Account.Get`
- `Automation Account.List`
- `Automation Account.ListByResourceGroup`
- `Automation Account.Update`
- `Automation Account.Delete`
- `Statistics.ListByAutomationAccount`
- `Usages.ListByAutomationAccount`
- `PrivateLinkResources.Automation`

Why first:
- this provides the parent resource required for nearly every other stage

### Stage 2: Core Asset Registry

Implement together:
- `Credential`
- `Certificate`
- `Connection`
- `Connection Type`
- `Variable`
- `Module`

Common behavior:
- create/update
- get
- list by automation account
- delete

Why grouped:
- they all share the same account-scoped child-resource lifecycle shape

### Stage 3: Runbooks, Drafts, Schedules, Job Schedules, Webhooks

Implement:
- `Runbook`
- `Runbook Draft`
- `Schedule`
- `Job Schedule`
- `Webhook`
- `convert Graph Runbook Content`

Minimum deterministic semantics:
- textual runbook content fixtures
- draft-vs-published separation
- webhook URI generation
- schedule recurrence fixture support
- job-schedule linkage between schedules and runbooks

### Stage 4: Jobs, Activities, Output, and Streams

Implement:
- `Job`
- `Job Stream`
- `Test Job`
- `Test Job Streams`
- `Activity`

Minimum deterministic semantics:
- job state transitions
- output content
- ordered stream records
- basic activity/module discovery for typed clients

### Stage 5: Runtime Environments and Packages

Implement:
- `Runtime Environments`
- `Package`
- `Python3Package`
- optionally `Python2Package` if targeted by coverage/examples

Why separate:
- these routes introduce another parent-child tree under automation accounts

### Stage 6: Source Control and Sync Jobs

Implement:
- `Source Control`
- `Source Control Sync Job`
- `Source Control Sync Job Streams`

Minimum deterministic semantics:
- source-control connection metadata
- sync job records generated from source-control resources
- sync job stream playback

### Stage 7: Network and Identity Tails

Implement:
- `Private Endpoint Connections`
- `Agent Registration Information`
- `Keys`
- `Linked Workspace`
- `deleted Automation Accounts`

Why later:
- these are important for coverage completeness, but not required for a first useful Automation example

### Stage 8: DSC and Hybrid Worker Surface

Implement:
- `Dsc Configuration`
- `Dsc Node`
- `Dsc Node Configuration`
- `Node Count Information`
- `Node Reports`
- `Hybrid Runbook Worker Group`
- `Hybrid Runbook Workers`
- `Fields`
- `Object Data Types`

Why later:
- this is another substantial subdomain with its own models and content payloads

### Stage 9: Watchers and Software Update Management

Implement:
- `Watcher`
- `Software Update Configurations`
- `Software Update Configuration Runs`
- `Software Update Configuration Machine Runs`

Why final:
- this tail is broader and more nested than the core account/runbook lifecycle, and the reviewed create payloads are comparatively large

## Suggested Validation Sequence

For each implemented stage:

1. add focused `internal/server` tests for CRUD, list, not-found, and validation failures
2. build the typed example:
   - `go build ./examples/azure/automation-2024-10-23`
3. run targeted coverage:
   - `python3 scripts/azure-contract-coverage.py --service automation --require-service automation --fail-on any`
   - `python3 scripts/azure-io-contract-coverage.py --service automation --require-service automation --fail-on any`
   - `python3 scripts/azure-doc-contract-coverage.py --service automation --require-service automation --fail-on any`
   - `python3 scripts/azure-endpoint-coverage.py --service automation --require-service automation --fail-on any`
4. once the service is complete, run:
   - `make ci`

## Recommended First Implementation Slice

Start with:
- Stage 0
- Stage 1
- Stage 2
- the smallest useful subset of Stage 3:
  - `Runbook`
  - `Runbook Draft`
  - `Schedule`
  - `Webhook`

That slice yields:
- a realistic parent resource
- several common account-scoped child resource families
- a typed SDK example with meaningful lifecycle coverage
- enough surface to add jobs in the next slice without reworking the storage model
