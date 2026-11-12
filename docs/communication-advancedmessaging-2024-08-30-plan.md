# Azure Communication - Advanced Messaging (`communication-advancedmessaging-2024-08-30`) Staged Plan

## Objective

Emulate the Azure Communication Services Advanced Messaging data-plane API for `2024-08-30` with deterministic local behavior for:

- outbound notification sends
- template listing by channel registration
- media download by stream ID
- an Azure Go SDK-based example under `examples/azure/communication-advancedmessaging-2024-08-30`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is a Communication Services endpoint data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the ACS resource endpoint, for example `{resource}.communication.azure.com/...`.
- This should be modeled as a separate Stackyard Azure service from future Communication management-plane surfaces.

Important SDK note:

- I did not confirm a clearly matching typed official Azure Go SDK package dedicated to this exact Advanced Messaging data-plane surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, instead of inventing a non-existent typed client.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/operation-groups?view=rest-communication-advancedmessaging-2024-08-30`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/notification-messages-operations?view=rest-communication-advancedmessaging-2024-08-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/template-operations?view=rest-communication-advancedmessaging-2024-08-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/stream-operations?view=rest-communication-advancedmessaging-2024-08-30`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/notification-messages-operations/send?view=rest-communication-advancedmessaging-2024-08-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/template-operations/list-templates?view=rest-communication-advancedmessaging-2024-08-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/advancedmessaging/stream-operations/get-media?view=rest-communication-advancedmessaging-2024-08-30`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_advancedmessaging`
- display name: `Azure Communication - Advanced Messaging`
- API version token: `2024-08-30`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_advancedmessaging.go`
- tests: `internal/server/provider_azure_communication_advancedmessaging_test.go`
- example: `examples/azure/communication-advancedmessaging-2024-08-30`
- plan doc: `docs/communication-advancedmessaging-2024-08-30-plan.md`

Recommended coverage aliases:

- `communication_advancedmessaging`
- `communication-advancedmessaging`
- `advancedmessaging`
- `advanced_messaging`
- `azure_communication_advancedmessaging`
- `acs_advancedmessaging`
- `rest_communication_advancedmessaging_2024_08_30`
- `communication_advancedmessaging_2024_08_30`

## Surface Summary

The reviewed REST surface contains 3 operation groups and 3 concrete methods:

1. `Notification Messages Operations`
2. `Template Operations`
3. `Stream Operations`

Documented route family:

- `POST {endpoint}/messages/notifications:send?api-version=2024-08-30`
- `GET {endpoint}/messages/channels/{channelId}/templates?api-version=2024-08-30`
- `GET {endpoint}/messages/streams/{id}?api-version=2024-08-30`

Recommended Stackyard route family:

- `/azure/communication-advancedmessaging/2024-08-30/messages/notifications:send`
- `/azure/communication-advancedmessaging/2024-08-30/messages/channels/{channelId}/templates`
- `/azure/communication-advancedmessaging/2024-08-30/messages/streams/{id}`

Important routing note:

- This API should not be mixed into the ARM Azure router chain that expects subscription and provider segments.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other endpoint-rooted Azure services.
- Every route still requires `api-version=2024-08-30`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:

- security supports Microsoft Entra OAuth2 with scope `communication.azure.com/.default`
- the reviewed pages also document API key authorization
- errors use `Azure.Core.Foundations.ErrorResponse`
- `x-ms-client-request-id` is supported across the reviewed methods
- the service endpoint path parameter is the ACS resource URI

### Notification Messages - Send

Reviewed method page:

- `Notification Messages Operations - Send`

Route:

- `POST {endpoint}/messages/notifications:send?api-version=2024-08-30`

Request traits from the reviewed page:

- repeatability headers are supported:
  - `Repeatability-Request-ID`
  - `Repeatability-First-Sent`
- the body is polymorphic by `kind`
- reviewed message kinds:
  - `text`
  - `template`
  - `image`
  - `audio`
  - `video`
  - `document`
  - deprecated `image_v0`
- all reviewed variants require:
  - `channelRegistrationId`
  - `to`
  - `kind`
- additional reviewed required fields:
  - `content` for `text`
  - `template` for `template`
  - `mediaUri` for media-bearing variants
- reviewed optional fields include:
  - `caption`
  - `fileName`

Response traits from the reviewed page:

- success status: `202 Accepted`
- success body: `SendMessageResult`
- example response contains `receipts[]` with:
  - `messageId`
  - `to`

Modeling implications:

- first-pass emulation should accept the documented `kind` discriminator family and validate the required shape for each kind
- the service should generate deterministic receipt rows, one per recipient in `to`
- repeatability headers should be honored enough to make duplicate sends stable when both headers match
- template sends should validate only the high-value shape from the reviewed page:
  - `template.name`
  - `template.language`
  - optional `template.values`
  - optional `template.bindings`

### Template Operations - List Templates

Reviewed method page:

- `Template Operations - List Templates`

Route:

- `GET {endpoint}/messages/channels/{channelId}/templates?api-version=2024-08-30`
- optional query:
  - `maxpagesize`

Request traits from the reviewed page:

- `channelId` is a required UUID path parameter

Response traits from the reviewed page:

- success status: `200 OK`
- success body: `Paged Message Template Item`
- reviewed payload shape:
  - `value`
  - `nextLink`
- reviewed item fields:
  - `name`
  - `language`
  - `status`
  - `kind`
  - `content`
- reviewed enum values include:
  - channel kind: `whatsApp`
  - template status: `approved`, `rejected`, `pending`, `paused`

Modeling implications:

- first-pass emulation should seed deterministic WhatsApp-oriented templates per channel
- pagination can be simple but should respect `maxpagesize`
- `nextLink` can be omitted or set to `null` until multi-page behavior is needed

### Stream Operations - Get Media

Reviewed method page:

- `Stream Operations - Get Media`

Route:

- `GET {endpoint}/messages/streams/{id}?api-version=2024-08-30`

Request traits from the reviewed page:

- `id` is the required stream ID path parameter

Response traits from the reviewed page:

- success status: `200 OK`
- success body is file content
- documented media types:
  - `application/octet-stream`
  - `application/json`

Modeling implications:

- this route should return raw bytes rather than a JSON envelope on success
- first-pass emulation can seed deterministic byte blobs keyed by stream ID
- when the stream ID is unknown, return the documented Azure error envelope instead of empty binary content

## State Model

Recommended in-memory state buckets:

- sent notification ledger keyed by repeatability identity or synthetic send ID
- template catalog keyed by `channelId`
- media blobs keyed by stream ID

Recommended persisted fixture shape:

- sent message record:
  - `kind`
  - `channelRegistrationId`
  - `to`
  - normalized payload details
  - generated receipt rows
  - timestamps
- template record:
  - `name`
  - `language`
  - `status`
  - `kind`
  - `content`
- media record:
  - `id`
  - `contentType`
  - raw bytes or deterministic encoded content

## Validation Priorities

High-value request validation from the reviewed pages:

- require `api-version=2024-08-30`
- require `channelId` to parse as UUID for template listing
- require non-empty stream ID
- for send:
  - require `kind`
  - require `channelRegistrationId`
  - require non-empty `to`
  - require `content` for `text`
  - require `template` for `template`
  - require `mediaUri` for media kinds
- reject malformed JSON with Azure-style `ErrorResponse`

High-value response validation:

- `POST ...notifications:send` returns `202` with `receipts`
- `GET ...templates` returns paged JSON with `value`
- `GET ...streams/{id}` returns non-JSON binary payload on success
- missing resources produce Azure-style error JSON with appropriate non-2xx status

## Staged Implementation Plan

### Stage 1: Route Skeleton And Shared Validation

Add a dedicated data-plane Azure router with:

- API version validation
- shared authorization/header handling consistent with other Azure data-plane shims
- error response helpers for `Azure.Core.Foundations.ErrorResponse`
- a small state store for sends, templates, and media blobs

Files:

- `internal/server/provider_azure_communication_advancedmessaging.go`
- `internal/server/http.go`

Exit criteria:

- the three route families dispatch correctly
- unknown routes do not fall through to `NotImplemented` for the covered surface

### Stage 2: Template And Media Read Routes

Implement:

- `GET /messages/channels/{channelId}/templates`
- `GET /messages/streams/{id}`

Behavior goals:

- deterministic seeded templates
- optional `maxpagesize` handling
- binary media download with stable content type

Why this stage first:

- it establishes both JSON and binary response handling before adding the more complex polymorphic send contract

### Stage 3: Notification Send

Implement:

- `POST /messages/notifications:send`

Behavior goals:

- per-kind validation for the documented message variants
- deterministic receipt generation
- stable repeatability behavior when repeatability headers are present

Representative success fixtures:

- text notification send
- template notification send
- media notification send

### Stage 4: Tests

Add focused provider tests covering:

- operations do not return `NotImplemented` for the reviewed routes
- text, template, and one media send path
- template listing
- media download
- invalid API version
- invalid UUID channel ID
- malformed send payloads
- missing media stream

Files:

- `internal/server/provider_azure_communication_advancedmessaging_test.go`

### Stage 5: Azure Go SDK Example

Add an example under:

- `examples/azure/communication-advancedmessaging-2024-08-30`

Recommended example flow:

1. send a text notification
2. send a template notification
3. list templates for a channel
4. download a media stream

Recommended implementation approach:

- use `azcore/runtime` and `azcore/policy`
- authenticate with a shared-key or API-key style header aligned with Stackyard’s local Azure conventions

Example support files:

- `main.go`
- `Dockerfile`
- `docker-compose.yml`

### Stage 6: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Required coverage updates:

- add the canonical service key `communication_advancedmessaging`
- add the aliases listed above
- map the service to `communication-advancedmessaging-2024-08-30-plan.md`

## Coverage Expectations

When implementation is complete, the following should pass:

- `go test ./internal/server -run 'TestAzureCommunicationAdvancedMessaging' -count=1`
- `go build ./examples/azure/communication-advancedmessaging-2024-08-30`
- `python3 scripts/azure-contract-coverage.py --service communication_advancedmessaging --require-service communication_advancedmessaging --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_advancedmessaging --require-service communication_advancedmessaging --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_advancedmessaging --require-service communication_advancedmessaging --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_advancedmessaging --require-service communication_advancedmessaging --fail-on any`

Success means:

- every reviewed Advanced Messaging endpoint is implemented
- no reviewed route falls through to `NotImplemented`
- the coverage scripts recognize the new service and map it to this plan doc
