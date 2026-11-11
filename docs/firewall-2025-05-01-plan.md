# Azure Firewall (`firewall_2025_05_01`) Staged Plan

## Objective

Implement an emulated Azure Firewall Resource Manager surface for API version `2025-05-01`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/firewall-2025-05-01`

This plan is based on full traversal of the Firewall operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/operation-groups?view=rest-firewall-2025-05-01`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewall-fqdn-tags?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-deployments?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-drafts?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-filter-values?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-overrides?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-group-drafts?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-groups?view=rest-firewall-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/firewall/web-categories?view=rest-firewall-2025-05-01`

## Reviewed Methods

- azure-firewall-fqdn-tags:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewall-fqdn-tags/list-all?view=rest-firewall-2025-05-01`
- azure-firewalls:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/create-or-update?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/delete?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/get?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/list?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/list-all?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/list-learned-prefixes?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/packet-capture?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/packet-capture-operation?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/azure-firewalls/update-tags?view=rest-firewall-2025-05-01`
- firewall-policies:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies/create-or-update?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies/delete?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies/get?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies/list?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies/list-all?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policies/update-tags?view=rest-firewall-2025-05-01`
- firewall-policy-deployments:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-deployments/deploy?view=rest-firewall-2025-05-01`
- firewall-policy-drafts:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-drafts/create-or-update?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-drafts/delete?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-drafts/get?view=rest-firewall-2025-05-01`
- firewall-policy-idps-signatures:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures/list?view=rest-firewall-2025-05-01`
- firewall-policy-idps-signatures-filter-values:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-filter-values/list?view=rest-firewall-2025-05-01`
- firewall-policy-idps-signatures-overrides:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-overrides/get?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-overrides/list?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-overrides/patch?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-idps-signatures-overrides/put?view=rest-firewall-2025-05-01`
- firewall-policy-rule-collection-group-drafts:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-group-drafts/create-or-update?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-group-drafts/delete?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-group-drafts/get?view=rest-firewall-2025-05-01`
- firewall-policy-rule-collection-groups:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-groups/create-or-update?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-groups/delete?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-groups/get?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/firewall-policy-rule-collection-groups/list?view=rest-firewall-2025-05-01`
- web-categories:
  - `https://learn.microsoft.com/en-us/rest/api/firewall/web-categories/get?view=rest-firewall-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/firewall/web-categories/list-by-subscription?view=rest-firewall-2025-05-01`

Total reviewed methods: `35`.

## Service Naming

- service key: `firewall_2025_05_01`
- display name: `Azure Firewall`
- API version: `2025-05-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_firewall_2025_05_01.go`
- tests: `internal/server/provider_azure_firewall_2025_05_01_test.go`
- example: `examples/azure/firewall-2025-05-01`
- plan doc: `docs/firewall-2025-05-01-plan.md`

## Route Envelope

Management-plane route families include:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/...`

Representative routes:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/azureFirewallFqdnTags?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/azureFirewalls/{azureFirewallName}?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/firewallPolicies/{firewallPolicyName}/ruleCollectionGroups/{ruleCollectionGroupName}?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/azureWebCategories/{name}?api-version=2025-05-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-05-01`
- reviewed methods document `Other Status Codes` with `ErrorResponse` envelopes

Observed response patterns:

- `200` appears in `33` methods
- `201` appears in `5` methods
- `202` appears in `8` methods
- `204` appears in `5` methods

Request-body coverage from reviewed pages:

- `13` methods include request bodies
- body-bearing methods:
  - `azure-firewalls/create-or-update`
  - `azure-firewalls/packet-capture`
  - `azure-firewalls/packet-capture-operation`
  - `azure-firewalls/update-tags`
  - `firewall-policies/create-or-update`
  - `firewall-policies/update-tags`
  - `firewall-policy-drafts/create-or-update`
  - `firewall-policy-idps-signatures/list`
  - `firewall-policy-idps-signatures-filter-values/list`
  - `firewall-policy-idps-signatures-overrides/patch`
  - `firewall-policy-idps-signatures-overrides/put`
  - `firewall-policy-rule-collection-group-drafts/create-or-update`
  - `firewall-policy-rule-collection-groups/create-or-update`

Operation-group method counts:

- azure-firewall-fqdn-tags: `1`
- azure-firewalls: `9`
- firewall-policies: `6`
- firewall-policy-deployments: `1`
- firewall-policy-drafts: `3`
- firewall-policy-idps-signatures: `1`
- firewall-policy-idps-signatures-filter-values: `1`
- firewall-policy-idps-signatures-overrides: `4`
- firewall-policy-rule-collection-group-drafts: `3`
- firewall-policy-rule-collection-groups: `4`
- web-categories: `2`

## SDK Reference

- Azure Go SDK runtime/pipeline used by the example:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_firewall_2025_05_01.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `firewall_2025_05_01 -> firewall-2025-05-01-plan.md`

### Stage 1: Request Envelope Validation

- validate required path/query parameters per method
- validate body requirements for create/update, packet-capture, IDPS filter/list and signature overrides
- normalize `ErrorResponse` payloads for invalid requests

### Stage 2: Route Families

- implement operation groups incrementally:
  - azure-firewall-fqdn-tags
  - azure-firewalls
  - firewall-policies
  - firewall-policy-drafts / deployments
  - firewall-policy-idps-signatures / filter-values / overrides
  - firewall-policy-rule-collection-groups / drafts
  - web-categories

### Stage 3: Deterministic Fixtures + State

- add deterministic fixtures for list/get/create/update/delete and draft resources
- maintain in-memory state keyed by firewall/policy ARM identifiers
- include deterministic long-running responses for `202` methods

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_firewall_2025_05_01_test.go` covering:
  - route matching and API version validation
  - success paths for each operation group
  - negative scenarios for body/param validation
  - async delete/deploy/suspend-like packet operations semantics

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `firewall_2025_05_01`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/firewall-2025-05-01/main.go` executes representative calls across all reviewed operation groups, including CRUD, drafts, IDPS listing/overrides, packet capture operations, and category listing.
