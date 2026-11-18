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
	impactAPIVersion    = "2026-01-01-preview"
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
		"impact-2026-01-01-preview",
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

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (any, int, error) {
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
		return nil, resp.StatusCode, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-impact-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	impactName := getenv("STACKYARD_AZURE_IMPACT_NAME", "stackyard-impact-001")

	fmt.Printf("Stackyard Azure Impact Reporting (impact-2026-01-01-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	ns := base + "/providers/Microsoft.Impact"
	impact := ns + "/workloadImpacts/" + impactName
	vm := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/vm1"
	q := "?api-version=" + impactAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListImpactCategories", method: http.MethodGet, path: ns + "/impactCategories" + q + "&resourceType=microsoft.compute/virtualmachines", statuses: []int{http.StatusOK}},
		{name: "GetImpactCategory", method: http.MethodGet, path: ns + "/impactCategories/Resource.Availability.Restart" + q, statuses: []int{http.StatusOK}},
		{name: "CreateConnector", method: http.MethodPut, path: ns + "/connectors/AzureMonitorConnector" + q, payload: map[string]any{"properties": map[string]any{"connectorType": "AzureMonitor"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateConnector", method: http.MethodPatch, path: ns + "/connectors/AzureMonitorConnector" + q, payload: map[string]any{"properties": map[string]any{"connectorType": "AzureMonitor"}}, statuses: []int{http.StatusOK}},
		{name: "ListConnectors", method: http.MethodGet, path: ns + "/connectors" + q, statuses: []int{http.StatusOK}},
		{name: "ReportImpact", method: http.MethodPut, path: impact + q, payload: map[string]any{"properties": map[string]any{"impactedResourceId": vm, "startDateTime": "2026-06-15T05:59:46Z", "impactCategory": "Resource.Connectivity.Inbound", "impactDescription": "connection failure", "connectivity": map[string]any{"protocol": "TCP", "port": 1443}, "workload": map[string]any{"context": "webapp/scenario1", "toolset": "SDK"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateInsight", method: http.MethodPut, path: impact + "/insights/insight001" + q, payload: map[string]any{"properties": map[string]any{"category": "repair", "status": "resolved", "content": map[string]any{"title": "Impact correlated to an outage", "description": "An outage affected the resource."}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetImpact", method: http.MethodGet, path: impact + q, statuses: []int{http.StatusOK}},
		{name: "ListInsights", method: http.MethodGet, path: impact + "/insights" + q, statuses: []int{http.StatusOK}},
		{name: "GetUploadToken", method: http.MethodPost, path: ns + "/getUploadToken" + q, statuses: []int{http.StatusOK}},
		{name: "ListImpacts", method: http.MethodGet, path: ns + "/workloadImpacts" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteInsight", method: http.MethodDelete, path: impact + "/insights/insight001" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteImpact", method: http.MethodDelete, path: impact + q, statuses: []int{http.StatusOK}},
		{name: "DeleteConnector", method: http.MethodDelete, path: ns + "/connectors/AzureMonitorConnector" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if resp != nil {
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
