# Azure Communication - Chat (`communication-chat-2025-03-15`) Staged Plan

## Objective

Emulate the Azure Communication Services Chat data-plane API for `2025-03-15` with deterministic local behavior for:

- chat thread creation, listing, get, update, and delete
- thread participant add, remove, and list
- thread message send, get, list, update, and delete
- read receipt send and list
- typing notification send
- an Azure Go SDK-based example under `examples/azure/communication-chat-2025-03-15`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/chat/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other Communication data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Chat `2025-03-15` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official package index.

Important documentation note:

- The Learn pages for this service are authorization-gated, but the operation-group pages and linked method pages were still reviewable enough to extract the route surface, primary request and response shapes, and major validation rules.
- The plan below is intentionally scoped to the methods explicitly visible from the reviewed `2025-03-15` pages and avoids inventing extra chat features that were not documented there.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/chat/operation-groups?view=rest-communication-chat-2025-03-15`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread?view=rest-communication-chat-2025-03-15`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat/create-chat-thread?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat/delete-chat-thread?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat/list-chat-threads?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/add-chat-participants?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/delete-chat-message?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/get-chat-message?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/get-chat-thread-properties?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/list-chat-messages?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/list-chat-participants?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/list-chat-read-receipts?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/remove-chat-participant?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/send-chat-message?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/send-chat-read-receipt?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/send-typing-notification?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/update-chat-message?view=rest-communication-chat-2025-03-15`
- `https://learn.microsoft.com/en-us/rest/api/communication/chat/chat-thread/update-chat-thread-properties?view=rest-communication-chat-2025-03-15`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_chat`
- display name: `Azure Communication - Chat`
- API version token: `2025-03-15`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_chat.go`
- tests: `internal/server/provider_azure_communication_chat_test.go`
- example: `examples/azure/communication-chat-2025-03-15`
- plan doc: `docs/communication-chat-2025-03-15-plan.md`

Recommended coverage aliases:

- `communication_chat`
- `communication-chat`
- `chat`
- `azure_communication_chat`
- `acs_chat`
- `communication_chat_2025_03_15`
- `rest_communication_chat_2025_03_15`

## Surface Summary

The reviewed REST surface contains 2 operation groups and 14 concrete methods:

1. `Chat`
2. `Chat Thread`

Documented route family:

- `POST {endpoint}/chat/threads?api-version=2025-03-15`
- `DELETE {endpoint}/chat/threads/{chatThreadId}?api-version=2025-03-15`
- `GET {endpoint}/chat/threads?api-version=2025-03-15`
- `GET {endpoint}/chat/threads?maxPageSize={maxPageSize}&startTime={startTime}&api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}?api-version=2025-03-15`
- `PATCH {endpoint}/chat/threads/{chatThreadId}?api-version=2025-03-15`
- `POST {endpoint}/chat/threads/{chatThreadId}/participants/:add?api-version=2025-03-15`
- `POST {endpoint}/chat/threads/{chatThreadId}/participants/:remove?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/participants?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/participants?maxPageSize={maxPageSize}&skip={skip}&api-version=2025-03-15`
- `POST {endpoint}/chat/threads/{chatThreadId}/messages?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/messages/{chatMessageId}?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/messages?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/messages?maxPageSize={maxPageSize}&startTime={startTime}&api-version=2025-03-15`
- `PATCH {endpoint}/chat/threads/{chatThreadId}/messages/{chatMessageId}?api-version=2025-03-15`
- `DELETE {endpoint}/chat/threads/{chatThreadId}/messages/{chatMessageId}?api-version=2025-03-15`
- `POST {endpoint}/chat/threads/{chatThreadId}/readReceipts?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/readReceipts?api-version=2025-03-15`
- `GET {endpoint}/chat/threads/{chatThreadId}/readReceipts?maxPageSize={maxPageSize}&skip={skip}&api-version=2025-03-15`
- `POST {endpoint}/chat/threads/{chatThreadId}/typing?api-version=2025-03-15`

Recommended Stackyard route family:

- `/azure/communication-chat/2025-03-15/chat/threads`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/participants/:add`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/participants/:remove`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/participants`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/messages`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/messages/{chatMessageId}`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/readReceipts`
- `/azure/communication-chat/2025-03-15/chat/threads/{chatThreadId}/typing`

Important routing note:

- This API should not be merged into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s existing Communication and Web PubSub data-plane services.
- Every route still requires `api-version=2025-03-15`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- authorization is documented as an ACS user access token in the `Authorization` header
- errors use `Communication Error Response`
- success semantics are standard HTTP rather than ARM async polling
- route parameters are endpoint-rooted and thread-centric
- list routes expose bounded paging controls such as `maxPageSize`, plus `startTime` or `skip` depending on the collection

### Thread Lifecycle

Reviewed method pages:

- `Create Chat Thread`
- `Delete Chat Thread`
- `List Chat Threads`
- `Get Chat Thread Properties`
- `Update Chat Thread Properties`

High-value route behavior from the reviewed pages:

- `Create Chat Thread`
  - `POST /chat/threads`
  - required body:
    - `topic`
  - optional body:
    - `metadata`
    - `participants`
    - `retentionPolicy`
  - repeatability header documented:
    - `repeatability-request-id`
  - success status: `201 Created`
  - response includes `chatThread` plus `invalidParticipants`
- `Delete Chat Thread`
  - `DELETE /chat/threads/{chatThreadId}`
  - success status: `204 No Content`
- `List Chat Threads`
  - `GET /chat/threads`
  - optional query:
    - `maxPageSize`
    - `startTime`
- `Get Chat Thread Properties`
  - `GET /chat/threads/{chatThreadId}`
- `Update Chat Thread Properties`
  - `PATCH /chat/threads/{chatThreadId}`
  - request uses merge-patch semantics in practice for mutable fields like `topic`

Modeling implications:

- first-pass emulation should maintain a deterministic thread store keyed by `chatThreadId`
- thread creation should seed `createdOn`, `createdByCommunicationIdentifier`, topic, metadata, and retention policy
- invalid participants do not need to be deeply simulated, but the response shape should reserve room for them
- list behavior should support stable ordering plus lightweight `maxPageSize` and `startTime` filtering

### Participants

Reviewed method pages:

- `Add Chat Participants`
- `List Chat Participants`
- `Remove Chat Participant`

High-value route behavior from the reviewed pages:

- `Add Chat Participants`
  - `POST /chat/threads/{chatThreadId}/participants/:add`
  - required body:
    - `participants`
  - participant shape includes:
    - `communicationIdentifier`
    - optional `displayName`
    - optional `shareHistoryTime`
  - success status: `201 Created`
  - response type: `Add Chat Participants Result`
- `List Chat Participants`
  - `GET /chat/threads/{chatThreadId}/participants`
  - optional query:
    - `maxPageSize`
    - `skip`
- `Remove Chat Participant`
  - `POST /chat/threads/{chatThreadId}/participants/:remove`
  - request removes by participant identity from body

Modeling implications:

- store participants as thread-scoped child records keyed by normalized identifier raw ID
- add should be idempotent enough that duplicate participants do not create duplicate rows
- `shareHistoryTime` can be retained and echoed but does not need deeper server behavior in the first pass
- list should support stable pagination with `skip`

### Messages

Reviewed method pages:

- `Send Chat Message`
- `Get Chat Message`
- `List Chat Messages`
- `Update Chat Message`
- `Delete Chat Message`

High-value route behavior from the reviewed pages:

- `Send Chat Message`
  - `POST /chat/threads/{chatThreadId}/messages`
  - required body:
    - `content`
  - optional body:
    - `metadata`
    - `senderDisplayName`
    - `type`
  - success status: `201 Created`
  - response type: `Send Chat Message Result`
  - `Location` header points to the new message resource
- `Get Chat Message`
  - `GET /chat/threads/{chatThreadId}/messages/{chatMessageId}`
- `List Chat Messages`
  - `GET /chat/threads/{chatThreadId}/messages`
  - optional query:
    - `maxPageSize`
    - `startTime`
- `Update Chat Message`
  - `PATCH /chat/threads/{chatThreadId}/messages/{chatMessageId}`
  - request content type is documented as `application/merge-patch+json`
  - mutable fields include:
    - `content`
    - `metadata`
  - success status: `204 No Content`
- `Delete Chat Message`
  - `DELETE /chat/threads/{chatThreadId}/messages/{chatMessageId}`
  - success status: `204 No Content`

Modeling implications:

- store thread messages separately from participants and threads so list/get/update/delete logic remains straightforward
- first-pass emulation should generate deterministic message IDs and `Location` headers
- the mutable-vs-deleted state should be reflected in message fixtures so update/delete tests can observe lifecycle changes
- `type` can be validated to a conservative documented subset such as `text` first, with room to widen later if exact enum coverage is needed during implementation

### Read Receipts And Typing

Reviewed method pages:

- `Send Chat Read Receipt`
- `List Chat Read Receipts`
- `Send Typing Notification`

High-value route behavior from the reviewed pages:

- `Send Chat Read Receipt`
  - `POST /chat/threads/{chatThreadId}/readReceipts`
  - required body:
    - `chatMessageId`
  - success status: `200 OK`
- `List Chat Read Receipts`
  - `GET /chat/threads/{chatThreadId}/readReceipts`
  - optional query:
    - `maxPageSize`
    - `skip`
- `Send Typing Notification`
  - `POST /chat/threads/{chatThreadId}/typing`
  - lightweight event route

Modeling implications:

- read receipts should be stored per thread as event rows with stable ordering
- first-pass typing behavior can be ephemeral or represented as a deterministic accepted event without a persisted feed
- the core value is route coverage and response correctness, not a full real-time system

## State Model

Recommended in-memory state buckets:

- thread records keyed by `chatThreadId`
- participant records keyed by `chatThreadId + participantRawId`
- message records keyed by `chatThreadId + chatMessageId`
- read receipt records keyed by `chatThreadId + receipt sequence`
- optional ephemeral typing-event ledger if tests need observable side effects

Recommended persisted fixture shape:

- thread record:
  - `id`
  - `topic`
  - `metadata`
  - `retentionPolicy`
  - `createdOn`
  - `createdByCommunicationIdentifier`
  - `deleted`
- participant record:
  - normalized identifier
  - `displayName`
  - `shareHistoryTime`
  - join timestamp
- message record:
  - `id`
  - `type`
  - `content`
  - `metadata`
  - `senderDisplayName`
  - sender identifier
  - `createdOn`
  - `editedOn`
  - `deletedOn`
- read receipt record:
  - `chatMessageId`
  - reader identifier
  - receipt timestamp

## Validation Priorities

High-value request validation from the reviewed pages:

- require `api-version=2025-03-15`
- require non-empty `chatThreadId` where applicable
- require non-empty `chatMessageId` where applicable
- require `topic` on thread creation
- require `participants` array for participant add
- require `content` for message send
- require merge-patch request shape for message updates
- require `chatMessageId` for read receipt send
- reject malformed `maxPageSize`, `skip`, and `startTime` query values

High-value response shape priorities:

- thread creation should include a `chatThread` object and optionally `invalidParticipants`
- list routes should return predictable paged envelopes
- message creation should include a created result plus `Location`
- delete and patch routes should use the documented no-content statuses where applicable
- read receipt send should return `200 OK`

## Staged Implementation

### Stage 1: Route Skeleton And Shared Data-Plane Primitives

Add:

- `internal/server/provider_azure_communication_chat.go`
- `internal/server/provider_azure_communication_chat_test.go`

Implement:

- router entrypoint for `/azure/communication-chat/2025-03-15/...`
- shared API-version guard
- shared Communication error response helper
- common identifier normalization helpers
- thread, participant, message, and receipt state buckets

Acceptance target:

- unrecognized Chat routes return deterministic Chat-flavored `PathNotFound` style errors rather than generic `NotImplemented`

### Stage 2: Core Thread Lifecycle

Implement:

- `POST /chat/threads`
- `GET /chat/threads`
- `GET /chat/threads/{chatThreadId}`
- `PATCH /chat/threads/{chatThreadId}`
- `DELETE /chat/threads/{chatThreadId}`

Acceptance target:

- threads can be created, listed, fetched, updated, and deleted with stable fixture output and list filtering by `startTime`

### Stage 3: Participants

Implement:

- `POST /chat/threads/{chatThreadId}/participants/:add`
- `POST /chat/threads/{chatThreadId}/participants/:remove`
- `GET /chat/threads/{chatThreadId}/participants`

Acceptance target:

- participant membership changes are observable through list results and validate the documented request bodies

### Stage 4: Messages

Implement:

- `POST /chat/threads/{chatThreadId}/messages`
- `GET /chat/threads/{chatThreadId}/messages`
- `GET /chat/threads/{chatThreadId}/messages/{chatMessageId}`
- `PATCH /chat/threads/{chatThreadId}/messages/{chatMessageId}`
- `DELETE /chat/threads/{chatThreadId}/messages/{chatMessageId}`

Acceptance target:

- message CRUD-style lifecycle is deterministic, `Location` headers are emitted for create, and merge-patch update semantics are accepted for the documented mutable fields

### Stage 5: Read Receipts And Typing

Implement:

- `POST /chat/threads/{chatThreadId}/readReceipts`
- `GET /chat/threads/{chatThreadId}/readReceipts`
- `POST /chat/threads/{chatThreadId}/typing`

Acceptance target:

- read receipts are observable through list results and typing routes return a stable documented success shape

### Stage 6: Focused Provider Tests

Add contract-style tests covering:

- thread lifecycle
- participant add/remove/list
- message send/get/list/update/delete
- read receipt send/list
- typing notification
- validation failures for missing required fields and malformed query parameters

Acceptance target:

- a focused `go test ./internal/server -run 'TestAzureCommunicationChat' -count=1` pass

### Stage 7: Azure Go SDK Example

Add:

- `examples/azure/communication-chat-2025-03-15/main.go`
- `examples/azure/communication-chat-2025-03-15/Dockerfile`
- `examples/azure/communication-chat-2025-03-15/docker-compose.yml`

Recommended example flow:

- create thread
- list threads
- add participants
- send message
- get/list/update/delete message
- send/list read receipts
- send typing notification

SDK guidance:

- use `azcore/runtime` and `azcore/policy` with a small ACS auth policy rather than inventing a fake typed client

Acceptance target:

- `go build ./examples/azure/communication-chat-2025-03-15`

### Stage 8: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add:

- canonical aliases for `communication_chat`
- plan-doc mapping for `communication-chat-2025-03-15-plan.md`

Acceptance target:

- coverage scripts discover the new service and do not report it as unmapped or undocumented

## Coverage Expectations

When implementation lands, the following should pass:

- `go test ./internal/server -run 'TestAzureCommunicationChat' -count=1`
- `go build ./examples/azure/communication-chat-2025-03-15`
- `python3 scripts/azure-contract-coverage.py --service communication_chat --require-service communication_chat --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_chat --require-service communication_chat --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_chat --require-service communication_chat --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_chat --require-service communication_chat --fail-on any`

## Notes For The Future Implementation Pass

- Preserve the ACS data-plane route style already used by the repo’s Communication services instead of threading these routes into the ARM provider dispatcher.
- Keep the first implementation conservative around message `type` and retention-policy variants unless exact enum coverage is needed for a failing contract or example.
- Prefer stable deterministic IDs and timestamps so repeat tests and coverage scripts remain reliable.
