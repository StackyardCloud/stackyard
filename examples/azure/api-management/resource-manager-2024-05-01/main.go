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

const apiManagementResourceManagerAPIVersion = "2024-05-01"

type apiManagementClient struct {
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

func newAPIManagementClient(endpoint, account, subscriptionKey string) *apiManagementClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"api-management-resource-manager-2024-05-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &apiManagementClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *apiManagementClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
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
	account := getenv("STACKYARD_AZURE_APIM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APIM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-apim")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	serviceName := getenv("STACKYARD_AZURE_APIM_SERVICE", "apim-a")
	apiID := getenv("STACKYARD_AZURE_APIM_API", "echo-api")
	operationID := getenv("STACKYARD_AZURE_APIM_OPERATION", "get-echo")
	productID := getenv("STACKYARD_AZURE_APIM_PRODUCT", "starter")
	userID := getenv("STACKYARD_AZURE_APIM_USER", "user-1")
	namedValueID := getenv("STACKYARD_AZURE_APIM_NAMED_VALUE", "kv-endpoint")
	gatewayID := getenv("STACKYARD_AZURE_APIM_GATEWAY", "gw-a")
	workspaceID := getenv("STACKYARD_AZURE_APIM_WORKSPACE", "ws-a")
	workspaceSubscriptionID := getenv("STACKYARD_AZURE_APIM_WORKSPACE_SUBSCRIPTION", "sub-a")
	notificationID := getenv("STACKYARD_AZURE_APIM_NOTIFICATION", "NewCommentNotificationMessage")
	recipientID := getenv("STACKYARD_AZURE_APIM_RECIPIENT", "dev-example-com")

	fmt.Printf("Stackyard Azure API Management Resource Manager (resource-manager-2024-05-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAPIManagementClient(endpoint, account, subscriptionKey)

	subscriptionScope := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ApiManagement"
	serviceScope := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ApiManagement/service/" + serviceName
	workspaceScope := serviceScope + "/workspaces/" + workspaceID
	apiScope := serviceScope + "/apis/" + apiID
	workspaceAPIScope := workspaceScope + "/apis/" + apiID
	operationScope := apiScope + "/operations/" + operationID

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.ApiManagement/operations?api-version=" + apiManagementResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSKUs",
			method:   http.MethodGet,
			path:     subscriptionScope + "/skus?api-version=" + apiManagementResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateService",
			method: http.MethodPut,
			path:   serviceScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"location": location,
				"sku": map[string]any{
					"name":     "Developer",
					"capacity": 1,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetService",
			method:   http.MethodGet,
			path:     serviceScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "HeadService",
			method:   http.MethodHead,
			path:     serviceScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ApplyNetworkConfigurationUpdates",
			method: http.MethodPost,
			path:   serviceScope + "/applynetworkconfigurationupdates?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"location": location,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAPI",
			method: http.MethodPut,
			path:   apiScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Echo API",
					"path":        "echo",
					"protocols":   []string{"https"},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAPIOperation",
			method: http.MethodPut,
			path:   operationScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Get Echo",
					"method":      "GET",
					"urlTemplate": "/echo",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAPIOperationPolicy",
			method: http.MethodPut,
			path:   operationScope + "/policies/policy?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"format": "rawxml",
					"value":  "<policies></policies>",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateProduct",
			method: http.MethodPut,
			path:   serviceScope + "/products/" + productID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName":          "Starter",
					"subscriptionRequired": false,
					"state":                "published",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateUser",
			method: http.MethodPut,
			path:   serviceScope + "/users/" + userID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"firstName": "Dev",
					"lastName":  "User",
					"email":     "dev@example.com",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateNamedValue",
			method: http.MethodPut,
			path:   serviceScope + "/namedValues/" + namedValueID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "KeyVaultEndpoint",
					"value":       "https://example.vault.azure.net/",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateGateway",
			method: http.MethodPut,
			path:   serviceScope + "/gateways/" + gatewayID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"description": "gateway a",
					"locationData": map[string]any{
						"name": location,
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspace",
			method: http.MethodPut,
			path:   workspaceScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"title": workspaceID,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspaceAPI",
			method: http.MethodPut,
			path:   workspaceAPIScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Echo API",
					"path":        "echo",
					"protocols":   []string{"https"},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspaceProduct",
			method: http.MethodPut,
			path:   workspaceScope + "/products/" + productID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Starter",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspaceSubscription",
			method: http.MethodPut,
			path:   workspaceScope + "/subscriptions/" + workspaceSubscriptionID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": workspaceSubscriptionID,
					"scope":       "/products/" + productID,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWorkspaceNamedValue",
			method: http.MethodPut,
			path:   workspaceScope + "/namedValues/" + namedValueID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "KeyVaultEndpoint",
					"value":       "https://example.vault.azure.net/",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "HeadNotificationRecipientEmail",
			method:   http.MethodHead,
			path:     serviceScope + "/notifications/" + notificationID + "/recipientEmails/" + recipientID + "?api-version=" + apiManagementResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteService",
			method:   http.MethodDelete,
			path:     serviceScope + "?api-version=" + apiManagementResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
