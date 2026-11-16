# Microsoft Sentinel (`securityinsights-2025-09-01`) Staged Plan

## Objective

Emulate the Microsoft Sentinel control plane (`Microsoft.SecurityInsights`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): onboarding, analytics rules and their actions, automation rules, data connectors, incidents with comments, relations and tasks, bookmarks, watchlists and their items, and content metadata. Sentinel resources are extension resources of a Log Analytics workspace (`internal/server/provider_azure_operationalinsights.go`). The emulator stores configuration only: rules never run and connectors ingest nothing.

## References

Microsoft Learn pages for the `2025-09-01` API version and the service limits it enforces:

- `https://learn.microsoft.com/en-us/rest/api/securityinsights/sentinel-onboarding-states/create`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/alert-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/actions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/alert-rule-templates/list`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/automation-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/data-connectors/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/incidents/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/incident-comments/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/incident-relations/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/incident-tasks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/bookmarks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/watchlists/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/watchlist-items/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/securityinsights/metadata/create`
- `https://learn.microsoft.com/en-us/azure/sentinel/create-analytics-rules`
- `https://learn.microsoft.com/en-us/azure/sentinel/watchlists`
- `https://learn.microsoft.com/en-us/azure/sentinel/create-manage-use-automation-rules`

## Service Naming

- service key: `securityinsights`
- display name: `Microsoft Sentinel`
- implementation: `internal/server/provider_azure_securityinsights.go`, `internal/server/provider_azure_securityinsights_incidents.go`
- tests: `internal/server/provider_azure_securityinsights_test.go`
- example: `examples/azure/securityinsights-2025-09-01/`

## Operations In Scope

All paths are under `{workspaceId}/providers/Microsoft.SecurityInsights`.

| Resource | Operations |
|---|---|
| `onboardingStates` | create/update, get, delete, list (the only name is `default`) |
| `alertRules` | create/update, get, delete, list (kinds `Scheduled`, `MicrosoftSecurityIncidentCreation`, `Fusion`) |
| `alertRules/{rule}/actions` | create/update, get, delete, list |
| `alertRuleTemplates` | list, get (the built-in Fusion template) |
| `automationRules` | create/update, get, delete, list |
| `dataConnectors` | create/update, get, delete, list |
| `incidents` | create/update, get, delete, list; POST `alerts`, `bookmarks`, `entities` |
| `incidents/{incident}/comments`, `relations`, `tasks` | create/update, get, delete, list |
| `bookmarks` | create/update, get, delete, list |
| `watchlists` | create/update, get, delete (204), list |
| `watchlists/{alias}/watchlistItems` | create/update, get, delete, list |
| `metadata` | create/update, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Sentinel types are extension types whose scope must be a stored Log Analytics workspace: other scopes get 400 `InvalidScope`, and a missing workspace gets 404 `ParentResourceNotFound`. Deleting the workspace deletes its Sentinel resources. Only metadata has PATCH; other types return 405. Service-specific rules:

- rules, automation rules, connectors, incidents, bookmarks and watchlists need the workspace's onboarding state `default` first (400 otherwise)
- scheduled rules need a display name, query, severity, trigger operator and a whole-number threshold of at least 0; they run every 5 minutes to 14 days over a lookback of 5 minutes to 14 days that is at least the interval, suppress for at most 24 hours, and map at most 10 entities of a recognized type; tactics, event grouping and incident grouping values must be known
- Fusion rules must use the Fusion template `f71aba3d-28fb-450b-b192-4e76a83015c8`, and take their display name, description, severity and tactics from it; incident creation rules need a known product filter
- rule actions need a Logic App workflow ID and an https trigger URI, which is write-only; the response's `workflowId` comes from the URI
- automation rules have an order from 1 to 1000, at most 50 conditions and 1 to 20 actions; rules triggered by alerts can only run playbooks
- data connectors check the required tenant ID and data types of their kind, and data type states are `Enabled` or `Disabled`
- incidents are numbered per workspace from 1, and keep their number when updated; closing one needs a classification, and the classification reason must belong to it (true positive: suspicious activity; benign positive: suspicious but expected; false positive: incorrect alert logic or inaccurate data)
- relations must name a bookmark in the incident's workspace; an incident's `bookmarks` action lists them, and `alerts` and `entities` are always empty
- relation names are 3-63 letters, digits or hyphens
- incident numbers are never reused, even after a delete
- watchlist aliases and display names are 3-64 characters, alphanumeric at both ends, with spaces, hyphens and underscores between; `rawContent` must be `text/csv` of at most 3.8 MB whose header (after `numberOfLinesToSkip` lines) includes `itemsSearchKey`, and a new watchlist gets one item per row; the emulator also caps one upload at 100,000 rows, since each becomes a stored item

Out of scope for this stage:
- running rules, raising alerts, firing automation rules or running playbooks (including the `runPlaybook` actions)
- the incident counts in `additionalData`, which stay 0
- items from `rawContent` on watchlist updates; only a new watchlist gets them
- `etag` optimistic concurrency
- workspace checks on the alert rule template list, which answers under any scope
- threat intelligence, content packages and templates, source control, ML analytics settings, data connector definitions and repositories
- percent-encoded names: the raw path is used, so watchlist aliases with spaces can't be addressed and metadata names with whitespace aren't rejected
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSentinel -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service securityinsights`
- `python3 scripts/azure-io-contract-coverage.py --service securityinsights`
- `python3 scripts/azure-doc-contract-coverage.py --service securityinsights`
