# Azure Communication (`communication-resourcemanager-2026-03-18`) Staged Plan

## Objective

Emulate the Azure Communication ARM management-plane API for `2026-03-18` with deterministic local behavior for:

- provider `operations`
- top-level `communicationServices`
- top-level `emailServices`
- nested `domains`
- nested `senderUsernames`
- nested `smtpUsernames`
- nested `suppressionLists`
- nested `suppressionListAddresses`
- an Azure Go SDK-based example under `examples/azure/communication-resourcemanager-2026-03-18`
- Azure coverage-script wiring for the new service key

Important scope note:

- This reviewed surface is the Azure Resource Manager control plane under `Microsoft.Communication`, not any of the Azure Communication Services data planes already implemented in the repo such as Chat, Email, Identity, Rooms, Phone Numbers, or Sip Routing.
- The route surface mixes two major resource trees:
  - `communicationServices` and their direct action routes like `listKeys`, `regenerateKey`, and `linkNotificationHub`
  - `emailServices` and the nested email hierarchy below them: `domains`, `senderUsernames`, `suppressionLists`, and `suppressionListAddresses`
- The staged plan stays within the verified `2026-03-18` management surface and does not invent unrelated ACS data-plane behavior.

Important SDK note:

- The official Azure Go SDK package for this provider family is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/communication/armcommunication/v2`.
- The currently indexed package is generated from `2025-09-01`, not this exact `2026-03-18` version.
- The future example should therefore prefer typed `armcommunication/v2` clients where the reviewed route shapes align, and use `azcore/runtime` fallback only if an exact `2026-03-18` route is missing from the typed package.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/operation-groups?view=rest-communication-resourcemanager-2026-03-18`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/sender-usernames?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-list-addresses?view=rest-communication-resourcemanager-2026-03-18`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/operations/list?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/check-name-availability?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/delete?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/link-notification-hub?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/list-by-resource-group?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/list-by-subscription?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/list-keys?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/regenerate-key?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/communication-services/update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/delete?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/list-by-resource-group?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/list-by-subscription?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/list-verified-exchange-online-domains?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/email-services/update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/cancel-verification?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/delete?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/initiate-verification?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/list-by-email-service-resource?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/domains/update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/sender-usernames/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/sender-usernames/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/sender-usernames/list-by-domains?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/smtp-usernames/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/smtp-usernames/delete?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/smtp-usernames/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-lists/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-lists/delete?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-lists/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-list-addresses/create-or-update?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-list-addresses/delete?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-list-addresses/get?view=rest-communication-resourcemanager-2026-03-18`
- `https://learn.microsoft.com/en-us/rest/api/communication/resourcemanager/suppression-list-addresses/list?view=rest-communication-resourcemanager-2026-03-18`

## Service Naming

Canonical Stackyard naming:

