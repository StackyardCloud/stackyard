# Azure Communication - Rooms (`communication-rooms-2025-03-13`) Staged Plan

## Objective

Emulate the Azure Communication Services Rooms data-plane API for `2025-03-13` with deterministic local behavior for:

- room creation, get, list, update, and delete
- room participant listing and merge-patch updates
- repeatable room creation behavior
- an Azure Go SDK-based example under `examples/azure/communication-rooms-2025-03-13`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/rooms/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Rooms `2025-03-13` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed Rooms client is later confirmed in the repo or official Azure package index.

Important documentation note:

- The reviewed `2025-03-13` operation-groups page exposes 2 operation groups:
  - `Rooms`
  - `Participants`
- The method pages are partially authorization-gated on Learn, but the operation-group pages plus the linked method pages were sufficient to confirm the route surface, primary request and response shapes, merge-patch semantics, and repeatability headers.
- This API version does not expose separate add/remove participant routes. Participant mutations are consolidated into `PATCH /rooms/{roomId}/participants`, and the sample payload shows `null` values as removals.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/operation-groups?view=rest-communication-rooms-2025-03-13`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/rooms?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/participants?view=rest-communication-rooms-2025-03-13`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/rooms/create?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/rooms/delete?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/rooms/get?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/rooms/list?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/rooms/update?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/participants/list?view=rest-communication-rooms-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/communication/rooms/participants/update?view=rest-communication-rooms-2025-03-13`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_rooms`
- display name: `Azure Communication - Rooms`
- API version token: `2025-03-13`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_rooms.go`
- tests: `internal/server/provider_azure_communication_rooms_test.go`
- example: `examples/azure/communication-rooms-2025-03-13`
- plan doc: `docs/communication-rooms-2025-03-13-plan.md`

Recommended coverage aliases:

- `communication_rooms`
- `communication-rooms`
- `rooms`
- `azure_communication_rooms`
- `acs_rooms`
- `rest_communication_rooms_2025_03_13`
- `communication_rooms_2025_03_13`

## Surface Summary

The reviewed REST surface contains 2 operation groups and 7 concrete methods:

1. `Rooms`
2. `Participants`

Documented route family:

- `POST {endpoint}/rooms?api-version=2025-03-13`
- `DELETE {endpoint}/rooms/{roomId}?api-version=2025-03-13`
- `GET {endpoint}/rooms/{roomId}?api-version=2025-03-13`
- `GET {endpoint}/rooms?api-version=2025-03-13`
- `PATCH {endpoint}/rooms/{roomId}?api-version=2025-03-13`
- `GET {endpoint}/rooms/{roomId}/participants?api-version=2025-03-13`
- `PATCH {endpoint}/rooms/{roomId}/participants?api-version=2025-03-13`

Recommended Stackyard route family:

- `/azure/communication-rooms/2025-03-13/rooms`
- `/azure/communication-rooms/2025-03-13/rooms/{roomId}`
- `/azure/communication-rooms/2025-03-13/rooms/{roomId}/participants`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s existing Communication services.
- Every route still requires `api-version=2025-03-13`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- security is documented for ACS endpoint authentication using HMAC-style shared-key headers rather than ARM bearer flows
- errors use `Communication Error Response`
- success semantics are standard HTTP rather than ARM-style async polling
- timestamps are RFC3339 date-times
- collection responses use `value` plus optional `nextLink`

### Room Lifecycle

Reviewed method pages:

- `Create`
- `Delete`
- `Get`
- `List`
- `Update`

High-value route behavior from the reviewed pages:

- `Create`
  - `POST /rooms`
  - optional body:
    - `participants`
    - `pstnDialOutEnabled`
    - `validFrom`
    - `validUntil`
  - repeatability headers documented:
    - `Repeatability-Request-ID`
    - `Repeatability-First-Sent`
  - success status: `201 Created`
  - response type: `Room Model`
- `Delete`
  - `DELETE /rooms/{roomId}`
  - success status: `204 No Content`
- `Get`
  - `GET /rooms/{roomId}`
  - success status: `200 OK`
  - response type: `Room Model`
- `List`
  - `GET /rooms`
  - success status: `200 OK`
  - response type: `Rooms Collection`
  - collection shape includes:
    - `value`
    - optional `nextLink`
- `Update`
  - `PATCH /rooms/{roomId}`
  - request media type:
    - `application/merge-patch+json`
  - mutable fields visible on the page:
    - `pstnDialOutEnabled`
    - `validFrom`
    - `validUntil`
  - success status: `200 OK`
  - response type: `Room Model`

Modeling implications:

- first-pass emulation should maintain a room store keyed by server-generated `roomId`
- room records should persist:
  - `id`
  - `createdAt`
  - `validFrom`
  - `validUntil`
  - `pstnDialOutEnabled`
- `Create` should support deterministic idempotency keyed by the repeatability headers when both are present
- `Update` should follow merge-patch semantics and only mutate provided fields
- `Delete` should hard-delete for first-pass simplicity

### Participants

Reviewed method pages:

- `List`
- `Update`

High-value route behavior from the reviewed pages:

- `List`
  - `GET /rooms/{roomId}/participants`
  - success status: `200 OK`
  - response type: `Participants Collection`
  - collection shape includes:
    - `value`
    - optional `nextLink`
  - participant shape includes:
    - `rawId`
    - `role`
- `Update`
  - `PATCH /rooms/{roomId}/participants`
  - request media type:
    - `application/merge-patch+json`
  - request body:
    - `participants` as a map of raw ACS participant IDs to `Participant Properties`
  - `Participant Properties` includes:
    - `role`
  - sample payload shows `null` participant values as removals
  - success status: `200 OK`
  - response body is an empty object

Reviewed participant roles:

- `Presenter`
- `Attendee`
- `Consumer`
- `Collaborator`

Modeling implications:

- participant state should be room-scoped and keyed by normalized ACS raw ID
- `PATCH /participants` should support:
  - add new participant when key does not already exist and value is an object
  - update participant role when key exists and value is an object
  - remove participant when key value is explicit `null`
- first-pass validation only needs to enforce:
  - valid room existence
  - non-empty ACS-style raw IDs
  - valid `role` enum values

## Stage 1: Route Skeleton And Shared Models

Add:

- `internal/server/provider_azure_communication_rooms.go`

Implement:

- service constants:
  - API version `2025-03-13`
  - route prefix `/azure/communication-rooms/2025-03-13`
- router entrypoint:
  - `handleAzureCommunicationRoomsRouter`
- shared helpers for:
  - path dispatch
  - JSON body decoding
  - ACS-style error payloads
  - RFC3339 parsing
  - repeatability header normalization

Add provider state in `internal/server/http.go` for:

- rooms
- room participants
- optional repeatability request cache

Exit criteria:

- the new route family is recognized
- unsupported paths return deterministic `PathNotFound`
- missing or wrong `api-version` returns deterministic validation errors

## Stage 2: Implement Room Lifecycle

Implement:

- `POST /rooms`
- `GET /rooms/{roomId}`
- `GET /rooms`
- `PATCH /rooms/{roomId}`
- `DELETE /rooms/{roomId}`

Recommended first-pass validation:

- `validFrom` and `validUntil` must parse as RFC3339 when provided
- `validUntil` should not be earlier than `validFrom`
- `pstnDialOutEnabled` must be boolean when present
- room IDs must be non-empty for room-scoped routes

Recommended deterministic behavior:

- generate stable sequential or digest-based room IDs
- default `validFrom` to current UTC time when omitted
- default `validUntil` to `validFrom + 180 days` when omitted
- default `pstnDialOutEnabled` to `false`
- return list results in stable creation order
- support `Repeatability-Request-ID` plus `Repeatability-First-Sent` by replaying the same created room when the same create request is repeated

Exit criteria:

- room CRUD works end to end
- create idempotency is deterministic
- list returns a `Rooms Collection` shape with `value`

## Stage 3: Implement Participant List And Merge-Patch Updates

Implement:

- `GET /rooms/{roomId}/participants`
- `PATCH /rooms/{roomId}/participants`

Recommended deterministic behavior:

- store participants as:
  - `rawId`
  - `role`
- on room creation, seed initial participants from the optional create payload
- on participant patch:
  - object value => upsert
  - `null` value => remove
- return an empty object on successful participant patch
- return a `Participants Collection` with stable ordering by raw ID or insertion order

Recommended validation:

- room must exist
- `participants` must be an object for update requests
- participant keys must be non-empty raw IDs
- role must be one of:
  - `Presenter`
  - `Attendee`
  - `Consumer`
  - `Collaborator`

Exit criteria:

- participant list and update behavior is deterministic
- create-room initial participants and later patch updates converge on the same underlying state

## Stage 4: Provider Tests

Add:

- `internal/server/provider_azure_communication_rooms_test.go`

Cover at minimum:

- room create/get/list/update/delete lifecycle
- defaults for omitted `validFrom`, `validUntil`, and `pstnDialOutEnabled`
- participant seed-on-create behavior
- participant patch add/update/remove behavior
- repeatability headers returning the same room on retried create
- validation failures for:
  - invalid API version
  - invalid JSON
  - invalid RFC3339 times
  - `validUntil < validFrom`
  - invalid participant role
  - missing room
  - unknown path

Exit criteria:

- deterministic positive fixtures exist for all reviewed route families
- negative tests prove no Rooms route falls through to `NotImplemented`

## Stage 5: Azure Go SDK Example

Add:

- `examples/azure/communication-rooms-2025-03-13/main.go`
- `examples/azure/communication-rooms-2025-03-13/Dockerfile`
- `examples/azure/communication-rooms-2025-03-13/docker-compose.yml`

Recommended example flow:

1. Create a room with one seeded participant.
2. List rooms.
3. Get the created room.
4. Update `pstnDialOutEnabled` or validity window.
5. List participants.
6. Patch participants to add one and remove one.
7. List participants again.
8. Delete the room.

Recommended SDK approach:

- use `azcore/runtime` and `azcore/policy`
- implement a small shared-key signing policy that mirrors the repo’s other ACS data-plane examples
- do not invent an `arm...` client because this is not ARM

Exit criteria:

- the example is runnable against Stackyard
- the example demonstrates both room and participant flows

## Stage 6: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for `communication_rooms`, including:

- `communication_rooms`
- `communication-rooms`
- `rooms`
- `azure_communication_rooms`
- `acs_rooms`
- `rest_communication_rooms_2025_03_13`
- `communication_rooms_2025_03_13`

Map the plan doc in `azure-doc-contract-coverage.py`:

- `communication_rooms` -> `communication-rooms-2025-03-13-plan.md`

Exit criteria:

- the service is discoverable by all Azure coverage scripts
- the plan doc is associated with the service key

## Stage 7: Final Validation

Run:

- `go test ./internal/server -run 'TestAzureCommunicationRooms' -count=1`
- `go build ./examples/azure/communication-rooms-2025-03-13`
- `python3 scripts/azure-contract-coverage.py --service communication_rooms --require-service communication_rooms --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_rooms --require-service communication_rooms --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_rooms --require-service communication_rooms --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_rooms --require-service communication_rooms --fail-on any`
- `git diff --check -- ...`

Success criteria:

- all reviewed Rooms endpoints are covered and successful
- no Rooms endpoint returns `NotImplemented`
- the example builds cleanly
- the new service is addressable through the repo’s standard Azure coverage aliases
