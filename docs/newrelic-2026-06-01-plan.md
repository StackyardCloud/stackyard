# Azure Native New Relic Service (`newrelic-2026-06-01`) Staged Plan

## Objective

Emulate Azure Native New Relic Service management (`NewRelic.Observability`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): monitors with their tag rules and monitored subscriptions, the monitor actions, and the subscription-level account, organization and plan lists. Nothing is sent to New Relic: a monitor's tag rules decide which stored resources report as sending metrics and logs, and no agents are ever installed.

## References

Microsoft Learn pages for the `2026-06-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/get`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/update`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/delete`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/get-metric-rules`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/get-metric-status`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/list-monitored-resources`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/list-hosts`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/list-app-services`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/vm-host-payload`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/refresh-ingestion-key`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/switch-billing`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/resubscribe`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/link-saa-s`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/latest-linked-saa-s`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitors/list-linked-resources`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/billing-info/get`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/connected-partner-resources/list`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/tag-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/tag-rules/get`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/tag-rules/update`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/tag-rules/delete`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/tag-rules/list-by-new-relic-monitor-resource`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitored-subscriptions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitored-subscriptions/get`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitored-subscriptions/update`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitored-subscriptions/delete`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/monitored-subscriptions/list`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/accounts/list`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/organizations/list`
- `https://learn.microsoft.com/en-us/rest/api/newrelic/plans/list`
- `https://learn.microsoft.com/en-us/azure/partner-solutions/new-relic/manage` (monitored subscriptions share the monitor's tag rules)
- `https://learn.microsoft.com/en-us/azure/partner-solutions/new-relic/faq` (the organization can't change; organizations created in New Relic are billed by New Relic)

## Service Naming

- service key: `newrelic`
- display name: `Azure Native New Relic Service`
- implementation: `internal/server/provider_azure_newrelic.go`, `internal/server/provider_azure_newrelic_validate.go`
- tests: `internal/server/provider_azure_newrelic_test.go`
- example: `examples/azure/newrelic-2026-06-01/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `NewRelic.Observability/monitors` | create or update, get, update (PATCH), delete, list by resource group and subscription; POST `getMetricRules`, `getMetricStatus`, `monitoredResources`, `listHosts`, `listAppServices`, `vmHostPayloads`, `refreshIngestionKey`, `switchBilling`, `resubscribe`, `linkSaaS`, `latestLinkedSaaS`, `listLinkedResources`, `getBillingInfo`, `listConnectedPartnerResources` |
| `monitors/{name}/tagRules` | create or update, get, update (PATCH), delete, list |
| `monitors/{name}/monitoredSubscriptions` | create or update, get, update (PATCH), delete, list (`default` only) |
| `/subscriptions/{id}/providers/NewRelic.Observability/accounts`, `organizations`, `plans` | GET |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- monitors check the reference's enums (`accountCreationSource` and `orgCreationSource` `LIFTR` or `NEWRELIC`, `usageType` `PAYG` or `COMMITTED`, `singleSignOnState`, identity `type`), `effectiveDate` as RFC 3339, the user's email against the reference's pattern, first and last names of at most 50 characters and phone numbers of at most 40, and a `saaSResourceId` that is a `Microsoft.SaaS/resources` ID
- `provisioningState`, `monitoringStatus` (`Enabled`), `marketplaceSubscriptionStatus` (`Active`), `liftrResourceCategory` (`MonitorLogs`) and `marketplaceSubscriptionId` (a GUID derived from the monitor ID) are server-owned
- linking an existing account (`accountCreationSource` `NEWRELIC`) needs its `accountId`, and linking an existing organization (`orgCreationSource` `NEWRELIC`) its `organizationId`; otherwise the monitor gets a new seven-digit account ID and an organization GUID, and the account region defaults to the monitor's location
- the organization can't change after creation; a PUT or PATCH that omits it keeps it
- PATCH takes the update model's fields only (`identity`, `tags`, and `accountCreationSource`, `newRelicAccountProperties`, `orgCreationSource`, `planData`, `saaSData`, `userInfo`)
- the account's ingestion key is never returned on GET; `vmHostPayloads` returns it. A PUT may set it; otherwise a monitor keeps its key, or takes the key of another monitor of the same account, or gets a new one. `refreshIngestionKey` (204) gives every monitor of the account a new key
- tag rules check `send*` flags (`Enabled` or `Disabled`), filtering tags (`name`, `value`, `action` `Include` or `Exclude`) and the metric rules' email; PATCH takes `logRules` and `metricRules` only
- a resource is captured by a tag rule's filtering tags unless an `Exclude` tag matches, and, when there are `Include` tags, only if one matches (no filtering tags capture everything); tag names compare case-insensitively and a tag without a value matches any value
- `monitoredResources` lists the stored top-level resources in the monitor's subscription and its monitored subscriptions that some tag rule captures for metrics (`sendMetrics` `Enabled`) or logs (`sendActivityLogs` `Enabled`), with `sendingMetrics` and `sendingLogs`
- `getMetricStatus` returns which of the named resources (all captured ones when none are named) send metrics; `getMetricRules` combines the tag rules' metric rules
- `getMetricRules`, `getMetricStatus`, `listHosts`, `listAppServices` and `switchBilling` require `userEmail`
- monitored subscriptions are named `default`; each list item names a subscription (a GUID, bare or as `/subscriptions/{id}`) and is `Active`. `patchOperation` applies to the stored list: `AddBegin` and `AddComplete` add the sent subscriptions, `DeleteBegin` and `DeleteComplete` remove them, and `Active` (or none) replaces the list; the add and delete operations need a non-empty list. Duplicates are dropped
- `linkSaaS` sets the monitor's SaaS resource and `latestLinkedSaaS` returns it (`isHiddenSaaS` false); `getBillingInfo` reports the Marketplace subscription ID, SaaS resource and billed subscription, and the organization
- `switchBilling` and `resubscribe` reject an `organizationId` other than the monitor's; `switchBilling` also rejects an `azureResourceId` other than the monitor's, takes new `planData` (200 with the monitor) and answers 204 without it; `resubscribe` takes `planId` as the plan's `planDetails`
- `listLinkedResources` lists the monitors of the same organization in any subscription; `listConnectedPartnerResources` lists the monitors billed through the same SaaS resource (just the monitor when it has none). Its optional body is a bare email string, as the reference defines it; no other action takes a string body
- `accounts` and `organizations` list those of monitors whose user email and location match the required `userEmail` and `location` query parameters; organizations created in New Relic (`orgCreationSource` `NEWRELIC`) have `billingSource` `NEWRELIC`, others `AZURE`. `plans` lists the plan data of the subscription's monitors, optionally only those of an `accountId` or `organizationId`
- deleting a monitor deletes its tag rules, monitored subscriptions and ingestion key

Out of scope for this stage:
- sending metrics or logs, installing agents: `listHosts` and `listAppServices` return no VMs or app services
- per-subscription `tagRules` on monitored subscription items: Learn says the monitor's tag rules apply to all of them, so they aren't repeated
- `accountName` and `organizationName`, which only New Relic knows, and monitor `Suspended` states
- the required `userEmail` query parameter of monitor DELETE, and 400 responses for `accounts` and `organizations` without their required query parameters (they list nothing): the shared handler doesn't read query parameters there
- resources the shared ARM handler doesn't store don't appear in `monitoredResources`
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzureNewRelic -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service newrelic`
- `python3 scripts/azure-io-contract-coverage.py --service newrelic`
- `python3 scripts/azure-doc-contract-coverage.py --service newrelic`
