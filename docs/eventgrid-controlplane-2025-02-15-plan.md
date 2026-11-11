# Azure Event Grid Control Plane (`eventgrid_controlplane_2025_02_15`) Staged Plan

## Objective

Emulate Azure Event Grid Control Plane Resource Manager REST API (`api-version=2025-02-15`) with deterministic local behavior for:

- namespace, namespace-topic, and event-subscription management surfaces
- domain, topic, and system-topic lifecycle and key-management surfaces
- partner resource families (registrations, namespaces, topics, channels, configs)
- private endpoint / private link and permission-binding surfaces
- topic-space, client, and CA-certificate surfaces
- provider operations and verified-partner discovery surfaces
- an Azure Go SDK example under `examples/azure/eventgrid-controlplane-2025-02-15`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/operation-groups?view=rest-eventgrid-controlplane-2025-02-15`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/ca-certificates?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/client-groups?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/clients?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topics?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/extension-topics?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/operations?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/permission-bindings?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-endpoint-connections?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-link-resources?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-spaces?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-types?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/verified-partners?view=rest-eventgrid-controlplane-2025-02-15`

Method pages reviewed directly: all linked methods from all operation groups (`170` total).

Ca Certificates methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/ca-certificates/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/ca-certificates/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/ca-certificates/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/ca-certificates/list-by-namespace?view=rest-eventgrid-controlplane-2025-02-15`

Channels methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels/list-by-partner-namespace?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/channels/update?view=rest-eventgrid-controlplane-2025-02-15`

Client Groups methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/client-groups/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/client-groups/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/client-groups/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/client-groups/list-by-namespace?view=rest-eventgrid-controlplane-2025-02-15`

Clients methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/clients/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/clients/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/clients/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/clients/list-by-namespace?view=rest-eventgrid-controlplane-2025-02-15`

Domain Event Subscriptions methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/list?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

Domain Topic Event Subscriptions methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/list?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topic-event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

Domain Topics methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topics/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topics/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topics/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domain-topics/list-by-domain?view=rest-eventgrid-controlplane-2025-02-15`

Domains methods (`8`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/list-shared-access-keys?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/regenerate-key?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/domains/update?view=rest-eventgrid-controlplane-2025-02-15`

