# Azure Marketplace Ordering (`marketplaceordering-2021-01-01`) Staged Plan

## Objective

Emulate Azure Marketplace Ordering (`Microsoft.MarketplaceOrdering`): reading and accepting the terms of marketplace virtual machine images, and signing, cancelling and listing the subscription's agreements to them, which tools such as Terraform's `azurerm_marketplace_agreement` need before deploying a marketplace image.

## References

Microsoft Learn pages for the `2021-01-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/marketplaceordering/marketplace-agreements/get`
- `https://learn.microsoft.com/en-us/rest/api/marketplaceordering/marketplace-agreements/create`
- `https://learn.microsoft.com/en-us/rest/api/marketplaceordering/marketplace-agreements/sign`
- `https://learn.microsoft.com/en-us/rest/api/marketplaceordering/marketplace-agreements/cancel`
- `https://learn.microsoft.com/en-us/rest/api/marketplaceordering/marketplace-agreements/get-agreement`
- `https://learn.microsoft.com/en-us/rest/api/marketplaceordering/marketplace-agreements/list`

## Service Naming

- service key: `marketplaceordering`
- display name: `Azure Marketplace Ordering`
- implementation: `internal/server/provider_azure_marketplaceordering.go`
- tests: `internal/server/provider_azure_marketplaceordering_test.go`
- example: `examples/azure/marketplaceordering-2021-01-01/`

## Operations In Scope

All paths are under `/subscriptions/{id}/providers/Microsoft.MarketplaceOrdering`.

| Resource | Operations |
|---|---|
| `offerTypes/virtualmachine/publishers/{publisher}/offers/{offer}/plans/{plan}/agreements/current` | get, create (PUT) |
| `agreements/{publisher}/offers/{offer}/plans/{plan}` | get; POST `sign`, `cancel` |
| `agreements` | list |

## Behavior

The service has its own router, since terms exist without being created; agreements are stored as records in the shared ARM store, under their subscription.

- terms exist for every publisher, offer and plan, unaccepted (with an empty `retrieveDatetime`) until accepted; the offer type must be `virtualmachine`
- PUT takes `properties.accepted` (true or false): accepting signs the agreement and declining cancels it; declining terms never accepted stores nothing
- agreements return their ID, publisher, offer and state; `sign` makes the agreement `Active` with a `signDate` and `cancel` makes it `Canceled` with a `cancelDate`; repeating either keeps the first date, and an agreement that was never signed can't be read or cancelled (404)
- the terms signature is stable per plan; the license, privacy and marketplace terms links are placeholders on a `.invalid` host

Out of scope for this stage:
- checking that a publisher, offer or plan exists, or that a PUT echoes the terms signature
- other offer types

## Quality Gates

- `go test ./internal/server -run TestAzureMarketplaceOrdering -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service marketplaceordering`
- `python3 scripts/azure-io-contract-coverage.py --service marketplaceordering`
- `python3 scripts/azure-doc-contract-coverage.py --service marketplaceordering`
