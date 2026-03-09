# Azure AI Bot Service - Direct Line (`bot-framework-direct-line-v1.1`) Staged Plan

## Objective

Emulate the Azure AI Bot Service Direct Line v1.1 API in Stackyard with deterministic local behavior for token, conversation, and message workflows.

Primary reference:
- `https://github.com/microsoft/botframework-sdk/blob/main/specs/botframework-protocol/directline-1.1.json`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement WebSocket streaming in this phase.

## Stackyard Route Envelope

Stackyard will expose Direct Line v1.1 routes under:
- `/azure/directline/v1.1/api/...`

This keeps Direct Line v1.1 routing explicit and separate from Connector routes under `/azure/botframework/v3/*` and Direct Line v3.0 routes under `/azure/directline/v3/directline/*`.

## API Surface (v1.1 Swagger)

Documented v1.1 operations to stage:

- `POST /api/conversations`
- `GET /api/conversations/{conversationId}/messages`
- `POST /api/conversations/{conversationId}/messages`
- `POST /api/conversations/{conversationId}/upload`
- `GET /api/tokens`
- `GET /api/tokens/{conversationId}/renew`
- `POST /api/tokens/conversation`

## Stage 0: Router Contract Skeleton

- Add Azure Direct Line v1.1 router surface in a dedicated Azure server file.
- Recognize documented route patterns and methods.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented operations.
- Add route recognition tests.

Exit criteria:
- Direct Line v1.1 route envelope is contract-locked in tests.

## Stage 1: Token and Conversation Bootstrap

- Implement `GET /api/tokens` and `POST /api/tokens/conversation`.
- Implement `GET /api/tokens/{conversationId}/renew`.
- Implement `POST /api/conversations`.
- Return stable token and conversation envelopes for local SDK workflow seeding.

Exit criteria:
- Token generation/renew and conversation start are deterministic and test-covered.

## Stage 2: Message Send and Read

- Implement `POST /api/conversations/{conversationId}/messages`.
- Implement `GET /api/conversations/{conversationId}/messages` with deterministic watermark semantics.
- Add validation and explicit not-found behavior for unknown conversations.

Exit criteria:
- Message send/read flow works with stable response envelopes and pagination signals.

## Stage 3: Upload Support and Hardening

- Implement `POST /api/conversations/{conversationId}/upload`.
- Add deterministic metadata envelopes for upload responses.
- Add validation for malformed payloads and missing conversation IDs.

Exit criteria:
- Remaining v1.1 operations are available with deterministic contract behavior.

## Stage 4: Example, Coverage, and Docs Integration

- Add `examples/azure/ai-bot-service/bot-framework-direct-line-v1.1` using Azure Go SDK `azcore` pipeline primitives.
- Update Azure coverage script aliases for `bot-framework-direct-line-v1.1` naming variants.
- Add service entry to Azure docs catalog and overview/services pages.

Exit criteria:
- New service is discoverable in docs and addressable by coverage scripts.

## Stage 5: CI and Regression Safety

- Add negative-path tests for token and conversation preconditions.
- Validate existing strict Azure gate list remains stable.
- Keep Direct Line v1.1 out of strict gate list until parity and test depth are complete.

Exit criteria:
- No regressions for existing providers; Direct Line v1.1 scaffold is stable for incremental expansion.