Event Subscriptions methods (`16`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-by-domain-topic?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-by-resource?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-global-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-global-by-resource-group-for-topic-type?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-global-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-global-by-subscription-for-topic-type?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-regional-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-regional-by-resource-group-for-topic-type?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-regional-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/list-regional-by-subscription-for-topic-type?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

Extension Topics methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/extension-topics/get?view=rest-eventgrid-controlplane-2025-02-15`

Namespace Topic Event Subscriptions methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/list-by-namespace-topic?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topic-event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

Namespace Topics methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/list-by-namespace?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/list-shared-access-keys?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/regenerate-key?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespace-topics/update?view=rest-eventgrid-controlplane-2025-02-15`

Namespaces methods (`9`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/list-shared-access-keys?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/regenerate-key?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/namespaces/validate-custom-domain-ownership?view=rest-eventgrid-controlplane-2025-02-15`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/operations/list?view=rest-eventgrid-controlplane-2025-02-15`

Partner Configurations methods (`8`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/authorize-partner?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/unauthorize-partner?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-configurations/update?view=rest-eventgrid-controlplane-2025-02-15`

Partner Namespaces methods (`8`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/list-shared-access-keys?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/regenerate-key?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-namespaces/update?view=rest-eventgrid-controlplane-2025-02-15`

Partner Registrations methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-registrations/update?view=rest-eventgrid-controlplane-2025-02-15`

Partner Topic Event Subscriptions methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/list-by-partner-topic?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topic-event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

Partner Topics methods (`8`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/activate?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/deactivate?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/partner-topics/update?view=rest-eventgrid-controlplane-2025-02-15`

Permission Bindings methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/permission-bindings/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/permission-bindings/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/permission-bindings/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/permission-bindings/list-by-namespace?view=rest-eventgrid-controlplane-2025-02-15`

Private Endpoint Connections methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-endpoint-connections/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-endpoint-connections/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-endpoint-connections/list-by-resource?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-endpoint-connections/update?view=rest-eventgrid-controlplane-2025-02-15`

Private Link Resources methods (`2`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-link-resources/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/private-link-resources/list-by-resource?view=rest-eventgrid-controlplane-2025-02-15`

System Topic Event Subscriptions methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/list-by-system-topic?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topic-event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

System Topics methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/system-topics/update?view=rest-eventgrid-controlplane-2025-02-15`

Topic Event Subscriptions methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/get-delivery-attributes?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/get-full-url?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/list?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-event-subscriptions/update?view=rest-eventgrid-controlplane-2025-02-15`

Topic Spaces methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-spaces/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-spaces/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-spaces/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-spaces/list-by-namespace?view=rest-eventgrid-controlplane-2025-02-15`

Topic Types methods (`3`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-types/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-types/list?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topic-types/list-event-types?view=rest-eventgrid-controlplane-2025-02-15`

Topics methods (`9`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/create-or-update?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/delete?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/list-by-resource-group?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/list-by-subscription?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/list-event-types?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/list-shared-access-keys?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/regenerate-key?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/topics/update?view=rest-eventgrid-controlplane-2025-02-15`

Verified Partners methods (`2`):

- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/verified-partners/get?view=rest-eventgrid-controlplane-2025-02-15`
- `https://learn.microsoft.com/en-us/rest/api/eventgrid/controlplane/verified-partners/list?view=rest-eventgrid-controlplane-2025-02-15`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventgrid/armeventgrid`

## Surface Summary

Reviewed operation groups and methods:

- Ca Certificates (4)
- Channels (6)
- Client Groups (4)
- Clients (4)
- Domain Event Subscriptions (7)
- Domain Topic Event Subscriptions (7)
- Domain Topics (4)
- Domains (8)
- Event Subscriptions (16)
- Extension Topics (1)
- Namespace Topic Event Subscriptions (7)
- Namespace Topics (7)
- Namespaces (9)
- Operations (1)
- Partner Configurations (8)
- Partner Namespaces (8)
- Partner Registrations (6)
- Partner Topic Event Subscriptions (7)
- Partner Topics (8)
- Permission Bindings (4)
- Private Endpoint Connections (4)
- Private Link Resources (2)
- System Topic Event Subscriptions (7)
- System Topics (6)
- Topic Event Subscriptions (7)
- Topic Spaces (4)
- Topic Types (3)
- Topics (9)
- Verified Partners (2)

Total reviewed methods: `170`.

## Service Naming

Canonical Stackyard naming:

- service key: `eventgrid_controlplane_2025_02_15`
- display name: `Azure Event Grid Control Plane`
- API version: `2025-02-15`

Recommended repo paths:

- implementation: `internal/server/provider_azure_eventgrid_controlplane_2025_02_15.go`
- tests: `internal/server/provider_azure_eventgrid_controlplane_2025_02_15_test.go`
- example: `examples/azure/eventgrid-controlplane-2025-02-15`
- plan doc: `docs/eventgrid-controlplane-2025-02-15-plan.md`

## Route Envelope

Management-plane route families to emulate include:

- `/azure/providers/Microsoft.EventGrid/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/...`
- `/azure/{scope}/providers/Microsoft.EventGrid/...` for scope-based event-subscription APIs

Representative routes:

- `/providers/Microsoft.EventGrid/operations?api-version=2025-02-15`
- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}?api-version=2025-02-15`
- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}?api-version=2025-02-15`
- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}?api-version=2025-02-15`
- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}?api-version=2025-02-15`
- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels/{channelName}?api-version=2025-02-15`
- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}?api-version=2025-02-15`
- `/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-02-15`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- reviewed methods document non-success paths as `Other Status Codes` with Event Grid error envelopes

Observed response patterns:

- `200` appears in `156` methods
- `201` appears in `36` methods
- `202` appears in `32` methods

Request-body coverage from reviewed pages:

- `47` methods include request bodies
- body-bearing methods:
  - `ca-certificates/create-or-update`
  - `channels/create-or-update`
  - `channels/update`
  - `client-groups/create-or-update`
  - `clients/create-or-update`
  - `domain-event-subscriptions/create-or-update`
  - `domain-event-subscriptions/update`
  - `domain-topic-event-subscriptions/create-or-update`
  - `domain-topic-event-subscriptions/update`
  - `domains/create-or-update`
  - `domains/regenerate-key`
  - `domains/update`
  - `event-subscriptions/create-or-update`
  - `event-subscriptions/update`
  - `namespace-topic-event-subscriptions/create-or-update`
  - `namespace-topic-event-subscriptions/update`
  - `namespace-topics/create-or-update`
  - `namespace-topics/regenerate-key`
  - `namespace-topics/update`
  - `namespaces/create-or-update`
  - `namespaces/regenerate-key`
  - `namespaces/update`
  - `partner-configurations/authorize-partner`
  - `partner-configurations/create-or-update`
  - `partner-configurations/unauthorize-partner`
  - `partner-configurations/update`
  - `partner-namespaces/create-or-update`
  - `partner-namespaces/regenerate-key`
  - `partner-namespaces/update`
  - `partner-registrations/create-or-update`
  - `partner-registrations/update`
  - `partner-topic-event-subscriptions/create-or-update`
  - `partner-topic-event-subscriptions/update`
  - `partner-topics/create-or-update`
  - `partner-topics/update`
  - `permission-bindings/create-or-update`
  - `private-endpoint-connections/update`
  - `system-topic-event-subscriptions/create-or-update`
  - `system-topic-event-subscriptions/update`
  - `system-topics/create-or-update`
  - `system-topics/update`
  - `topic-event-subscriptions/create-or-update`
  - `topic-event-subscriptions/update`
  - `topic-spaces/create-or-update`
  - `topics/create-or-update`
  - `topics/regenerate-key`
  - `topics/update`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add stage-0 provider skeleton at `internal/server/provider_azure_eventgrid_controlplane_2025_02_15.go`
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key resolves as `eventgrid_controlplane_2025_02_15`
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.EventGrid` route families for this API version
- enforce `api-version=2025-02-15` for handled routes
- preserve behavior for unrelated Azure providers

### Stage 2: Read/List and Discovery Surfaces

- implement `Operations_List` and other discovery/list/get methods across all operation groups
- implement key and URL retrieval actions (`listKeys`, `regenerateKey`, `getFullUrl`, `getDeliveryAttributes`)
- implement private-link and verified-partner read/list surfaces

### Stage 3: Mutating Surfaces

- implement create/update/delete lifecycles across namespaces, domains, topics, partner resources, clients, and certificates
- implement authorization and activation/deactivation actions
- implement scope-based and resource-based event-subscription mutations
- preserve documented `200`/`201`/`202` patterns per method contract

### Stage 4: Payload and Query Validation

- validate body requirements for create/update/action operations
- validate identifier segments and scope-based routing parameters
- validate key/action payload enums and required fields

### Stage 5: Tests

- add provider tests `internal/server/provider_azure_eventgrid_controlplane_2025_02_15_test.go` covering:
  - route ownership and API-version validation
  - read/list/discovery endpoints by operation group
  - mutation flows with expected status codes and envelopes
  - deterministic negative-contract behavior
