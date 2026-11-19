# Azure Marketplace Catalog (`marketplacecatalog-2025-05-01`) Staged Plan

## Objective

Emulate the Azure Marketplace Catalog Resource Manager API (`Microsoft.Marketplace`): the product catalog at tenant, subscription, billing account and billing profile scope, the tenant-level catalog search, and the subscription's Discovery API keys. The catalog is a fixed seed of five products; keys are stored.

## References

Microsoft Learn pages for the Resource Manager API. Products use `2025-05-01`, search uses `2022-02-02` and the key operations use `2023-01-01-preview`, the versions Learn documents for each:

- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/` (overview, OData query support)
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-list/list-by-tenant`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-list/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-list/list-by-billing-account`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-list/list-by-billing-profile`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-get/get-by-tenant`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-get/get-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-get/get-by-billing-account`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/product-get/get-by-billing-profile`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/search/post-arm`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/create-api-key/post`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/get-api-keys/list`
- `https://learn.microsoft.com/en-us/rest/api/marketplacecatalog/resourcemanager/revoke-api-key/post`

## Service Naming

- service key: `marketplacecatalog`
- display name: `Azure Marketplace Catalog`
- implementation: `internal/server/provider_azure_marketplacecatalog.go`, `internal/server/provider_azure_marketplacecatalog_validate.go`
- tests: `internal/server/provider_azure_marketplacecatalog_test.go`
- example: `examples/azure/marketplacecatalog-2025-05-01/`

## Operations In Scope

| Operation | Path |
|---|---|
| list and get products | `{scope}/providers/Microsoft.Marketplace/products[/{productId}]`, where scope is the tenant (empty), `/subscriptions/{id}`, `/providers/Microsoft.Billing/billingAccounts/{id}` or `.../billingAccounts/{id}/billingProfiles/{id}` |
| search | POST `/providers/Microsoft.Marketplace/search` |
| list keys | GET `/subscriptions/{id}/providers/Microsoft.Marketplace/keys` |
| create and revoke a key | POST `/subscriptions/{id}/providers/Microsoft.Marketplace/keys/{apiKeyAlias}/create` and `/revoke` |

## Behavior

The service has its own router (`handleAzureMarketplaceCatalogRouter`), since key aliases sit in the middle of action paths and search and the key operations read a request header. It claims only the paths above and leaves other `Microsoft.Marketplace` paths to later routers. Errors use the `{"error": {"code", "message"}}` payload with the codes `BadRequest`, `NotFound` and `Conflict`.

- lists return `ProductSummary` items with `PlanSummary` plans and a null `nextLink` (the seed fits one page). Hidden and stop-sold plans aren't listed
- `$filter` takes `eq` and `ne` comparisons of the fields Learn lists, joined by `and` and `or` (`and` binds tighter); string literals must be quoted, with `''` for a quote, and numbers compare as numbers. An array field matches when it holds the value, and `plan.*` fields match when any listed plan does. Other fields, functions and parentheses are rejected
- `$select` names `ProductSummary` properties; `$orderBy` takes one of the fields Learn lists, optionally with `asc` or `desc`, and rejects more than one field; `$search` matches the display name, publisher display name or summaries without regard to case
- `excludePublicOffersAndPublicPlans=true` keeps only private plans and the products that have one; `market` (tenant list) and `locations` (billing scope lists, repeated or comma-separated, matching any) keep the products sold there
- `language` must be one of Learn's codes (content isn't translated); boolean parameters must be `true` or `false`
- a product is found by `uniqueProductId` without regard to case. Get returns all its details; hidden and stop-sold plans need `includeHiddenPlans` and `includeStopSoldPlans`, service instruction templates need `includeServiceInstructionTemplates`, and `planId` and `skuId` narrow the plans
- the tenant get requires a two-letter `market` the product is sold in (404 otherwise) and takes `pricingAudience` `Direct-Commercial` (default) or `Partner-Commercial`
- search and the key operations require the `x-ms-client-tenant-id` header
- search requires `languages` (Learn's codes) and `stores` (`Appsource`, `AMP`, `Ibiza`, `Cosell`, `DakotaDownstream`); `pricingTypes`, `privateStoreAllowances`, `select`, `facets` and `sortBy` must use Learn's values and `SearchItemDto` fields, and `skip` and `take` must be non-negative integers. `searchQuery`, `publisherName`, `offerTypes`, `pricingTypes`, `operatingSystems`, the category and industry lists and the boolean flags filter the results. `totalCount` counts every match before `skip` and `take`; each requested facet counts the matches by value
- there is no private store, so every product is `AllowedByDefault`: a search whose `privateStoreAllowances` leaves that out finds nothing
- key aliases are alphanumeric characters separated by dashes and unique in the subscription (409 otherwise). Create returns the alias and a new key; list returns the subscription's active keys with their values, as Learn's sample shows; revoke (200) removes a key, and an unknown alias is 404

Out of scope for this stage:
- the data-plane API on `catalogapi.azure.com` (products, SKUs, search, facets, suggestions, reviews and comments)
- the operations list
- private stores and per-tenant private offers, translated content and paging
- checking that billing accounts and profiles exist

## Quality Gates

- `go test ./internal/server -run TestAzureMarketplaceCatalog -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service marketplacecatalog`
- `python3 scripts/azure-io-contract-coverage.py --service marketplacecatalog`
- `python3 scripts/azure-doc-contract-coverage.py --service marketplacecatalog`
