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

const dynatraceAPIVersion = "2021-09-01"
const dynatracePrefix = "/azure"

type dynatraceClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dynatraceSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type dynatraceCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *dynatraceSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDynatraceClient(endpoint, account, subscriptionKey string) *dynatraceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"dynatrace-2021-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dynatraceSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &dynatraceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dynatraceClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return dynatracePrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DYNATRACE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DYNATRACE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-dynatrace")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	monitorName := getenv("STACKYARD_AZURE_DYNATRACE_MONITOR_NAME", "myMonitor")
	ruleSetName := getenv("STACKYARD_AZURE_DYNATRACE_RULE_SET_NAME", "default")
	configurationName := getenv("STACKYARD_AZURE_DYNATRACE_SSO_CONFIGURATION_NAME", "default")
	userPrincipal := getenv("STACKYARD_AZURE_DYNATRACE_USER_PRINCIPAL", "alice@microsoft.com")
	tenantID := getenv("STACKYARD_AZURE_TENANT_ID", "00000000-0000-0000-0000-000000000000")
	region := getenv("STACKYARD_AZURE_DYNATRACE_REGION", "East US")

	client := newDynatraceClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":    subscriptionID,
		"resourceGroupName": resourceGroupName,
		"monitorName":       monitorName,
		"ruleSetName":       ruleSetName,
		"configurationName": configurationName,
	}

	monitorCreatePayload := map[string]any{
		"location": location,
		"identity": map[string]any{
			"type": "SystemAssigned",
		},
		"properties": map[string]any{
			"monitoringStatus":              "Enabled",
			"marketplaceSubscriptionStatus": "Active",
			"dynatraceEnvironmentProperties": map[string]any{
				"accountInfo":            map[string]any{},
				"environmentInfo":        map[string]any{},
				"singleSignOnProperties": map[string]any{},
			},
			"userInfo": map[string]any{
				"firstName":    "Alice",
				"lastName":     "Bobab",
				"emailAddress": userPrincipal,
				"phoneNumber":  "123456",
				"country":      location,
			},
			"planData": map[string]any{
				"usageType":     "Committed",
				"billingCycle":  "Monthly",
				"planDetails":   "dynatraceapitestplan",
				"effectiveDate": "2019-08-30T15:14:33+02:00",
			},
		},
		"tags": map[string]any{
			"Environment": "Local",
			"Service":     "Dynatrace",
		},
	}

	monitorUpdatePayload := map[string]any{
		"monitoringStatus":              "Enabled",
		"marketplaceSubscriptionStatus": "Active",
		"dynatraceEnvironmentProperties": map[string]any{
			"accountInfo":            map[string]any{},
			"environmentInfo":        map[string]any{},
			"singleSignOnProperties": map[string]any{},
		},
		"userInfo": map[string]any{
			"firstName":    "Alice",
			"lastName":     "Updated",
			"emailAddress": userPrincipal,
			"phoneNumber":  "123456",
			"country":      location,
		},
		"planData": map[string]any{
			"usageType":     "Committed",
			"billingCycle":  "Monthly",
			"planDetails":   "dynatraceapitestplan-updated",
			"effectiveDate": "2019-08-30T15:14:33+02:00",
		},
		"tags": map[string]any{
			"Environment": "LocalUpdated",
			"Service":     "Dynatrace",
		},
	}

	getSSODetailsPayload := map[string]any{
		"userPrincipal": userPrincipal,
	}

	listLinkableEnvironmentsPayload := map[string]any{
		"tenantId":      tenantID,
		"userPrincipal": userPrincipal,
		"region":        region,
	}

	singleSignOnCreatePayload := map[string]any{
		"properties": map[string]any{
			"singleSignOnState": "Enable",
			"enterpriseAppId":   "00000000-0000-0000-0000-000000000000",
			"singleSignOnUrl":   "https://www.dynatrace.io",
			"aadDomains":        []any{"example.onmicrosoft.com"},
		},
	}

	tagRulesCreatePayload := map[string]any{
		"properties": map[string]any{
			"logRules": map[string]any{
				"sendAadLogs":          "Enabled",
				"sendSubscriptionLogs": "Enabled",
				"sendActivityLogs":     "Enabled",
				"filteringTags": []any{
					map[string]any{
						"name":   "Environment",
						"value":  "Prod",
						"action": "Include",
					},
					map[string]any{
						"name":   "Environment",
						"value":  "Dev",
						"action": "Exclude",
					},
				},
			},
			"metricRules": map[string]any{
				"filteringTags": []any{
					map[string]any{
						"name":   "Environment",
						"value":  "Prod",
						"action": "Include",
					},
				},
			},
		},
	}

	tagRulesUpdatePayload := map[string]any{
		"logRules": map[string]any{
			"sendAadLogs":          "Enabled",
			"sendSubscriptionLogs": "Enabled",
			"sendActivityLogs":     "Enabled",
			"filteringTags": []any{
				map[string]any{
					"name":   "Environment",
					"value":  "Prod",
					"action": "Include",
				},
			},
		},
		"metricRules": map[string]any{
			"filteringTags": []any{
				map[string]any{
					"name":   "Environment",
					"value":  "Prod",
					"action": "Include",
				},
			},
		},
	}

	calls := []dynatraceCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Dynatrace.Observability/operations?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list_by_subscription_id",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Dynatrace.Observability/monitors?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01",
			payload:  monitorCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "monitors_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01",
			payload:  monitorUpdatePayload,
			statuses: []int{200},
		},
		{
			name:     "monitors_get_account_credentials",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/getAccountCredentials?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_get_sso_details",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/getSSODetails?api-version=2021-09-01",
			payload:  getSSODetailsPayload,
			statuses: []int{200, 401},
		},
		{
			name:     "monitors_get_vm_host_payload",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/getVMHostPayload?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list_app_services",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listAppServices?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list_hosts",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listHosts?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "monitors_list_linkable_environments",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listLinkableEnvironments?api-version=2021-09-01",
			payload:  listLinkableEnvironmentsPayload,
			statuses: []int{200},
		},
		{
			name:     "monitors_list_monitored_resources",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/listMonitoredResources?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "single_sign_on_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations/{configurationName}?api-version=2021-09-01",
			payload:  singleSignOnCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "single_sign_on_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations/{configurationName}?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "single_sign_on_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/singleSignOnConfigurations?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "tag_rules_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01",
			payload:  tagRulesCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "tag_rules_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "tag_rules_list",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules?api-version=2021-09-01",
			statuses: []int{200},
		},
		{
			name:     "tag_rules_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01",
			payload:  tagRulesUpdatePayload,
			statuses: []int{200},
		},
		{
			name:     "tag_rules_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2021-09-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "monitors_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dynatrace.Observability/monitors/{monitorName}?api-version=2021-09-01",
			statuses: []int{200, 202, 204},
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
