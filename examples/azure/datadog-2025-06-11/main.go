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

const datadogAPIVersion = "2025-06-11"

type datadogClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type datadogSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *datadogSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDatadogClient(endpoint, account, subscriptionKey string) *datadogClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datadog-2025-06-11",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&datadogSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &datadogClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *datadogClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATADOG_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATADOG_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datadog")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus")
	monitorName := getenv("STACKYARD_AZURE_DATADOG_MONITOR_NAME", "datadog-monitor-example")
	agreementName := getenv("STACKYARD_AZURE_DATADOG_AGREEMENT_NAME", "default")
	configurationName := getenv("STACKYARD_AZURE_DATADOG_MONITORED_SUBSCRIPTION_CONFIGURATION", "default")
	ruleSetName := getenv("STACKYARD_AZURE_DATADOG_TAG_RULE_SET_NAME", "default")
	ssoConfigurationName := getenv("STACKYARD_AZURE_DATADOG_SSO_CONFIGURATION_NAME", "default")
	datadogOrganizationID := getenv("STACKYARD_AZURE_DATADOG_ORG_ID", "00000000-0000-0000-0000-000000000000")
	marketplacePlanName := getenv("STACKYARD_AZURE_DATADOG_PLAN_NAME", "free_Monthly")

	monitorBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Datadog/monitors/%s",
		subscriptionID,
		resourceGroup,
		monitorName,
	)
	agreementBase := fmt.Sprintf(
		"/azure/subscriptions/%s/providers/Microsoft.Datadog/agreements/%s",
		subscriptionID,
		agreementName,
	)
	creationSupportedBase := fmt.Sprintf(
		"/azure/subscriptions/%s/providers/Microsoft.Datadog/subscriptionStatuses",
		subscriptionID,
	)
	monitoredSubscriptionBase := monitorBase + "/monitoredSubscriptions/" + configurationName
	tagRuleBase := monitorBase + "/tagRules/" + ruleSetName
	ssoBase := monitorBase + "/singleSignOnConfigurations/" + ssoConfigurationName

	monitoredSubscriptionID := fmt.Sprintf("/subscriptions/%s", subscriptionID)

	monitoredSubscriptionPayload := map[string]any{
		"properties": map[string]any{
			"monitoredSubscriptionList": []any{
				map[string]any{
					"subscriptionId": monitoredSubscriptionID,
					"tagRules": map[string]any{
						"automuting": true,
						"logRules": map[string]any{
							"sendAadLogs":          false,
							"sendResourceLogs":     true,
							"sendSubscriptionLogs": true,
							"filteringTags": []any{
								map[string]any{
									"name":   "Environment",
									"action": "Include",
									"value":  "Prod",
								},
							},
						},
						"metricRules": map[string]any{
							"filteringTags": []any{},
						},
					},
				},
			},
		},
	}

	tagRulePayload := map[string]any{
		"properties": map[string]any{
			"automuting": true,
			"logRules": map[string]any{
				"sendAadLogs":          false,
				"sendResourceLogs":     true,
				"sendSubscriptionLogs": true,
				"filteringTags": []any{
					map[string]any{
						"name":   "Environment",
						"action": "Include",
						"value":  "Prod",
					},
					map[string]any{
						"name":   "Environment",
						"action": "Exclude",
						"value":  "Dev",
					},
				},
			},
			"metricRules": map[string]any{
				"filteringTags": []any{},
			},
		},
	}

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
			path:     "/azure/providers/Microsoft.Datadog/operations?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCreationSupported",
			method:   http.MethodGet,
			path:     creationSupportedBase + "?api-version=" + datadogAPIVersion + "&datadogOrganizationId=" + datadogOrganizationID,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetCreationSupported",
			method:   http.MethodGet,
			path:     creationSupportedBase + "/default?api-version=" + datadogAPIVersion + "&datadogOrganizationId=" + datadogOrganizationID,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateMarketplaceAgreement",
			method: http.MethodPut,
			path:   agreementBase + "?api-version=" + datadogAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"accepted": true,
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMarketplaceAgreements",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.Datadog/agreements?api-version=%s", subscriptionID, datadogAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateMonitor",
			method: http.MethodPut,
			path:   monitorBase + "?api-version=" + datadogAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{
					"type": "SystemAssigned",
				},
				"sku": map[string]any{
					"name": marketplacePlanName,
				},
				"tags": map[string]any{
					"environment": "local",
					"service":     "datadog",
				},
				"properties": map[string]any{
					"monitoringStatus": "Enabled",
					"datadogOrganizationProperties": map[string]any{
						"name":               "myOrg",
						"id":                 "myOrg123",
						"linkingAuthCode":    "someAuthCode",
						"linkingClientId":    datadogOrganizationID,
						"enterpriseAppId":    datadogOrganizationID,
						"cspm":               false,
						"resourceCollection": false,
					},
					"userInfo": map[string]any{
						"name":         "Alice",
						"emailAddress": "alice@example.com",
						"phoneNumber":  "123-456-7890",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetMonitor",
			method:   http.MethodGet,
			path:     monitorBase + "?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMonitorsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Datadog/monitors?api-version=%s", subscriptionID, resourceGroup, datadogAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMonitorsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.Datadog/monitors?api-version=%s", subscriptionID, datadogAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateMonitor",
			method: http.MethodPatch,
			path:   monitorBase + "?api-version=" + datadogAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"monitoringStatus": "Enabled",
				},
				"tags": map[string]any{
					"environment": "local-updated",
					"service":     "datadog",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetMonitorDefaultKey",
			method:   http.MethodPost,
			path:     monitorBase + "/getDefaultKey?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMonitorAPIKeys",
			method:   http.MethodPost,
			path:     monitorBase + "/listApiKeys?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SetMonitorDefaultKey",
			method: http.MethodPost,
			path:   monitorBase + "/setDefaultKey?api-version=" + datadogAPIVersion,
			payload: map[string]any{
				"name": "default-key",
				"key":  "1111111111111111aaaaaaaaaaaaaaaa",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "RefreshSetPasswordLink",
			method:   http.MethodPost,
			path:     monitorBase + "/refreshSetPasswordLink?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMonitorHosts",
			method:   http.MethodPost,
			path:     monitorBase + "/listHosts?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListLinkedResources",
			method:   http.MethodPost,
			path:     monitorBase + "/listLinkedResources?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMonitoredResources",
			method:   http.MethodPost,
			path:     monitorBase + "/listMonitoredResources?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetBillingInfo",
			method:   http.MethodPost,
			path:     monitorBase + "/getBillingInfo?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateMonitoredSubscriptions",
			method:   http.MethodPut,
			path:     monitoredSubscriptionBase + "?api-version=" + datadogAPIVersion,
			payload:  monitoredSubscriptionPayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetMonitoredSubscriptions",
			method:   http.MethodGet,
			path:     monitoredSubscriptionBase + "?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMonitoredSubscriptions",
			method:   http.MethodGet,
			path:     monitorBase + "/monitoredSubscriptions?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateMonitoredSubscriptions",
			method:   http.MethodPatch,
			path:     monitoredSubscriptionBase + "?api-version=" + datadogAPIVersion,
			payload:  monitoredSubscriptionPayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateTagRules",
			method:   http.MethodPut,
			path:     tagRuleBase + "?api-version=" + datadogAPIVersion,
			payload:  tagRulePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetTagRules",
			method:   http.MethodGet,
			path:     tagRuleBase + "?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTagRules",
			method:   http.MethodGet,
			path:     monitorBase + "/tagRules?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateSingleSignOnConfiguration",
			method: http.MethodPut,
			path:   ssoBase + "?api-version=" + datadogAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"enterpriseAppId":   datadogOrganizationID,
					"singleSignOnState": "Enable",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetSingleSignOnConfiguration",
			method:   http.MethodGet,
			path:     ssoBase + "?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSingleSignOnConfigurations",
			method:   http.MethodGet,
			path:     monitorBase + "/singleSignOnConfigurations?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ResubscribeOrganization",
			method: http.MethodPost,
			path:   monitorBase + "/resubscribe?api-version=" + datadogAPIVersion,
			payload: map[string]any{
				"azureSubscriptionId": subscriptionID,
				"resourceGroup":       resourceGroup,
				"sku": map[string]any{
					"name": marketplacePlanName,
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
	}

	client := newDatadogClient(endpoint, account, subscriptionKey)

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
			name:     "DeleteMonitoredSubscriptions",
			method:   http.MethodDelete,
			path:     monitoredSubscriptionBase + "?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteMonitor",
			method:   http.MethodDelete,
			path:     monitorBase + "?api-version=" + datadogAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
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
