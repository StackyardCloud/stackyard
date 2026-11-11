# Azure ExpressRoute (`expressroute_2025_05_01`) Staged Plan

## Objective

Implement an emulated Azure ExpressRoute Resource Manager surface for API version `2025-05-01`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/expressroute-2025-05-01`

This plan is based on full traversal of the ExpressRoute operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/operation-groups?view=rest-expressroute-2025-05-01`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/expressroute/bgp-service-communities?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-authorizations?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-connections?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-peerings?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connection-peerings?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-links?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-port-authorizations?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports-locations?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-provider-port?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-provider-ports-location?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-service-providers?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/peer-express-route-circuit-connections?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filter-rules?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies?view=rest-expressroute-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policy-definitions?view=rest-expressroute-2025-05-01`

## Reviewed Methods

- bgp-service-communities:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/bgp-service-communities/list?view=rest-expressroute-2025-05-01`
- express-route-circuit-authorizations:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-authorizations/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-authorizations/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-authorizations/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-authorizations/list?view=rest-expressroute-2025-05-01`
- express-route-circuit-connections:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-connections/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-connections/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-connections/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-connections/list?view=rest-expressroute-2025-05-01`
- express-route-circuit-peerings:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-peerings/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-peerings/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-peerings/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuit-peerings/list?view=rest-expressroute-2025-05-01`
- express-route-circuits:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/get-peering-stats?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/get-stats?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/list?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/list-all?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/list-arp-table?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/list-routes-table?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/list-routes-table-summary?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-circuits/update-tags?view=rest-expressroute-2025-05-01`
- express-route-cross-connection-peerings:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connection-peerings/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connection-peerings/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connection-peerings/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connection-peerings/list?view=rest-expressroute-2025-05-01`
- express-route-cross-connections:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/list?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/list-arp-table?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/list-by-resource-group?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/list-routes-table?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/list-routes-table-summary?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-cross-connections/update-tags?view=rest-expressroute-2025-05-01`
- express-route-links:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-links/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-links/list?view=rest-expressroute-2025-05-01`
- express-route-port-authorizations:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-port-authorizations/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-port-authorizations/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-port-authorizations/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-port-authorizations/list?view=rest-expressroute-2025-05-01`
- express-route-ports:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/generate-loa?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/list?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/list-by-resource-group?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports/update-tags?view=rest-expressroute-2025-05-01`
- express-route-ports-locations:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports-locations/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-ports-locations/list?view=rest-expressroute-2025-05-01`
- express-route-provider-port:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-provider-port/express-route-provider-port?view=rest-expressroute-2025-05-01`
- express-route-provider-ports-location:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-provider-ports-location/list?view=rest-expressroute-2025-05-01`
- express-route-service-providers:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/express-route-service-providers/list?view=rest-expressroute-2025-05-01`
- peer-express-route-circuit-connections:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/peer-express-route-circuit-connections/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/peer-express-route-circuit-connections/list?view=rest-expressroute-2025-05-01`
- route-filter-rules:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filter-rules/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filter-rules/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filter-rules/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filter-rules/list-by-route-filter?view=rest-expressroute-2025-05-01`
- route-filters:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters/list?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters/list-by-resource-group?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/route-filters/update-tags?view=rest-expressroute-2025-05-01`
- service-endpoint-policies:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies/list?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies/list-by-resource-group?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policies/update-tags?view=rest-expressroute-2025-05-01`
- service-endpoint-policy-definitions:
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policy-definitions/create-or-update?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policy-definitions/delete?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policy-definitions/get?view=rest-expressroute-2025-05-01`
  - `https://learn.microsoft.com/en-us/rest/api/expressroute/service-endpoint-policy-definitions/list-by-resource-group?view=rest-expressroute-2025-05-01`

Total reviewed methods: `76`.

## Service Naming

