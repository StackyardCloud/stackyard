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
	newRelicAPIVersion  = "2026-06-01"
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
		"newrelic-2026-06-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-newrelic-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	monitorName := getenv("STACKYARD_AZURE_NEWRELIC_MONITOR", "stackyard-newrelic")
	email := getenv("STACKYARD_AZURE_NEWRELIC_EMAIL", "alice@contoso.com")

	fmt.Printf("Stackyard Azure Native New Relic Service (newrelic-2026-06-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	mon := base + "/resourceGroups/" + resourceGroup + "/providers/NewRelic.Observability/monitors/" + monitorName
	q := "?api-version=" + newRelicAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateMonitor", method: http.MethodPut, path: mon + q, payload: map[string]any{"location": region, "tags": map[string]any{"env": "prod"}, "properties": map[string]any{"accountCreationSource": "LIFTR", "orgCreationSource": "LIFTR", "planData": map[string]any{"billingCycle": "Monthly", "planDetails": "newrelic-pay-as-you-go", "usageType": "PAYG"}, "userInfo": map[string]any{"emailAddress": email, "firstName": "Alice", "lastName": "Smith"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateMonitor", method: http.MethodPatch, path: mon + q, payload: map[string]any{"tags": map[string]any{"env": "test"}, "properties": map[string]any{"userInfo": map[string]any{"emailAddress": email, "phoneNumber": "+1 555 0100"}}}, statuses: []int{http.StatusOK}},
		{name: "CreateTagRule", method: http.MethodPut, path: mon + "/tagRules/default" + q, payload: map[string]any{"properties": map[string]any{"logRules": map[string]any{"sendActivityLogs": "Enabled", "sendSubscriptionLogs": "Enabled", "sendAadLogs": "Disabled"}, "metricRules": map[string]any{"sendMetrics": "Enabled", "userEmail": email, "filteringTags": []any{map[string]any{"name": "env", "value": "test", "action": "Include"}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateTagRule", method: http.MethodPatch, path: mon + "/tagRules/default" + q, payload: map[string]any{"properties": map[string]any{"logRules": map[string]any{"sendActivityLogs": "Enabled", "filteringTags": []any{map[string]any{"name": "env", "value": "dev", "action": "Exclude"}}}}}, statuses: []int{http.StatusOK}},
		{name: "ListTagRules", method: http.MethodGet, path: mon + "/tagRules" + q, statuses: []int{http.StatusOK}},
		{name: "AddMonitoredSubscriptions", method: http.MethodPut, path: mon + "/monitoredSubscriptions/default" + q, payload: map[string]any{"properties": map[string]any{"patchOperation": "AddBegin", "monitoredSubscriptionList": []any{map[string]any{"subscriptionId": "/subscriptions/" + subscriptionID}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UpdateMonitoredSubscriptions", method: http.MethodPatch, path: mon + "/monitoredSubscriptions/default" + q, payload: map[string]any{"properties": map[string]any{"patchOperation": "AddComplete", "monitoredSubscriptionList": []any{map[string]any{"subscriptionId": "/subscriptions/" + subscriptionID}}}}, statuses: []int{http.StatusOK}},
		{name: "ListMonitoredSubscriptions", method: http.MethodGet, path: mon + "/monitoredSubscriptions" + q, statuses: []int{http.StatusOK}},
		{name: "ListMonitoredResources", method: http.MethodPost, path: mon + "/monitoredResources" + q, statuses: []int{http.StatusOK}},
		{name: "GetMetricRules", method: http.MethodPost, path: mon + "/getMetricRules" + q, payload: map[string]any{"userEmail": email}, statuses: []int{http.StatusOK}},
		{name: "GetMetricStatus", method: http.MethodPost, path: mon + "/getMetricStatus" + q, payload: map[string]any{"userEmail": email}, statuses: []int{http.StatusOK}},
		{name: "ListHosts", method: http.MethodPost, path: mon + "/listHosts" + q, payload: map[string]any{"userEmail": email}, statuses: []int{http.StatusOK}},
		{name: "ListAppServices", method: http.MethodPost, path: mon + "/listAppServices" + q, payload: map[string]any{"userEmail": email}, statuses: []int{http.StatusOK}},
		{name: "VMHostPayload", method: http.MethodPost, path: mon + "/vmHostPayloads" + q, statuses: []int{http.StatusOK}},
		{name: "RefreshIngestionKey", method: http.MethodPost, path: mon + "/refreshIngestionKey" + q, statuses: []int{http.StatusNoContent}},
		{name: "LinkSaaS", method: http.MethodPost, path: mon + "/linkSaaS" + q, payload: map[string]any{"saaSResourceId": "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.SaaS/resources/newrelic-saas"}, statuses: []int{http.StatusOK}},
		{name: "LatestLinkedSaaS", method: http.MethodPost, path: mon + "/latestLinkedSaaS" + q, statuses: []int{http.StatusOK}},
		{name: "GetBillingInfo", method: http.MethodPost, path: mon + "/getBillingInfo" + q, statuses: []int{http.StatusOK}},
		{name: "ListConnectedPartnerResources", method: http.MethodPost, path: mon + "/listConnectedPartnerResources" + q, payload: email, statuses: []int{http.StatusOK}},
		{name: "ListLinkedResources", method: http.MethodPost, path: mon + "/listLinkedResources" + q, statuses: []int{http.StatusOK}},
		{name: "SwitchBilling", method: http.MethodPost, path: mon + "/switchBilling" + q, payload: map[string]any{"userEmail": email, "planData": map[string]any{"usageType": "COMMITTED", "billingCycle": "Yearly", "planDetails": "newrelic-annual"}}, statuses: []int{http.StatusOK}},
		{name: "Resubscribe", method: http.MethodPost, path: mon + "/resubscribe" + q, payload: map[string]any{"planId": "newrelic-pay-as-you-go", "termId": "P1M"}, statuses: []int{http.StatusOK}},
		{name: "ListAccounts", method: http.MethodGet, path: base + "/providers/NewRelic.Observability/accounts" + q + "&userEmail=" + email + "&location=" + region, statuses: []int{http.StatusOK}},
		{name: "ListOrganizations", method: http.MethodGet, path: base + "/providers/NewRelic.Observability/organizations" + q + "&userEmail=" + email + "&location=" + region, statuses: []int{http.StatusOK}},
		{name: "ListPlans", method: http.MethodGet, path: base + "/providers/NewRelic.Observability/plans" + q, statuses: []int{http.StatusOK}},
		{name: "ListMonitorsByResourceGroup", method: http.MethodGet, path: base + "/resourceGroups/" + resourceGroup + "/providers/NewRelic.Observability/monitors" + q, statuses: []int{http.StatusOK}},
		{name: "ListMonitorsBySubscription", method: http.MethodGet, path: base + "/providers/NewRelic.Observability/monitors" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteMonitoredSubscriptions", method: http.MethodDelete, path: mon + "/monitoredSubscriptions/default" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteTagRule", method: http.MethodDelete, path: mon + "/tagRules/default" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteMonitor", method: http.MethodDelete, path: mon + q + "&userEmail=" + email, statuses: []int{http.StatusOK}},
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
