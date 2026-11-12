# Azure AI Foundry - Model Inference (`model-inference-2024-05-01-preview`) Staged Plan

## Objective

Emulate Azure AI Foundry Model Inference (`2024-05-01-preview`) with deterministic local behavior for chat completions, text embeddings, image embeddings, and model metadata inference endpoints.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/operation-groups?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-chat-completions?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-embeddings?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-image-embeddings?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-model-info?view=rest-aifoundry-model-inference-2024-05-01-preview`

Method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-chat-completions/get-chat-completions?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-embeddings/get-embeddings?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-image-embeddings/get-image-embeddings?view=rest-aifoundry-model-inference-2024-05-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aifoundry/model-inference/get-model-info/get-model-info?view=rest-aifoundry-model-inference-2024-05-01-preview`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full model-runtime semantics, safety filters, or deterministic token generation in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/ai-foundry/model-inference/models/chat/completions`
- `/azure/ai-foundry/model-inference/models/embeddings`
- `/azure/ai-foundry/model-inference/models/images/embeddings`
- `/azure/ai-foundry/model-inference/models/info`

Target API version:
- `api-version=2024-05-01-preview`

## API Surface and Contract Notes

Operation groups reviewed:
- Get Chat Completions
- Get Embeddings
- Get Image Embeddings
- Get Model Info

Documented operations and routes:
- `POST /models/chat/completions`
- `POST /models/embeddings`
- `POST /models/images/embeddings`
- `GET /models/info`

Common contract characteristics from method pages:
- Success response is `200 OK`.
- Error envelope uses `Azure.Core.Foundations.ErrorResponse` for non-200 status classes.
- `api-version` is required query input and route-level request body contracts vary by operation (chat messages, text/image embedding inputs, optional model override).
- Auth supports API key and OAuth scopes for Cognitive Services.

## Stage 0: Contract Skeleton

- Add dedicated Azure AI Foundry Model Inference router surface.
- Recognize all documented model-inference route envelopes.
- Add route-recognition tests for each documented operation.

Exit criteria:
- Route ownership and deterministic staging are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method routing for operation endpoints.
- Return deterministic invalid-request payload for malformed API version input.

Exit criteria:
- Invalid query patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- All recognized routes return deterministic success fixtures under contract tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example in `examples/azure/ai-foundry/model-inference-2024-05-01-preview`.
- Exercise representative calls for all four operations in the service.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add service aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for the new service key.

Exit criteria:
- Coverage tooling resolves `ai_foundry_model_inference` and versioned aliases consistently.
