# Azure Traffic Manager (`trafficmanager-2022-04-01`) Staged Plan

## Objective

Emulate Azure Traffic Manager profiles and endpoints (`Microsoft.Network/trafficManagerProfiles`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Other `Microsoft.Network` types keep their existing handlers: the shared router only claims registered type paths.

## References

Microsoft Learn pages for the `2022-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/trafficmanager/profiles`
- `https://learn.microsoft.com/en-us/rest/api/trafficmanager/profiles/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/trafficmanager/profiles/check-traffic-manager-name-availability-v2`
- `https://learn.microsoft.com/en-us/rest/api/trafficmanager/endpoints`

## Service Naming

- service key: `trafficmanager`
- display name: `Azure Traffic Manager`
- implementation: `internal/server/provider_azure_trafficmanager.go`
- tests: `internal/server/provider_azure_trafficmanager_test.go`
- example: `examples/azure/trafficmanager-2022-04-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `trafficManagerProfiles` | tracked (global) | create/update, get, update (PATCH), delete, list by group and subscription |
| `trafficManagerProfiles/externalEndpoints` | proxy child | create/update, get, update, delete, list |
| `trafficManagerProfiles/azureEndpoints` | proxy child | create/update, get, update, delete, list |
| `trafficManagerProfiles/nestedEndpoints` | proxy child | create/update, get, update, delete, list |
| `checkTrafficManagerNameAvailabilityV2` (subscription) and `checkTrafficManagerNameAvailability` (tenant) | action | available / `AlreadyExists` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- profiles must use location `global`, a `trafficRoutingMethod` of `Performance`, `Priority`, `Weighted`, `Geographic`, `MultiValue` or `Subnet`, and a `dnsConfig.relativeName` (400 `InvalidRequestContent`)
- `dnsConfig.fqdn` is server-owned: `{lower(relativeName)}.trafficmanager.net`, recomputed when PATCH changes the relative name
- external endpoints require `properties.target`; Azure and nested endpoints require `properties.targetResourceId`
- name checks compare relative DNS names across all profiles, case-insensitively
- the request path segment is matched case-insensitively (SDKs send `trafficmanagerprofiles`); IDs and types use `trafficManagerProfiles`

Out of scope for this stage:
- DNS resolution of profile FQDNs and endpoint health monitoring (`monitorStatus` is not reported)
- the profile's inline `properties.endpoints` array (endpoints are read through their own routes)
- rejecting a PUT whose relative name another profile already uses
- heat maps, real user metrics keys and user metrics

## Quality Gates

- `go test ./internal/server -run TestAzureTrafficManager -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service trafficmanager`
- `python3 scripts/azure-io-contract-coverage.py --service trafficmanager`
- `python3 scripts/azure-doc-contract-coverage.py --service trafficmanager`