- service key: `expressroute_2025_05_01`
- display name: `Azure ExpressRoute`
- API version: `2025-05-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_expressroute_2025_05_01.go`
- tests: `internal/server/provider_azure_expressroute_2025_05_01_test.go`
- example: `examples/azure/expressroute-2025-05-01`
- plan doc: `docs/expressroute-2025-05-01-plan.md`

## Route Envelope

Management-plane route families include:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/...`

Representative routes:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/bgpServiceCommunities?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCircuits/{circuitName}?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/expressRouteCrossConnections/{crossConnectionName}?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ExpressRoutePorts/{expressRoutePortName}?api-version=2025-05-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/serviceEndpointPolicies/{serviceEndpointPolicyName}?api-version=2025-05-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-05-01`
- reviewed methods document `Other Status Codes` with `Common.CloudError` response envelopes

Observed response patterns:

- `200` appears in `76` methods
- `201` appears in `11` methods
- `202` appears in `17` methods
- `204` appears in `11` methods

Request-body coverage from reviewed pages:

- `18` methods include request bodies
- body-bearing methods:
  - `express-route-circuit-authorizations/create-or-update`
  - `express-route-circuit-connections/create-or-update`
  - `express-route-circuit-peerings/create-or-update`
  - `express-route-circuits/create-or-update`
  - `express-route-circuits/update-tags`
  - `express-route-cross-connection-peerings/create-or-update`
  - `express-route-cross-connections/create-or-update`
  - `express-route-cross-connections/update-tags`
  - `express-route-port-authorizations/create-or-update`
  - `express-route-ports/create-or-update`
  - `express-route-ports/generate-loa`
  - `express-route-ports/update-tags`
  - `route-filter-rules/create-or-update`
  - `route-filters/create-or-update`
  - `route-filters/update-tags`
  - `service-endpoint-policies/create-or-update`
  - `service-endpoint-policies/update-tags`
  - `service-endpoint-policy-definitions/create-or-update`

Operation-group method counts:

- bgp-service-communities: `1`
- express-route-circuit-authorizations: `4`
- express-route-circuit-connections: `4`
- express-route-circuit-peerings: `4`
- express-route-circuits: `11`
- express-route-cross-connection-peerings: `4`
- express-route-cross-connections: `8`
- express-route-links: `2`
- express-route-port-authorizations: `4`
- express-route-ports: `7`
- express-route-ports-locations: `2`
- express-route-provider-port: `1`
- express-route-provider-ports-location: `1`
- express-route-service-providers: `1`
- peer-express-route-circuit-connections: `2`
- route-filter-rules: `4`
- route-filters: `6`
- service-endpoint-policies: `6`
- service-endpoint-policy-definitions: `4`

## SDK Reference

- Azure Go SDK runtime/pipeline used by the example:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_expressroute_2025_05_01.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `expressroute_2025_05_01 -> expressroute-2025-05-01-plan.md`

### Stage 1: Request Envelope Validation

- validate required path/query parameters per method
- validate request body requirements for create/update and LOA generation methods
- normalize `Common.CloudError` responses for invalid requests

### Stage 2: Route Families

- implement primary route families incrementally:
  - Express Route Circuits + Authorizations + Peerings + Connections
  - Cross Connections + Cross Connection Peerings
  - Express Route Ports + Port Authorizations + Links + Locations
  - Route Filters + Route Filter Rules
  - Service Endpoint Policies + Definitions
  - BGP communities / Service providers / Provider ports

### Stage 3: Deterministic Fixtures + State

- add deterministic success fixtures for list/get/create/update/delete operations
- maintain in-memory state keyed by ARM identifiers for nested resources
- include deterministic async-capable responses for `202` operations

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_expressroute_2025_05_01_test.go` covering:
  - route matching and version validation
  - success paths for each operation group
  - negative scenarios for body/param validation
  - async-style delete/list-routes/arp-table operations

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `expressroute_2025_05_01`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/expressroute-2025-05-01/main.go` should execute representative calls across all reviewed operation groups, including create/get/list/update/delete operations plus route/ARP table retrieval and LOA generation.
