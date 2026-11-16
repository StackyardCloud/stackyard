# Azure Virtual Network (`network-2024-05-01`) Staged Plan

## Objective

Emulate the core Azure networking control plane (`Microsoft.Network`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): virtual networks with subnets and peerings, network security groups with security rules, application security groups, public IP addresses, network interfaces, load balancers, route tables with routes, and (stage 2, `internal/server/provider_azure_network_private.go`) NAT gateways, private endpoints with DNS zone groups, and private link services. No packets flow: addresses, rules and routes are bookkeeping only. Application Gateway, Front Door, DNS, Private DNS, DNS Resolver and Traffic Manager are covered by their own plans.

## References

Microsoft Learn pages for the `2024-05-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-network-peerings/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/security-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/route-tables/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/virtualnetwork/routes/create-or-update`

## Service Naming

- service key: `network`
- display name: `Azure Virtual Network`
- implementation: `internal/server/provider_azure_network.go`
- tests: `internal/server/provider_azure_network_test.go`
- example: `examples/azure/network-2024-05-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `virtualNetworks` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `virtualNetworks/subnets` | proxy child, inline as `subnets` | create/update, get, delete, list |
| `virtualNetworks/virtualNetworkPeerings` | proxy child, inline as `virtualNetworkPeerings` | create/update, get, delete, list |
| `networkSecurityGroups` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `networkSecurityGroups/securityRules` | proxy child, inline as `securityRules` | create/update, get, delete, list |
| `publicIPAddresses` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `networkInterfaces` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `loadBalancers` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `routeTables` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `routeTables/routes` | proxy child, inline as `routes` | create/update, get, delete, list |
| `applicationSecurityGroups` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `natGateways` | tracked | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `privateEndpoints` | tracked | create/update, get, delete, list by group and subscription |
| `privateEndpoints/privateDnsZoneGroups` | proxy child | create/update, get, delete, list |
| `privateLinkServices` | tracked | create/update, get, delete, list by group and subscription, `checkPrivateLinkServiceVisibility` (subscription and resource group location) |

## Behavior

CRUD, listing, cascade, inline-child sync and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Subnets, peerings, security rules and routes can be sent inline in their parent's PUT or created on their own, and the parent always lists the current ones. A parent PUT that sets the list replaces the children; one that omits it (or sends null) keeps them, and is rejected if they would become invalid, such as subnets outside a shrunk address space. Child names cannot contain `/`, `?` or `#`. Service-specific rules:

