# Azure Support (`support-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Support control plane (`Microsoft.Support`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): support tickets with communications and chat transcripts, file workspaces with files, and the services and problem classifications tickets are filed against, at subscription and tenant scope. No engineer ever answers.

## References

Microsoft Learn pages for the `2024-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/support/support-tickets/create`
- `https://learn.microsoft.com/en-us/rest/api/support/communications/create`
- `https://learn.microsoft.com/en-us/rest/api/support/files/create`
- `https://learn.microsoft.com/en-us/rest/api/support/problem-classifications/list`

## Service Naming

- service key: `support`
- display name: `Azure Support`
- implementation: `internal/server/provider_azure_support.go`
- tests: `internal/server/provider_azure_support_test.go`
- example: `examples/azure/support-2024-04-01/`

## Operations In Scope

Tickets and file workspaces live at `/subscriptions/{id}/providers/Microsoft.Support/...` and at tenant scope `/providers/Microsoft.Support/...`.

| Resource | Kind | Operations |
|---|---|---|
| `supportTickets` | subscription and tenant | create, get, update (PATCH), list, `checkNameAvailability` (communications) |
| `supportTickets/communications` | child | create, get, list |
| `supportTickets/chatTranscripts` | read-only nested view | list, get (always empty) |
| `fileWorkspaces` | subscription and tenant | create, get |
| `fileWorkspaces/files` | child | create, get, list, `upload` |
| `services`, `services/problemClassifications` | read-only tenant views | list, get |
| `checkNameAvailability` | subscription and tenant action | POST (tickets) |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- a ticket needs a `title`, `description`, a support service ID, a problem classification of that service, a `severity` (`minimal`, `moderate`, `critical` or `highestcriticalimpact`), `advancedDiagnosticConsent` `Yes` or `No`, and contact details: first and last name, an email address, contact method (`email` or `phone`), time zone, country and language, plus a phone number for phone contact or `critical` and `highestcriticalimpact` severity; `problemStartTime` must be RFC 3339 and secondary consents `Yes` or `No`
- ticket and communication creates answer 200, never 201, like the API; tickets report a stable 16-digit `supportTicketId`, a `Standard` support plan, the response SLA for their severity, catalog display names, and `status` `Open`; only a PATCH changes the status, so a PUT never closes or reopens a ticket
- PATCH takes the update model's top-level `severity`, `status` (`open` or `closed`), `contactDetails`, `advancedDiagnosticConsent` and `secondaryConsent`, checking the enums, email and contact method; contact details merge with the stored ones
- communications need a `subject` and `body` (and an email `sender` when set), are `web` and `inbound`, and can't be added to a closed ticket; they have no PATCH (405)
- files need an integer `fileSize` up to 5 MB, `chunkSize` up to 2.5 MB and `numberOfChunks` of 1 or 2; `upload` takes a base64 chunk no larger than `chunkSize` with a `chunkIndex` below `numberOfChunks` and answers 204; workspace names are letters, digits, spaces, `_`, `-` and `.`, and workspaces expire 30 days after creation
- the service catalog is a sample of five services, each with a few problem classifications whose GUIDs the emulator generates; tickets may name any service and problem classification GUID
- ticket names are checked across all scopes (subscriptions and tenant) and communication names within their ticket

Out of scope for this stage:
- the full service catalog, real problem classification GUIDs, quota ticket payloads, and support plans
- engineer responses, chat transcripts and ticket assignment
- workspace and file names with spaces (paths are not percent-decoded)
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSupport -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service support`
- `python3 scripts/azure-io-contract-coverage.py --service support`
- `python3 scripts/azure-doc-contract-coverage.py --service support`