- service key: `communication`
- display name: `Azure Communication`
- API version token: `2026-03-18`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication.go`
- tests: `internal/server/provider_azure_communication_test.go`
- example: `examples/azure/communication-resourcemanager-2026-03-18`
- plan doc: `docs/communication-resourcemanager-2026-03-18-plan.md`

Recommended coverage aliases:

- `communication`
- `communication-resourcemanager`
- `communication_resourcemanager`
- `azure_communication`
- `microsoft_communication`
- `rest_communication_resourcemanager_2026_03_18`
- `communication_resourcemanager_2026_03_18`

## Surface Summary

The reviewed REST surface contains 8 operation groups:

1. `Operations`
2. `Communication Services`
3. `Email Services`
4. `Domains`
5. `Sender Usernames`
6. `Smtp Usernames`
7. `Suppression Lists`
8. `Suppression List Addresses`

High-value route families:

- provider root:
  - `GET /providers/Microsoft.Communication/operations`
- communication services:
  - subscription-scoped `checkNameAvailability`
  - resource CRUD and list
  - `listKeys`
  - `regenerateKey`
  - `linkNotificationHub`
- email services:
  - resource CRUD and list
  - subscription-scoped `listVerifiedExchangeOnlineDomains`
- domains:
  - nested CRUD/list under `emailServices`
  - `initiateVerification`
  - `cancelVerification`
- sender usernames:
  - nested create/get/list/delete under `domains`
- smtp usernames:
  - nested create/get/list/delete under `communicationServices`
- suppression lists:
  - nested create/get/list/delete under `domains`
- suppression list addresses:
  - nested create/get/list/delete under `suppressionLists`

## Contract Notes From Reviewed Pages

Shared traits across the reviewed pages:

- security is ARM OAuth rather than ACS endpoint shared-key auth
- all routes use the ARM management endpoint shape under `management.azure.com`
- top-level resources follow normal ARM `PUT/GET/PATCH/DELETE/list` patterns
- several create and delete routes are long-running and document `Azure-AsyncOperation` plus `Retry-After`

### Communication Services

Reviewed method pages show:

- `checkNameAvailability` is subscription-scoped and returns `nameAvailable`, `reason`, and `message`
- `Create Or Update` requires:
  - `location`
  - `properties.dataLocation`
  - optional `identity`
  - optional `properties.disableLocalAuth`
  - optional `properties.linkedDomains`
- `Update` can patch:
  - `identity`
  - `properties.disableLocalAuth`
  - `properties.linkedDomains`
  - `properties.publicNetworkAccess`
  - `tags`
- `linkNotificationHub` is a distinct action route
- `listKeys` and `regenerateKey` return key material and must model primary/secondary key rotation

Modeling implications:

- the emulator should persist communication services as first-class ARM resources with deterministic `hostName`, `version`, key material, and `systemData`
- key actions should update stored key state instead of returning stateless fixtures
- `linkedDomains` should store ARM IDs of linked email domain resources

### Email Services

Reviewed method pages show:

- `Create Or Update` requires:
  - `location`
  - `properties.dataLocation`
  - optional `tags`
- `Update` is effectively a tag patch
- `listVerifiedExchangeOnlineDomains` is a provider/subscription action route returning a simple `string[]`

Modeling implications:

- email services should be a separate top-level ARM resource family keyed independently from communication services
- the verified Exchange Online domain action can be deterministic and subscription-scoped rather than coupled to resource creation

### Domains

Reviewed method pages show:

- `Create Or Update` requires:
  - `location`
  - `properties.domainManagement`
  - optional `properties.userEngagementTracking`
  - optional `tags`
- `Update` can patch:
  - `properties.userEngagementTracking`
  - `tags`
- `initiateVerification` and `cancelVerification` are POST actions on a domain
- domains are listed under an email service

Modeling implications:

- the emulator should store DNS-verification state on each domain resource
- verification actions should transition deterministic per-record verification status instead of returning empty placeholders
- the resource should expose enough read-only verification metadata for nested email resources to reference the parent domain cleanly

### Sender Usernames, SMTP Usernames, Suppression Lists, Suppression List Addresses

Reviewed method pages show:

- sender usernames are nested under a domain and accept:
  - required `properties.username`
  - optional `properties.displayName`
- SMTP usernames are nested under a communication service and accept:
  - required `properties.entraApplicationId`
  - required `properties.tenantId`
  - required `properties.username`
- suppression lists are nested under a domain and accept:
  - `properties.listName`
- suppression list addresses are nested under a suppression list and accept:
  - required `properties.email`
  - optional `properties.firstName`
  - optional `properties.lastName`
  - optional `properties.notes`

Modeling implications:

- nested-child storage should follow normalized parent IDs rather than flattening all email children into one bucket
- sender usernames and suppression lists should validate against the parent domain context
- suppression list names and addresses should be persisted so list routes and `GET` routes are backed by the same store

## Stage 1: Route Skeleton And Provider Operations

Add:

- `internal/server/provider_azure_communication.go`

Implement first:

- `handleAzureCommunicationRouter`
- `GET /providers/Microsoft.Communication/operations`

Add provider state in `internal/server/http.go` for:

- communication services
- email services
- nested domain resources
- nested sender usernames
- nested SMTP usernames
- nested suppression lists
- nested suppression list addresses

Exit criteria:

- the Azure router recognizes `Microsoft.Communication` resource-manager routes
- provider operations return deterministic `value[]`
- unknown `Microsoft.Communication` paths do not fall through to `NotImplemented`

## Stage 2: Communication Services Foundation

Implement:

- `POST /subscriptions/{subscriptionId}/providers/Microsoft.Communication/checkNameAvailability`
- `PUT/GET/PATCH/DELETE` for `communicationServices`
- `list-by-resource-group`
- `list-by-subscription`

Model:

- `location`
- `properties.dataLocation`
- `properties.disableLocalAuth`
- `properties.linkedDomains`
- `properties.publicNetworkAccess`
- `properties.hostName`
- `properties.version`
- `identity`
- `tags`
- `systemData`
- `provisioningState`

Exit criteria:

- communication services behave like normal ARM resources with deterministic IDs and read-backs
- name availability supports both available and unavailable responses
- create/delete support repo-consistent long-running semantics where appropriate

## Stage 3: Communication Service Actions

Implement:

- `POST .../linkNotificationHub`
- `POST .../listKeys`
- `POST .../regenerateKey`

Model:

- stored primary and secondary keys
- connection strings derived from the stored hostname and keys
- linked notification hub configuration

Exit criteria:

- action routes mutate or return stored state rather than synthetic stateless payloads
- regenerated keys are deterministic but actually change the requested key slot

## Stage 4: Email Services Foundation

Implement:

- `PUT/GET/PATCH/DELETE` for `emailServices`
- list by resource group
- list by subscription
- `POST /subscriptions/{subscriptionId}/providers/Microsoft.Communication/listVerifiedExchangeOnlineDomains`

Model:

- `location`
- `properties.dataLocation`
- `tags`
- `provisioningState`
- deterministic verified Exchange Online domain inventory

Exit criteria:

- email services are first-class ARM resources
- subscription-wide verified-domain action returns stable data for examples and coverage

## Stage 5: Domains And Verification Actions

Implement:

- `PUT/GET/PATCH/DELETE` for `domains`
- `list-by-email-service-resource`
- `POST .../initiateVerification`
- `POST .../cancelVerification`

Model:

- `properties.domainManagement`
- `properties.userEngagementTracking`
- verification records and verification state
- derived inheritance points for child resources

Exit criteria:

- domains support lifecycle, read-back, list, and verification state transitions
- child resources can be keyed under the normalized domain ARM ID

## Stage 6: Sender Usernames And SMTP Usernames

Implement:

- sender usernames:
  - create/get/list/delete under a domain
- SMTP usernames:
  - create/get/list/delete under a communication service

Model:

- sender username `username` and `displayName`
- SMTP username `entraApplicationId`, `tenantId`, and `username`

Exit criteria:

- both child families have CRUD/list coverage with deterministic parent-child linkage
- parent existence validation is enforced

## Stage 7: Suppression Lists And Suppression List Addresses

Implement:

- suppression lists:
  - create/get/list/delete under a domain
- suppression list addresses:
  - create/get/list/delete under a suppression list

Model:

- suppression list `listName`
- suppression address `email`, `firstName`, `lastName`, `notes`
- inherited domain/email-service context

Exit criteria:

- nested list and get routes are backed by persisted child state
- delete semantics are deterministic and coverage-friendly

## Stage 8: Tests

Add focused tests in:

- `internal/server/provider_azure_communication_test.go`

Cover:

- provider operations
- communication service lifecycle and action routes
- email service lifecycle
- domain lifecycle and verification action transitions
- sender username lifecycle
- SMTP username lifecycle
- suppression list lifecycle
- suppression list address lifecycle
- name validation, missing parent resources, and wrong API version cases

Exit criteria:

- a focused `TestAzureCommunication` suite passes
- every reviewed route family has both success and negative coverage

## Stage 9: Azure Go SDK Example

Add:

- `examples/azure/communication-resourcemanager-2026-03-18/main.go`
- `examples/azure/communication-resourcemanager-2026-03-18/Dockerfile`
- `examples/azure/communication-resourcemanager-2026-03-18/docker-compose.yml`

Example flow should exercise:

- provider `checkNameAvailability`
- create/get/update/list for a communication service
- `listKeys` and `regenerateKey`
- create/get for an email service
- create/get/list for a domain
- create/get/list for one sender username or SMTP username
- create/get/list for one suppression list address chain

SDK approach:

- prefer `armcommunication/v2` clients from a client factory
- if a reviewed `2026-03-18` route is not exposed cleanly by the currently published package, use `azcore/runtime` for that one route only

Exit criteria:

- the example builds cleanly with the repo toolchain
- the example demonstrates the management-plane split between communication services and email services

## Stage 10: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for:

- `communication`
- `communication-resourcemanager`
- `communication_resourcemanager`
- `azure_communication`
- `microsoft_communication`
- `rest_communication_resourcemanager_2026_03_18`
- `communication_resourcemanager_2026_03_18`

Map the plan doc:

- `communication-resourcemanager-2026-03-18-plan.md`

Exit criteria:

- the new service is discoverable by all three Azure coverage scripts
- doc coverage recognizes the plan file and the official Learn sources

## Stage 11: Final Validation

Run:

- `go test ./internal/server -run 'TestAzureCommunication' -count=1`
- `go build ./examples/azure/communication-resourcemanager-2026-03-18`
- `python3 scripts/azure-contract-coverage.py --service communication --require-service communication --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication --require-service communication --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication --require-service communication --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication --require-service communication --fail-on any`
- `make ci`

Success criteria:

- all reviewed Communication ARM endpoints are covered by the new provider
- the Communication coverage scripts pass without `NotImplemented`
- any remaining `make ci` failures are unrelated pre-existing repo issues
