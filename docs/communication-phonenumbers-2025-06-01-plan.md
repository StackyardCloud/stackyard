# Azure Communication - Phone Numbers (`communication-phonenumbers-2025-06-01`) Staged Plan

## Objective

Emulate the Azure Communication Services Phone Numbers data-plane API for `2025-06-01` with deterministic local behavior for:

- available-number discovery and search
- reservation lifecycle
- purchased phone-number inventory and capability updates
- operation-status tracking for long-running phone-number workflows
- operator-information lookup
- an Azure Go SDK-based example under `examples/azure/communication-phonenumbers-2025-06-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/availablePhoneNumbers/...` and `{resource}.communication.azure.com/phoneNumbers/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Phone Numbers `2025-06-01` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official Azure package index.

Important documentation note:

- The reviewed Learn surface exposes a single `Phone Numbers` operation group with 20 operations.
- Several `2025-06-01` method pages are authorization-gated on Learn, but the operation-group page plus indexed method pages and search-visible excerpts were sufficient to confirm the route surface, major request and response contracts, and long-running-operation headers.
- Where an exact `2025-06-01` method page was not fully retrievable, I used the closest visible Learn page for the same operation family only as a route-and-shape fallback and kept the plan conservative.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/operation-groups?view=rest-communication-phonenumbers-2025-06-01`

Operation-group page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers?view=rest-communication-phonenumbers-2025-06-01`

Exact `2025-06-01` method pages reviewed or search-verified:

- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/browse-available-numbers?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/cancel-operation?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/create-or-update-reservation?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/get-by-number?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/get-operation?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/get-reservation?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/get-search-result?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/list-area-codes?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/list-available-countries?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/list-available-localities?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/list-offerings?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/list-phone-numbers?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/list-reservations?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/purchase-phone-numbers?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/release-phone-number?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/search-available-phone-numbers?view=rest-communication-phonenumbers-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/update-capabilities?view=rest-communication-phonenumbers-2025-06-01`

Closest Learn fallback pages used only where the exact `2025-06-01` method page was not fully retrievable:

- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/delete-reservation?view=rest-communication-phonenumbers-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/operator-information-search?view=rest-communication-phonenumbers-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/phonenumbers/phone-numbers/purchase-reservation?view=rest-communication-phonenumbers-2025-04-01`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_phonenumbers`
- display name: `Azure Communication - Phone Numbers`
- API version token: `2025-06-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_phonenumbers.go`
- tests: `internal/server/provider_azure_communication_phonenumbers_test.go`
- example: `examples/azure/communication-phonenumbers-2025-06-01`
- plan doc: `docs/communication-phonenumbers-2025-06-01-plan.md`

Recommended coverage aliases:

- `communication_phonenumbers`
- `communication-phonenumbers`
- `phonenumbers`
- `phone_numbers`
- `azure_communication_phonenumbers`
- `acs_phonenumbers`
- `rest_communication_phonenumbers_2025_06_01`
- `communication_phonenumbers_2025_06_01`

## Surface Summary

The reviewed REST surface contains 1 operation group and 20 concrete methods:

1. `Phone Numbers`

Documented route family:

- `POST {endpoint}/availablePhoneNumbers/countries/{countryCode}/:browse?api-version=2025-06-01`
- `DELETE {endpoint}/phoneNumbers/operations/{operationId}?api-version=2025-06-01`
- `PATCH {endpoint}/availablePhoneNumbers/reservations/{reservationId}?api-version=2025-06-01`
- `DELETE {endpoint}/availablePhoneNumbers/reservations/{reservationId}?api-version=2025-06-01`
- `GET {endpoint}/phoneNumbers/{phoneNumber}?api-version=2025-06-01`
- `GET {endpoint}/phoneNumbers/operations/{operationId}?api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/reservations/{reservationId}?api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/searchResults/{searchId}?api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/countries/{countryCode}/areaCodes?phoneNumberType={phoneNumberType}&api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/countries?api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/countries/{countryCode}/localities?api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/countries/{countryCode}/offerings?api-version=2025-06-01`
- `GET {endpoint}/phoneNumbers?api-version=2025-06-01`
- `GET {endpoint}/availablePhoneNumbers/reservations?api-version=2025-06-01`
- `POST {endpoint}/operatorInformation/:search?api-version=2025-06-01`
- `POST {endpoint}/availablePhoneNumbers/:purchase?api-version=2025-06-01`
- `POST {endpoint}/availablePhoneNumbers/reservations/{reservationId}/:purchase?api-version=2025-06-01`
- `DELETE {endpoint}/phoneNumbers/{phoneNumber}?api-version=2025-06-01`
- `POST {endpoint}/availablePhoneNumbers/countries/{countryCode}/:search?api-version=2025-06-01`
- `PATCH {endpoint}/phoneNumbers/{phoneNumber}/capabilities?api-version=2025-06-01`

