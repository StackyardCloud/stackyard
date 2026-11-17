# Azure Partner Admin Link (`managementpartner-2018-02-01`) Staged Plan

## Objective

Emulate Partner Admin Link (`Microsoft.ManagementPartner`): linking a Microsoft partner ID to the signed-in account. Each account links at most one partner ID in a tenant, so the API is a singleton with its own router (`handleAzureManagementPartnerRouter`) rather than shared ARM handler resources.

## References

The `2018-02-01` API has no REST reference pages on Microsoft Learn; these cover its behavior:

- `https://learn.microsoft.com/en-us/azure/cost-management-billing/manage/link-partner-id`
- `https://learn.microsoft.com/en-us/cli/azure/managementpartner`
- `https://learn.microsoft.com/en-us/powershell/module/az.managementpartner/new-azmanagementpartner`
- `https://learn.microsoft.com/en-us/powershell/module/az.managementpartner/update-azmanagementpartner`

## Service Naming

- service key: `managementpartner`
- display name: `Azure Partner Admin Link`
- implementation: `internal/server/provider_azure_managementpartner.go`
- tests: `internal/server/provider_azure_managementpartner_test.go`
- example: `examples/azure/managementpartner-2018-02-01/`

## Operations In Scope

| Operation | Path | Behavior |
|---|---|---|
| link | `PUT /providers/Microsoft.ManagementPartner/partners/{partnerId}` | 200; the same ID again is a no-op, another ID while one is linked is 409 `Conflict` |
| get | `GET /providers/Microsoft.ManagementPartner/partners/{partnerId}` | 200 when that ID is linked, else 404 |
| get the link | `GET /providers/Microsoft.ManagementPartner/partners` | the one link (not a list), or 404 |
| change | `PATCH /providers/Microsoft.ManagementPartner/partners/{partnerId}` | moves the link to this ID and bumps `version` and `etag`; 404 with no link |
| remove | `DELETE /providers/Microsoft.ManagementPartner/partners/{partnerId}` | 200 when that ID is linked, else 404 |

## Behavior

- partner IDs are numeric (1-10 digits); anything else is 400 `BadRequest`
- the emulator has one caller, so it keeps one link; `tenantId` is the emulator tenant and `objectId` a fixed caller ID
- `partnerName` is `Partner {partnerId}`, since the emulator has no partner directory

Out of scope for this stage:
- checking that a partner ID exists in the Microsoft Cloud Partner Program
- separate links per user, service principal or tenant
- the `operations` list

## Quality Gates

- `go test ./internal/server -run TestAzureManagementPartner -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service managementpartner`
- `python3 scripts/azure-io-contract-coverage.py --service managementpartner`
- `python3 scripts/azure-doc-contract-coverage.py --service managementpartner`
