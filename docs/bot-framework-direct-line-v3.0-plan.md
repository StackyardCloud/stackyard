# Azure AI Bot Service - Direct Line (`bot-framework-direct-line-v3.0`) Staged Plan

## Objective

Emulate the Azure AI Bot Service Direct Line v3.0 API in Stackyard with deterministic local behavior for token, conversation, and activity workflows.

Primary reference:
- `https://raw.githubusercontent.com/microsoft/botframework-sdk/refs/heads/main/specs/botframework-protocol/directline-3.0.json`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement WebSocket streaming in this phase.

## Stackyard Route Envelope

Stackyard will expose Direct Line routes under:
- `/azure/directline/v3/directline/...`

This keeps Direct Line service routing explicit and avoids collision with `/azure/botframework/v3/*` connector routes.

## API Surface (v3.0 Swagger)

Documented v3.0 operations to stage:

- `GET /v3/directline/session/getsessionid`
- `POST /v3/directline/conversations`
- `GET /v3/directline/conversations/{conversationId}`
- `GET /v3/directline/conversations/{conversationId}/activities`
- `POST /v3/directline/conversations/{conversationId}/activities`
- `POST /v3/directline/conversations/{conversationId}/upload`
- `POST /v3/directline/tokens/refresh`
- `POST /v3/directline/tokens/generate`

## Stage 0: Router Contract Skeleton

- Add Azure Direct Line router surface in a dedicated Azure server file.
- Recognize documented route patterns and methods.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented operations.
- Add route recognition tests.

Exit criteria:
- Direct Line route envelope is contract-locked in tests.

## Stage 1: Token and Conversation Bootstrap

- Implement `POST /v3/directline/tokens/generate`.
- Implement `POST /v3/directline/tokens/refresh`.
- Implement `POST /v3/directline/conversations`.
- Return stable token and conversation envelopes for local SDK workflow seeding.

Exit criteria:
- Token generation/refresh and conversation start are deterministic and test-covered.

## Stage 2: Activity and Reconnect Reads

- Implement `POST /v3/directline/conversations/{conversationId}/activities`.
- Implement `GET /v3/directline/conversations/{conversationId}/activities` with deterministic watermark behavior.
- Implement `GET /v3/directline/conversations/{conversationId}` for reconnect semantics.

Exit criteria:
- Send/read/reconnect flow works with stable pagination/watermark semantics.

## Stage 3: Session and Upload Support

- Implement `GET /v3/directline/session/getsessionid`.
- Implement `POST /v3/directline/conversations/{conversationId}/upload`.
- Add validation and explicit error envelopes for invalid payloads and unknown conversations.

Exit criteria:
- Remaining v3.0 operations are available with deterministic contract behavior.

## Stage 4: Example, Coverage, and Docs Integration

- Add `examples/azure/ai-bot-service/bot-framework-direct-line-v3.0` using Azure Go SDK `azcore` pipeline primitives.
- Update Azure coverage script aliases for `bot-framework-direct-line-v3.0` naming variants.
- Add service entry to Azure docs catalog and overview/services pages.

Exit criteria:
- New service is discoverable in docs and addressable by coverage scripts.

## Stage 5: Hardening and CI Gates

- Add negative-path tests for missing tokens, invalid conversation IDs, and malformed payloads.
- Validate existing strict Azure gate list remains stable.
- Keep Direct Line out of strict gate list until parity and test depth are complete.

Exit criteria:
- No regressions for existing services; Direct Line scaffold is stable for incremental expansion.
