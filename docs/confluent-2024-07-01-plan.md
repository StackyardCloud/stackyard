# Azure Confluent (`confluent-2024-07-01`) Staged Plan

## Objective

Emulate the Azure Confluent ARM management API for `2024-07-01` with deterministic local behavior for:

- provider `operations`
- subscription-scoped `marketplace agreements`
- core `organizations`
- organization-scoped `access` action routes
- nested `environments`, `clusters`, and `topics`
- an Azure Go SDK example under `examples/azure/confluent-2024-07-01`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed Learn surface is the ARM management plane under `Microsoft.Confluent`, not a direct Kafka data plane.
- This API mixes normal ARM resources such as `organizations` with provider and child action routes like `access/default/*`, API key management, and topic management under environment and cluster identifiers.
- Several operation-group and method pages are authorization-gated on Microsoft Learn, but they still expose enough route, parameter, request-body, and response metadata to stage the emulator cleanly.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/confluent/operation-groups?view=rest-confluent-2024-07-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/confluent/organization?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/access?view=rest-confluent-2024-07-01`

Representative linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/confluent/organization-operations/list?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/organization/create?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/organization/get?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/organization/update?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/access/list-clusters?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/environment/create-or-update?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/cluster/create-or-update?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/topics/create?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/topics/get?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/topics/list?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/marketplace-agreements/create?view=rest-confluent-2024-07-01`
- `https://learn.microsoft.com/en-us/rest/api/confluent/marketplace-agreements/list?view=rest-confluent-2024-07-01`

## SDK Selection Note

- The Azure Go SDK provides a dedicated module for this surface: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/confluent/armconfluent/v2`.
- The future example should prefer typed `armconfluent/v2` clients for:
  - `Organization`
  - `Access`
  - `MarketplaceAgreements`
  - nested environment and cluster operations if exposed by the generated package
- If the local module revision does not expose every reviewed `2024-07-01` child route cleanly, a narrow `azcore/runtime` fallback is acceptable for the remaining tails.

## Service Naming

Canonical Stackyard naming:

- service key: `confluent`
- display name: `Azure Confluent`
- API version token: `2024-07-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_confluent.go`
- tests: `internal/server/provider_azure_confluent_test.go`
- example: `examples/azure/confluent-2024-07-01`
- plan doc: `docs/confluent-2024-07-01-plan.md`

Recommended coverage aliases:

- `confluent`
- `azure_confluent`
- `microsoft_confluent`
- `confluent_2024_07_01`
- `rest_confluent_2024_07_01`

## API Surface Summary

The reviewed pages expose several route families:

- provider-level:
  - `/providers/Microsoft.Confluent/operations`
- subscription-level marketplace-agreement routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Confluent/agreements`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Confluent/agreements/default`
- core ARM organization routes:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Confluent/organizations`
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Confluent/organizations/{organizationName}`
  - subscription list variant for organizations
- organization action routes under `access/default`:
  - `listClusters`
  - `listEnvironments`
  - `listInvitations`
  - `listRoleBindingNameList`
  - `listRoleBindings`
  - `listServiceAccounts`
  - `listUsers`
  - `inviteUser`
  - `createRoleBinding`
  - `deleteRoleBinding`
- nested child resource families under organizations:
  - `/environments/{environmentId}`
  - `/environments/{environmentId}/clusters/{clusterId}`
  - `/environments/{environmentId}/clusters/{clusterId}/topics`
  - API-key and schema-registry tails surfaced on the organization group page

Modeling implications:

- this is a normal ARM surface at the organization layer, with deterministic child resources keyed below each organization
- environments, clusters, and topics use identifiers chosen by the caller rather than generated ARM resource names
- the access routes are RPC-style action endpoints and should be modeled separately from CRUD resources

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use `ResourceProviderDefaultErrorResponse`
- list routes return `value` and sometimes `nextLink`
- path parameters validate `subscriptionId` as UUID and `resourceGroupName` as ARM-valid resource group name

### Provider Operations

From `Organization Operations - List`:

- route:
  - `GET /providers/Microsoft.Confluent/operations?api-version=2024-07-01`
- response:
  - `value`
  - each operation includes `name`, `display`, and `isDataAction`

The sample list at least confirms the normal ARM organization verbs:

- read
- write
- delete

Implementation implication:

- the provider operation fixture should include organization CRUD plus the action and child-resource families surfaced on the reviewed pages

### Marketplace Agreements

From `Marketplace Agreements - Create` and `List`:

- routes:
  - `PUT /subscriptions/{subscriptionId}/providers/Microsoft.Confluent/agreements/default`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Confluent/agreements`
- resource shape:
  - `id`, `name`, `type`, `properties`, and `systemData`
- key properties:
  - `accepted`
  - `licenseTextLink`
  - `plan`
  - `privacyPolicyLink`
  - `product`
  - `publisher`
  - `retrieveDatetime`
  - `signature`

Implementation implication:

- this can be modeled as a simple subscription-scoped singleton plus list wrapper

### Organizations

From `Organization - Create`, `Get`, and `Update`:

- routes:
  - `PUT .../organizations/{organizationName}`
  - `GET .../organizations/{organizationName}`
  - `PATCH .../organizations/{organizationName}`
  - `DELETE .../organizations/{organizationName}`
  - list by resource group
  - list by subscription
- create request requires:
  - `location`
  - `properties.offerDetail`
- patch request primarily updates tags

Implementation implication:

- first pass should treat `organizations` as the root durable Confluent resource and hang every child family beneath it

### Access Routes

From `Access - REST API` and `Access - List Clusters`:

- routes sit under:
  - `/organizations/{organizationName}/access/default/...`
