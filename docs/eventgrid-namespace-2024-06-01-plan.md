# Azure Event Grid Namespace (`eventgrid_namespace_2024_06_01`) Staged Plan

## Objective

Implement an emulated Azure Event Grid Namespace surface for API version `2024-06-01`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/eventgrid-namespace-2024-06-01`

This plan is based on full traversal of the Event Grid Namespace operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/operation-groups?view=rest-eventgrid-namespace-2024-06-01`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/acknowledge-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/publish-cloud-event?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/publish-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/receive-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/reject-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/release-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/renew-cloud-event-locks?view=rest-eventgrid-namespace-2024-06-01`

## Reviewed Methods

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/acknowledge-cloud-events/acknowledge-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/publish-cloud-event/publish-cloud-event?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/publish-cloud-events/publish-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/receive-cloud-events/receive-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/reject-cloud-events/reject-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/release-cloud-events/release-cloud-events?view=rest-eventgrid-namespace-2024-06-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/namespace/renew-cloud-event-locks/renew-cloud-event-locks?view=rest-eventgrid-namespace-2024-06-01`

Total reviewed methods: `7`.

## Service Naming

- service key: `eventgrid_namespace_2024_06_01`
- display name: `Azure Event Grid Namespace`
- API version: `2024-06-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_eventgrid_namespace_2024_06_01.go`
- tests: `internal/server/provider_azure_eventgrid_namespace_2024_06_01_test.go`
- example: `examples/azure/eventgrid-namespace-2024-06-01`
- plan doc: `docs/eventgrid-namespace-2024-06-01-plan.md`

## Route Envelope

Namespace methods are endpoint-template based in the spec:

- `POST {endpoint}/topics/{topicName}:publish?api-version=2024-06-01`
- `POST {endpoint}/topics/{topicName}:publish?_overload=publishCloudEvents&api-version=2024-06-01`
- `POST {endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:receive?api-version=2024-06-01`
- `POST {endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:acknowledge?api-version=2024-06-01`
- `POST {endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:reject?api-version=2024-06-01`
- `POST {endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:release?api-version=2024-06-01`
- `POST {endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:renewLock?api-version=2024-06-01`

For Stackyard emulator routing, use:

- `/azure/eventgrid-namespace/2024-06-01/{endpoint}/topics/...`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods are `POST`
- all reviewed methods require `api-version=2024-06-01`
- all reviewed methods return `200` on success
- all reviewed methods include `Other Status Codes` with Azure error response envelopes

Request-body coverage:

- body-bearing methods: `6 / 7`
- methods with request body:
  - `acknowledge-cloud-events/acknowledge-cloud-events`
  - `publish-cloud-event/publish-cloud-event`
  - `publish-cloud-events/publish-cloud-events`
  - `reject-cloud-events/reject-cloud-events`
  - `release-cloud-events/release-cloud-events`
  - `renew-cloud-event-locks/renew-cloud-event-locks`
- method without request body:
  - `receive-cloud-events/receive-cloud-events`

Operation-specific notes:

- `publish-cloud-event` uses media type `application/cloudevents+json; charset=utf-8`
- `publish-cloud-events` uses media type `application/cloudevents-batch+json; charset=utf-8` and `_overload=publishCloudEvents`
- `receive-cloud-events` supports optional query params `maxEvents` and `maxWaitTime`
- acknowledge/reject/release/renew-lock operations use request body with `lockTokens: string[]`

## SDK Reference

- Azure Go SDK pipeline/runtime used by examples:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_eventgrid_namespace_2024_06_01.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `eventgrid_namespace_2024_06_01 -> eventgrid-namespace-2024-06-01-plan.md`

### Stage 1: Request Envelope Validation

- validate:
  - API version query value
  - required path variables (`endpoint`, `topicName`, `eventSubscriptionName` where applicable)
  - method-specific query semantics (`_overload`, `maxEvents`, `maxWaitTime`)
- enforce body requirements and basic shape validation for lock-token and CloudEvent payloads

### Stage 2: Namespace Operation Semantics

- implement publish operations:
  - single CloudEvent publish
  - batch CloudEvents publish
- implement subscription operations:
  - receive
  - acknowledge
  - reject
  - release
  - renew lock
- return deterministic success envelopes and modeled error envelopes

### Stage 3: Stateful Behavior

- add in-memory store for:
  - published events per topic
  - receive queue simulation per event subscription
  - lock token lifecycle for acknowledge/reject/release/renew
- keep deterministic fixture behavior for contract testing

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_eventgrid_namespace_2024_06_01_test.go` covering:
  - successful flows for all 7 methods
  - invalid API version
  - invalid/missing payload shape
  - invalid overload
  - lock token failure scenarios
  - receive query bounds (`maxEvents`, `maxWaitTime`)

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `eventgrid_namespace_2024_06_01`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/eventgrid-namespace-2024-06-01/main.go` should invoke all reviewed operations:

- publish single CloudEvent
- publish batch CloudEvents
- receive CloudEvents
- acknowledge CloudEvents
- reject CloudEvents
- release CloudEvents
- renew CloudEvent locks
