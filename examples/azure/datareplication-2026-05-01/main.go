package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const dataReplicationAPIVersion = "2026-05-01"

type dataReplicationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataReplicationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataReplicationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataReplicationClient(endpoint, account, subscriptionKey string) *dataReplicationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datareplication-2026-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dataReplicationSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &dataReplicationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataReplicationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DATAREPLICATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATAREPLICATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datareplication")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus2")
	vaultName := getenv("STACKYARD_AZURE_DATAREPLICATION_VAULT_NAME", "dr-vault-example")
	fabricName := getenv("STACKYARD_AZURE_DATAREPLICATION_FABRIC_NAME", "dr-fabric-example")
	fabricAgentName := getenv("STACKYARD_AZURE_DATAREPLICATION_FABRIC_AGENT_NAME", "dr-agent-example")
	policyName := getenv("STACKYARD_AZURE_DATAREPLICATION_POLICY_NAME", "dr-policy-example")
	replicationExtensionName := getenv("STACKYARD_AZURE_DATAREPLICATION_EXTENSION_NAME", "dr-extension-example")
	protectedItemName := getenv("STACKYARD_AZURE_DATAREPLICATION_PROTECTED_ITEM_NAME", "dr-protected-item-example")
	jobName := getenv("STACKYARD_AZURE_DATAREPLICATION_JOB_NAME", "dr-job-example")
	eventName := getenv("STACKYARD_AZURE_DATAREPLICATION_EVENT_NAME", "dr-event-example")
	recoveryPointName := getenv("STACKYARD_AZURE_DATAREPLICATION_RECOVERY_POINT_NAME", "dr-rp-example")
	emailConfigurationName := getenv("STACKYARD_AZURE_DATAREPLICATION_EMAIL_CONFIGURATION_NAME", "dr-alerts")
	privateEndpointConnectionProxyName := getenv("STACKYARD_AZURE_DATAREPLICATION_PRIVATE_ENDPOINT_CONNECTION_PROXY_NAME", "dr-pec-proxy")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_DATAREPLICATION_PRIVATE_ENDPOINT_CONNECTION_NAME", "dr-pec")
	deploymentID := getenv("STACKYARD_AZURE_DATAREPLICATION_DEPLOYMENT_ID", "dr-deployment-01")
	operationID := getenv("STACKYARD_AZURE_DATAREPLICATION_OPERATION_ID", "dr-op-01")

	client := newDataReplicationClient(endpoint, account, subscriptionKey)

	vaultBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/replicationVaults/%s",
		subscriptionID,
		resourceGroup,
		vaultName,
	)
	fabricBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/replicationFabrics/%s",
		subscriptionID,
		resourceGroup,
		fabricName,
	)
	fabricAgentBase := fabricBase + "/fabricAgents/" + fabricAgentName
	policyBase := vaultBase + "/replicationPolicies/" + policyName
	replicationExtensionBase := vaultBase + "/replicationExtensions/" + replicationExtensionName
	protectedItemBase := vaultBase + "/protectedItems/" + protectedItemName
	recoveryPointBase := protectedItemBase + "/recoveryPoints/" + recoveryPointName
	jobBase := vaultBase + "/jobs/" + jobName
	eventBase := vaultBase + "/events/" + eventName
	emailConfigurationBase := vaultBase + "/alertSettings/" + emailConfigurationName
	privateEndpointConnectionProxyBase := vaultBase + "/privateEndpointConnectionProxies/" + privateEndpointConnectionProxyName
	privateEndpointConnectionBase := vaultBase + "/privateEndpointConnections/" + privateEndpointConnectionName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DataReplication/operations?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DataReplication/locations/%s/checkNameAvailability?api-version=%s",
				subscriptionID,
				location,
				dataReplicationAPIVersion,
			),
			payload: map[string]any{
				"name": vaultName,
				"type": "Microsoft.DataReplication/replicationVaults",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "DeploymentPreflight",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/deployments/%s/preflight?api-version=%s",
				subscriptionID,
				resourceGroup,
				deploymentID,
				dataReplicationAPIVersion,
			),
			payload: map[string]any{
				"resources": []any{
					map[string]any{
						"name":       vaultName,
						"type":       "Microsoft.DataReplication/replicationVaults",
						"location":   location,
						"apiVersion": dataReplicationAPIVersion,
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateVault",
			method: http.MethodPut,
			path:   vaultBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"vaultType": "DisasterRecovery",
				},
				"tags": map[string]any{
					"environment": "local",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetVault",
			method:   http.MethodGet,
			path:     vaultBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListVaultsInResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/replicationVaults?api-version=%s", subscriptionID, resourceGroup, dataReplicationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListVaultsInSubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataReplication/replicationVaults?api-version=%s", subscriptionID, dataReplicationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateVault",
			method: http.MethodPatch,
			path:   vaultBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateFabric",
			method: http.MethodPut,
			path:   fabricBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"customProperties": map[string]any{
						"instanceType": "FabricModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetFabric",
			method:   http.MethodGet,
			path:     fabricBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFabricsInResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/replicationFabrics?api-version=%s", subscriptionID, resourceGroup, dataReplicationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFabricsInSubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataReplication/replicationFabrics?api-version=%s", subscriptionID, dataReplicationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateFabric",
			method: http.MethodPatch,
			path:   fabricBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
				},
				"properties": map[string]any{
					"customProperties": map[string]any{
						"instanceType": "FabricModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateFabricAgent",
			method: http.MethodPut,
			path:   fabricAgentBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"machineId":   "machine-001",
					"machineName": "machine-001",
					"authenticationIdentity": map[string]any{
						"tenantId":      "tenant-id",
						"applicationId": "app-id",
						"objectId":      "object-id",
						"audience":      "audience",
						"aadAuthority":  "https://login.microsoftonline.com/",
					},
					"resourceAccessIdentity": map[string]any{
						"tenantId":      "tenant-id",
						"applicationId": "app-id",
						"objectId":      "object-id",
						"audience":      "audience",
						"aadAuthority":  "https://login.microsoftonline.com/",
					},
					"customProperties": map[string]any{
						"instanceType": "FabricAgentModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetFabricAgent",
			method:   http.MethodGet,
			path:     fabricAgentBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFabricAgents",
			method:   http.MethodGet,
			path:     fabricBase + "/fabricAgents?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreatePolicy",
			method: http.MethodPut,
			path:   policyBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"customProperties": map[string]any{
						"instanceType":                      "PolicyModelCustomProperties",
						"appConsistentFrequencyInMinutes":   60,
						"crashConsistentFrequencyInMinutes": 15,
						"recoveryPointHistoryInMinutes":     1440,
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetPolicy",
			method:   http.MethodGet,
			path:     policyBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPolicies",
			method:   http.MethodGet,
			path:     vaultBase + "/replicationPolicies?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateReplicationExtension",
			method: http.MethodPut,
			path:   replicationExtensionBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"customProperties": map[string]any{
						"instanceType": "ReplicationExtensionModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetReplicationExtension",
			method:   http.MethodGet,
			path:     replicationExtensionBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListReplicationExtensions",
			method:   http.MethodGet,
			path:     vaultBase + "/replicationExtensions?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateEmailConfiguration",
			method: http.MethodPut,
			path:   emailConfigurationBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"sendToOwners": true,
					"customEmailAddresses": []any{
						"ops@example.com",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetEmailConfiguration",
			method:   http.MethodGet,
			path:     emailConfigurationBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEmailConfigurations",
			method:   http.MethodGet,
			path:     vaultBase + "/alertSettings?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreatePrivateEndpointConnectionProxy",
			method: http.MethodPut,
			path:   privateEndpointConnectionProxyBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"groupId": "default",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetPrivateEndpointConnectionProxy",
			method:   http.MethodGet,
			path:     privateEndpointConnectionProxyBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnectionProxies",
			method:   http.MethodGet,
			path:     vaultBase + "/privateEndpointConnectionProxies?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateEndpointConnection",
			method:   http.MethodGet,
			path:     privateEndpointConnectionBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnections",
			method:   http.MethodGet,
			path:     vaultBase + "/privateEndpointConnections?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResources",
			method:   http.MethodGet,
			path:     vaultBase + "/privateLinkResources?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateProtectedItem",
			method: http.MethodPut,
			path:   protectedItemBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"policyName":               policyName,
					"replicationExtensionName": replicationExtensionName,
					"customProperties": map[string]any{
						"instanceType": "ProtectedItemModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetProtectedItem",
			method:   http.MethodGet,
			path:     protectedItemBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProtectedItems",
			method:   http.MethodGet,
			path:     vaultBase + "/protectedItems?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateProtectedItem",
			method: http.MethodPatch,
			path:   protectedItemBase + "?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"customProperties": map[string]any{
						"instanceType": "ProtectedItemModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "PlannedFailoverProtectedItem",
			method: http.MethodPost,
			path:   protectedItemBase + "/plannedFailover?api-version=" + dataReplicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"customProperties": map[string]any{
						"instanceType": "PlannedFailoverModelCustomProperties",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "ListRecoveryPoints",
			method:   http.MethodGet,
			path:     protectedItemBase + "/recoveryPoints?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetRecoveryPoint",
			method:   http.MethodGet,
			path:     recoveryPointBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListJobs",
			method:   http.MethodGet,
			path:     vaultBase + "/jobs?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetJob",
			method:   http.MethodGet,
			path:     jobBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEvents",
			method:   http.MethodGet,
			path:     vaultBase + "/events?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEvent",
			method:   http.MethodGet,
			path:     eventBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GetLocationBasedOperationResult",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/locations/%s/operationResults/%s?api-version=%s",
				subscriptionID,
				resourceGroup,
				location,
				operationID,
				dataReplicationAPIVersion,
			),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GetOperationResult",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataReplication/operationResults/%s/subscriptions/%s/resourceGroups/%s/%s?api-version=%s",
				subscriptionID,
				resourceGroup,
				operationID,
				subscriptionID,
				resourceGroup,
				operationID,
				dataReplicationAPIVersion,
			),
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeleteProtectedItem",
			method:   http.MethodDelete,
			path:     protectedItemBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeletePrivateEndpointConnectionProxy",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionProxyBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteReplicationExtension",
			method:   http.MethodDelete,
			path:     replicationExtensionBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeletePolicy",
			method:   http.MethodDelete,
			path:     policyBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteFabricAgent",
			method:   http.MethodDelete,
			path:     fabricAgentBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteFabric",
			method:   http.MethodDelete,
			path:     fabricBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteVault",
			method:   http.MethodDelete,
			path:     vaultBase + "?api-version=" + dataReplicationAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		payload, status, err := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
