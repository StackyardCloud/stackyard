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
	resourcesAPIVersion    = "2021-04-01"
	actionGroupsAPIVersion = "2023-01-01"
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
		"monitor-2023-01-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-monitor-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")

	fmt.Printf("Stackyard Azure Monitor (monitor-2023-01-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	insights := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Insights"
	groupScope := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup
	workspaceID := groupScope + "/providers/Microsoft.OperationalInsights/workspaces/stackyard-logs"
	q := "?api-version=" + actionGroupsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateActionGroup", method: http.MethodPut, path: insights + "/actionGroups/oncall" + q, payload: map[string]any{"location": "Global", "properties": map[string]any{"groupShortName": "oncall", "enabled": true, "emailReceivers": []any{map[string]any{"name": "ops", "emailAddress": "ops@example.com"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateMetricAlert", method: http.MethodPut, path: insights + "/metricAlerts/cpu-high?api-version=2018-03-01", payload: map[string]any{"location": "global", "properties": map[string]any{"severity": 2, "enabled": true, "scopes": []string{groupScope}, "evaluationFrequency": "PT1M", "windowSize": "PT5M", "criteria": map[string]any{"odata.type": "Microsoft.Azure.Monitor.MultipleResourceMultipleMetricCriteria", "allOf": []any{}}, "actions": []any{map[string]any{"actionGroupId": groupScope + "/providers/Microsoft.Insights/actionGroups/oncall"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateActivityLogAlert", method: http.MethodPut, path: insights + "/activityLogAlerts/group-deletes?api-version=2020-10-01", payload: map[string]any{"location": "Global", "properties": map[string]any{"scopes": []string{groupScope}, "condition": map[string]any{"allOf": []any{map[string]any{"field": "operationName", "equals": "Microsoft.Resources/subscriptions/resourceGroups/delete"}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateSubscriptionDiagnosticSetting", method: http.MethodPut, path: base + "/providers/Microsoft.Insights/diagnosticSettings/activity-to-logs?api-version=2021-05-01-preview", payload: map[string]any{"properties": map[string]any{"workspaceId": workspaceID, "logs": []any{map[string]any{"category": "Administrative", "enabled": true}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListSubscriptionDiagnosticSettings", method: http.MethodGet, path: base + "/providers/Microsoft.Insights/diagnosticSettings?api-version=2021-05-01-preview", statuses: []int{http.StatusOK}},
		{name: "DeleteSubscriptionDiagnosticSetting", method: http.MethodDelete, path: base + "/providers/Microsoft.Insights/diagnosticSettings/activity-to-logs?api-version=2021-05-01-preview", statuses: []int{http.StatusOK}},
		{name: "ListMetricAlerts", method: http.MethodGet, path: base + "/providers/Microsoft.Insights/metricAlerts?api-version=2018-03-01", statuses: []int{http.StatusOK}},
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
