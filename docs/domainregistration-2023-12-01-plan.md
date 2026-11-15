# App Service Domains (`domainregistration-2023-12-01`) Staged Plan

## Objective

Emulate the App Service domains control plane (`Microsoft.DomainRegistration`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): domains, their ownership identifiers, availability checks and top-level domains. No domain is registered with a registrar, and the reported Azure DNS name servers do not answer for it.

## References

Microsoft Learn pages for the `2023-12-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/appservice/domains/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/appservice/domains/renew`
- `https://learn.microsoft.com/en-us/rest/api/appservice/domains/check-availability`
- `https://learn.microsoft.com/en-us/rest/api/appservice/domains/create-or-update-ownership-identifier`
- `https://learn.microsoft.com/en-us/rest/api/appservice/top-level-domains/list`
- `https://learn.microsoft.com/en-us/rest/api/appservice/top-level-domains/list-agreements`

## Service Naming

- service key: `domainregistration`
- display name: `App Service Domains`
- implementation: `internal/server/provider_azure_domainregistration.go`
- tests: `internal/server/provider_azure_domainregistration_test.go`
- example: `examples/azure/domainregistration-2023-12-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `domains` | tracked (global) | create/update, get, update (PATCH), delete, list by group and subscription, `renew` |
| `domains/domainOwnershipIdentifiers` | proxy child | create/update, get, update (PATCH), delete, list |
| `checkDomainAvailability` | subscription action | `available` |
| `listDomainRecommendations` | subscription action | empty list |
| `topLevelDomains` | subscription view | list, get |
| `topLevelDomains/{name}/listAgreements` | subscription action | registration and privacy agreements |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- domain names are a label of letters, digits or hyphens under a supported top-level domain: `com`, `net`, `org`, `info`, `biz`, `co.uk`, `org.uk`, `nl` or `in` (400 `InvalidResourceName`)
- domains need location `global`; the admin, billing, registrant and technical contacts each need `email`, `nameFirst`, `nameLast` and `phone`; `consent` needs `agreementKeys` and `agreedBy`, and `agreedAt`, when set, must be an RFC 3339 time; `dnsType` must be `AzureDns` or `DefaultDomainRegistrarDns`
- the four contacts and `authCode` are accepted but never returned, so a client that PUTs back a domain it read must send the contacts again
- `registrationStatus` (`Active`), `nameServers`, `createdTime`, `lastRenewedTime`, `expirationTime` (one year after creation), `readyForDnsRecordManagement`, `managedHostNames` and `domainNotRenewableReasons` are server-owned; `autoRenew`, `privacy` and `dnsType` default to `true`, `false` and `AzureDns`
- `renew` returns 204 and does nothing; ownership identifiers need an `ownershipId`
- `checkDomainAvailability` reports a domain unavailable when its name is invalid or any subscription holds it
- `topLevelDomains` lists the supported top-level domains, and `listAgreements` returns the same agreements for each

Out of scope for this stage:
- registering, renewing and transferring domains: no registrar, WHOIS or DNS zone
- `transferOut`, single sign-on requests and recommendations: `listDomainRecommendations` returns no domains
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureDomain -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service domainregistration`
- `python3 scripts/azure-io-contract-coverage.py --service domainregistration`
- `python3 scripts/azure-doc-contract-coverage.py --service domainregistration`
