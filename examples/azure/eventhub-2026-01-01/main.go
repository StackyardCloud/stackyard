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

const eventHubAPIVersion = "2026-01-01"
const eventHubPrefix = "/azure"

type eventHubClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type eventHubSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type eventHubCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *eventHubSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEventHubClient(endpoint, account, subscriptionKey string) *eventHubClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"eventhub-2026-01-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&eventHubSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &eventHubClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *eventHubClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
	}
	return eventHubPrefix + path
}

func payloadForMethod(methodPath, location, partnerNamespaceID, privateEndpointID string) any {
	skuStandard := map[string]any{"name": "Standard", "tier": "Standard"}
	resourceTags := map[string]any{"env": "local", "service": "eventhub"}

	switch methodPath {
	case "application-group/create-or-update-application-group":
		return map[string]any{
			"properties": map[string]any{
				"clientAppGroupIdentifier": "SASKeyName=RootManageSharedAccessKey",
				"isEnabled":                true,
				"policies": []any{
					map[string]any{
						"name":               "throttle-policy",
						"type":               "ThrottlingPolicy",
						"rateLimitThreshold": 1000,
					},
				},
			},
		}
	case "clusters/create-or-update":
		return map[string]any{
			"location":   location,
			"sku":        map[string]any{"name": "Dedicated_1", "capacity": 1},
			"tags":       resourceTags,
			"properties": map[string]any{"supportsScaling": true, "zoneRedundant": false},
		}
	case "clusters/update":
		return map[string]any{
			"sku":        map[string]any{"name": "Dedicated_1", "capacity": 1},
			"tags":       map[string]any{"env": "local-updated"},
			"properties": map[string]any{"supportsScaling": true},
		}
	case "configuration/patch":
		return map[string]any{"settings": map[string]any{"clusterSku": "Dedicated_1"}}
	case "consumer-groups/create-or-update":
		return map[string]any{"properties": map[string]any{"userMetadata": "consumer-group-local"}}
	case "disaster-recovery-configs/check-name-availability", "namespaces/check-name-availability":
		return map[string]any{"name": "eventhub-name-check"}
	case "disaster-recovery-configs/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"alternateName":    "eventhub-dr-alias-alt",
				"partnerNamespace": partnerNamespaceID,
			},
		}
	case "event-hubs/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"partitionCount":         2,
				"messageRetentionInDays": 1,
				"status":                 "Active",
				"userMetadata":           "eventhub-local",
			},
		}
	case "event-hubs/create-or-update-authorization-rule", "namespaces/create-or-update-authorization-rule":
		return map[string]any{"properties": map[string]any{"rights": []any{"Listen", "Send"}}}
	case "event-hubs/regenerate-keys", "namespaces/regenerate-keys":
		return map[string]any{"keyType": "PrimaryKey"}
	case "namespaces/create-or-update":
		return map[string]any{
			"location": location,
			"sku":      skuStandard,
			"tags":     resourceTags,
			"properties": map[string]any{
				"isAutoInflateEnabled":   true,
				"maximumThroughputUnits": 4,
				"kafkaEnabled":           true,
			},
		}
	case "namespaces/create-or-update-network-rule-set":
		return map[string]any{
			"properties": map[string]any{
				"defaultAction":               "Allow",
				"publicNetworkAccess":         "Enabled",
				"trustedServiceAccessEnabled": true,
				"ipRules":                     []any{map[string]any{"ipMask": "10.0.0.0/24", "action": "Allow"}},
			},
		}
	case "namespaces/failover":
		return map[string]any{"properties": map[string]any{"force": false, "primaryLocation": location}}
	case "namespaces/update":
		return map[string]any{"tags": map[string]any{"env": "local-updated"}, "properties": map[string]any{"kafkaEnabled": true}}
	case "private-endpoint-connections/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"privateEndpoint": map[string]any{"id": privateEndpointID},
				"privateLinkServiceConnectionState": map[string]any{
					"status":          "Approved",
					"description":     "approved by local example",
					"actionsRequired": "None",
				},
			},
		}
	case "schema-registry/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"schemaType":          "Avro",
				"schemaCompatibility": "Forward",
				"groupProperties":     map[string]any{"team": "platform"},
			},
		}
	}
	return map[string]any{}
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EVENTHUB_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EVENTHUB_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-eventhub")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	namespaceName := getenv("STACKYARD_AZURE_EVENTHUB_NAMESPACE_NAME", "eventhub-namespace01")
	eventHubName := getenv("STACKYARD_AZURE_EVENTHUB_NAME", "eventhub01")
	consumerGroupName := getenv("STACKYARD_AZURE_EVENTHUB_CONSUMER_GROUP_NAME", "cg01")
	authorizationRuleName := getenv("STACKYARD_AZURE_EVENTHUB_AUTHORIZATION_RULE_NAME", "authrule01")
	applicationGroupName := getenv("STACKYARD_AZURE_EVENTHUB_APPLICATION_GROUP_NAME", "appgroup01")
	clusterName := getenv("STACKYARD_AZURE_EVENTHUB_CLUSTER_NAME", "ehcluster01")
	alias := getenv("STACKYARD_AZURE_EVENTHUB_ALIAS", "ehalias01")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_EVENTHUB_PRIVATE_ENDPOINT_CONNECTION_NAME", "connection01")
	schemaGroupName := getenv("STACKYARD_AZURE_EVENTHUB_SCHEMA_GROUP_NAME", "schemagroup01")
	resourceAssociationName := getenv("STACKYARD_AZURE_EVENTHUB_RESOURCE_ASSOCIATION_NAME", "assoc01")

	partnerNamespaceID := getenv(
		"STACKYARD_AZURE_EVENTHUB_PARTNER_NAMESPACE_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.EventHub/namespaces/%s-secondary", subscriptionID, resourceGroupName, namespaceName),
	)
	privateEndpointID := getenv(
		"STACKYARD_AZURE_EVENTHUB_PRIVATE_ENDPOINT_ID",
		fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/eventhub-pe", subscriptionID, resourceGroupName),
	)

	client := newEventHubClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":                subscriptionID,
		"resourceGroupName":             resourceGroupName,
		"namespaceName":                 namespaceName,
		"eventHubName":                  eventHubName,
		"consumerGroupName":             consumerGroupName,
		"authorizationRuleName":         authorizationRuleName,
		"applicationGroupName":          applicationGroupName,
		"clusterName":                   clusterName,
		"alias":                         alias,
		"privateEndpointConnectionName": privateEndpointConnectionName,
		"schemaGroupName":               schemaGroupName,
		"resourceAssociationName":       resourceAssociationName,
	}

	calls := []eventHubCall{
		{
			name:     "application_group_create_or_update_application_group",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/applicationGroups/{applicationGroupName}?api-version=2026-01-01",
			payload:  payloadForMethod("application-group/create-or-update-application-group", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "application_group_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/applicationGroups/{applicationGroupName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "application_group_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/applicationGroups/{applicationGroupName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "application_group_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/applicationGroups?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clusters_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}?api-version=2026-01-01",
			payload:  payloadForMethod("clusters/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201, 202},
		},
		{
			name:     "clusters_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "clusters_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clusters_list_available_cluster_region",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventHub/availableClusterRegions?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clusters_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clusters_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventHub/clusters?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clusters_list_namespaces",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}/namespaces?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "clusters_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}?api-version=2026-01-01",
			payload:  payloadForMethod("clusters/update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201, 202},
		},
		{
			name:     "configuration_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}/quotaConfiguration/default?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "configuration_patch",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/clusters/{clusterName}/quotaConfiguration/default?api-version=2026-01-01",
			payload:  payloadForMethod("configuration/patch", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201, 202},
		},
		{
			name:     "consumer_groups_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/consumergroups/{consumerGroupName}?api-version=2026-01-01",
			payload:  payloadForMethod("consumer-groups/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "consumer_groups_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/consumergroups/{consumerGroupName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "consumer_groups_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/consumergroups/{consumerGroupName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "consumer_groups_list_by_event_hub",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/consumergroups?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_break_pairing",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}/breakPairing?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_check_name_availability",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/checkNameAvailability?api-version=2026-01-01",
			payload:  payloadForMethod("disaster-recovery-configs/check-name-availability", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}?api-version=2026-01-01",
			payload:  payloadForMethod("disaster-recovery-configs/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201},
		},
		{
			name:     "disaster_recovery_configs_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "disaster_recovery_configs_fail_over",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}/failover?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_get_authorization_rule",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_list_authorization_rules",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}/authorizationRules?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "disaster_recovery_configs_list_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/disasterRecoveryConfigs/{alias}/authorizationRules/{authorizationRuleName}/listKeys?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_hubs_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}?api-version=2026-01-01",
			payload:  payloadForMethod("event-hubs/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "event_hubs_create_or_update_authorization_rule",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  payloadForMethod("event-hubs/create-or-update-authorization-rule", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "event_hubs_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "event_hubs_delete_authorization_rule",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "event_hubs_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_hubs_get_authorization_rule",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_hubs_list_authorization_rules",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/authorizationRules?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_hubs_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_hubs_list_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/authorizationRules/{authorizationRuleName}/listKeys?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "event_hubs_regenerate_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/eventhubs/{eventHubName}/authorizationRules/{authorizationRuleName}/regenerateKeys?api-version=2026-01-01",
			payload:  payloadForMethod("event-hubs/regenerate-keys", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "namespaces_check_name_availability",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventHub/checkNameAvailability?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/check-name-availability", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "namespaces_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201, 202},
		},
		{
			name:     "namespaces_create_or_update_authorization_rule",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/create-or-update-authorization-rule", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "namespaces_create_or_update_network_rule_set",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/networkRuleSets/default?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/create-or-update-network-rule-set", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "namespaces_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "namespaces_delete_authorization_rule",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "namespaces_failover",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/failover?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/failover", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{202},
		},
		{
			name:     "namespaces_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_get_authorization_rule",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/authorizationRules/{authorizationRuleName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_get_network_rule_set",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/networkRuleSets/default?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EventHub/namespaces?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_authorization_rules",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/authorizationRules?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/authorizationRules/{authorizationRuleName}/listKeys?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_list_network_rule_set",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/networkRuleSets?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "namespaces_regenerate_keys",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/authorizationRules/{authorizationRuleName}/regenerateKeys?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/regenerate-keys", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "namespaces_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}?api-version=2026-01-01",
			payload:  payloadForMethod("namespaces/update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201, 202},
		},
		{
			name:     "network_security_perimeter_configuration_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/networkSecurityPerimeterConfigurations?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "network_security_perimeter_configurations_create_or_update",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/networkSecurityPerimeterConfigurations/{resourceAssociationName}/reconcile?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 202},
		},
		{
			name:     "network_security_perimeter_configurations_get_resource_association_name",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/networkSecurityPerimeterConfigurations/{resourceAssociationName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EventHub/operations?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_endpoint_connections_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-01-01",
			payload:  payloadForMethod("private-endpoint-connections/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200, 201, 202},
		},
		{
			name:     "private_endpoint_connections_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 202, 204},
		},
		{
			name:     "private_endpoint_connections_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_endpoint_connections_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/privateEndpointConnections?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "private_link_resources_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/privateLinkResources?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schema_registry_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/schemagroups/{schemaGroupName}?api-version=2026-01-01",
			payload:  payloadForMethod("schema-registry/create-or-update", location, partnerNamespaceID, privateEndpointID),
			statuses: []int{200},
		},
		{
			name:     "schema_registry_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/schemagroups/{schemaGroupName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200, 204},
		},
		{
			name:     "schema_registry_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/schemagroups/{schemaGroupName}?api-version=2026-01-01",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "schema_registry_list_by_namespace",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EventHub/namespaces/{namespaceName}/schemagroups?api-version=2026-01-01",
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