- some are read/list actions:
  - `listClusters`
  - `listEnvironments`
  - `listInvitations`
  - `listRoleBindings`
  - `listServiceAccounts`
  - `listUsers`
- some are mutation actions:
  - `inviteUser`
  - `createRoleBinding`
  - `deleteRoleBinding`
- `listClusters` uses `POST` with a request body carrying `searchFilters`

Implementation implication:

- these should not be forced into fake ARM resource CRUD; they are better represented as organization-scoped action handlers over deterministic in-memory invitation, role-binding, service-account, user, environment, and cluster fixtures

### Environments, Clusters, and Topics

From the `Environment`, `Cluster`, and `Topics` pages:

- nested routes:
  - `PUT/GET/DELETE .../environments/{environmentId}`
  - `PUT/GET/DELETE .../clusters/{clusterId}`
  - `PUT/GET/DELETE .../topics/{topicName}`
  - `GET .../topics`
- these pages use caller-provided identifiers:
  - `environmentId`
  - `clusterId`
  - `topicName`
- topic list supports pagination-style query parameters:
  - `pageSize`
  - `pageToken`

Implementation implication:

- environments and clusters can be modeled as typed nested resources
- topics are a deeper child family under cluster scope, with deterministic paged list behavior in the first pass

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the new service shell and alias wiring:

- reserve service key `confluent`
- add provider registration in `internal/server/http.go`
- add provider scaffolding in `internal/server/provider_azure_confluent.go`
- add plan-doc mapping and aliases in the three Azure coverage scripts
- create the example directory scaffold under `examples/azure/confluent-2024-07-01`

Deliverable:

- the service is discoverable by the Azure coverage tooling before the deeper routes are filled in

### Stage 1: Provider Operations and Marketplace Agreements

Implement:

- `GET /providers/Microsoft.Confluent/operations`
- `GET /subscriptions/{subscriptionId}/providers/Microsoft.Confluent/agreements`
- `PUT /subscriptions/{subscriptionId}/providers/Microsoft.Confluent/agreements/default`

Validation targets:

- UUID validation for subscription id
- required `properties.accepted` and basic agreement property normalization

Deliverable:

- provider discovery and subscription bootstrap routes work

### Stage 2: Core Organizations

Implement the durable ARM resource layer:

- organization create/get/list/patch/delete
- list by resource group
- list by subscription

Model the root organization with:

- `id`, `name`, `type`, `location`
- `tags`
- `properties.offerDetail`
- `properties.provisioningState`
- `systemData`

Deliverable:

- the service has a stable parent resource on which all nested Confluent entities depend

### Stage 3: Access Actions

Implement the action family under `access/default`:

- list routes:
  - clusters
  - environments
  - invitations
  - role bindings
  - role binding name list
  - service accounts
  - users
- mutation routes:
  - invite user
  - create role binding
  - delete role binding

Behavior targets:

- deterministic action responses using organization-scoped fixtures
- `listClusters` should accept a request body with `searchFilters`

Deliverable:

- the non-CRUD organization management surface works without inventing fake ARM children for every access concept

### Stage 4: Environments and Clusters

Implement nested resource families:

- `environments` create/get/list/delete
- `clusters` create/get/list/delete
- supporting get-by-id helpers surfaced on the organization group page

Validation targets:

- organization must exist before child creation
- `environmentId` and `clusterId` must be non-empty

Deliverable:

- the core Confluent hierarchy under organization is in place

### Stage 5: Topics

Implement the deeper child family:

- `PUT/GET/DELETE .../topics/{topicName}`
- `GET .../topics`

Behavior targets:

- deterministic pagination with `pageSize` and `pageToken`
- topic resource payloads should be nested under cluster scope

Deliverable:

- the emulator supports the most concrete Kafka-like child resource family documented on Learn

### Stage 6: Organization Tail Actions

Implement the remaining organization-scoped tails surfaced on the group page as coverage requires:

- create/get/delete cluster API key
- get cluster by id
- get environment by id
- list schema registry clusters
- get schema registry cluster by id
- list regions

Recommendation:

- keep API keys and schema-registry clusters as deterministic synthetic children rather than over-modeling real Confluent internals

Deliverable:

- the documented organization tail endpoints no longer fall through to `NotImplemented`

### Stage 7: Example Program

Add `examples/azure/confluent-2024-07-01`:

- prefer `armconfluent/v2`
- show:
  - listing provider operations
  - creating a marketplace agreement
  - creating an organization
  - creating an environment
  - creating a cluster
  - listing topics or access clusters

Suggested example flow:

1. list provider operations
2. accept marketplace agreement
3. create organization
4. create environment
5. create cluster
6. list clusters through `access/default/listClusters`

Deliverable:

- an example that exercises both ARM CRUD and Confluent-specific action routes

### Stage 8: Tests and Coverage

Add focused provider tests in `internal/server/provider_azure_confluent_test.go` for:

- operations list
- marketplace agreement create/list
- organization lifecycle
- access action success and validation cases
- environment and cluster lifecycle
- topic lifecycle and pagination
- representative not-found and bad-request cases

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Validation target after implementation:

- `python3 scripts/azure-contract-coverage.py --service confluent --require-service confluent --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service confluent --require-service confluent --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service confluent --require-service confluent --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service confluent --require-service confluent --fail-on any`

## Recommended First Implementation Slice

If the goal is a mergeable first version with the highest signal-to-effort ratio:

1. provider `operations`
2. marketplace agreements
3. organization CRUD/list
4. access list routes
5. environments and clusters
6. one topic list/get/create path
7. example and coverage wiring

That slice establishes the real Confluent hierarchy and the action-heavy management surface quickly, and the remaining API-key and schema-registry tails can reuse the same organization-scoped fixture model.
