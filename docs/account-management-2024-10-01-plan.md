# Azure AI Services - Account Management (`account-management-2024-10-01`) Staged Plan

## Objective

Emulate Azure AI Services Account Management (`2024-10-01`) with deterministic local behavior for account lifecycle, deployments, networking, policy, capacity, and related management-plane workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/operation-groups?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/accounts?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/check-domain-availability?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/check-sku-availability?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/commitment-plans?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/commitment-tiers?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/defender-for-ai-settings?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/deleted-accounts?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/deployments?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/encryption-scopes?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/location-based-model-capacities?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/model-capacities?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/models?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/network-security-perimeter-configurations?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/operations?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/private-endpoint-connections?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/private-link-resources?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/rai-blocklist-items?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/rai-blocklists?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/rai-content-filters?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/rai-policies?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/resource-skus?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/usages?view=rest-aiservices-accountmanagement-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/accountmanagement/usages/calculate-model-capacity?view=rest-aiservices-accountmanagement-2024-10-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full platform-side provisioning, billing, and quota enforcement in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.CognitiveServices/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.CognitiveServices/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.CognitiveServices/*`

Target API version:
- `api-version=2024-10-01`

## API Surface and Contract Notes

Operation groups and method pages reviewed:

- Accounts: create/update/delete/get/list/list keys/regenerate keys/list skus/update/list deleted/restore.
- Check Domain Availability: post domain-availability validation.
- Check Sku Availability: post sku-availability validation.
- Commitment Plans: create/delete/get/list/update/list associations/move associations/list resources.
- Commitment Tiers: list commitment tiers.
- Defender for AI Settings: get/update.
- Deleted Accounts: list/get/purge.
- Deployments: create/update/delete/get/list/list skus/list upgrades/validate/purge.
- Encryption Scopes: create/delete/get/list/update.
- Location Based Model Capacities: list.
- Model Capacities: list.
- Models: list.
- Network Security Perimeter Configurations: list/get/reconcile.
- Operations: list provider operations.
- Private Endpoint Connections: list/get/create/update/delete.
- Private Link Resources: list/get.
- RAI Blocklist Items: add/delete/get/list.
- RAI Blocklists: create/delete/get/list/update.
- RAI Content Filters: get/list.
- RAI Policies: create/delete/get/list/update.
- Resource Skus: list.
- Usages: list and `calculateModelCapacity`.

Common contract characteristics from method pages:
- ARM-style management-plane endpoint patterns (`subscriptions`, `resourceGroups`, `providers/Microsoft.CognitiveServices`).
- Common success codes include `200/201/202/204` depending on operation semantics.
- Common error classes include invalid request shape/parameters, authorization failures, not found, conflict, and service failures with typed error envelopes.

## Stage 0: Contract Skeleton

- Add dedicated Azure Account Management router surface.
- Recognize all ARM route envelopes scoped to `Microsoft.CognitiveServices`.
- Add route-recognition tests spanning all operation groups.

Exit criteria:
- Route envelope and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Validate minimum routing shape for provider/subscription/resource-group scoped paths.
- Return deterministic invalid-request contract for malformed query shape.

Exit criteria:
- Invalid query/method patterns return stable validation payloads.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized account-management routes.
- Preserve stable `provider/path/status` payload shape used across Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/account-management-2024-10-01`.
- Exercise representative account-management flows (accounts, deployments, keys/skus/operations, usage/capacity).
- Update Azure contract, IO, and doc coverage scripts with service aliases and plan mapping.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
