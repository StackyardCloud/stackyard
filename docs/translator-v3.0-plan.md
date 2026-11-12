# Azure Translator (`translator-v3.0`) Staged Plan

## Objective

Emulate Azure Translator Text REST API (`v3.0`) with deterministic local behavior for translate, detect, sentence-break, dictionary lookup/examples, language metadata, and transliteration flows.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/translator/operation-groups?view=rest-translator-v3.0`

Operation-group reference:
- `https://learn.microsoft.com/en-us/rest/api/translator/translator?view=rest-translator-v3.0`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/translate?view=rest-translator-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/detect?view=rest-translator-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/break-sentence?view=rest-translator-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/dictionary-lookup?view=rest-translator-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/dictionary-examples?view=rest-translator-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/languages?view=rest-translator-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/translator/translator/transliterate?view=rest-translator-v3.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate production-grade language model outputs in this phase.

## Route Envelope

Documented endpoint route families:
- `{endpoint}/translate?api-version=3.0`
- `{endpoint}/detect?api-version=3.0`
- `{endpoint}/breaksentence?api-version=3.0`
- `{endpoint}/dictionary/lookup?api-version=3.0&from={from}&to={to}`
- `{endpoint}/dictionary/examples?api-version=3.0&from={from}&to={to}`
- `{endpoint}/languages?api-version=3.0`
- `{endpoint}/transliterate?api-version=3.0&language={language}&fromScript={fromScript}&toScript={toScript}`

Stackyard emulation prefix:
- `/azure/translator/*`

## API Surface and Contract Notes

Operations reviewed:
- Translate
- Detect
- Break sentence
- Dictionary lookup
- Dictionary examples
- Languages
- Transliterate

Contract characteristics from method pages:
- Required query parameter: `api-version=3.0`.
- Required request header: `Ocp-Apim-Subscription-Key`.
- Optional/conditional request header in Azure deployments: `Ocp-Apim-Subscription-Region`.
- JSON request media type for POST methods: `application/json; charset=UTF-8`.
- Success response patterns:
  - `200 OK` across Translator v3.0 operations.
- Error contract:
  - non-success responses include an error payload shape documented as `Error Response`.

## Stage 0: Contract Skeleton

- Add dedicated Azure Translator router surface under `/azure/translator/*`.
- Recognize documented method paths and HTTP methods.
- Add route-recognition tests for all operations.

Exit criteria:
- Route ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate required query parameters for:
  - dictionary lookup/examples: `from`, `to`
  - transliterate: `language`, `fromScript`, `toScript`
- Preserve deterministic staged behavior for unknown nested routes.

Exit criteria:
- Invalid query forms return deterministic `InvalidRequest` contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized Translator routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes return deterministic fixtures and pass provider contract tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/translator-v3.0`.
- Exercise all Translator operations with representative request bodies and query parameters.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Translator API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `translator-v3.0` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and include it in coverage reporting.