Common optional query traits visible from the reviewed pages:

- `skip`
- `top`
- `maxPageSize`
- `assignmentType`
- `phoneNumberType`
- `administrativeDivision`
- `locality`

Recommended Stackyard route family:

- `/azure/communication-phonenumbers/2025-06-01/availablePhoneNumbers/...`
- `/azure/communication-phonenumbers/2025-06-01/phoneNumbers/...`
- `/azure/communication-phonenumbers/2025-06-01/operatorInformation/:search`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other Communication endpoint-rooted services.
- Every route still requires `api-version=2025-06-01`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- security is documented for ACS endpoint authentication rather than ARM bearer flows
- errors use `CommunicationErrorResponse`
- some workflows are standard synchronous reads
- search, purchase, release, and capability updates are long-running-operation flavored and rely on:
  - `Operation-Location`
  - `Location`
  - `operation-id`
- list routes expose lightweight paging controls

### Available Number Discovery

Reviewed method pages:

- `List Available Countries`
- `List Area Codes`
- `List Available Localities`
- `List Offerings`
- `Browse Available Numbers`

High-value route behavior from the reviewed pages:

- these routes are read-only discovery endpoints under `/availablePhoneNumbers`
- `List Available Countries`
  - paged `GET`
  - optional `skip`, `maxPageSize`
- `List Area Codes`
  - `GET /availablePhoneNumbers/countries/{countryCode}/areaCodes`
  - requires `phoneNumberType`
  - optional `assignmentType`, `locality`, `administrativeDivision`, `skip`, `maxPageSize`
- `List Available Localities`
  - `GET /availablePhoneNumbers/countries/{countryCode}/localities`
  - optional `administrativeDivision`, `phoneNumberType`, `skip`, `maxPageSize`
- `List Offerings`
  - `GET /availablePhoneNumbers/countries/{countryCode}/offerings`
  - optional `phoneNumberType`, `assignmentType`, `skip`, `maxPageSize`
- `Browse Available Numbers`
  - `POST /availablePhoneNumbers/countries/{countryCode}/:browse`
  - not paginated
  - randomized upstream behavior, but Stackyard should be deterministic
  - request body includes `phoneNumberType`
  - optional `assignmentType`, `capabilities`, `phoneNumberPrefixes`

Modeling implications:

- first-pass emulation should keep fixed synthetic country, locality, area-code, and offering catalogs
- browsing should return a deterministic list of available numbers rather than upstream randomization
- number capabilities and costs should be consistent across discovery, search, reservation, and purchase flows

### Search And Search Results

Reviewed method pages:

- `Search Available Phone Numbers`
- `Get Search Result`
- `Get Operation`
- `Cancel Operation`

High-value route behavior from the reviewed pages:

- `Search Available Phone Numbers`
  - `POST /availablePhoneNumbers/countries/{countryCode}/:search`
  - request body includes required:
    - `assignmentType`
    - `capabilities`
    - `phoneNumberType`
  - optional:
    - `areaCode`
    - `quantity`
  - success status: `202 Accepted`
  - returns headers:
    - `Operation-Location`
    - `Location`
    - `operation-id`
    - `search-id`
- `Get Search Result`
  - `GET /availablePhoneNumbers/searchResults/{searchId}`
  - returns the final search result payload
- `Get Operation`
  - `GET /phoneNumbers/operations/{operationId}`
  - returns operation state and a final `resourceLocation` / `Location` for completed workflows
- `Cancel Operation`
  - `DELETE /phoneNumbers/operations/{operationId}`
  - success status: `204 No Content`

Modeling implications:

- searches should be represented as long-running operations plus a separately addressable search-result resource
- first-pass deterministic model:
  - search request creates an operation with status `running`
  - first operation poll can remain `running`
  - later poll transitions to `succeeded`
  - `Get Search Result` returns a stable result immediately after creation or after first poll, depending on the emulator choice
- cancellations should transition only cancelable operations to `canceled`

### Reservations

Reviewed method pages:

- `Create Or Update Reservation`
- `Get Reservation`
- `List Reservations`
- `Delete Reservation`
- `Purchase Reservation`

High-value route behavior from the reviewed pages:

- `Create Or Update Reservation`
  - `PATCH /availablePhoneNumbers/reservations/{reservationId}`
  - idempotent
  - active reservations only
  - payload adds numbers when values are present and removes them when values are explicit `null`
  - updates extend expiration by 15 minutes after last change, capped to 2 hours from creation
  - partial success is possible with `207`
