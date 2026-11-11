# Azure Communication - Email (`communication-email-2025-09-01`) Staged Plan

## Objective

Emulate the Azure Communication Services Email data-plane API for `2025-09-01` with deterministic local behavior for:

- email send submission
- email send operation-status polling
- an Azure Go SDK-based example under `examples/azure/communication-email-2025-09-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Email `2025-09-01` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official package index.

Important documentation note:

- The `2025-09-01` operation-groups page exposes a single `Email` group.
- The concrete method pages reviewed under that group document only 2 operations:
  - `Send`
  - `Get Send Result`
- The plan below intentionally stays at that verified scope and does not invent mailbox, domain, suppression-list, or template-management routes that were not part of the reviewed Email data-plane surface.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/email/operation-groups?view=rest-communication-email-2025-09-01`

Operation-group page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/email/email?view=rest-communication-email-2025-09-01`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/email/email/send?view=rest-communication-email-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/email/email/get-send-result?view=rest-communication-email-2025-09-01`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_email`
- display name: `Azure Communication - Email`
- API version token: `2025-09-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_email.go`
- tests: `internal/server/provider_azure_communication_email_test.go`
- example: `examples/azure/communication-email-2025-09-01`
- plan doc: `docs/communication-email-2025-09-01-plan.md`

Recommended coverage aliases:

- `communication_email`
- `communication-email`
- `email`
- `azure_communication_email`
- `acs_email`
- `rest_communication_email_2025_09_01`
- `communication_email_2025_09_01`

## Surface Summary

The reviewed REST surface contains 1 operation group and 2 concrete methods:

1. `Email`

Documented route family:

- `POST {endpoint}/emails:send?api-version=2025-09-01`
- `GET {endpoint}/emails/operations/{operationId}?api-version=2025-09-01`

Recommended Stackyard route family:

- `/azure/communication-email/2025-09-01/emails:send`
- `/azure/communication-email/2025-09-01/emails/operations/{operationId}`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other Communication endpoint-rooted services.
- Every route still requires `api-version=2025-09-01`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:

- security is documented as HMAC-SHA256 authorization via the `Authorization` header
- `x-ms-client-request-id` is supported on send
- errors use `Error Response`
- the service is endpoint-rooted and long-running-operation flavored rather than ARM-async flavored
- the send flow relies on:
  - request header `Operation-Id`
  - response header `Operation-Location`
  - optional response header `retry-after`

### Email - Send

Reviewed method page:

- `Email - Send`

Route:

- `POST {endpoint}/emails:send?api-version=2025-09-01`

Request traits from the reviewed page:

- required body fields:
  - `senderAddress`
  - `content`
  - `recipients`
- `content` includes:
  - required `subject`
  - optional `plainText`
  - optional `html`
- `recipients` includes recipient lists:
  - `to`
  - optional `cc`
  - optional `bcc`
- optional top-level fields:
  - `attachments`
  - `headers`
  - `replyTo`
  - `userEngagementTrackingDisabled`
- attachment shape includes:
  - `name`
  - `contentType`
  - `contentInBase64`
  - optional `contentId`
- documented request header:
  - `Operation-Id`
  - if omitted, the service generates one

Response traits from the reviewed page:

- success status: `202 Accepted`
- success body: `Email Send Result`
- response headers:
  - `Operation-Location`
  - `retry-after`
- sample accepted body contains:
  - `id`
  - `status`
- reviewed initial accepted status:
  - `Running`

Modeling implications:

- first-pass emulation should maintain a deterministic email-operation store keyed by operation ID
- if `Operation-Id` is supplied, use it as the primary idempotency key
- if `Operation-Id` is omitted, generate a deterministic UUID-like operation ID
- validate the minimum documented structure:
  - `senderAddress` present
  - `content.subject` present
  - at least one `recipients.to` entry
- echo accepted operation state immediately as `Running`
- return an `Operation-Location` header pointing at the poll route inside Stackyard

### Email - Get Send Result

Reviewed method page:

- `Email - Get Send Result`

Route:

- `GET {endpoint}/emails/operations/{operationId}?api-version=2025-09-01`

Request traits from the reviewed page:

- `operationId` is the path parameter
- it is documented as the GUID returned from a previous send call

Response traits from the reviewed page:

- success status: `200 OK`
- success body: `Email Send Result`
- optional response header:
  - `retry-after`
- reviewed result fields:
  - `id`
  - `status`
  - optional `error`
- reviewed status enum values:
  - `NotStarted`
  - `Running`
  - `Succeeded`
  - `Failed`
  - `Canceled`
- reviewed sample terminal error shape:
  - `error.code`
  - `error.message`

Modeling implications:

- first-pass emulation should support deterministic polling behavior:
  - newly accepted operations start as `Running`
  - later polls may remain `Running` or transition deterministically to `Succeeded`
- the simplest credible model is:
  - send creates `Running`
  - first poll returns `Running` with `retry-after`
  - second and later polls return `Succeeded`
- retain the ability to seed explicit failure states in tests so error payloads are covered
- unknown operation IDs should return a documented-style error envelope rather than `NotImplemented`

## Recommended Emulator Shape

State model:

- `email operations` keyed by `operationId`
- each operation stores:
  - request digest or raw request for deterministic idempotency
  - sender address
  - recipient summary
  - content summary
  - status
  - poll count
  - optional error detail
  - timestamps such as `createdAt` and `lastUpdatedAt`

Suggested helper behaviors:

- normalize email addresses to a stable lookup key for summaries
- compute a deterministic operation ID when the client omits `Operation-Id`
- keep polling deterministic so repeated tests get the same status transitions
- cap request validation to the documented high-value shape, not full MIME or RFC email validation

Error-handling expectations:

- reject wrong or missing `api-version`
- reject invalid JSON bodies
- reject missing required send fields
- reject missing or malformed `Operation-Id` only if validation is explicitly enforced
- return a consistent ACS-style error envelope for unknown operations and validation failures

## Staged Implementation Plan

### Stage 1: Route Skeleton And Shared Validation

Add the new data-plane route family and service registration:

- implement `internal/server/provider_azure_communication_email.go`
- wire the router into `internal/server/http.go`
- enforce `api-version=2025-09-01`
- add shared helpers for JSON decoding, error envelopes, operation ID creation, and operation lookup

Exit criteria:

- both documented routes resolve into the new provider
- unsupported methods return proper method errors rather than `NotImplemented`

### Stage 2: Send Operation

Implement `POST /emails:send`:

- validate the minimum documented email message shape
- create or reuse an operation by `Operation-Id`
- persist deterministic operation state
- return:
  - `202 Accepted`
  - accepted body with `id` and `status`
  - `Operation-Location`
  - `retry-after`

Exit criteria:

- repeated sends with the same explicit `Operation-Id` are stable
- accepted responses match the documented shape closely enough for contract tests

### Stage 3: Operation Polling

Implement `GET /emails/operations/{operationId}`:

- return the stored operation state
- support deterministic status progression from `Running` to `Succeeded`
- emit `retry-after` for non-terminal states
- support seeded failure states in tests

Exit criteria:

- polling a known ID never falls through to `NotImplemented`
- unknown IDs return a stable error contract

### Stage 4: Provider Tests

Add focused tests in `internal/server/provider_azure_communication_email_test.go` for:

- successful send with generated operation ID
- successful send with explicit `Operation-Id`
- idempotent replay on the same `Operation-Id`
- poll progression from `Running` to `Succeeded`
- invalid body validation
- missing required fields
- unknown operation ID
- incorrect `api-version`

Exit criteria:

- the new tests cover every documented route and the main validation paths

### Stage 5: Azure Go SDK Example

Add `examples/azure/communication-email-2025-09-01`:

- use `azcore/runtime` and `azcore/policy`
- authenticate with the documented HMAC-style authorization pattern used by ACS endpoint data planes
- send an email
- parse `Operation-Location`
- poll until terminal success

Example flow should demonstrate:

- constructing the endpoint-rooted request
- setting `Operation-Id`
- printing the accepted result
- following the status URL until completion

Exit criteria:

- the example builds cleanly
- the example exercises both documented routes

### Stage 6: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Required coverage additions:

- map the canonical service key `communication_email`
- add the alias set listed above
- map the plan doc in the doc-coverage script

Exit criteria:

- coverage tooling recognizes `communication_email`
- the service can be targeted explicitly by the Azure coverage scripts

## Example Scope Recommendation

The example should stay intentionally small and exercise the highest-signal path:

1. submit an email send request
2. capture the accepted operation id and `Operation-Location`
3. poll `GET /emails/operations/{operationId}` until the operation becomes `Succeeded`

That is enough to prove:

- request signing and endpoint construction
- send acceptance behavior
- long-running status polling behavior

## Risks And Edge Cases To Handle Deliberately

- `Operation-Id` idempotency behavior should be deterministic rather than probabilistic
- invalid attachment objects should fail consistently if required fields are missing
- the attachment size limit is documented as 10 MB, but a first implementation can treat that as a later strictness enhancement if byte-accurate size accounting adds too much complexity
- `replyTo`, custom `headers`, and `userEngagementTrackingDisabled` should be accepted and echoed or persisted even if they do not affect later behavior
- operation polling should not invent delivery-event semantics beyond the documented send-operation statuses

## Definition Of Done

The service is ready when all of the following are true:

- the two documented `2025-09-01` Email routes are implemented
- no reviewed Email route returns `NotImplemented`
- deterministic provider tests pass
- the example under `examples/azure/communication-email-2025-09-01` builds and runs against Stackyard
- the Azure coverage scripts recognize and validate the new `communication_email` service
