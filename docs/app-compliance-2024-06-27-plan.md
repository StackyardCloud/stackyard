# Azure App Compliance (`app-compliance-2024-06-27`) Staged Plan

## Objective

Emulate Azure App Compliance management-plane APIs (`2024-06-27`) with deterministic local behavior for reports, evidences, scoping configurations, snapshots, webhooks, and provider actions.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/operation-groups?view=rest-appcompliance-2024-06-27`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/evidence?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/operations?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/provider-actions?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/report?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/scoping-configuration?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/snapshot?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/webhook?view=rest-appcompliance-2024-06-27`

Representative method-page references:
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/operations/list?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/provider-actions/check-name-availability?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/provider-actions/trigger-evaluation?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/report/create-or-update?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/report/update?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/report/verify?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/evidence/create-or-update?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/evidence/download?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/scoping-configuration/create-or-update?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/snapshot/download?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/webhook/create-or-update?view=rest-appcompliance-2024-06-27`
- `https://learn.microsoft.com/en-us/rest/api/appcompliance/webhook/update?view=rest-appcompliance-2024-06-27`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate backend report-evaluation internals in this stage.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.AppComplianceAutomation/*`

Target API version:
- `api-version=2024-06-27`

## API Surface and Contract Notes

Operation groups reviewed:
- 7 operation-group pages linked from the operation-groups root page.

Method pages reviewed:
- 34 method pages linked from the operation-group pages.
- Crawl output captured at `/tmp/app_compliance_2024_06_27_methods.json`.

Observed contract characteristics:
- Route family anchored at `providers/Microsoft.AppComplianceAutomation`.
- Methods span `GET`, `POST`, `PUT`, `PATCH`, and `DELETE`.
- Documented success statuses include `200`, `201`, and `202` depending on operation semantics.
- Non-success responses are documented with typed ARM error payloads.

## Stage 0: Contract Skeleton

- Add dedicated App Compliance router surface.
- Recognize `providers/Microsoft.AppComplianceAutomation` route ownership.
- Add route-recognition tests across all operation groups.

Exit criteria:
- Route ownership is locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating from observed method set.
- Return deterministic invalid-request payloads for malformed `api-version`.

Exit criteria:
- Invalid query and unsupported method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized App Compliance routes.
- Preserve standard Azure staged response shape (`provider`, `path`, `status`).

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/app-compliance-2024-06-27`.
- Exercise representative workflows for operations, provider actions, report lifecycle, evidence/scoping/snapshot/webhook subresources, and deletion paths.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add App Compliance aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for canonical service key `app_compliance`.

Exit criteria:
- Coverage tooling resolves `app_compliance` and versioned aliases consistently.
