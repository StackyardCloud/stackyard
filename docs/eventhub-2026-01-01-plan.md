# Azure Event Hubs (`eventhub_2026_01_01`) Staged Plan

## Objective

Implement an emulated Azure Event Hubs Resource Manager surface for API version `2026-01-01`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/eventhub-2026-01-01`

This plan is based on full traversal of the Event Hubs operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/operation-groups?view=rest-eventhub-2026-01-01`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/eventhub/application-group?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/configuration?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/network-security-perimeter-configuration?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/network-security-perimeter-configurations?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/operations?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/private-endpoint-connections?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/private-link-resources?view=rest-eventhub-2026-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventhub/schema-registry?view=rest-eventhub-2026-01-01`

## Reviewed Methods

- application-group:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/application-group/create-or-update-application-group?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/application-group/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/application-group/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/application-group/list-by-namespace?view=rest-eventhub-2026-01-01`
- clusters:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/list-available-cluster-region?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/list-by-resource-group?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/list-by-subscription?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/list-namespaces?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/clusters/update?view=rest-eventhub-2026-01-01`
- configuration:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/configuration/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/configuration/patch?view=rest-eventhub-2026-01-01`
- consumer-groups:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/consumer-groups/list-by-event-hub?view=rest-eventhub-2026-01-01`
- disaster-recovery-configs:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/break-pairing?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/check-name-availability?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/fail-over?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/get-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/list?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/list-authorization-rules?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/disaster-recovery-configs/list-keys?view=rest-eventhub-2026-01-01`
- event-hubs:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/create-or-update-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/delete-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/get-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/list-authorization-rules?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/list-by-namespace?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/list-keys?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/event-hubs/regenerate-keys?view=rest-eventhub-2026-01-01`
- namespaces:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/check-name-availability?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/create-or-update-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/create-or-update-network-rule-set?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/delete-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/failover?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/get-authorization-rule?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/get-network-rule-set?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-authorization-rules?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-by-resource-group?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-keys?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/list-network-rule-set?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/regenerate-keys?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/namespaces/update?view=rest-eventhub-2026-01-01`
- network-security-perimeter-configuration:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/network-security-perimeter-configuration/list?view=rest-eventhub-2026-01-01`
- network-security-perimeter-configurations:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/network-security-perimeter-configurations/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/network-security-perimeter-configurations/get-resource-association-name?view=rest-eventhub-2026-01-01`
- operations:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/operations/list?view=rest-eventhub-2026-01-01`
- private-endpoint-connections:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/private-endpoint-connections/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/private-endpoint-connections/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/private-endpoint-connections/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/private-endpoint-connections/list?view=rest-eventhub-2026-01-01`
- private-link-resources:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/private-link-resources/get?view=rest-eventhub-2026-01-01`
- schema-registry:
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/schema-registry/create-or-update?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/schema-registry/delete?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/schema-registry/get?view=rest-eventhub-2026-01-01`
  - `https://learn.microsoft.com/en-us/rest/api/eventhub/schema-registry/list-by-namespace?view=rest-eventhub-2026-01-01`

Total reviewed methods: `68`.

## Service Naming

- service key: `eventhub_2026_01_01`
- display name: `Azure Event Hubs`
- API version: `2026-01-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_eventhub_2026_01_01.go`
- tests: `internal/server/provider_azure_eventhub_2026_01_01_test.go`
- example: `examples/azure/eventhub-2026-01-01`
- plan doc: `docs/eventhub-2026-01-01-plan.md`

## Route Envelope

Management-plane route families include:

- `/azure/providers/Microsoft.EventHub/...`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EventHub/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/...`

Representative routes:

- `/azure/providers/Microsoft.EventHub/operations?api-version=2026-01-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}?api-version=2026-01-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}?api-version=2026-01-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/consumergroups/{consumerGroupName}?api-version=2026-01-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}?api-version=2026-01-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2026-01-01`
- reviewed methods document `Other Status Codes` with Event Hubs error response envelopes

Observed response patterns:

- `200` appears in `67` methods
- `201` appears in `7` methods
- `202` appears in `11` methods
- `204` appears in `10` methods

Request-body coverage from reviewed pages:

- `19` methods include request bodies
- body-bearing methods:
  - `application-group/create-or-update-application-group`
  - `clusters/create-or-update`
  - `clusters/update`
  - `configuration/patch`
  - `consumer-groups/create-or-update`
  - `disaster-recovery-configs/check-name-availability`
  - `disaster-recovery-configs/create-or-update`
  - `event-hubs/create-or-update`
  - `event-hubs/create-or-update-authorization-rule`
  - `event-hubs/regenerate-keys`
  - `namespaces/check-name-availability`
  - `namespaces/create-or-update`
  - `namespaces/create-or-update-authorization-rule`
  - `namespaces/create-or-update-network-rule-set`
  - `namespaces/failover`
  - `namespaces/regenerate-keys`
  - `namespaces/update`
  - `private-endpoint-connections/create-or-update`
  - `schema-registry/create-or-update`

Operation-group method counts:

- application-group: `4`
- clusters: `8`
- configuration: `2`
- consumer-groups: `4`
- disaster-recovery-configs: `10`
- event-hubs: `10`
- namespaces: `17`
- network-security-perimeter-configuration: `1`
- network-security-perimeter-configurations: `2`
- operations: `1`
- private-endpoint-connections: `4`
- private-link-resources: `1`
- schema-registry: `4`

## SDK Reference

- Azure Go SDK runtime/pipeline used by the example:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_eventhub_2026_01_01.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `eventhub_2026_01_01 -> eventhub-2026-01-01-plan.md`

### Stage 1: Request Envelope Validation

- validate required path and query params per method
- validate create/update/regenerate/check-name request body shape
- normalize Event Hubs error responses for invalid requests

### Stage 2: Route Families

- implement primary route families incrementally:
  - Namespaces
  - Event Hubs + Consumer Groups + Authorization Rules
  - Clusters + Configuration
  - Disaster Recovery Configs
  - Network/Private Endpoint/Schema Registry/Operations

### Stage 3: Deterministic Fixtures + State

- add deterministic success fixtures for list/get/create/update/delete operations
- maintain in-memory state keyed by ARM identifiers for nested resources
- include key regeneration and failover deterministic responses

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_eventhub_2026_01_01_test.go` covering:
  - route matching and version validation
  - success paths for each operation group
  - negative scenarios for body/param validation
  - key regeneration and authorization-rule behavior

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `eventhub_2026_01_01`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/eventhub-2026-01-01/main.go` should execute representative calls across all reviewed operation groups, including create/get/list/update/delete and key/regeneration/failover style operations.
