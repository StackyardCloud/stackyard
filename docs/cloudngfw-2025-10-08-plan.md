# Azure Cloud NGFW (`cloudngfw-2025-10-08`) Staged Plan

## Objective

Emulate the Azure Cloud NGFW ARM management API for `2025-10-08` with deterministic local behavior for:

- provider `operations`
- top-level `firewalls`
- firewall child `statuses`
- firewall action routes such as log-profile and support/global-rulestack lookups
- subscription/resource-group-scoped `localRulestacks`
- local-rulestack child resources:
  - `localRules`
  - `prefixlists`
  - `fqdnlists`
  - `certificates`
- provider/tenant-scoped `globalRulestacks`
- global-rulestack child resources:
  - `prefixlists`
  - `fqdnlists`
  - `certificates`
- rulestack utility/action routes such as `commit`, `revert`, changelog, support info, and catalog lists
- an Azure Go SDK example under `examples/azure/cloudngfw-2025-10-08`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the ARM management plane under `PaloAltoNetworks.Cloudngfw`.
- This is not a packet-processing data plane and not the separate Palo Alto AWS SDK.
- The provider is unusual in that it mixes:
  - normal subscription/resource-group ARM resources (`firewalls`, `localRulestacks`)
  - tenant/provider-scoped resources with no subscription segment (`globalRulestacks`)
  - many nested object families under both local and global rulestacks

Important staging note:

- The full surface is broad enough that trying to emulate every nested object family in one pass would be low leverage.
- The staged plan below treats the service in four practical layers:
  - provider discovery and top-level firewalls
  - local rulestacks and their action/catalog routes
  - local child object families plus rule counters
  - global rulestacks and global child object families

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/operation-groups?view=rest-cloudngfw-2025-10-08`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/operations?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewalls?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewall-status?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rulestacks?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rules?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/prefix-list-local-rulestack?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/fqdn-list-local-rulestack?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/certificate-object-local-rulestack?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/global-rulestack?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/prefix-list-global-rulestack?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/fqdn-list-global-rulestack?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/certificate-object-global-rulestack?view=rest-cloudngfw-2025-10-08`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/operations/list?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewalls/create-or-update?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewalls/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewalls/list-by-resource-group?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewalls/list-by-subscription?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewall-status/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/firewall-status/list-by-firewalls?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rulestacks/create-or-update?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rulestacks/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rulestacks/update?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rules/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rules/list-by-local-rulestacks?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/local-rules/get-counters?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/fqdn-list-local-rulestack/create-or-update?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/fqdn-list-local-rulestack/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/prefix-list-local-rulestack/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/certificate-object-local-rulestack/create-or-update?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/global-rulestack/list?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/fqdn-list-global-rulestack/create-or-update?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/fqdn-list-global-rulestack/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/prefix-list-global-rulestack/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/certificate-object-global-rulestack/get?view=rest-cloudngfw-2025-10-08`
- `https://learn.microsoft.com/en-us/rest/api/cloudngfw/certificate-object-global-rulestack/list?view=rest-cloudngfw-2025-10-08`

