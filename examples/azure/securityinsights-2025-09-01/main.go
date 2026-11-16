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
	resourcesAPIVersion        = "2021-04-01"
	securityInsightsAPIVersion = "2025-09-01"
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
		"securityinsights-2025-09-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-sentinel-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	workspaceName := getenv("STACKYARD_AZURE_SENTINEL_WORKSPACE", "soc-workspace")

	fmt.Printf("Stackyard Microsoft Sentinel (securityinsights-2025-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	workspace := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.OperationalInsights/workspaces/" + workspaceName
	sentinel := workspace + "/providers/Microsoft.SecurityInsights"
	armSentinel := strings.TrimPrefix(sentinel, "/azure")
	rule := sentinel + "/alertRules/failed-sign-ins"
	incident := sentinel + "/incidents/73e01a99-5cd7-4139-a149-9f2736ff2ab5"
	watchlist := sentinel + "/watchlists/highValueAsset"
	q := "?api-version=" + securityInsightsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateWorkspace", method: http.MethodPut, path: workspace + "?api-version=2022-10-01", payload: map[string]any{"location": region, "properties": map[string]any{"sku": map[string]any{"name": "PerGB2018"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "OnboardSentinel", method: http.MethodPut, path: sentinel + "/onboardingStates/default" + q, payload: map[string]any{"properties": map[string]any{"customerManagedKey": false}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateScheduledRule", method: http.MethodPut, path: rule + q, payload: map[string]any{"kind": "Scheduled", "properties": map[string]any{
			"displayName": "Failed sign-ins", "enabled": true, "query": "SigninLogs | where ResultType != 0",
			"queryFrequency": "PT1H", "queryPeriod": "P1D", "severity": "Medium", "triggerOperator": "GreaterThan", "triggerThreshold": 10,
			"suppressionDuration": "PT5H", "suppressionEnabled": false, "tactics": []string{"CredentialAccess"},
			"incidentConfiguration": map[string]any{"createIncident": true},
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AddRuleAction", method: http.MethodPut, path: rule + "/actions/notify" + q, payload: map[string]any{"properties": map[string]any{
			"logicAppResourceId": strings.TrimPrefix(base, "/azure") + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Logic/workflows/notify",
			"triggerUri":         "https://prod-31.eastus.logic.azure.com:443/workflows/0b5c6c1d/triggers/manual/paths/invoke",
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListAlertRuleTemplates", method: http.MethodGet, path: sentinel + "/alertRuleTemplates" + q, statuses: []int{http.StatusOK}},
		{name: "CreateAutomationRule", method: http.MethodPut, path: sentinel + "/automationRules/triage" + q, payload: map[string]any{"properties": map[string]any{
			"displayName": "Triage new incidents", "order": 1,
			"triggeringLogic": map[string]any{"isEnabled": true, "triggersOn": "Incidents", "triggersWhen": "Created"},
			"actions":         []any{map[string]any{"order": 1, "actionType": "ModifyProperties", "actionConfiguration": map[string]any{"status": "Active"}}},
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ConnectEntraID", method: http.MethodPut, path: sentinel + "/dataConnectors/entra-id" + q, payload: map[string]any{"kind": "AzureActiveDirectory", "properties": map[string]any{
			"tenantId": "72f988bf-86f1-41af-91ab-2d7cd011db47", "dataTypes": map[string]any{"alerts": map[string]any{"state": "Enabled"}},
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateIncident", method: http.MethodPut, path: incident + q, payload: map[string]any{"properties": map[string]any{"title": "Suspicious sign-ins", "severity": "High", "status": "New"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CommentOnIncident", method: http.MethodPut, path: incident + "/comments/first" + q, payload: map[string]any{"properties": map[string]any{"message": "Investigating."}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateBookmark", method: http.MethodPut, path: sentinel + "/bookmarks/evidence" + q, payload: map[string]any{"properties": map[string]any{"displayName": "Sign-in evidence", "query": "SigninLogs | take 10"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "RelateBookmark", method: http.MethodPut, path: incident + "/relations/evidence" + q, payload: map[string]any{"properties": map[string]any{"relatedResourceId": armSentinel + "/bookmarks/evidence"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListIncidentBookmarks", method: http.MethodPost, path: incident + "/bookmarks" + q, statuses: []int{http.StatusOK}},
		{name: "CloseIncident", method: http.MethodPut, path: incident + q, payload: map[string]any{"properties": map[string]any{
			"title": "Suspicious sign-ins", "severity": "High", "status": "Closed",
			"classification": "FalsePositive", "classificationReason": "IncorrectAlertLogic",
		}}, statuses: []int{http.StatusOK}},
		{name: "CreateWatchlist", method: http.MethodPut, path: watchlist + q, payload: map[string]any{"properties": map[string]any{
			"displayName": "High Value Assets", "provider": "Microsoft", "source": "assets.csv", "itemsSearchKey": "hostname",
			"contentType": "text/csv", "rawContent": "hostname,owner\ndc01,identity-team\nsql01,data-team",
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListWatchlistItems", method: http.MethodGet, path: watchlist + "/watchlistItems" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteWatchlist", method: http.MethodDelete, path: watchlist + q, statuses: []int{http.StatusNoContent}},
		{name: "DeleteIncident", method: http.MethodDelete, path: incident + q, statuses: []int{http.StatusOK}},
		{name: "DeleteRule", method: http.MethodDelete, path: rule + q, statuses: []int{http.StatusOK}},
		{name: "DeleteWorkspace", method: http.MethodDelete, path: workspace + "?api-version=2022-10-01", statuses: []int{http.StatusOK}},
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
