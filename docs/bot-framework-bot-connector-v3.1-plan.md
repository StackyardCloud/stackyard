# Azure AI Bot Service - Bot Connector (`bot-framework-bot-connector-v3.1`) Staged Plan

## Objective

Emulate the Azure AI Bot Service Bot Connector v3.1 API surface in Stackyard with deterministic local behavior for bot conversation and activity workflows.

Primary reference:
- `https://raw.githubusercontent.com/microsoft/botframework-sdk/refs/heads/main/specs/botframework-protocol/botframework-channel.json`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement channel adapters, OAuth token exchange flows, or WebSocket transport in this phase.

## Stackyard Route Envelope

Stackyard will continue to expose Bot Connector routes under:
- `/azure/botframework/v3/...`

The `bot-framework-bot-connector-v3.1` service maps to the existing Azure Bot Framework connector route family so existing clients remain compatible.

## API Surface (v3.1 Swagger)

Documented v3.1 operations to stage:

- `GET /v3/attachments/{attachmentId}`
- `GET /v3/attachments/{attachmentId}/views/{viewId}`
- `GET /v3/conversations`
- `POST /v3/conversations`
- `POST /v3/conversations/{conversationId}/activities`
- `POST /v3/conversations/{conversationId}/activities/history`
- `PUT /v3/conversations/{conversationId}/activities/{activityId}`
- `POST /v3/conversations/{conversationId}/activities/{activityId}`
- `DELETE /v3/conversations/{conversationId}/activities/{activityId}`
- `GET /v3/conversations/{conversationId}/members`
- `GET /v3/conversations/{conversationId}/members/{memberId}`
- `DELETE /v3/conversations/{conversationId}/members/{memberId}`
- `GET /v3/conversations/{conversationId}/pagedmembers`
- `GET /v3/conversations/{conversationId}/activities/{activityId}/members`
- `POST /v3/conversations/{conversationId}/attachments`

## Stage 0: Compatibility Mapping and Contract Skeleton

- Keep `provider_azure_botframework.go` as the canonical implementation target.
- Add a service alias mapping for `bot-framework-bot-connector-v3.1` in Azure coverage tooling.
- Keep recognized-but-unimplemented routes deterministic (`501 NotImplemented`) until each stage lands.

Exit criteria:
- The new service name resolves to the existing bot connector implementation in coverage and docs.

## Stage 1: Conversation and Activity Baseline

- Ensure stable support for:
  - `POST /v3/conversations`
  - `POST /v3/conversations/{conversationId}/activities`
  - `POST /v3/conversations/{conversationId}/activities/{activityId}`
  - `PUT /v3/conversations/{conversationId}/activities/{activityId}`
  - `DELETE /v3/conversations/{conversationId}/activities/{activityId}`
- Keep deterministic IDs and response shapes for local test repeatability.

Exit criteria:
- End-to-end conversation/activity lifecycle passes integration tests and example flow.

## Stage 2: Membership and Paging

- Ensure stable support for:
  - `GET /v3/conversations/{conversationId}/members`
  - `GET /v3/conversations/{conversationId}/pagedmembers`
  - `GET /v3/conversations/{conversationId}/activities/{activityId}/members`
- Preserve deterministic pagination token behavior.

Exit criteria:
- Member list and paged-member flows are repeatable and test-covered.

## Stage 3: Swagger Gap-Closure Backlog

- Add staged implementation for currently uncovered v3.1 operations:
  - `GET /v3/conversations`
  - `POST /v3/conversations/{conversationId}/activities/history`
  - `GET /v3/conversations/{conversationId}/members/{memberId}`
  - `DELETE /v3/conversations/{conversationId}/members/{memberId}`
  - Attachment download/upload operations.
- Add deterministic fixture shaping and negative-path validation for each new operation.

Exit criteria:
- Remaining Swagger operations have explicit implementation or tracked staged backlog with tests.

## Stage 4: Example and Docs Integration

- Add example at `examples/azure/ai-bot-service/bot-framework-bot-connector-v3.1` using Azure Go SDK `azcore` pipeline primitives.
- Add service entry to Azure docs catalog and overview pages.
- Wire docs links to this staged plan and runnable example compose file.

Exit criteria:
- New service appears in docs web pages and has a runnable local example.

## Stage 5: Coverage and CI Gates

- Update Azure coverage script aliases to accept `bot-framework-bot-connector-v3.1`.
- Validate both coverage scripts can target the new service name and resolve strict gates through the bot connector implementation.
- Keep existing strict Azure service list unchanged unless explicitly expanded.

Exit criteria:
- Coverage scripts pass for the new service name without regressions to existing providers.
