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

const analysisServicesAPIVersion = "2017-08-01"

type analysisServicesClient struct {
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

func newAnalysisServicesClient(endpoint, account, subscriptionKey string) *analysisServicesClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"analysis-services-2017-08-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &analysisServicesClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *analysisServicesClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-analysis-services")
	location := getenv("STACKYARD_AZURE_REGION", "eastus")
	serverName := getenv("STACKYARD_AZURE_ANALYSIS_SERVICES_SERVER", "stackyard-aas")
	operationID := getenv("STACKYARD_AZURE_ANALYSIS_SERVICES_OPERATION_ID", "operation-123")

	fmt.Printf("Stackyard Azure Analysis Services (analysis-services-2017-08-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAnalysisServicesClient(endpoint, account, subscriptionKey)
	serverScope := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AnalysisServices/servers/" + serverName
	subscriptionScope := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.AnalysisServices"
	locationScope := subscriptionScope + "/locations/" + location

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
			path:     "/azure/providers/Microsoft.AnalysisServices/operations?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   locationScope + "/checkNameAvailability?api-version=" + analysisServicesAPIVersion,
			payload: map[string]any{
				"name": serverName,
				"type": "Microsoft.AnalysisServices/servers",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateServer",
			method: http.MethodPut,
			path:   serverScope + "?api-version=" + analysisServicesAPIVersion,
			payload: map[string]any{
				"location": location,
				"sku": map[string]any{
					"name": "S1",
					"tier": "Standard",
				},
				"properties": map[string]any{
					"asAdministrators": map[string]any{
						"members": []string{"admin@example.com"},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetServer",
			method:   http.MethodGet,
			path:     serverScope + "?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateServer",
			method: http.MethodPatch,
			path:   serverScope + "?api-version=" + analysisServicesAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env": "dev",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "ListServersBySubscription",
			method:   http.MethodGet,
			path:     subscriptionScope + "/servers?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListServersByResourceGroup",
			method:   http.MethodGet,
			path:     "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AnalysisServices/servers?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSkusForServer",
			method:   http.MethodGet,
			path:     serverScope + "/skus?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSkus",
			method:   http.MethodGet,
			path:     subscriptionScope + "/skus?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListGatewayStatus",
			method:   http.MethodPost,
			path:     serverScope + "/listGatewayStatus?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DissociateGateway",
			method:   http.MethodPost,
			path:     serverScope + "/dissociateGateway?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SuspendServer",
			method:   http.MethodPost,
			path:     serverScope + "/suspend?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ResumeServer",
			method:   http.MethodPost,
			path:     serverScope + "/resume?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetOperationResult",
			method:   http.MethodGet,
			path:     locationScope + "/operationresults/" + operationID + "?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetOperationStatus",
			method:   http.MethodGet,
			path:     locationScope + "/operationstatuses/" + operationID + "?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteServer",
			method:   http.MethodDelete,
			path:     serverScope + "?api-version=" + analysisServicesAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	fmt.Println("Azure Analysis Services staged routes completed successfully.")
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