- `Get Reservation`
  - returns full reservation with `phoneNumbers`, `expiresAt`, and `status`
- `List Reservations`
  - paged `GET`
  - collection payload omits the embedded `phoneNumbers` details
- `Delete Reservation`
  - `204 No Content`
  - active reservations only
- `Purchase Reservation`
  - long-running `POST`
  - requires active reservation with at least one phone number
  - may require `agreeToNotResell`
  - returns `202 Accepted` and operation-tracking headers

Modeling implications:

- reservations should be durable in-memory resources with:
  - `id`
  - `status`
  - `expiresAt`
  - `createdAt`
  - `phoneNumbers` map keyed by number id
- partial success can be modeled conservatively:
  - first pass may return `200` for fully valid reservation updates
  - `207` should still be represented in tests for mixed valid/invalid reservation patch bodies if implemented
- reservation purchase should promote reserved numbers into purchased inventory and mark the reservation `completed`

### Purchased Phone Numbers

Reviewed method pages:

- `List Phone Numbers`
- `Get By Number`
- `Release Phone Number`
- `Update Capabilities`
- `Purchase Phone Numbers`

High-value route behavior from the reviewed pages:

- `List Phone Numbers`
  - paged `GET`
  - optional `skip`, `top`
- `Get By Number`
  - returns a single purchased phone-number record
- `Release Phone Number`
  - `DELETE /phoneNumbers/{phoneNumber}`
  - `202 Accepted`
  - returns `Operation-Location`
- `Update Capabilities`
  - `PATCH /phoneNumbers/{phoneNumber}/capabilities`
  - merge-patch body with `calling` and `sms`
  - `202 Accepted`
  - returns `Operation-Location`, `Location`, `operation-id`, `capabilities-id`
- `Purchase Phone Numbers`
  - `POST /availablePhoneNumbers/:purchase`
  - long-running purchase flow for directly selected numbers

Modeling implications:

- purchased inventory should be durable in-memory state keyed by normalized E.164 number
- release and capability updates should create operation records and eventually mutate inventory
- purchase operations should move numbers from discovery/search/reservation contexts into purchased inventory

### Operator Information Search

Reviewed method page:

- `Operator Information Search`

High-value route behavior from the fallback Learn page:

- `POST /operatorInformation/:search`
- required body:
  - `phoneNumbers`
- success status: `200 OK`
- returns per-number operator and formatting information

Modeling implications:

- first-pass emulation can synthesize deterministic carrier and formatting metadata from the E.164 input
- this route is synchronous and independent from reservation/purchase state

## Recommended Emulator Shape

State model:

- `available phone numbers` derived from fixed country/locality/offering catalogs
- `search results` keyed by `searchId`
- `reservations` keyed by `reservationId`
- `purchased phone numbers` keyed by normalized E.164 number
- `operations` keyed by `operationId`

Primary response models:

- available-country collection
- area-code collection
- locality collection
- offering collection
- browse result
- search result
- reservation
- purchased phone number
- operation
- operator-information result

Operation model:

- `id`
- `operationType`
- `status`
- `resourceLocation`
- `createdDateTime`
- `lastActionDateTime`

Deterministic fixture guidance:

- use a small fixed catalog such as:
  - `US`
  - `GB`
  - `CA`
- derive number ids and synthetic phone numbers from country, locality, type, and assignment
- use stable costs and capabilities across related flows
- normalize purchased and reserved numbers in E.164 form
- implement simple long-running progression:
  - create operation as `running`
  - first poll may stay `running`
  - second poll becomes `succeeded` unless seeded otherwise

Error model:

- reuse the ACS-style Communication error envelope pattern already present in the repo
- return stable codes such as:
  - `InvalidApiVersion`
  - `InvalidRequest`
  - `PathNotFound`
  - `ReservationNotFound`
  - `PhoneNumberNotFound`
  - `OperationNotFound`
  - `SearchResultNotFound`

## Validation Rules To Encode

Request validation:

- reject missing or incorrect `api-version`
- reject invalid JSON
- reject malformed `countryCode`
- reject malformed reservation ids where the path requires UUID semantics
- reject malformed E.164 numbers on purchased-number routes
- reject invalid enum values for:
  - `phoneNumberType`
  - `assignmentType`
  - capability values
- reject invalid paging parameters

Response validation:

- discovery routes return stable typed collections
- search and purchase-style routes return long-running headers instead of `NotImplemented`
- `Get Operation` returns consistent lifecycle state
- reservation and purchased-number reads return stable typed objects

## Staged Implementation Plan

### Stage 1: Route Skeleton, Service Registration, And Shared Models

Add the service shell and shared state.

