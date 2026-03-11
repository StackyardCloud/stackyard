# Azure AI Foundry - Account Management (`account-management-2025-06-01`) Staged Plan

## Objective

Emulate Azure AI Foundry Account Management (`2025-06-01`) with deterministic local behavior for management-plane account, deployment, project, connection, capacity, security, and policy workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/operation-groups?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/account-capability-hosts?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/account-connections?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/accounts?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/check-domain-availability?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/check-sku-availability?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/commitment-plans?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/commitment-tiers?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/defender-for-ai-settings?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/deleted-accounts?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/deployments?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/encryption-scopes?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/location-based-model-capacities?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/model-capacities?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/models?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/network-security-perimeter-configurations?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/operations?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/private-endpoint-connections?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/private-link-resources?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/project-capability-hosts?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/project-connections?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/projects?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/rai-blocklist-items?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/rai-blocklists?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/rai-content-filters?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/rai-policies?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/resource-skus?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/usages?view=rest-aifoundry-accountmanagement-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/accountmanagement/calculate-model-capacity?view=rest-aifoundry-accountmanagement-2025-06-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full Azure control-plane provisioning internals in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.CognitiveServices/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.CognitiveServices/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.CognitiveServices/*`

Target API version:
- `api-version=2025-06-01`

## API Surface and Contract Notes

Operation-group and method documentation reviewed end-to-end:
- 28 operation-group pages.
- 99 method pages linked from those groups.

Common contract characteristics across method pages:
- ARM management-plane path patterns using `subscriptions`, `resourceGroups`, and `providers/Microsoft.CognitiveServices`.
- Success status families across operations include `200`, `201`, `202`, and `204`.
- Common typed error classes include invalid request/input, authorization failures, not found, conflict, and service/internal failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure AI Foundry Account Management router surface.
- Recognize provider-level and subscription/resource-group management-plane envelopes.
- Add route-recognition tests for operations, accounts, projects, connections, capability hosts, vector-store connections, and usages.

Exit criteria:
- Recognized route envelopes are deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating for management-plane routes.
- Return deterministic invalid-request payloads for malformed requests.

Exit criteria:
- Invalid `api-version` and malformed request patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes return deterministic success fixtures in tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example in `examples/azure/ai-foundry/account-management-2025-06-01`.
- Exercise representative management calls across operations, accounts, projects, connections, capability hosts, and usages.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add service aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for the new service key.

Exit criteria:
- Coverage tools resolve `ai_foundry_account_management` and versioned aliases consistently.