- virtual networks need at least one canonical CIDR prefix in `addressSpace.addressPrefixes`; `dhcpOptions.dnsServers` must be IP addresses; `resourceGuid` is server-owned
- subnets need canonical CIDR prefixes (`addressPrefix` or `addressPrefixes`) inside the network's address space and not overlapping another subnet; `networkSecurityGroup` and `routeTable` references must be IDs of those types, delegations need a `serviceName`, and `privateEndpointNetworkPolicies` must be a known value; it defaults to `Disabled` and `privateLinkServiceNetworkPolicies` to `Enabled`
- peerings need a virtual network `remoteVirtualNetwork.id` and cannot both use remote gateways and allow gateway transit; they report `peeringState` `Connected` and `peeringSyncLevel` `FullyInSync` at once, with Azure's access defaults
- security rules need a `protocol` (`Tcp`, `Udp`, `Icmp`, `Esp`, `Ah` or `*`), `access`, `direction`, an integer `priority` from 100 to 4096 unique per direction within the group, source and destination port ranges (`*`, a port or a range) and source and destination address prefixes or application security groups; groups report Azure's six `defaultSecurityRules`
- public IP addresses take `sku.name` `Basic` or `Standard` and `sku.tier` `Regional` or `Global`; Standard addresses must be `Static`, which is also their default (Basic defaults to `Dynamic`); `idleTimeoutInMinutes` is 4-30 (default 4) and a `dnsSettings.domainNameLabel` gives `fqdn` `{label}.{location}.cloudapp.azure.com`; static addresses get a stable `ipAddress` derived from the resource ID, and dynamic ones none
- network interfaces need at least one IP configuration, each with a name and a subnet ID, and exactly one primary when there are several; `Static` allocation needs a `privateIPAddress`, and public IP and security group references must be IDs of those types; configurations get IDs and allocation defaults (a single configuration is always primary), and the interface a stable `macAddress`
- load balancers take `sku.name` `Basic`, `Standard` or `Gateway` and each frontend needs exactly one of a public IP, a public IP prefix or a subnet; item names are unique per collection; probes need a protocol (`Tcp`, `Http` or `Https`), a port and, for HTTP(S), a `requestPath`; load-balancing and inbound NAT rules need a protocol, a backend port, a frontend port (or, for NAT rules on a backend pool, `frontendPortRangeStart` and `frontendPortRangeEnd`) and a frontend, and their backend pool and probe references must name items of the load balancer (only the item name is compared, not the load balancer in the ID); every item in the five main collections gets its ID
- routes need a CIDR prefix or service tag and a `nextHopType`; `VirtualAppliance` hops need a `nextHopIpAddress`, which other hop types must not set
- subnets accept a `natGateway` reference, which must be a NAT gateway ID; application security groups and NAT gateways get a `resourceGuid`
- NAT gateways take `sku.name` `Standard`, at most one zone, an integer `idleTimeoutInMinutes` from 4 to 120, and public IP address and prefix references of those types
- private endpoints need a subnet ID and at least one automatic or manual connection, each with a name and a `privateLinkServiceId` (a resource ID or a private link service alias); targets other than private link services need `groupIds`; a `customNetworkInterfaceName` is 2-80 letters, digits, `.`, `_` or `-`; static `ipConfigurations` need a name, `groupId`, `memberName` and an address in the subnet; application security group references must be of that type
- a private endpoint gets a network interface (`customNetworkInterfaceName`, or `{name}.nic.{guid}`) in its resource group, referencing it in `privateEndpoint`, with the lowest free IPv4 address in the subnet after Azure's first four (or the static address, which must be usable and not taken by another interface); the address is kept across PUTs, the interface is replaced if renamed and deleted with the endpoint, and a missing subnet or one without free addresses rejects the PUT; private endpoints and private link services have no PATCH (405)
- automatic connections are `Approved` at once and manual ones `Pending`; each connection gets its ID, and `customDnsConfigs` (unless sent) maps the target's private name, for common sub-resources such as `blob`, `vault` or `sqlServer`, to the endpoint's address, replacing any configs the client sends
- DNS zone groups need 1 to 5 `privateDnsZoneConfigs`, each with a name and a private DNS zone ID
- private link services need load balancer frontend references and 1 to 8 IP configurations with a subnet, exactly one primary when there are several; `visibility` subscriptions are IDs or `*` and `autoApproval` subscriptions are IDs; they get an `alias` `{name}.{guid}.{location}.azure.privatelinkservice`, which `checkPrivateLinkServiceVisibility` reports visible

Out of scope for this stage:
- PATCH changes tags and other top-level fields only; like other shared-handler types it skips validation, including the subnet and rule checks against the parent
- inbound NAT pools, outbound rules and multi-pool `backendAddressPools` arrays on rules are stored as sent, without IDs or reference checks
- name rules per resource type, the /29 minimum subnet size, overlapping prefixes within one address space, and self-peering
- back-references such as a subnet's `ipConfigurations` or a security group's `networkInterfaces`, dynamic private IP allocation, and refusing to delete resources in use
- cross-resource checks that referenced subnets, public IPs or remote networks exist, and peering state that depends on the remote network
- load balancer child resources (`backendAddressPools`, `inboundNatRules` at their own URLs) and NIC `effectiveRouteTable` and `effectiveNetworkSecurityGroups`
- private link service `privateEndpointConnections` (listing and approving endpoints), its network interfaces, `autoApprovedPrivateLinkServices` and `availablePrivateEndpointTypes`; DNS zone group `recordSets`; addresses of dynamic interfaces created directly, which private endpoint allocation does not see
- more than one static `ipConfigurations` entry: only the first gets an address on the interface
- checks that NAT gateway public IPs are Standard, that private link service frontends are internal, and that endpoint targets exist; refusing to delete a private endpoint's interface directly
- VPN and ExpressRoute gateways, Azure Firewall, Bastion, DDoS plans, Network Watcher and virtual WAN
- long-running operation polling: create, update and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureNetwork -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service network`
- `python3 scripts/azure-io-contract-coverage.py --service network`
- `python3 scripts/azure-doc-contract-coverage.py --service network`
