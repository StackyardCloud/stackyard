# Azure Health Insights (`health-insights-2024-10-01`) Staged Plan

## Objective

Emulate Azure Health Insights REST API (`2024-10-01`) with deterministic local behavior for Radiology Insights job creation and retrieval workflows.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/healthinsights/operation-groups?view=rest-healthinsights-2024-10-01`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/healthinsights/radiology-insights?view=rest-healthinsights-2024-10-01`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/healthinsights/radiology-insights/create-job?view=rest-healthinsights-2024-10-01`
- `https://learn.microsoft.com/en-us/rest/api/healthinsights/radiology-insights/get-job?view=rest-healthinsights-2024-10-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full clinical/radiology inference semantics in this phase.

## Route Envelope

Documented endpoint route family:
- `{endpoint}/health-insights/radiology-insights/jobs/{id}`

Stackyard emulation prefix:
- `/azure/health-insights/radiology-insights/jobs/{id}`

Target API version:
- `api-version=2024-10-01` (query parameter)

## API Surface and Contract Notes

Operation group and documented operations reviewed:
- Radiology Insights
  - Create Job: `PUT /health-insights/radiology-insights/jobs/{id}?api-version=2024-10-01` with optional `expand` query parameter.
  - Get Job: `GET /health-insights/radiology-insights/jobs/{id}?api-version=2024-10-01` with optional `expand` query parameter.

Contract characteristics from method pages:
- `Create Job` accepts `jobData` (`RadiologyInsightsData`) in the request body.
- `Create Job` success responses include `200 OK` and `201 Created`, with `Operation-Location` and `x-ms-request-id` headers.
- `Get Job` success response includes `200 OK`, with `Retry-After` and `x-ms-request-id` headers.
- Error contract uses `HealthInsightsErrorResponse` with `x-ms-error-code` and `x-ms-request-id` headers.
- Security supports `Ocp-Apim-Subscription-Key` and Entra ID OAuth2 token flows.

## Stage 0: Contract Skeleton

- Add dedicated Azure Health Insights router surface.
- Recognize the documented route family under `/azure/health-insights/radiology-insights/jobs/*`.
- Add route-recognition tests for representative `PUT` and `GET` operations.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate top-level operation-group routing for `radiology-insights/jobs`.
- Preserve deterministic behavior for unknown nested routes under the Health Insights prefix.

Exit criteria:
- Invalid query forms return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized Health Insights routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and validated by provider tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/health-insights-2024-10-01`.
- Exercise representative `Create Job` and `Get Job` calls.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Health Insights API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `health-insights-2024-10-01` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and include it in coverage reporting.
