# Azure Bot Framework (`azure-bot-service-4.0`) Staged Plan

## Objective

Emulate the Azure Bot Framework Connector REST API in Stackyard with predictable local behavior for conversation and activity workflows.

Primary reference:
- `https://docs.azure.cn/en-us/bot-service/rest-api/bot-framework-rest-connector-api-reference?view=azure-bot-service-4.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not implement full Bot Framework channel adapters in this phase.
- Do not require live Azure resources for local examples.

## Stackyard Route Envelope

Stackyard will expose Connector routes under:
- `/azure/botframework/v3/...`

This keeps Azure service routing explicit and prevents collision with existing Blob routes under `/azure/{account}/...`.

## API Surface (Initial Target)

From the Bot Framework Connector reference, stage coverage targets:

- `POST /v3/conversations`
- `POST /v3/conversations/{conversationId}`
- `POST /v3/conversations/{conversationId}/activities`
- `POST /v3/conversations/{conversationId}/activities/{activityId}`
- `PUT /v3/conversations/{conversationId}/activities/{activityId}`
- `DELETE /v3/conversations/{conversationId}/activities/{activityId}`
- `GET /v3/conversations/{conversationId}/members`
- `GET /v3/conversations/{conversationId}/pagedmembers`
- `GET /v3/conversations/{conversationId}/activities/{activityId}/members`

## Stage 0: Router Contract Skeleton

- Add Azure Bot Framework router file.
- Recognize all target route patterns.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented routes.
- Add route-recognition tests.

Exit criteria:
- Connector route set is contract-locked in tests.

## Stage 1: Conversation Lifecycle

- Implement `POST /v3/conversations`.
- Implement `POST /v3/conversations/{conversationId}`.
- Parse request body and seed deterministic conversation state.
- Return stable response envelopes (`id`, `activityId`, `serviceUrl`).

Exit criteria:
- Conversation creation works with deterministic IDs.

## Stage 2: Activity Lifecycle

- Implement `POST /v3/conversations/{conversationId}/activities`.
- Implement `POST /v3/conversations/{conversationId}/activities/{activityId}`.
- Implement `PUT /v3/conversations/{conversationId}/activities/{activityId}`.
- Implement `DELETE /v3/conversations/{conversationId}/activities/{activityId}`.
- Add input validation and not-found responses.

Exit criteria:
- Activity create/reply/update/delete workflow is covered by tests.

## Stage 3: Membership Reads

- Implement `GET /v3/conversations/{conversationId}/members`.
- Implement `GET /v3/conversations/{conversationId}/pagedmembers`.
- Implement `GET /v3/conversations/{conversationId}/activities/{activityId}/members`.
- Add deterministic pagination token behavior for paged members.

Exit criteria:
- Member enumeration responses are stable and test-covered.

## Stage 4: Coverage, Example, and Docs

- Add `examples/azure/ai-bot-service/bot-service-4.0` example using Azure Go SDK primitives.
- Update `scripts/azure-contract-coverage.py` aliases for `azure-bot-service-4.0`.
- Update `scripts/azure-io-contract-coverage.py` aliases for `azure-bot-service-4.0`.
- Add catalog entry in `docs/web/assets/azure-catalog.js`.

Exit criteria:
- Service appears in Azure catalog and is selectable by coverage scripts.

## Stage 5: CI and Hardening

- Add additional negative-path tests (invalid JSON, missing members, missing activity IDs).
- Validate strict Azure gates remain green for existing strict services.
- Keep Bot Framework out of strict list until higher parity is complete.

Exit criteria:
- No regressions for existing providers; new service scaffold is stable.
