# Azure DNS Resolver (`dns_dnsresolver_2025_05_01`) Staged Plan

## Objective

Emulate Azure DNS Resolver Resource Manager REST API (`api-version=2025-05-01`) with deterministic local behavior for:

- DNS resolvers and endpoint lifecycle (resolvers, inbound endpoints, outbound endpoints)
- DNS forwarding ruleset lifecycle (rulesets, forwarding rules, virtual network links)
- DNS resolver policy lifecycle (policies, policy virtual network links, DNS security rules, domain lists)
- an Azure Go SDK example under `examples/azure/dns-dnsresolver-2025-05-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/operation-groups?view=rest-dns-dnsresolver-2025-05-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-security-rules?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/forwarding-rules?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/inbound-endpoints?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/outbound-endpoints?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/virtual-network-links?view=rest-dns-dnsresolver-2025-05-01`

Method pages reviewed directly: all linked methods from all operation groups (`58` total).

dns-forwarding-rulesets methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/list-by-resource-group?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/list-by-virtual-network?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-forwarding-rulesets/update?view=rest-dns-dnsresolver-2025-05-01`

dns-resolver-domain-lists methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/bulk?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/list-by-resource-group?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-domain-lists/update?view=rest-dns-dnsresolver-2025-05-01`

dns-resolver-policies methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/list-by-resource-group?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/list-by-virtual-network?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policies/update?view=rest-dns-dnsresolver-2025-05-01`

dns-resolver-policy-virtual-network-links methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolver-policy-virtual-network-links/update?view=rest-dns-dnsresolver-2025-05-01`

dns-resolvers methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/list-by-resource-group?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/list-by-virtual-network?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-resolvers/update?view=rest-dns-dnsresolver-2025-05-01`

dns-security-rules methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-security-rules/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-security-rules/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-security-rules/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-security-rules/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/dns-security-rules/update?view=rest-dns-dnsresolver-2025-05-01`

forwarding-rules methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/forwarding-rules/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/forwarding-rules/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/forwarding-rules/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/forwarding-rules/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/forwarding-rules/update?view=rest-dns-dnsresolver-2025-05-01`

inbound-endpoints methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/inbound-endpoints/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/inbound-endpoints/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/inbound-endpoints/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/inbound-endpoints/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/inbound-endpoints/update?view=rest-dns-dnsresolver-2025-05-01`

outbound-endpoints methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/outbound-endpoints/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/outbound-endpoints/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/outbound-endpoints/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/outbound-endpoints/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/outbound-endpoints/update?view=rest-dns-dnsresolver-2025-05-01`

virtual-network-links methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/virtual-network-links/create-or-update?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/virtual-network-links/delete?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/virtual-network-links/get?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/virtual-network-links/list?view=rest-dns-dnsresolver-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/dnsresolver/virtual-network-links/update?view=rest-dns-dnsresolver-2025-05-01`

## Surface Summary

Reviewed operation groups and methods:

- dns-forwarding-rulesets (7)
- dns-resolver-domain-lists (7)
- dns-resolver-policies (7)
- dns-resolver-policy-virtual-network-links (5)
- dns-resolvers (7)
- dns-security-rules (5)
- forwarding-rules (5)
- inbound-endpoints (5)
- outbound-endpoints (5)
- virtual-network-links (5)

Total reviewed methods: `58`.

## Service Naming

Canonical Stackyard naming:

- service key: `dns_dnsresolver_2025_05_01`
- display name: `Azure DNS Resolver`
- API version: `2025-05-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_dns_dnsresolver_2025_05_01.go`
- tests: `internal/server/provider_azure_dns_dnsresolver_2025_05_01_test.go`
- example: `examples/azure/dns-dnsresolver-2025-05-01`
- plan doc: `docs/dns-dnsresolver-2025-05-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsResolvers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsForwardingRulesets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsResolverPolicies`

Representative routes:

- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsForwardingRulesets?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets?api-version=2025-05-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsForwardingRulesets?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}?api-version=2025-05-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}/bulk?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsResolverDomainLists?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverDomainLists/{dnsResolverDomainListName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsResolverPolicies?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies?api-version=2025-05-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsResolverPolicies?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/virtualNetworkLinks/{dnsResolverPolicyVirtualNetworkLinkName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/dnsResolvers?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers?api-version=2025-05-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/listDnsResolvers?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolverPolicies/{dnsResolverPolicyName}/dnsSecurityRules/{dnsSecurityRuleName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/forwardingRules/{forwardingRuleName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/inboundEndpoints/{inboundEndpointName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsResolvers/{dnsResolverName}/outboundEndpoints/{outboundEndpointName}?api-version=2025-05-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks?api-version=2025-05-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/dnsForwardingRulesets/{dnsForwardingRulesetName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2025-05-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-05-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- non-success responses are documented as `Other Status Codes | CloudError`
- asynchronous acceptance patterns appear across multiple create/update/delete methods

Observed response patterns:

- `200` appears in `54` methods
- `201` appears in `10` methods
- `202` appears in `24` methods
- `204` appears in `10` methods

Request-body coverage from reviewed pages:

- `21` methods include a request body
- request-body required fields vary by method; common required fields include `location` and key subresources such as virtual networks, subnets, outbound endpoints, target DNS servers, and domain list/action inputs

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add `docs/dns-dnsresolver-2025-05-01-plan.md` with operation-group/method inventory
- add `internal/server/provider_azure_dns_dnsresolver_2025_05_01.go` stage-0 router skeleton
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key is discoverable as `dns_dnsresolver_2025_05_01` in coverage tooling
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.Network/dnsResolvers`, `dnsForwardingRulesets`, `dnsResolverDomainLists`, and `dnsResolverPolicies` route families for this API version
- include list-by-virtual-network POST route families
- enforce `api-version=2025-05-01` for handled routes

### Stage 2: Resolver Core + Endpoint Surfaces

- implement `dns-resolvers` methods (7)
- implement `inbound-endpoints` methods (5)
- implement `outbound-endpoints` methods (5)
- support async `202` patterns on create/update/delete where documented

### Stage 3: Forwarding Ruleset Surfaces

- implement `dns-forwarding-rulesets` methods (7)
- implement `forwarding-rules` methods (5)
- implement `virtual-network-links` methods (5)
- support list-by-virtual-network selector operation

### Stage 4: Policy + Domain List Surfaces

- implement `dns-resolver-domain-lists` methods (7), including `bulk`
- implement `dns-resolver-policies` methods (7)
- implement `dns-resolver-policy-virtual-network-links` methods (5)
- implement `dns-security-rules` methods (5)
- support list-by-virtual-network selector operation

### Stage 5: Tests

- add provider tests (`internal/server/provider_azure_dns_dnsresolver_2025_05_01_test.go`) covering:
  - api-version validation
  - positive lifecycle for all 10 operation groups
  - negative paths (missing params, invalid references, unknown resources)
  - async-header assertions for representative `202` responses

### Stage 6: Example and Validation

- add `examples/azure/dns-dnsresolver-2025-05-01` using Azure Go SDK core pipeline
- include one call per reviewed method with expected status assertions
- verify build and coverage script discoverability

Exit criteria:

- example compiles and runs against emulated routes once handlers are implemented
- coverage scripts include `dns_dnsresolver_2025_05_01` and mapped plan doc

## Example Scope for `examples/azure/dns-dnsresolver-2025-05-01`

The example should exercise all reviewed methods (`58` total) with environment-configurable subscription, resource group, resolver, ruleset, endpoint, forwarding-rule, domain-list, policy, and virtual-network link identifiers.