Official Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/paloaltonetworksngfw/armpanngfw/v2`

## SDK Selection Note

- The official Azure Go SDK package for this surface is `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/paloaltonetworksngfw/armpanngfw/v2`.
- The package exposes typed clients aligned with the reviewed surface, including families such as:
  - `FirewallsClient`
  - `FirewallStatusClient`
  - `LocalRulestacksClient`
  - `LocalRulesClient`
  - `PrefixListLocalRulestackClient`
  - `FqdnListLocalRulestackClient`
  - `CertificateObjectLocalRulestackClient`
  - `GlobalRulestackClient`
  - `PrefixListGlobalRulestackClient`
  - `FqdnListGlobalRulestackClient`
  - `CertificateObjectGlobalRulestackClient`
  - `OperationsClient`
- The future example should therefore prefer typed `armpanngfw/v2` clients and use raw `azcore/runtime` only as a narrow fallback for any reviewed action route the local package revision does not expose cleanly.

## Service Naming

Canonical Stackyard naming:

- service key: `cloudngfw`
- display name: `Azure Cloud NGFW`
- API version token: `2025-10-08`

Recommended repo paths:

- implementation: `internal/server/provider_azure_cloudngfw.go`
- tests: `internal/server/provider_azure_cloudngfw_test.go`
- example: `examples/azure/cloudngfw-2025-10-08`
- plan doc: `docs/cloudngfw-2025-10-08-plan.md`

Recommended coverage aliases:

- `cloudngfw`
- `cloud_ngfw`
- `cloud-ngfw`
- `azure_cloudngfw`
- `paloaltonetworks_cloudngfw`
- `rest_cloudngfw_2025_10_08`
- `cloudngfw_2025_10_08`

## API Surface Summary

The reviewed pages expose four major route trunks:

- provider discovery:
  - `/providers/PaloAltoNetworks.Cloudngfw/operations`
- subscription/resource-group tracked resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/PaloAltoNetworks.Cloudngfw/firewalls/{firewallName}`
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/{localRulestackName}`
- nested local-rulestack resources:
  - `.../localRules/{priority}`
  - `.../prefixlists/{name}`
  - `.../fqdnlists/{name}`
  - `.../certificates/{name}`
- provider/tenant-scoped resources:
  - `/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/{globalRulestackName}`
  - `.../globalRulestacks/{globalRulestackName}/prefixlists/{name}`
  - `.../globalRulestacks/{globalRulestackName}/fqdnlists/{name}`
  - `.../globalRulestacks/{globalRulestackName}/certificates/{name}`

Modeling implications:

- the emulator needs both subscription-scoped and provider-scoped route handling
- `localRulestacks` and `globalRulestacks` share many concepts, so their object models should reuse helpers where practical
- counters, changelog, commit, and revert routes are action-style operations better modeled as deterministic fixtures plus small state transitions
- many create/update operations document long-running semantics, so the implementation can use stable `202 Accepted` responses with deterministic completion state where needed

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value` and often `nextLink`
- tracked resources use standard ARM fields such as:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.*`
- several create operations document `201 Created` with `Azure-AsyncOperation` and `Retry-After`

### Operations

Reviewed method page:

- `Operations.List`

Key traits from the reviewed page:

- route:
  - `/providers/PaloAltoNetworks.Cloudngfw/operations`
- returns `OperationListResult`

Modeling implications:

- expose a deterministic provider operation list matching the routes actually implemented
- do not pad the response with unsupported nested object families just to look complete

### Firewalls and Firewall Status

Reviewed method pages:

- `Firewalls.Create Or Update`
- `Firewalls.Get`
- `Firewalls.List By Resource Group`
- `Firewalls.List By Subscription`
- `Firewall Status.Get`
- `Firewall Status.List By Firewalls`

Key traits from the reviewed pages:

- firewall route:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/PaloAltoNetworks.Cloudngfw/firewalls/{firewallName}`
- `firewallName` pattern:
  - `^(?![-_])(?!.*[-_]{2})(?!.*[-_]$)[a-zA-Z0-9][a-zA-Z0-9-]{0,127}$`
- firewall status is a child collection under `statuses`, with `default` as the documented get route
- firewall operation group also includes:
  - `get Global Rulestack`
  - `get Log Profile`
  - `save Log Profile`
  - `get Support Info`

Modeling implications:

- firewalls should be one canonical tracked-resource store keyed by ARM ID
- status routes can be fixture-driven and derived from firewall state
- log-profile and support/global-rulestack lookup routes are good candidates for deterministic action/read helpers rather than full extra resource types

### Local Rulestacks and Local Rules

Reviewed method pages:

- `Local Rulestacks.Create Or Update`
- `Local Rulestacks.Get`
- `Local Rulestacks.Update`
- `Local Rules.Get`
- `Local Rules.List By Local Rulestacks`
- `Local Rules.get Counters`

Key traits from the reviewed pages:

