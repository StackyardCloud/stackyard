# Azure Internet Peering and Peering Service (`peering-2025-05-01`) Staged Plan

## Objective

Emulate the `Microsoft.Peering` control plane on the shared ARM handler (`internal/server/azure_resource_manager.go`): peer ASNs, Direct and Exchange peerings with registered ASNs and prefixes, and Peering Service resources with prefixes and connection monitor tests. Microsoft approves ASNs and connections by hand; the emulator approves them at once, and no BGP session runs.

## References

Microsoft Learn pages for the `2025-05-01` API version and the peering walkthroughs:

- `https://learn.microsoft.com/en-us/rest/api/peering/peer-asns/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/peering/peerings/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/peering/registered-prefixes/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/peering/peering-services/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/peering/prefixes/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/peering/connection-monitor-tests/create-or-update`
- `https://learn.microsoft.com/en-us/previous-versions/azure/internet-peering/howto-direct-powershell`
- `https://learn.microsoft.com/en-us/previous-versions/azure/internet-peering/howto-exchange-powershell`

## Service Naming

- service key: `peering`
- display name: `Azure Peering`
- implementation: `internal/server/provider_azure_peering.go`
- tests: `internal/server/provider_azure_peering_test.go`
- example: `examples/azure/peering-2025-05-01/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `peerAsns` (subscription) | create/update, get, delete, list |
| `peerings` | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `peerings/registeredAsns`, `peerings/registeredPrefixes` | create/update, get, delete, list; POST `validate` on a prefix |
| `peeringServices` | create/update, get, update tags (PATCH), delete, list by group and subscription |
| `peeringServices/prefixes`, `peeringServices/connectionMonitorTests` | create/update, get, delete, list |
| `peeringLocations`, `peeringServiceLocations`, `peeringServiceProviders`, `peeringServiceCountries` | list (sample catalog) |
| `legacyPeerings`, `cdnPeeringPrefixes`, `peerings/receivedRoutes`, `peerings/rpUnbilledPrefixes` | list (always empty) |
| `checkServiceProviderAvailability`, `initializeConnectionMonitor` | POST |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- peer ASNs need an ASN from 1 to 4294967295, a peer name and contacts with a known role and an email; `validationState` is `Approved` at once
- peerings need a kind (`Direct` or `Exchange`), a sku of that kind (such as `Basic_Direct_Free`; `tier`, `family` and `size` are derived from it), the peering location, at least one connection, and the ID of a peer ASN that exists in the subscription
- Direct connections need a facility at the location, bandwidth in 10 Gbps steps and, when the peer provides session addresses, a `/31` IPv4 or `/127` IPv6 session prefix; Microsoft takes the first address and the peer the second
- Exchange connections need an exchange at the location and peer session addresses inside the exchange's prefixes; sessions use the exchange's Microsoft addresses
- connections are `Active` with sessions `Established`, and `provisionedBandwidthInMbps` equals the requested bandwidth
- registered ASNs and prefixes get a `peeringServicePrefixKey`; a Peering Service prefix that quotes a key and prefix a provider registered is `Verified` (`ViaServiceProvider`), and any other is `Failed` with "Prefix is not announced by the service provider to Microsoft."
- Peering Services need a known location and provider that serves it; the provider's primary and backup peering locations must be ones it peers at and must differ
- the sample catalog has Seattle (Direct facility 71 with 10 and 100 Gbps offers; the Equinix Seattle exchange, ID 11), Peering Service locations Washington and Ontario, and one provider, Contoso Networks, which serves Washington
- PATCH on peerings and Peering Services changes tags only

Out of scope for this stage:
- filtering the catalog by `kind`, `directPeeringType` or `country`: views can't read query parameters, so they list everything
- re-validating Peering Service prefixes: a prefix is checked when it is written, so one written before the provider registers it stays `Failed` until it is written again
- looking glass commands, received routes and connection monitor results
- approval workflows (`Pending` states), legacy peering conversion, and blocking deletion of an ASN that peerings use
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzurePeering -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service peering`
- `python3 scripts/azure-io-contract-coverage.py --service peering`
- `python3 scripts/azure-doc-contract-coverage.py --service peering`
