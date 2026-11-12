package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const eventGridControlPlanePrefix = "/azure"

type eventGridControlPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type eventGridControlPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type eventGridControlPlaneCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *eventGridControlPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEventGridControlPlaneClient(endpoint, account, subscriptionKey string) *eventGridControlPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"eventgrid-controlplane-2025-02-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&eventGridControlPlaneSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &eventGridControlPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *eventGridControlPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func buildPath(template string, vars map[string]string) string {
	path := template
	for key, value := range vars {
		replacement := url.PathEscape(value)
		if key == "scope" {
			replacement = strings.TrimPrefix(value, "/")
		}
		path = strings.ReplaceAll(path, "{"+key+"}", replacement)
	}
	return eventGridControlPlanePrefix + path
}

func payloadForMethod(methodPath, location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName string) any {
	resourceCreate := map[string]any{
		"location": location,
		"tags": map[string]any{
			"env":     "local",
			"service": "eventgrid",
		},
		"properties": map[string]any{},
	}

	resourceUpdate := map[string]any{
		"tags": map[string]any{
			"env": "local-updated",
		},
		"properties": map[string]any{},
	}

	eventSubscriptionCreate := map[string]any{
		"properties": map[string]any{
			"destination": map[string]any{
				"endpointType": "WebHook",
				"properties": map[string]any{
					"endpointUrl": webhookEndpoint,
				},
			},
			"filter": map[string]any{
				"isSubjectCaseSensitive": false,
				"subjectBeginsWith":      "",
				"subjectEndsWith":        "",
			},
		},
	}

	eventSubscriptionUpdate := map[string]any{
		"properties": map[string]any{
			"destination": map[string]any{
				"endpointType": "WebHook",
				"properties": map[string]any{
					"endpointUrl": webhookEndpoint,
				},
			},
		},
	}

	privateEndpointConnectionUpdate := map[string]any{
		"properties": map[string]any{
			"privateEndpoint": map[string]any{
				"id": privateEndpointID,
			},
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Approved",
				"description":     "approving connection",
				"actionsRequired": "None",
			},
		},
	}

	partnerAuthorizationPayload := map[string]any{
		"partnerRegistrationImmutableId":   partnerRegistrationImmutableID,
		"partnerName":                      partnerName,
		"authorizationExpirationTimeInUtc": "2027-01-28T01:20:55.142Z",
	}

	regenerateKeyPayload := map[string]any{
		"keyName": "key1",
	}

	switch methodPath {
	case "partner-configurations/authorize-partner", "partner-configurations/unauthorize-partner":
		return partnerAuthorizationPayload
	case "private-endpoint-connections/update":
		return privateEndpointConnectionUpdate
	case "domains/regenerate-key", "namespace-topics/regenerate-key", "namespaces/regenerate-key", "partner-namespaces/regenerate-key", "topics/regenerate-key":
		return regenerateKeyPayload
	}

	if strings.Contains(methodPath, "event-subscriptions/") {
		if strings.HasSuffix(methodPath, "/create-or-update") {
			return eventSubscriptionCreate
		}
		if strings.HasSuffix(methodPath, "/update") {
			return eventSubscriptionUpdate
		}
	}

	if strings.HasSuffix(methodPath, "/create-or-update") {
		return resourceCreate
	}
	if strings.HasSuffix(methodPath, "/update") {
		return resourceUpdate
	}

	return map[string]any{}
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EVENTGRID_CONTROLPLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EVENTGRID_CONTROLPLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-eventgrid")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus2")
	namespaceName := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_NAME", "eventgrid-namespace01")
	caCertificateName := getenv("STACKYARD_AZURE_EVENTGRID_CA_CERTIFICATE_NAME", "cacert01")
	partnerNamespaceName := getenv("STACKYARD_AZURE_EVENTGRID_PARTNER_NAMESPACE_NAME", "partnernamespace01")
	channelName := getenv("STACKYARD_AZURE_EVENTGRID_CHANNEL_NAME", "channel01")
	clientGroupName := getenv("STACKYARD_AZURE_EVENTGRID_CLIENT_GROUP_NAME", "clientgroup01")
	clientName := getenv("STACKYARD_AZURE_EVENTGRID_CLIENT_NAME", "client01")
	domainName := getenv("STACKYARD_AZURE_EVENTGRID_DOMAIN_NAME", "eventgriddomain01")
	domainTopicName := getenv("STACKYARD_AZURE_EVENTGRID_DOMAIN_TOPIC_NAME", "domaintopic01")
	eventSubscriptionName := getenv("STACKYARD_AZURE_EVENTGRID_EVENT_SUBSCRIPTION_NAME", "eventsub01")
	topicName := getenv("STACKYARD_AZURE_EVENTGRID_TOPIC_NAME", "eventgridtopic01")
	topicSpaceName := getenv("STACKYARD_AZURE_EVENTGRID_TOPIC_SPACE_NAME", "topicspace01")
	topicTypeName := getenv("STACKYARD_AZURE_EVENTGRID_TOPIC_TYPE_NAME", "Microsoft.Storage.StorageAccounts")
	permissionBindingName := getenv("STACKYARD_AZURE_EVENTGRID_PERMISSION_BINDING_NAME", "permissionbinding01")
	partnerRegistrationName := getenv("STACKYARD_AZURE_EVENTGRID_PARTNER_REGISTRATION_NAME", "partnerregistration01")
	partnerTopicName := getenv("STACKYARD_AZURE_EVENTGRID_PARTNER_TOPIC_NAME", "partnertopic01")
	systemTopicName := getenv("STACKYARD_AZURE_EVENTGRID_SYSTEM_TOPIC_NAME", "systemtopic01")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_EVENTGRID_PRIVATE_ENDPOINT_CONNECTION_NAME", "connection01")
	privateLinkResourceName := getenv("STACKYARD_AZURE_EVENTGRID_PRIVATE_LINK_RESOURCE_NAME", "topic")
	verifiedPartnerName := getenv("STACKYARD_AZURE_EVENTGRID_VERIFIED_PARTNER_NAME", "Contoso")
	providerNamespace := getenv("STACKYARD_AZURE_EVENTGRID_PROVIDER_NAMESPACE", "Microsoft.Storage")
	resourceTypeName := getenv("STACKYARD_AZURE_EVENTGRID_RESOURCE_TYPE_NAME", "storageAccounts")
	resourceName := getenv("STACKYARD_AZURE_EVENTGRID_RESOURCE_NAME", "storageaccount01")
	parentType := getenv("STACKYARD_AZURE_EVENTGRID_PARENT_TYPE", "topics")
	parentName := getenv("STACKYARD_AZURE_EVENTGRID_PARENT_NAME", topicName)
	webhookEndpoint := getenv("STACKYARD_AZURE_EVENTGRID_WEBHOOK_ENDPOINT", "https://example.com/eventgrid-hook")
	partnerRegistrationImmutableID := getenv("STACKYARD_AZURE_EVENTGRID_PARTNER_REGISTRATION_IMMUTABLE_ID", "941892bc-f5d0-4d1c-8fb5-477570fc2b71")
	partnerName := getenv("STACKYARD_AZURE_EVENTGRID_PARTNER_NAME", "Contoso.Finance")
	privateEndpointID := getenv(
		"STACKYARD_AZURE_EVENTGRID_PRIVATE_ENDPOINT_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/eventgrid-pe", subscriptionID, resourceGroupName),
	)
	scope := getenv(
		"STACKYARD_AZURE_EVENTGRID_SCOPE",
		fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/Microsoft.EventGrid/topics/%s", subscriptionID, resourceGroupName, topicName),
	)

	client := newEventGridControlPlaneClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":                subscriptionID,
		"resourceGroupName":             resourceGroupName,
		"location":                      location,
		"namespaceName":                 namespaceName,
		"caCertificateName":             caCertificateName,
		"partnerNamespaceName":          partnerNamespaceName,
		"channelName":                   channelName,
		"clientGroupName":               clientGroupName,
		"clientName":                    clientName,
		"domainName":                    domainName,
		"domainTopicName":               domainTopicName,
		"eventSubscriptionName":         eventSubscriptionName,
		"topicName":                     topicName,
		"topicSpaceName":                topicSpaceName,
		"topicTypeName":                 topicTypeName,
		"permissionBindingName":         permissionBindingName,
		"partnerRegistrationName":       partnerRegistrationName,
		"partnerTopicName":              partnerTopicName,
		"systemTopicName":               systemTopicName,
		"privateEndpointConnectionName": privateEndpointConnectionName,
		"privateLinkResourceName":       privateLinkResourceName,
		"verifiedPartnerName":           verifiedPartnerName,
		"providerNamespace":             providerNamespace,
		"resourceTypeName":              resourceTypeName,
		"resourceName":                  resourceName,
		"parentType":                    parentType,
		"parentName":                    parentName,
		"scope":                         scope,
	}

	calls := []eventGridControlPlaneCall{
		{
			name:     "ca_certificates_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/caCertificates/{caCertificateName}?api-version=2025-02-15",
			payload:  payloadForMethod("ca-certificates/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "ca_certificates_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/caCertificates/{caCertificateName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "ca_certificates_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/caCertificates/{caCertificateName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "ca_certificates_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/caCertificates?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "channels_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels/{channelName}?api-version=2025-02-15",
			payload:  payloadForMethod("channels/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "channels_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels/{channelName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "channels_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels/{channelName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "channels_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels/{channelName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "channels_list_by_partner_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "channels_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/channels/{channelName}?api-version=2025-02-15",
			payload:  payloadForMethod("channels/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200},
		},
		{
			name:     "client_groups_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clientGroups/{clientGroupName}?api-version=2025-02-15",
			payload:  payloadForMethod("client-groups/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "client_groups_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clientGroups/{clientGroupName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "client_groups_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clientGroups/{clientGroupName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "client_groups_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clientGroups?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clients_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clients/{clientName}?api-version=2025-02-15",
			payload:  payloadForMethod("clients/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "clients_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clients/{clientName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "clients_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clients/{clientName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clients_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/clients?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("domain-event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "domain_event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "domain_event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_event_subscriptions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_event_subscriptions_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("domain-event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "domain_topic_event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("domain-topic-event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "domain_topic_event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "domain_topic_event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_topic_event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_topic_event_subscriptions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_topic_event_subscriptions_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_topic_event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("domain-topic-event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "domain_topics_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{domainTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{201},
		},
		{
			name:     "domain_topics_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{domainTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "domain_topics_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{domainTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domain_topics_list_by_domain",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domains_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}?api-version=2025-02-15",
			payload:  payloadForMethod("domains/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "domains_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "domains_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domains_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domains_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/domains?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domains_list_shared_access_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/listKeys?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "domains_regenerate_key",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/regenerateKey?api-version=2025-02-15",
			payload:  payloadForMethod("domains/regenerate-key", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200},
		},
		{
			name:     "domains_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}?api-version=2025-02-15",
			payload:  payloadForMethod("domains/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_get",
			method:   http.MethodGet,
			template: "/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_by_domain_topic",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/domains/{domainName}/topics/{topicName}/providers/Microsoft.EventGrid/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_by_resource",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{providerNamespace}/{resourceTypeName}/{resourceName}/providers/Microsoft.EventGrid/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_global_by_resource_group_for_topic_type",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topicTypes/{topicTypeName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_global_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_global_by_subscription_for_topic_type",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/topicTypes/{topicTypeName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_global_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_regional_by_resource_group_for_topic_type",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/locations/{location}/topicTypes/{topicTypeName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_regional_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/locations/{location}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_regional_by_subscription_for_topic_type",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/locations/{location}/topicTypes/{topicTypeName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_list_regional_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/locations/{location}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/{scope}/providers/Microsoft.EventGrid/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "extension_topics_get",
			method:   http.MethodGet,
			template: "/{scope}/providers/Microsoft.EventGrid/extensionTopics/default?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topic_event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("namespace-topic-event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "namespace_topic_event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "namespace_topic_event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topic_event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topic_event_subscriptions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topic_event_subscriptions_list_by_namespace_topic",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topic_event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("namespace-topic-event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "namespace_topics_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}?api-version=2025-02-15",
			payload:  payloadForMethod("namespace-topics/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "namespace_topics_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "namespace_topics_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topics_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topics_list_shared_access_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/listKeys?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespace_topics_regenerate_key",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}/regenerateKey?api-version=2025-02-15",
			payload:  payloadForMethod("namespace-topics/regenerate-key", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "namespace_topics_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topics/{topicName}?api-version=2025-02-15",
			payload:  payloadForMethod("namespace-topics/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "namespaces_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}?api-version=2025-02-15",
			payload:  payloadForMethod("namespaces/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "namespaces_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "namespaces_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/namespaces?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_shared_access_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/listKeys?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_regenerate_key",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/regenerateKey?api-version=2025-02-15",
			payload:  payloadForMethod("namespaces/regenerate-key", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "namespaces_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}?api-version=2025-02-15",
			payload:  payloadForMethod("namespaces/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "namespaces_validate_custom_domain_ownership",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/validateCustomDomainOwnership?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventGrid/operations?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_configurations_authorize_partner",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations/default/authorizePartner?api-version=2025-02-15",
			payload:  payloadForMethod("partner-configurations/authorize-partner", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200},
		},
		{
			name:     "partner_configurations_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations/default?api-version=2025-02-15",
			payload:  payloadForMethod("partner-configurations/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "partner_configurations_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations/default?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "partner_configurations_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations/default?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_configurations_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_configurations_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/partnerConfigurations?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_configurations_unauthorize_partner",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations/default/unauthorizePartner?api-version=2025-02-15",
			payload:  payloadForMethod("partner-configurations/unauthorize-partner", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200},
		},
		{
			name:     "partner_configurations_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerConfigurations/default?api-version=2025-02-15",
			payload:  payloadForMethod("partner-configurations/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "partner_namespaces_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-namespaces/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "partner_namespaces_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "partner_namespaces_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_namespaces_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_namespaces_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/partnerNamespaces?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_namespaces_list_shared_access_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/listKeys?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_namespaces_regenerate_key",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}/regenerateKey?api-version=2025-02-15",
			payload:  payloadForMethod("partner-namespaces/regenerate-key", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200},
		},
		{
			name:     "partner_namespaces_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerNamespaces/{partnerNamespaceName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-namespaces/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "partner_registrations_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerRegistrations/{partnerRegistrationName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-registrations/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "partner_registrations_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerRegistrations/{partnerRegistrationName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "partner_registrations_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerRegistrations/{partnerRegistrationName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_registrations_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerRegistrations?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_registrations_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/partnerRegistrations?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_registrations_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerRegistrations/{partnerRegistrationName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-registrations/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "partner_topic_event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-topic-event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "partner_topic_event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "partner_topic_event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topic_event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topic_event_subscriptions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topic_event_subscriptions_list_by_partner_topic",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topic_event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-topic-event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "partner_topics_activate",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/activate?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topics_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-topics/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "partner_topics_deactivate",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}/deactivate?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topics_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "partner_topics_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topics_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topics_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/partnerTopics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "partner_topics_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/partnerTopics/{partnerTopicName}?api-version=2025-02-15",
			payload:  payloadForMethod("partner-topics/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "permission_bindings_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/permissionBindings/{permissionBindingName}?api-version=2025-02-15",
			payload:  payloadForMethod("permission-bindings/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "permission_bindings_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/permissionBindings/{permissionBindingName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "permission_bindings_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/permissionBindings/{permissionBindingName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "permission_bindings_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/permissionBindings?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_endpoint_connections_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/{parentType}/{parentName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{202},
		},
		{
			name:     "private_endpoint_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/{parentType}/{parentName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_endpoint_connections_list_by_resource",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/{parentType}/{parentName}/privateEndpointConnections?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_endpoint_connections_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/{parentType}/{parentName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-02-15",
			payload:  payloadForMethod("private-endpoint-connections/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "private_link_resources_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/{parentType}/{parentName}/privateLinkResources/{privateLinkResourceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_link_resources_list_by_resource",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/{parentType}/{parentName}/privateLinkResources?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topic_event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("system-topic-event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "system_topic_event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "system_topic_event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topic_event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topic_event_subscriptions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topic_event_subscriptions_list_by_system_topic",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topic_event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("system-topic-event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "system_topics_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}?api-version=2025-02-15",
			payload:  payloadForMethod("system-topics/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "system_topics_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "system_topics_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topics_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topics_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/systemTopics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "system_topics_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/systemTopics/{systemTopicName}?api-version=2025-02-15",
			payload:  payloadForMethod("system-topics/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "topic_event_subscriptions_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("topic-event-subscriptions/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "topic_event_subscriptions_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "topic_event_subscriptions_get_delivery_attributes",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}/getDeliveryAttributes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_event_subscriptions_get_full_url",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}/getFullUrl?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_event_subscriptions_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_event_subscriptions_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_event_subscriptions_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/eventSubscriptions/{eventSubscriptionName}?api-version=2025-02-15",
			payload:  payloadForMethod("topic-event-subscriptions/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "topic_spaces_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topicSpaces/{topicSpaceName}?api-version=2025-02-15",
			payload:  payloadForMethod("topic-spaces/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "topic_spaces_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topicSpaces/{topicSpaceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "topic_spaces_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topicSpaces/{topicSpaceName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_spaces_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/namespaces/{namespaceName}/topicSpaces?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_types_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventGrid/topicTypes/{topicTypeName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_types_list_event_types",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventGrid/topicTypes/{topicTypeName}/eventTypes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topic_types_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventGrid/topicTypes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topics_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}?api-version=2025-02-15",
			payload:  payloadForMethod("topics/create-or-update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{201},
		},
		{
			name:     "topics_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{202},
		},
		{
			name:     "topics_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topics_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topics_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventGrid/topics?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topics_list_event_types",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{providerNamespace}/{resourceTypeName}/{resourceName}/providers/Microsoft.EventGrid/eventTypes?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topics_list_shared_access_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/listKeys?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "topics_regenerate_key",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}/regenerateKey?api-version=2025-02-15",
			payload:  payloadForMethod("topics/regenerate-key", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 202},
		},
		{
			name:     "topics_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventGrid/topics/{topicName}?api-version=2025-02-15",
			payload:  payloadForMethod("topics/update", location, webhookEndpoint, privateEndpointID, partnerRegistrationImmutableID, partnerName),
			statuses: []int{200, 201},
		},
		{
			name:     "verified_partners_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventGrid/verifiedPartners/{verifiedPartnerName}?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "verified_partners_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventGrid/verifiedPartners?api-version=2025-02-15",
			payload:  nil,
			statuses: []int{200},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d\n", call.name, status)
		if len(response) > 0 {
			encoded, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s marshal response: %v\n", call.name, err)
				os.Exit(1)
			}
			fmt.Println(string(encoded))
		}
	}
}

func ensureStatus(actual int, body []byte, expected ...int) error {
	for _, code := range expected {
		if actual == code {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d, expected one of %v, body: %s", actual, expected, strings.TrimSpace(string(body)))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
