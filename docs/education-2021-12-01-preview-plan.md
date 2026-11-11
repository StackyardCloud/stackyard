# Azure Education (`education_2021_12_01_preview`) Staged Plan

## Objective

Emulate Azure Education Resource Manager REST API (`api-version=2021-12-01-preview`) with deterministic local behavior for:

- grants read/list surfaces at billing and provider scopes
- lab lifecycle and invite-code generation flows
- join request read/list and approve/deny actions
- student lifecycle and list surfaces under labs
- student-lab read/list surfaces for caller context
- redeem invitation code action
- provider operations discovery endpoint
- an Azure Go SDK example under `examples/azure/education-2021-12-01-preview`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/education/operation-groups?view=rest-education-2021-12-01-preview`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/education/grants?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/join-requests?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/labs?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/operations?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/redeem-invitation-code?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/student-labs?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/students?view=rest-education-2021-12-01-preview`

Method pages reviewed directly: all linked methods from all operation groups (`21` total).

Grants methods (`3`):

- `https://learn.microsoft.com/en-us/rest/api/education/grants/get?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/grants/list?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/grants/list-all?view=rest-education-2021-12-01-preview`

Join Requests methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/education/join-requests/approve?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/join-requests/deny?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/join-requests/get?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/join-requests/list?view=rest-education-2021-12-01-preview`

Labs methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/education/labs/create-or-update?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/labs/delete?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/labs/generate-invite-code?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/labs/get?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/labs/list?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/labs/list-all?view=rest-education-2021-12-01-preview`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/education/operations/list?view=rest-education-2021-12-01-preview`

Redeem Invitation Code methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/education/redeem-invitation-code/redeem-invitation-code?view=rest-education-2021-12-01-preview`

Student Labs methods (`2`):

- `https://learn.microsoft.com/en-us/rest/api/education/student-labs/get?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/student-labs/list-all?view=rest-education-2021-12-01-preview`

Students methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/education/students/create-or-update?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/students/delete?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/students/get?view=rest-education-2021-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/education/students/list?view=rest-education-2021-12-01-preview`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/education/armeducation`

## Surface Summary

Reviewed operation groups and methods:

- Grants (3)
- Join Requests (4)
- Labs (6)
- Operations (1)
- Redeem Invitation Code (1)
- Student Labs (2)
- Students (4)

Total reviewed methods: `21`.

## Service Naming

Canonical Stackyard naming:

- service key: `education_2021_12_01_preview`
- display name: `Azure Education`
- API version: `2021-12-01-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_education_2021_12_01_preview.go`
- tests: `internal/server/provider_azure_education_2021_12_01_preview_test.go`
- example: `examples/azure/education-2021-12-01-preview`
- plan doc: `docs/education-2021-12-01-preview-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Education/operations`
- `/azure/providers/Microsoft.Education/grants`
- `/azure/providers/Microsoft.Education/redeemInvitationCode`
- `/azure/providers/Microsoft.Education/studentLabs`
- `/azure/providers/Microsoft.Education/studentLabs/{studentLabName}`
- `/azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/providers/Microsoft.Education/grants`
- `/azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/providers/Microsoft.Education/labs`
- `/azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default`
- `/azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/generateInviteCode`
- `/azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/joinRequests`
- `/azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/students`

Representative routes:

- `GET /azure/providers/Microsoft.Education/operations?api-version=2021-12-01-preview`
- `GET /azure/providers/Microsoft.Education/grants?api-version=2021-12-01-preview`
- `GET /azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/providers/Microsoft.Education/grants/default?api-version=2021-12-01-preview`
- `PUT /azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default?api-version=2021-12-01-preview`
- `POST /azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/generateInviteCode?api-version=2021-12-01-preview`
- `POST /azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/joinRequests/{joinRequestName}/approve?api-version=2021-12-01-preview`
- `POST /azure/providers/Microsoft.Education/redeemInvitationCode?api-version=2021-12-01-preview`
- `PUT /azure/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/students/{studentAlias}?api-version=2021-12-01-preview`
- `GET /azure/providers/Microsoft.Education/studentLabs/{studentLabName}?api-version=2021-12-01-preview`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2021-12-01-preview`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods document non-success paths using `ErrorResponseBody`

Observed response patterns:

- `200` appears in `21` methods
- `201` appears in `2` methods
- `204` appears in `2` methods

Request-body coverage from reviewed pages:

- `4` methods include request bodies
- body-bearing methods:
  - `Labs_CreateOrUpdate`
  - `Labs_GenerateInviteCode`
  - `RedeemInvitationCode_RedeemInvitationCode`
  - `Students_CreateOrUpdate`

Notable optional query parameters:

- `includeAllocatedBudget` on `Grants_Get`
- `includeBudget` on `Labs_Get`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add stage-0 provider skeleton at `internal/server/provider_azure_education_2021_12_01_preview.go`
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key resolves as `education_2021_12_01_preview`
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.Education` route families (including billing-scoped routes)
- enforce `api-version=2021-12-01-preview` on owned routes
- preserve behavior for unrelated Azure providers

### Stage 2: Provider Ops + Grants/Labs Read Surfaces

- implement `Operations_List`
- implement grants `Get`, `List`, `ListAll`
- implement labs `Get`, `List`, `ListAll`
- implement student-labs `Get`, `ListAll`

### Stage 3: Lab + Student Lifecycle

- implement labs `CreateOrUpdate`, `Delete`, and `GenerateInviteCode`
- implement students `CreateOrUpdate`, `Get`, `List`, and `Delete`
- model deterministic budget, status, and date fields

### Stage 4: Join Requests + Redeem Action

- implement join-requests `Get`, `List`, `Approve`, and `Deny`
- implement `RedeemInvitationCode`
- model deterministic request status transitions (`Pending`, `Approved`, `Denied`)

### Stage 5: Tests

- add provider tests `internal/server/provider_azure_education_2021_12_01_preview_test.go` covering:
  - API-version validation and route ownership
  - grants/labs/students/student-labs read and list scopes
  - lab and student lifecycle flows
  - join request approve/deny transitions
  - redeem and invite-code action payload validation
  - delete status behavior (`200`/`204`)

### Stage 6: Example and Validation

- add `examples/azure/education-2021-12-01-preview` using Azure Go SDK core pipeline
- include one call per reviewed method (`21` calls)
- verify build and coverage script discoverability

Exit criteria:

- example compiles and is ready for end-to-end runs once handlers are implemented
- coverage scripts include `education_2021_12_01_preview` aliases and plan-doc mapping

