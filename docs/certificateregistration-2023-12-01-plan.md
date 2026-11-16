# App Service Certificates (`certificateregistration-2023-12-01`) Staged Plan

## Objective

Emulate the App Service certificates control plane (`Microsoft.CertificateRegistration`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): certificate orders and the Key Vault bindings of their certificates. No certificate is purchased or issued by a certificate authority.

## References

Microsoft Learn pages for the `2023-12-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-certificate-orders/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-certificate-orders/verify-domain-ownership`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-certificate-orders/retrieve-site-seal`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-certificate-orders/create-or-update-certificate`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-certificate-orders/validate-purchase-information`

## Service Naming

- service key: `certificateregistration`
- display name: `App Service Certificates`
- implementation: `internal/server/provider_azure_certificateregistration.go`
- tests: `internal/server/provider_azure_certificateregistration_test.go`
- example: `examples/azure/certificateregistration-2023-12-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `certificateOrders` | tracked (global) | create/update, get, update (PATCH), delete, list by group and subscription, `verifyDomainOwnership`, `renew`, `reissue`, `resendEmail`, `resendRequestEmails`, `retrieveSiteSeal`, `retrieveCertificateActions`, `retrieveEmailHistory` |
| `certificateOrders/certificates` | tracked child (global) | create/update, get, update (PATCH), delete, list |
| `validateCertificateRegistrationInformation` | subscription action | 204 or 400 |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- orders and certificates need location `global`
- orders need `productType` `StandardDomainValidatedSsl` or `StandardDomainValidatedWildCardSsl`; `distinguishedName`, when set, must start with `CN=`, `validityInYears` must be 1, 2 or 3 and `keySize` 2048 or 4096; they default to 1 year, 2048 bits and `autoRenew` `true`
- `csr` is accepted but never returned
- `status` is `Pendingissuance` for a new order and only `verifyDomainOwnership` changes it, to `Issued`; PUT and PATCH never change it
- `domainVerificationToken`, `serialNumber`, `expirationTime` (`validityInYears` after creation), `lastCertificateIssuanceTime`, `nextAutoRenewalTimeStamp` and `appServiceCertificateNotRenewableReasons` are server-owned
- `renew`, `reissue`, `resendEmail` and `resendRequestEmails` return 204 and do nothing; `retrieveSiteSeal` returns placeholder HTML, and `retrieveCertificateActions` and `retrieveEmailHistory` return empty lists
- certificates need a `keyVaultId` naming a key vault and a `keyVaultSecretName`; the vault is not required to exist
- `validateCertificateRegistrationInformation` applies the order rules to its body

Out of scope for this stage:
- purchasing, issuing and storing certificates: no CA, certificate content or Key Vault secret
- domain verification by email, website or DNS
- the order's `signedCertificate`, `root` and `intermediate` details and the certificate status of each binding
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureCertificateOrder -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service certificateregistration`
- `python3 scripts/azure-io-contract-coverage.py --service certificateregistration`
- `python3 scripts/azure-doc-contract-coverage.py --service certificateregistration`