- local rulestack route:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/PaloAltoNetworks.Cloudngfw/localRulestacks/{localRulestackName}`
- local-rule route:
  - `.../localRules/{priority}`
- local-rule counters are action POSTs with optional `firewallName` query filtering
- local rulestack operation group also includes:
  - `commit`
  - `revert`
  - `get Change Log`
  - `get Support Info`
  - `list Firewalls`
  - catalog routes for countries, app IDs, predefined URL categories, security services, and advanced security objects

Modeling implications:

- local rulestacks should maintain a small mutable revision state so `commit`, `revert`, and changelog can return coherent payloads
- local rules can be keyed by local rulestack plus priority
- counter routes can be deterministic fixtures tied to a rule and optionally shaped by `firewallName`

### Local Object Families

Reviewed method pages:

- `Prefix List Local Rulestack.Get`
- `Prefix List Local Rulestack.List By Local Rulestacks`
- `Fqdn List Local Rulestack.Create Or Update`
- `Fqdn List Local Rulestack.Get`
- `Certificate Object Local Rulestack.Create Or Update`
- `Certificate Object Local Rulestack.Get`
- `Certificate Object Local Rulestack.List By Local Rulestacks`

Key traits from the reviewed pages:

- local object routes are nested under a local rulestack:
  - `prefixlists/{name}`
  - `fqdnlists/{name}`
  - `certificates/{name}`
- the reviewed FQDN and certificate create/update pages confirm typical object properties such as:
  - `description`
  - `auditComment`
  - `etag`
  - family-specific payload fields like `fqdnList` or `certificateSelfSigned`

Modeling implications:

- these families should be stored independently but share the same generic child-resource helper patterns
- FQDN lists and prefix lists are especially good early child-resource candidates because their payloads are straightforward

### Global Rulestacks and Global Object Families

Reviewed method pages:

- `Global Rulestack.List`
- `Fqdn List Global Rulestack.Create Or Update`
- `Fqdn List Global Rulestack.Get`
- `Prefix List Global Rulestack.Get`
- `Certificate Object Global Rulestack.Get`
- `Certificate Object Global Rulestack.List`

Key traits from the reviewed pages:

- global rulestacks are provider-scoped:
  - `/providers/PaloAltoNetworks.Cloudngfw/globalRulestacks/{globalRulestackName}`
- global object families mirror the local ones but omit subscription/resource-group path segments
- the global rulestack operation group also includes:
  - `commit`
  - `revert`
  - `get Change Log`
  - `list Firewalls`
  - catalog routes similar to local rulestacks

Modeling implications:

- a separate provider-scoped store is needed for global rulestacks and their child objects
- helper logic should reuse the same normalization and fixture generation as local child families where possible

## Recommended State Model

Keep canonical stores for:

- firewalls keyed by `{subscriptionId}/{resourceGroup}/{firewallName}`
- firewall statuses keyed by firewall ID plus status name
- local rulestacks keyed by `{subscriptionId}/{resourceGroup}/{localRulestackName}`
- local rules keyed by local rulestack ID plus priority
- local child objects keyed by local rulestack ID plus `{collection}/{name}`
- global rulestacks keyed by `{globalRulestackName}`
- global child objects keyed by global rulestack ID plus `{collection}/{name}`

Recommended adjunct state:

- provider operations fixture
- rulestack revision/changelog state
- firewall log-profile fixtures
- catalog fixtures for countries, app IDs, predefined URL categories, security services, and advanced security objects
- rule counter fixtures

Recommended derived behavior:

- `commit` increments a deterministic revision and appends a changelog entry
- `revert` rolls back to the prior revision if present
- `list Firewalls` under a rulestack derives from firewalls linked to that rulestack
- firewall status derives from firewall provisioning and attachment state rather than random values

## Staged Implementation Plan

### Stage 1: Route Skeleton and Coverage Wiring

Create the initial provider and route registration for:

- provider `operations`
- `firewalls`
- `firewall statuses`
- `localRulestacks`
- tenant-scoped `globalRulestacks`

Include:

- initial `cloudngfw` coverage aliases in the Azure coverage scripts
- server wiring in `internal/server/http.go`
- shared helpers for provider-scoped versus subscription-scoped routes

### Stage 2: Firewalls and Firewall Status

Implement:

- `Operations.List`
- `Firewalls` create, get, list-by-resource-group, list-by-subscription, update, delete
- `Firewall Status` get and list-by-firewalls
- firewall support/read action routes:
  - `get Global Rulestack`
  - `get Log Profile`
  - `save Log Profile`
  - `get Support Info`

Recommended outcome:

- the service is discoverable and top-level firewall lifecycle works
- firewall status and log/support routes return stable deterministic payloads

### Stage 3: Local Rulestacks and Local Action Routes

Implement:

- `Local Rulestacks` create, get, list-by-resource-group, list-by-subscription, update, delete
- `commit`
- `revert`
- `get Change Log`
- `get Support Info`
- catalog routes:
  - `list Firewalls`
  - `list Countries`
  - `list App Ids`
  - `list Predefined Url Categories`
  - `list Security Services`
  - `list Advanced Security Objects`

Recommended outcome:

- local rulestack lifecycle works independently of child objects
- commit/revert/changelog semantics are coherent enough for coverage and examples

### Stage 4: Local Child Objects and Rule Counters

Implement:

- `Local Rules` create, get, list-by-local-rulestacks, delete
- `Local Rules.get Counters`
- `Local Rules.refresh Counters`
- `Local Rules.reset Counters`
- `Prefix List Local Rulestack` create, get, list, delete
- `Fqdn List Local Rulestack` create, get, list, delete
- `Certificate Object Local Rulestack` create, get, list, delete

Recommended outcome:

- the main mutable local child-resource families round-trip correctly
- counters return deterministic values tied to the rule and optional firewall filter

### Stage 5: Global Rulestacks, Global Child Objects, Example, and Final Coverage Validation

Implement:

- `Global Rulestack` create, get, list, update, delete
- `commit`
- `revert`
- `get Change Log`
- `list Firewalls`
- provider-scoped catalog routes matching the reviewed global rulestack surface
- `Prefix List Global Rulestack` create, get, list, delete
- `Fqdn List Global Rulestack` create, get, list, delete
- `Certificate Object Global Rulestack` create, get, list, delete
- the Azure Go SDK example in `examples/azure/cloudngfw-2025-10-08`
- final coverage-script doc mapping

Recommended example flow:

- list provider operations
- create a firewall
- create a local rulestack
- create one FQDN or prefix list under the local rulestack
- commit the local rulestack
- list or get a global rulestack object family using the typed SDK

## Testing Strategy

Target focused lifecycle and validation tests rather than a single oversized end-to-end suite.

Minimum test coverage should include:

- provider operations
- firewall create/get/list/update/delete
- firewall status read routes
- local rulestack create/get/list/update/delete plus commit/revert/changelog
- local rule create/get/list and counter actions
- local FQDN/prefix/certificate child resource create/get/list/delete
- global rulestack list/get and one global child family

Recommended negative cases:

- invalid `firewallName` against the documented regex
- local child-resource create under a missing local rulestack
- global child-resource get under a missing global rulestack
- counter action on a missing local rule
- malformed or unsupported payloads for FQDN list and certificate objects

## Coverage Script Updates To Make During Implementation

Update the following scripts:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for:

- `cloudngfw`
- `cloud_ngfw`
- `cloud-ngfw`
- `azure_cloudngfw`
- `paloaltonetworks_cloudngfw`
- `rest_cloudngfw_2025_10_08`
- `cloudngfw_2025_10_08`

Add the plan doc mapping:

- `cloudngfw`: `cloudngfw-2025-10-08-plan.md`

## Validation Targets For The Future Implementation

After implementation, validate with:

- `go test ./internal/server -run 'TestAzureCloudNGFW' -count=1`
- `go build ./examples/azure/cloudngfw-2025-10-08`
- `python3 scripts/azure-contract-coverage.py --service cloudngfw --require-service cloudngfw --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service cloudngfw --require-service cloudngfw --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service cloudngfw --require-service cloudngfw --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service cloudngfw --require-service cloudngfw --fail-on any`

Expected quality bar:

- no Cloud NGFW endpoint falls through to `NotImplemented`
- the example builds cleanly against the chosen Azure Go SDK package
- the coverage scripts recognize `cloudngfw` and map it to this plan doc