Implementation work:

- add `internal/server/provider_azure_communication_phonenumbers.go`
- register the router in `internal/server/http.go`
- add server state for:
  - operations
  - search results
  - reservations
  - purchased phone numbers
- isolate routing under:
  - `/azure/communication-phonenumbers/2025-06-01`

Acceptance target:

- all Phone Numbers route families are recognized
- unknown routes return structured `PathNotFound`

### Stage 2: Discovery And Browse/Search

Implement the non-destructive catalog and search surface.

Implementation work:

- implement:
  - `List Available Countries`
  - `List Area Codes`
  - `List Available Localities`
  - `List Offerings`
  - `Browse Available Numbers`
  - `Search Available Phone Numbers`
  - `Get Search Result`
  - `Get Operation`
  - `Cancel Operation`
- add deterministic operation creation for search requests

Acceptance target:

- browse is synchronous and deterministic
- search is asynchronous and pollable
- no search route falls through to `NotImplemented`

### Stage 3: Reservation Lifecycle

Implement reservation creation and management.

Implementation work:

- implement:
  - `Create Or Update Reservation`
  - `Get Reservation`
  - `List Reservations`
  - `Delete Reservation`
  - `Purchase Reservation`
- model reservation expiry extension and active/completed status transitions

Acceptance target:

- reservations are durable and queryable
- reservation purchase produces an operation and final purchased inventory

### Stage 4: Purchased Inventory And Capability Updates

Implement purchased-number inventory routes.

Implementation work:

- implement:
  - `List Phone Numbers`
  - `Get By Number`
  - `Purchase Phone Numbers`
  - `Release Phone Number`
  - `Update Capabilities`
- wire operation tracking for release and capability updates

Acceptance target:

- purchased inventory behaves as a durable store
- release and capability operations are pollable and mutate final state

### Stage 5: Operator Information Search

Implement the synchronous operator-information lookup route.

Implementation work:

- implement:
  - `POST /operatorInformation/:search`
- synthesize deterministic operator and formatting metadata for each requested number

Acceptance target:

- route returns typed operator info for valid numbers
- invalid requests return ACS-style errors

### Stage 6: Focused Provider Tests

Cover the reviewed surface with provider tests.

Implementation work:

- add `internal/server/provider_azure_communication_phonenumbers_test.go`
- cover:
  - discovery lists
  - browse
  - search + operation polling + search result
  - reservation patch/get/list/delete/purchase
  - list/get purchased number
  - direct purchase
  - release
  - update capabilities
  - operator-information search
  - invalid API version
  - invalid JSON
  - invalid enums
  - invalid E.164 path values
  - not-found paths and resources

Acceptance target:

- tests prove every reviewed route family is implemented
- no Phone Numbers request returns `NotImplemented`

### Stage 7: Azure Go SDK Example

Add a runnable example that exercises the primary flows.

Implementation work:

- add `examples/azure/communication-phonenumbers-2025-06-01/main.go`
- add matching `Dockerfile`
- add matching `docker-compose.yml`
- use the official Azure Go SDK core runtime pipeline:
  - `azcore/runtime`
  - `azcore/policy`
- demonstrate:
  - listing countries or offerings
  - starting a search
  - polling the operation
  - fetching the search result
  - optionally creating a reservation or reading purchased inventory

Acceptance target:

- example builds successfully
- example demonstrates the Stackyard route family and required `api-version`

### Stage 8: Coverage Script Wiring

Register the new service in the Azure coverage helpers.

Implementation work:

- update `scripts/azure-contract-coverage.py`
- update `scripts/azure-io-contract-coverage.py`
- update `scripts/azure-doc-contract-coverage.py`
- add aliases for `communication_phonenumbers`
- map the service to this plan doc

Acceptance target:

- coverage scripts discover the service under canonical and alias names
- doc coverage recognizes the Learn references in this plan doc

### Stage 9: Final Validation

Run the service-specific validation gates.

Validation commands to run during implementation:

- `go test ./internal/server -run 'TestAzureCommunicationPhoneNumbers' -count=1`
- `go build ./examples/azure/communication-phonenumbers-2025-06-01`
- `python3 scripts/azure-contract-coverage.py --service communication_phonenumbers --require-service communication_phonenumbers --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_phonenumbers --require-service communication_phonenumbers --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_phonenumbers --require-service communication_phonenumbers --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_phonenumbers --require-service communication_phonenumbers --fail-on any`
- `make ci`

Expected outcome:

- all Phone Numbers endpoints in the reviewed surface are covered and successful
- no Phone Numbers endpoint returns `NotImplemented`
- any remaining `make ci` failures should be reported separately if they are unrelated pre-existing repo issues
