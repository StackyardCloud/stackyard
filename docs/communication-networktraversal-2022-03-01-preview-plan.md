# Azure Communication - Network Traversal (`communication-networktraversal-2022-03-01-preview`) Staged Plan

## Objective

Emulate the Azure Communication Services Network Traversal data-plane API for `2022-03-01-preview` with deterministic local behavior for:

- relay-configuration issuance for STUN/TURN negotiation
- an Azure Go SDK-based example under `examples/azure/communication-networktraversal-2022-03-01-preview`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented route is rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/networkTraversal/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Network Traversal `2022-03-01-preview` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official Azure package index.

Important documentation note:

- The reviewed Learn surface exposes a single operation group, `Communication Network Traversal`.
- The concrete method pages reviewed under that group document only 1 operation:
  - `Issue Relay Configuration`
- The plan below intentionally stays at that verified scope and does not invent TURN allocation management, token refresh, identity lifecycle, or generic WebRTC signaling routes that were not part of the reviewed Network Traversal data-plane surface.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/networktraversal/operation-groups?view=rest-communication-networktraversal-2022-03-01-preview`

Operation-group page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/networktraversal/communication-network-traversal?view=rest-communication-networktraversal-2022-03-01-preview`

Linked method page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/networktraversal/communication-network-traversal/issue-relay-configuration?view=rest-communication-networktraversal-2022-03-01-preview`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_networktraversal`
- display name: `Azure Communication - Network Traversal`
- API version token: `2022-03-01-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_networktraversal.go`
- tests: `internal/server/provider_azure_communication_networktraversal_test.go`
- example: `examples/azure/communication-networktraversal-2022-03-01-preview`
- plan doc: `docs/communication-networktraversal-2022-03-01-preview-plan.md`

Recommended coverage aliases:

- `communication_networktraversal`
- `communication-networktraversal`
- `networktraversal`
- `network_traversal`
- `azure_communication_networktraversal`
- `acs_networktraversal`
- `rest_communication_networktraversal_2022_03_01_preview`
- `communication_networktraversal_2022_03_01_preview`

## Surface Summary

The reviewed REST surface contains 1 operation group and 1 concrete method:

1. `Communication Network Traversal`

Documented route family:

- `POST {endpoint}/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview`

Recommended Stackyard route family:

- `/azure/communication-networktraversal/2022-03-01-preview/networkTraversal/:issueRelayConfiguration`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other Communication endpoint-rooted services.
- Every route still requires `api-version=2022-03-01-preview`.

## Contract Notes From Reviewed Method Page

Shared traits across the reviewed method page:

- security is documented for ACS endpoint authentication rather than ARM bearer flows
- errors use `CommunicationErrorResponse`
- success semantics are standard HTTP rather than ARM async polling
- the service is endpoint-rooted and returns ephemeral relay credentials rather than tracked resources

### Communication Network Traversal - Issue Relay Configuration

Reviewed method page:

- `Issue Relay Configuration`

Route:

- `POST {endpoint}/networkTraversal/:issueRelayConfiguration?api-version=2022-03-01-preview`

Request traits from the reviewed page:

- optional body fields:
  - `id`
  - `routeType`
  - `ttl`
- `id`
  - an ACS user identity associated with telemetry for relayed data
  - must be an existing ACS user identity if supplied
- `routeType`
  - enum values reviewed:
    - `any`
    - `nearest`
- `ttl`
  - integer seconds
  - minimum: `0`
  - maximum: `172800`
  - documented default value: `172800`
  - if the given value exceeds the default, the default is used

Response traits from the reviewed page:

- success status: `200 OK`
- success body type: `CommunicationRelayConfiguration`
- reviewed success fields:
  - `expiresOn`
  - `iceServers`
- each `iceServers` entry contains:
  - `urls`
  - `username`
  - `credential`
  - `routeType`

Error traits from the reviewed page:

- non-success responses use `CommunicationErrorResponse`
- error envelope includes:
  - `error.code`
  - `error.message`
  - optional nested details and target fields

Modeling implications:

- first-pass emulation should maintain no durable resource hierarchy because the reviewed surface only returns short-lived relay credentials
- the route should synthesize deterministic ICE server credentials from request inputs instead of creating persistent ACS resources
- `id` should be optional, but when provided it should be shape-validated as an ACS-style identity string
- `routeType` should be limited to the reviewed enum values
- `ttl` should be clamped to the documented maximum and should influence `expiresOn`
- if `routeType` is omitted, the reviewed page says all route types can be returned in separate ICE servers; first pass can either:
  - return both `any` and `nearest` server entries, or
  - return a deterministic default set with both route types represented
- the response should look like usable TURN credentials, with deterministic but fake:
  - `turn:` and `stun:` URLs
  - `username`
  - `credential`
- unknown routes must return a structured error instead of `NotImplemented`

## Recommended Emulator Shape

State model:

- no persistent top-level ACS resources are required
- optional in-memory issuance log keyed by deterministic request digest can help:
  - keep test assertions stable
  - support deterministic repeated responses for identical payloads
  - preserve seeded response timestamps within a single test process

