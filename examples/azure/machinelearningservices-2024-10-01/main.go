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

const (
	resourcesAPIVersion = "2021-04-01"
	machineLearningAPIVersion = "2024-10-01"
)

type armClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyAndSubscriptionPolicy struct {
	account         string
	subscriptionKey string
}

func (p *sharedKeyAndSubscriptionPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newARMClient(endpoint, account, subscriptionKey string) *armClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"machinelearningservices-2024-10-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &armClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}
	if len(expectedStatuses) > 0 {
		ok := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-ml-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	workspaceName := getenv("STACKYARD_AZURE_ML_WORKSPACE", "stackyard-research")

	fmt.Printf("Stackyard Azure Machine Learning (machinelearningservices-2024-10-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	workspace := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.MachineLearningServices/workspaces/" + workspaceName
	deps := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers"
	scorer := workspace + "/onlineEndpoints/stackyard-scorer"
	q := "?api-version=" + machineLearningAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateWorkspace", method: http.MethodPut, path: workspace + q, payload: map[string]any{"location": region, "identity": map[string]any{"type": "SystemAssigned"}, "properties": map[string]any{"friendlyName": "Research", "storageAccount": deps + "/Microsoft.Storage/storageAccounts/stackyardml", "keyVault": deps + "/Microsoft.KeyVault/vaults/stackyard-ml-kv", "applicationInsights": deps + "/Microsoft.Insights/components/stackyard-ml"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListWorkspaceKeys", method: http.MethodPost, path: workspace + "/listKeys" + q, statuses: []int{http.StatusOK}},
		// The admin password is accepted but never returned.
		{name: "CreateComputeCluster", method: http.MethodPut, path: workspace + "/computes/cpu-cluster" + q, payload: map[string]any{"location": region, "properties": map[string]any{"computeType": "AmlCompute", "properties": map[string]any{"vmSize": "STANDARD_DS3_V2", "scaleSettings": map[string]any{"minNodeCount": 0, "maxNodeCount": 4, "nodeIdleTimeBeforeScaleDown": "PT120S"}, "userAccountCredentials": map[string]any{"adminUserName": "azureuser", "adminUserPassword": "Stackyard-Example-1"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateComputeInstance", method: http.MethodPut, path: workspace + "/computes/notebook-1" + q, payload: map[string]any{"location": region, "properties": map[string]any{"computeType": "ComputeInstance", "properties": map[string]any{"vmSize": "STANDARD_DS11_V2"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "StopComputeInstance", method: http.MethodPost, path: workspace + "/computes/notebook-1/stop" + q, statuses: []int{http.StatusAccepted}},
		{name: "CreateDatastore", method: http.MethodPut, path: workspace + "/datastores/training" + q, payload: map[string]any{"properties": map[string]any{"datastoreType": "AzureBlob", "accountName": "stackyardml", "containerName": "training", "credentials": map[string]any{"credentialsType": "None"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateOnlineEndpoint", method: http.MethodPut, path: scorer + q, payload: map[string]any{"location": region, "identity": map[string]any{"type": "SystemAssigned"}, "properties": map[string]any{"authMode": "Key"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDeployment", method: http.MethodPut, path: scorer + "/deployments/blue" + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "Default", "capacity": 1}, "properties": map[string]any{"endpointComputeType": "Managed", "model": "azureml:sklearn-model:1", "instanceType": "Standard_DS3_v2"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "RouteTraffic", method: http.MethodPatch, path: scorer + q, payload: map[string]any{"properties": map[string]any{"traffic": map[string]any{"blue": 100}}}, statuses: []int{http.StatusOK}},
		{name: "ListEndpointKeys", method: http.MethodPost, path: scorer + "/listKeys" + q, statuses: []int{http.StatusOK}},
		{name: "GetDeploymentLogs", method: http.MethodPost, path: scorer + "/deployments/blue/getLogs" + q, payload: map[string]any{"tail": 100}, statuses: []int{http.StatusOK}},
		{name: "DeleteWorkspace", method: http.MethodDelete, path: workspace + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if len(resp) != 0 {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
