# Azure Event Grid Data Plane (`eventgrid_dataplane_2018_01_01`) Staged Plan

## Objective

Implement an emulated Azure Event Grid Data Plane surface for API version `2018-01-01`, including:

- provider routing scaffold in `internal/server`
- coverage script service registration
- example client under `examples/azure/eventgrid-dataplane-2018-01-01`

This plan is based on full traversal of the Event Grid data-plane operation-group page and every linked operation page.

## Documentation Sources

- Operation groups root:
  - `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/operation-groups?view=rest-eventgrid-dataplane-2018-01-01`

## Operation Groups

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/publish-cloud-event-events?view=rest-eventgrid-dataplane-2018-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/publish-custom-event-events?view=rest-eventgrid-dataplane-2018-01-01`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/publish-event-grid-events?view=rest-eventgrid-dataplane-2018-01-01`

## Reviewed Methods

- Publish Cloud Event Events:
  - `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/publish-cloud-event-events/publish-cloud-event-events?view=rest-eventgrid-dataplane-2018-01-01`
- Publish Custom Event Events:
  - `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/publish-custom-event-events/publish-custom-event-events?view=rest-eventgrid-dataplane-2018-01-01`
- Publish Event Grid Events:
  - `https://learn.microsoft.com/en-us/rest/api/eventgrid/dataplane/publish-event-grid-events/publish-event-grid-events?view=rest-eventgrid-dataplane-2018-01-01`

Total reviewed methods: `3`.

## Service Naming

- service key: `eventgrid_dataplane_2018_01_01`
- display name: `Azure Event Grid Data Plane`
- API version: `2018-01-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_eventgrid_dataplane_2018_01_01.go`
- tests: `internal/server/provider_azure_eventgrid_dataplane_2018_01_01_test.go`
- example: `examples/azure/eventgrid-dataplane-2018-01-01`
- plan doc: `docs/eventgrid-dataplane-2018-01-01-plan.md`

## Route Envelope

Event Grid data-plane operations are host-based in the official spec:

- `POST {topicHostname}?overload=cloudEvent&api-version=2018-01-01`
- `POST {topicHostname}?overload=customEvent&api-version=2018-01-01`
- `POST {topicHostname}?overload=EventGridEvent&api-version=2018-01-01`

For Stackyard emulator routing, use:

- `/azure/eventgrid-dataplane/2018-01-01/{topicHostname}?overload=...&api-version=2018-01-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods are `POST`
- all reviewed methods require `api-version=2018-01-01`
- all reviewed methods return `200` on success
- all reviewed methods document `Other Status Codes` including `400`, `500`, `429`, and `404`
- all reviewed methods require request body arrays

Method-specific payload notes:

- `publish-cloud-event-events/publish-cloud-event-events`:
  - CloudEvents batch payload
  - media type `application/cloudevents-batch+json; charset=utf-8`
  - optional `aeg-channel-name` request header for partner namespace routing mode
- `publish-custom-event-events/publish-custom-event-events`:
  - custom event payload array (`object[]`)
- `publish-event-grid-events/publish-event-grid-events`:
  - Event Grid schema payload array (`EventGridEvent[]`)

## SDK Reference

- Azure Go SDK base pipeline/runtime used by examples:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

## Staged Implementation Plan

### Stage 0: Skeleton + Registration

- add stage-0 provider skeleton at `internal/server/provider_azure_eventgrid_dataplane_2018_01_01.go`
- wire router in `internal/server/http.go`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map docs in `azure-doc-contract-coverage.py`:
  - `eventgrid_dataplane_2018_01_01 -> eventgrid-dataplane-2018-01-01-plan.md`

### Stage 1: Request Envelope Validation

- validate:
  - API version query value
  - `overload` query value (`cloudEvent`, `customEvent`, `EventGridEvent`)
  - payload is a non-empty JSON array
- apply media-type rule for cloud-event overload
- normalize error envelope/shape for invalid requests

### Stage 2: Publish Endpoint Behavior

- implement `POST` publish endpoint logic keyed by overload
- return deterministic acceptance fixture with `200`
- return modeled errors for contract cases (`400/404/429/500`)

### Stage 3: Storage + Deterministic Fixtures

- persist published batches in provider store for traceability and debug assertions
- include schema-specific fixture fields that mirror docs terminology

### Stage 4: Tests

- add provider tests `internal/server/provider_azure_eventgrid_dataplane_2018_01_01_test.go` covering:
  - success for all three overloads
  - invalid api-version
  - invalid overload
  - invalid body type/content
  - overload-specific content-type checks
  - representative error paths

### Stage 5: Coverage Gate Tightening

- ensure service key resolves as `eventgrid_dataplane_2018_01_01`
- incrementally satisfy:
  - request/input validation signals
  - output fixture signals
  - negative contract test signals

## Example Scope

Example `examples/azure/eventgrid-dataplane-2018-01-01/main.go` should invoke:

- publish Event Grid schema batch
- publish CloudEvents batch
- publish custom event batch

with shared-key style headers consistent with other Stackyard Azure examples.