Primary response model:

- `CommunicationRelayConfiguration`
  - `expiresOn`
  - `iceServers: []CommunicationIceServer`
- `CommunicationIceServer`
  - `urls: []string`
  - `username`
  - `credential`
  - `routeType`

Error model:

- reuse the existing ACS-style error envelope patterns already present in the repo’s Communication data-plane emulators
- return stable codes such as:
  - `InvalidApiVersion`
  - `InvalidRequest`
  - `PathNotFound`

Deterministic fixture guidance:

- derive relay usernames and credentials from a stable digest of:
  - `id`
  - normalized `routeType`
  - effective `ttl`
- derive `expiresOn` from `time.Now().UTC()` plus effective TTL, while keeping tests tolerant to runtime timestamps or anchoring with deterministic seeded clocks if the repo already has that pattern
- use fixed hostname patterns for ICE URLs, for example:
  - `turn:relay.communication.azure.local:3478`
  - `stun:relay.communication.azure.local:3478`
- when `routeType` is omitted, return two ICE server entries:
  - one `any`
  - one `nearest`

## Validation Rules To Encode

Request validation:

- reject missing or incorrect `api-version`
- reject invalid JSON
- reject invalid `routeType`
- reject non-integer or out-of-range `ttl`
- reject malformed `id` values when present

Response validation:

- `200 OK` for valid issuance requests
- response body always contains:
  - `expiresOn`
  - non-empty `iceServers`
- every ICE server entry contains:
  - non-empty `urls`
  - non-empty `username`
  - non-empty `credential`
  - reviewed `routeType`

## Staged Implementation Plan

### Stage 1: Route Skeleton And Service Registration

Add the service shell and route entry points.

Implementation work:

- add `internal/server/provider_azure_communication_networktraversal.go`
- register the router in `internal/server/http.go`
- add any minimal server state needed for deterministic issuance bookkeeping
- ensure routing is isolated under:
  - `/azure/communication-networktraversal/2022-03-01-preview`

Acceptance target:

- unknown Network Traversal routes return structured `PathNotFound`
- valid route family is recognized and never falls through to generic `NotImplemented`

### Stage 2: Relay Configuration Issuance

Implement the single reviewed REST method end to end.

Implementation work:

- implement:
  - `POST /networkTraversal/:issueRelayConfiguration`
- validate request body shape and query version
- clamp TTL to the documented maximum
- synthesize deterministic relay credentials and ICE URLs
- return `CommunicationRelayConfiguration`

Acceptance target:

- success path matches the reviewed response shape
- invalid payloads return ACS-style error envelopes

### Stage 3: Focused Provider Tests

Cover the entire reviewed surface with provider tests.

Implementation work:

- add `internal/server/provider_azure_communication_networktraversal_test.go`
- cover:
  - successful issuance with no body
  - successful issuance with `id`
  - successful issuance with `routeType=any`
  - successful issuance with `routeType=nearest`
  - successful issuance with oversized TTL clamped to default
  - invalid API version
  - invalid JSON
  - invalid `routeType`
  - invalid `ttl`
  - invalid `id`
  - unknown path

Acceptance target:

- tests prove every reviewed endpoint is implemented
- no Network Traversal request returns `NotImplemented`

### Stage 4: Azure Go SDK Example

Add a runnable example that exercises the reviewed data-plane route.

Implementation work:

- add `examples/azure/communication-networktraversal-2022-03-01-preview/main.go`
- add matching `Dockerfile`
- add matching `docker-compose.yml`
- use the official Azure Go SDK core runtime pipeline:
  - `azcore/runtime`
  - `azcore/policy`
- send a relay-configuration request and print the returned ICE servers

Acceptance target:

- example builds successfully
- example demonstrates the exact Stackyard route family and required `api-version`

### Stage 5: Coverage Script Wiring

Register the new service in the Azure coverage helpers.

Implementation work:

- update `scripts/azure-contract-coverage.py`
- update `scripts/azure-io-contract-coverage.py`
- update `scripts/azure-doc-contract-coverage.py`
- add aliases for `communication_networktraversal`
- map the service to this plan doc

Acceptance target:

- the coverage scripts discover the service under canonical and alias names
- doc coverage recognizes the Learn references in this plan doc

### Stage 6: Final Validation

Run the service-specific validation gates.

Validation commands to run during implementation:

- `go test ./internal/server -run 'TestAzureCommunicationNetworkTraversal' -count=1`
- `go build ./examples/azure/communication-networktraversal-2022-03-01-preview`
- `python3 scripts/azure-contract-coverage.py --service communication_networktraversal --require-service communication_networktraversal --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_networktraversal --require-service communication_networktraversal --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_networktraversal --require-service communication_networktraversal --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_networktraversal --require-service communication_networktraversal --fail-on any`
- `make ci`

Expected outcome:

- all Network Traversal endpoints are covered and successful
- no Network Traversal endpoint returns `NotImplemented`
- any remaining `make ci` failures should be reported separately if they are unrelated pre-existing repo issues
