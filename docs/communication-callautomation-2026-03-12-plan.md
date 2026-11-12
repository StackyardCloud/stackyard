# Azure Communication - Call Automation (`communication-callautomation-2026-03-12`) Staged Plan

## Objective

Emulate the Azure Communication Services Call Automation data-plane API for `2026-03-12` with deterministic local behavior for:

- inbound call entrypoints
- outbound call creation and call-connection establishment
- call-connection lifecycle and participant management
- call media actions
- call recording actions
- an Azure Go SDK-based example under `examples/azure/communication-callautomation-2026-03-12`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The routes are rooted at the Communication resource endpoint and use `/calling/...`.
- This should be modeled as a separate Stackyard Azure service from any future ACS management-plane APIs.

Important SDK note:

- I did not confirm a clearly matching typed official Go SDK package for this exact Call Automation `2026-03-12` data-plane surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official package index.

Important documentation note:

- Several `2026-03-12` method pages are access-restricted on Learn and were only available through search-result snippets or related operation-group pages during review.
- Where that happened, this plan keeps the route family but avoids overcommitting to deep field-level behavior unless it was visible on an exact method page.
- `Move Participants` is the clearest example, so it is placed in a later reconciliation stage instead of the first implementation slice.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/operation-groups?view=rest-communication-callautomation-2026-03-12`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/answer-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-recording?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/connect?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/create-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/redirect-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/reject-call?view=rest-communication-callautomation-2026-03-12`

Exact method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/answer-call/answer-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/create-call/create-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/connect/connect?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/reject-call/reject-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/add-participant?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/cancel-add-participant?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/get-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/get-participant?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/get-participants?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/hangup-call?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/mute?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/remove-participant?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/transfer-to-participant?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/cancel-all-media-operations?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/hold?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/play?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/recognize?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/send-dtmf-tones?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/start-continuous-dtmf-recognition?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/start-media-streaming?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/start-transcription?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/stop-continuous-dtmf-recognition?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/stop-media-streaming?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/stop-transcription?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/update-transcription?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-media/unhold?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-recording/get-recording-properties?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-recording/pause-recording?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-recording/resume-recording?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-recording/start-recording?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-recording/stop-recording?view=rest-communication-callautomation-2026-03-12`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/redirect-call/redirect-call?view=rest-communication-callautomation-2026-03-12`

Related fallback references used only where the exact `2026-03-12` method page was not retrievable but the `2026-03-12` operation-group page clearly advertised the method:

- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/move-participants?view=rest-communication-callautomation-2025-08-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/communication/callautomation/call-connection/terminate-call?view=rest-communication-callautomation-2025-06-15`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_callautomation`
- display name: `Azure Communication - Call Automation`
- API version token: `2026-03-12`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_callautomation.go`
- tests: `internal/server/provider_azure_communication_callautomation_test.go`
- example: `examples/azure/communication-callautomation-2026-03-12`
- plan doc: `docs/communication-callautomation-2026-03-12-plan.md`

Recommended coverage aliases:

- `communication_callautomation`
- `communication-callautomation`
- `callautomation`
- `call_automation`
- `azure_communication_callautomation`
- `acs_callautomation`
- `rest_communication_callautomation_2026_03_12`
- `communication_callautomation_2026_03_12`

## Surface Summary

The reviewed REST surface contains 8 operation groups:

1. `Answer Call`
2. `Call Connection`
3. `Call Media`
4. `Call Recording`
5. `Connect`
6. `Create Call`
7. `Redirect Call`
8. `Reject Call`

Route families visible from the reviewed method pages:

- `POST {endpoint}/calling/callConnections:answer?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections:connect?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections:redirect?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections:reject?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}/participants:add?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}/participants:cancelAddParticipant?api-version=2026-03-12`
- `GET {endpoint}/calling/callConnections/{callConnectionId}?api-version=2026-03-12`
- `GET {endpoint}/calling/callConnections/{callConnectionId}/participants/{participantRawId}?api-version=2026-03-12`
- `GET {endpoint}/calling/callConnections/{callConnectionId}/participants?api-version=2026-03-12`
- `DELETE {endpoint}/calling/callConnections/{callConnectionId}?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:terminate?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}/participants:remove?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:transferToParticipant?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:cancelAllMediaOperations?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:hold?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:play?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:recognize?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:sendDtmfTones?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:startContinuousDtmfRecognition?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:startMediaStreaming?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:startTranscription?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:stopContinuousDtmfRecognition?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:stopMediaStreaming?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:stopTranscription?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:unhold?api-version=2026-03-12`
- `POST {endpoint}/calling/callConnections/{callConnectionId}:updateTranscription?api-version=2026-03-12`
- `POST {endpoint}/calling/recordings?api-version=2026-03-12`
- `GET {endpoint}/calling/recordings/{recordingId}?api-version=2026-03-12`
- `POST {endpoint}/calling/recordings/{recordingId}:pause?api-version=2026-03-12`
- `POST {endpoint}/calling/recordings/{recordingId}:resume?api-version=2026-03-12`
- `DELETE {endpoint}/calling/recordings/{recordingId}?api-version=2026-03-12`

Recommended Stackyard route family:

- `/azure/communication-callautomation/2026-03-12/calling/...`

Important routing note:

- This API should be implemented as a data-plane style Azure route family, not merged into the ARM provider router.
- Every route still requires `api-version=2026-03-12`.
- The endpoint path itself should be stable inside Stackyard while preserving the upstream `calling/...` route envelope.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:

- authorization is documented as an ACS user access token in the `Authorization` header
- repeatability headers are supported on create and many mutation routes:
  - `Repeatability-Request-ID`
  - `Repeatability-First-Sent`
- errors use `Communication Error Response`
- many mutation routes are asynchronous in the sense that they return success immediately and rely on callback events for completion
- callback-related fields recur across the surface:
  - `callbackUri`
  - `operationCallbackUri`
  - `operationContext`

### Call Entry And Connection Bootstrap

Reviewed method pages:

- `Answer Call`
- `Create Call`
- `Connect`
- `Redirect Call`
- `Reject Call`

High-value route behavior from the reviewed pages:

- `Answer Call`
  - `POST /calling/callConnections:answer`
  - required body:
    - `incomingCallContext`
    - `callbackUri`
  - optional fields include:
    - `answeredBy`
    - `operationContext`
    - `mediaStreamingOptions`
    - `transcriptionOptions`
    - `callIntelligenceOptions`
  - success status: `200 OK`
- `Create Call`
  - `POST /calling/callConnections`
  - creates an outbound call
  - repeatability headers supported
  - success status is creation-style success with connection properties
- `Connect`
  - `POST /calling/callConnections:connect`
  - creates a connection to a `CallLocator`
- `Redirect Call`
  - `POST /calling/callConnections:redirect`
- `Reject Call`
  - `POST /calling/callConnections:reject`
  - request includes:
    - `incomingCallContext`
    - optional `callRejectReason`
  - reviewed enum values:
    - `none`
    - `busy`
    - `forbidden`
  - success status: `204 No Content`

Modeling implications:

- implement a deterministic call store keyed by `callConnectionId`
- answer/create/connect should all return normalized call-connection properties and seed call state
- redirect and reject can remain lighter-weight request handlers that mutate inbound-call state and return no-content success or a simple accepted result
- repeatability support should at minimum deduplicate identical create/bootstrap mutations when both repeatability headers are supplied

### Call Connection And Participants

Reviewed method pages:

- `Add Participant`
- `Cancel Add Participant`
- `Get Call`
- `Get Participant`
- `Get Participants`
- `Hangup Call`
- `Mute`
- `Remove Participant`
- `Transfer To Participant`

Operation-group-only or fallback-confirmed items:

- `Terminate Call`
- `Move Participants`

High-value contract points from the reviewed pages:

- `Get Call`
  - `GET /calling/callConnections/{callConnectionId}`
  - returns call connection properties
- `Add Participant`
  - `POST /calling/callConnections/{callConnectionId}/participants:add`
  - required body:
    - `participantToAdd`
  - optional fields include:
    - `operationContext`
    - `operationCallbackUri`
    - `invitationTimeoutInSeconds`
    - `sourceCallerIdNumber`
    - `sourceDisplayName`
    - `customCallingContext`
  - success status: `202 Accepted`
  - response includes:
    - `invitationId`
    - `participant`
    - `operationContext`
- `Cancel Add Participant`
  - `POST /participants:cancelAddParticipant`
- `Get Participant`
  - `GET /participants/{participantRawId}`
- `Get Participants`
  - `GET /participants`
  - recording and transcription bots are omitted from this list
- `Hangup Call`
  - `DELETE /calling/callConnections/{callConnectionId}`
  - success status: `204 No Content`
- `Mute`
  - mutation route over participant identifiers
- `Remove Participant`
  - `POST /participants:remove`
- `Transfer To Participant`
  - mutation route that hands the call to another participant

Modeling implications:

- persist participants separately from the top-level call object
- keep `participantRawId` as a stable lookup key
- model participant flags at least for:
  - `isMuted`
  - `isOnHold`
- keep `invitationId` and an operation ledger for add/cancel semantics
- treat `Terminate Call` and `Move Participants` as later additions unless the exact latest method page is revalidated during implementation

### Call Media

Reviewed method pages:

- `Cancel All Media Operations`
- `Hold`
- `Play`
- `Recognize`
- `Send Dtmf Tones`
- `Start Continuous Dtmf Recognition`
- `Start Media Streaming`
- `Start Transcription`
- `Stop Continuous Dtmf Recognition`
- `Stop Media Streaming`
- `Stop Transcription`
- `Unhold`
- `Update Transcription`

High-value contract points from the reviewed pages:

- `Cancel All Media Operations`
  - `POST /calling/callConnections/{callConnectionId}:cancelAllMediaOperations`
  - success status: `202 Accepted`
- `Hold`
  - requires `targetParticipant`
- `Play`
  - requires `playSources[]`
  - optional:
    - `playTo[]`
    - `playOptions.loop`
    - `interruptCallMediaOperation`
    - `operationContext`
    - `operationCallbackUri`
  - success status: `202 Accepted`
  - reviewed play source kinds include:
    - `file`
    - `text`
    - `ssml`
- `Recognize`
  - media recognition action with a richer polymorphic body
- `Send Dtmf Tones`
  - DTMF mutation route
- `Start Continuous Dtmf Recognition`
  - subscribes to tones for a participant
- `Start Media Streaming`
  - turns streaming on for a call
- `Start Transcription`
  - starts transcription
- `Stop Continuous Dtmf Recognition`
  - corresponding stop route
- `Stop Media Streaming`
  - corresponding stop route
- `Stop Transcription`
  - corresponding stop route
- `Unhold`
  - requires `targetParticipant`
- `Update Transcription`
  - update route for live transcription options

Modeling implications:

- maintain a per-call media state object with flags such as:
  - `activePlayOperations`
  - `continuousDtmfEnabled`
  - `mediaStreamingEnabled`
  - `transcriptionEnabled`
- media routes can be deterministic state toggles on first implementation; they do not need full media processing
- use an operation log keyed by `operationContext` for later test assertions and stable fixtures

### Call Recording

Reviewed method pages:

- `Start Recording`
- `Get Recording Properties`
- `Pause Recording`
- `Resume Recording`
- `Stop Recording`

High-value contract points from the reviewed pages:

- `Start Recording`
  - `POST /calling/recordings`
  - repeatability headers supported
  - success status: `202 Accepted`
- `Get Recording Properties`
  - `GET /calling/recordings/{recordingId}`
  - success status: `200 OK`
- `Pause Recording`
  - `POST /calling/recordings/{recordingId}:pause`
  - success status: `202 Accepted`
- `Resume Recording`
  - `POST /calling/recordings/{recordingId}:resume`
  - success status: `202 Accepted`
- `Stop Recording`
  - `DELETE /calling/recordings/{recordingId}`
  - success status: `204 No Content`

Modeling implications:

- maintain a separate recording store keyed by `recordingId`
- store recording state with values like:
  - `active`
  - `paused`
  - `stopped`
- keep lightweight linkage from recording back to `callConnectionId`

## State Model

Recommended in-memory state buckets:

- inbound-call contexts
- active call connections
- participants per call
- per-call media state
- per-call action history
- recordings

Recommended minimal fixture shape:

- call:
  - `callConnectionId`
  - `serverCallId`
  - `callbackUri`
  - `state`
  - `operationContext`
  - `createdDateTime`
- participant:
  - `rawId`
  - `identifier`
  - `isMuted`
  - `isOnHold`
- media state:
  - `continuousDtmfEnabled`
  - `mediaStreamingEnabled`
  - `transcriptionEnabled`
  - `lastPlayOperation`
- recording:
  - `recordingId`
  - `callConnectionId`
  - `recordingState`
  - `createdDateTime`

## Validation Priorities

High-value request validation from the reviewed pages:

- require `api-version=2026-03-12`
- require non-empty `callConnectionId` and `recordingId` where applicable
- for `Answer Call`:
  - require `incomingCallContext`
  - require `callbackUri`
- for `Reject Call`:
  - require `incomingCallContext`
- for `Add Participant`:
  - require `participantToAdd`
- for `Hold` and `Unhold`:
  - require `targetParticipant`
- for `Play`:
  - require non-empty `playSources`
- validate repeatability headers as a pair when present
- reject malformed JSON with the documented communication error envelope

High-value response validation:

- return `200` for call/participant/recording reads
- return `202` for accepted asynchronous mutations where documented
- return `204` for no-content routes like `Reject Call`, `Hangup Call`, and `Stop Recording`
- missing resources should return a consistent communication error envelope rather than generic `NotImplemented`

## Staged Implementation Plan

### Stage 1: Route Skeleton And Shared Data-Plane Primitives

Add a dedicated Azure data-plane router for:

- `api-version` validation
- endpoint-rooted `/calling/...` route parsing
- communication-style error responses
- repeatability handling helpers
- shared call and recording stores

Files:

- `internal/server/provider_azure_communication_callautomation.go`
- `internal/server/http.go`

Exit criteria:

- the covered route families dispatch correctly
- no reviewed route falls through to `NotImplemented`

### Stage 2: Call Entry Routes

Implement:

- `Answer Call`
- `Create Call`
- `Connect`
- `Redirect Call`
- `Reject Call`

Behavior goals:

- deterministic call-connection creation
- repeatability support for create-style mutations
- stable call identifiers and callback URIs in fixtures

Why this stage first:

- these routes seed the call state that the rest of the surface depends on

### Stage 3: Call Connection And Participants

Implement:

- `Get Call`
- `Get Participant`
- `Get Participants`
- `Add Participant`
- `Cancel Add Participant`
- `Remove Participant`
- `Mute`
- `Hangup Call`
- `Transfer To Participant`

Behavior goals:

- stable participant roster management
- invitation-operation ledger for add/cancel flows
- lifecycle flags for call state and participant state

Deferred tail for later reconciliation:

- `Terminate Call`
- `Move Participants`

Reason for defer:

- the operation-group page advertises them for `2026-03-12`, but the exact latest method pages were not fully retrievable during review

### Stage 4: Call Media

Implement:

- `Cancel All Media Operations`
- `Hold`
- `Play`
- `Recognize`
- `Send Dtmf Tones`
- `Start Continuous Dtmf Recognition`
- `Start Media Streaming`
- `Start Transcription`
- `Stop Continuous Dtmf Recognition`
- `Stop Media Streaming`
- `Stop Transcription`
- `Unhold`
- `Update Transcription`

Behavior goals:

- deterministic media-state toggles
- accepted responses with operation context echoes where appropriate
- enough payload normalization to exercise the documented polymorphic shapes

### Stage 5: Call Recording

Implement:

- `Start Recording`
- `Get Recording Properties`
- `Pause Recording`
- `Resume Recording`
- `Stop Recording`

Behavior goals:

- recording store keyed by `recordingId`
- stable recording state transitions
- simple linkage to the originating call

### Stage 6: Tests

Add focused provider tests covering:

- answer/create/connect/reject happy paths
- call get plus participant add/list/get/remove
- media actions for play, hold/unhold, DTMF, streaming, and transcription toggles
- recording start/get/pause/resume/stop
- invalid API version
- malformed JSON
- missing required fields
- missing call or recording IDs

Files:

- `internal/server/provider_azure_communication_callautomation_test.go`

### Stage 7: Azure Go SDK Example

Add an example under:

- `examples/azure/communication-callautomation-2026-03-12`

Recommended example flow:

1. answer or create a call
2. add a participant
3. get the call and participants
4. play audio and start transcription
5. start, inspect, and stop a recording

Recommended implementation approach:

- use `azcore/runtime` and `azcore/policy`
- use Stackyard’s local Azure shared-key or API-key conventions for local auth shimming

Support files:

- `main.go`
- `Dockerfile`
- `docker-compose.yml`

### Stage 8: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Required coverage updates:

- add the canonical service key `communication_callautomation`
- add the aliases listed above
- map the service to `communication-callautomation-2026-03-12-plan.md`

## Coverage Expectations

When implementation is complete, the following should pass:

- `go test ./internal/server -run 'TestAzureCommunicationCallAutomation' -count=1`
- `go build ./examples/azure/communication-callautomation-2026-03-12`
- `python3 scripts/azure-contract-coverage.py --service communication_callautomation --require-service communication_callautomation --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_callautomation --require-service communication_callautomation --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_callautomation --require-service communication_callautomation --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_callautomation --require-service communication_callautomation --fail-on any`

Success means:

- every reviewed Call Automation endpoint in the implemented stages returns a concrete success or contract error
- no reviewed route falls through to `NotImplemented`
- the coverage scripts recognize the new service and map it to this plan doc
