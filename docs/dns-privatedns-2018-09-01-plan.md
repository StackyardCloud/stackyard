# Azure Private DNS (`dns_privatedns_2018_09_01`) Staged Plan

## Objective

Emulate Azure Private DNS Resource Manager REST API (`api-version=2018-09-01`) with deterministic local behavior for:

- private DNS zone lifecycle (create/get/list/update/delete)
- private DNS record-set lifecycle and type-scoped listings
- virtual-network link lifecycle for private zones
- an Azure Go SDK example under `examples/azure/dns-privatedns-2018-09-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/operation-groups?view=rest-dns-privatedns-2018-09-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links?view=rest-dns-privatedns-2018-09-01`

Method pages reviewed directly: all linked methods from all operation groups (`17` total).

Private Zones method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones/create-or-update?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones/delete?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones/get?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones/list?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones/list-by-resource-group?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/private-zones/update?view=rest-dns-privatedns-2018-09-01`

Record Sets method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets/create-or-update?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets/delete?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets/get?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets/list?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets/list-by-type?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/record-sets/update?view=rest-dns-privatedns-2018-09-01`

Virtual Network Links method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links/create-or-update?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links/delete?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links/get?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links/list?view=rest-dns-privatedns-2018-09-01`
- `https://learn.microsoft.com/en-us/rest/api/dns/privatedns/virtual-network-links/update?view=rest-dns-privatedns-2018-09-01`

## Surface Summary

Reviewed operation groups and methods:

- Private Zones (6)
- Record Sets (6)
- Virtual Network Links (5)

Total reviewed methods: `17`.

## Service Naming

Canonical Stackyard naming:

- service key: `dns_privatedns_2018_09_01`
- display name: `Azure Private DNS`
- API version: `2018-09-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_dns_privatedns_2018_09_01.go`
- tests: `internal/server/provider_azure_dns_privatedns_2018_09_01_test.go`
- example: `examples/azure/dns-privatedns-2018-09-01`
- plan doc: `docs/dns-privatedns-2018-09-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}`

Representative routes:

- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Network/privateDnsZones?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones?api-version=2018-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}?api-version=2018-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/ALL?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}?api-version=2018-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/{recordType}/{relativeRecordSetName}?api-version=2018-09-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks?api-version=2018-09-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateDnsZones/{privateZoneName}/virtualNetworkLinks/{virtualNetworkLinkName}?api-version=2018-09-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2018-09-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- non-success responses are documented as `Other Status Codes | CloudError`
- private-zone and virtual-network-link create/update/delete paths include asynchronous acceptance patterns

Observed response patterns:

- `200` appears in `17` methods
- `201` appears in `3` methods
- `202` appears in `6` methods
- `204` appears in `3` methods
- private-zone and virtual-network-link create operations can return `202 Accepted` with async headers
- private-zone and virtual-network-link updates can return `200` or `202`
- delete operations include idempotent `204` outcomes

Required body fields from reviewed tables:

- none are marked as universally required in the table metadata, but sample payloads consistently include `location` on create-or-update methods

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add `docs/dns-privatedns-2018-09-01-plan.md` with operation-group/method inventory
- add `internal/server/provider_azure_dns_privatedns_2018_09_01.go` stage-0 router skeleton
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key is discoverable as `dns_privatedns_2018_09_01` in coverage tooling
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.Network/privateDnsZones` routes for this api-version
- enforce `api-version=2018-09-01` for handled routes
- preserve existing public DNS and non-DNS route behavior

### Stage 2: Private Zones Core Surface

Implement methods:

- `private-zones/create-or-update`
- `private-zones/get`
- `private-zones/list`
- `private-zones/list-by-resource-group`
- `private-zones/update`
- `private-zones/delete`

Behavior goals:

- deterministic zone IDs/types/etag and provisioning envelopes
- optional async acceptance headers for create/update/delete flows returning `202`
- idempotent delete semantics across `200/202/204` outcomes

### Stage 3: Record Sets Surface

Implement methods:

- `record-sets/create-or-update`
- `record-sets/get`
- `record-sets/list`
- `record-sets/list-by-type`
- `record-sets/update`
- `record-sets/delete`

Behavior goals:

- canonicalized record-type handling (`A`, `AAAA`, `CNAME`, `MX`, `PTR`, `SOA`, `SRV`, `TXT`)
- deterministic record-set response shape (`fqdn`, `etag`, `isAutoRegistered`, TTL/records arrays)
- consistent parent-zone existence validation

### Stage 4: Virtual Network Links Surface

Implement methods:

- `virtual-network-links/create-or-update`
- `virtual-network-links/get`
- `virtual-network-links/list`
- `virtual-network-links/update`
- `virtual-network-links/delete`

Behavior goals:

- persist `properties.virtualNetwork.id` and `properties.registrationEnabled`
- deterministic link state/provisioning state placeholders
- support both sync `200` and async `202` update/delete/create responses

### Stage 5: Tests

- add provider tests (`internal/server/provider_azure_dns_privatedns_2018_09_01_test.go`) covering:
  - api-version validation
  - positive lifecycle for zones, record sets, and virtual network links
  - negative paths (missing params, unsupported record type, unknown resources)
  - async-header assertions for representative `202` responses

### Stage 6: Example and Validation

- add `examples/azure/dns-privatedns-2018-09-01` using Azure Go SDK core pipeline
- include one call per reviewed method with expected status assertions
- verify build and coverage script discoverability

Exit criteria:

- example compiles and runs against emulated routes once handlers are implemented
- coverage scripts include `dns_privatedns_2018_09_01` and mapped plan doc

## Example Scope for `examples/azure/dns-privatedns-2018-09-01`

The example should exercise all reviewed methods (`17` total) with environment-configurable subscription, resource group, private zone, record type, record set name, and virtual network link name values.
