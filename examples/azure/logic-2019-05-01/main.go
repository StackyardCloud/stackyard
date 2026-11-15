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
	logicAPIVersion     = "2019-05-01"
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
		"logic-2019-05-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-logic-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	workflowName := getenv("STACKYARD_AZURE_LOGIC_WORKFLOW", "stackyard-orders")
	accountName := getenv("STACKYARD_AZURE_LOGIC_INTEGRATION_ACCOUNT", "stackyard-b2b")

	fmt.Printf("Stackyard Azure Logic Apps (logic-2019-05-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	workflow := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Logic/workflows/" + workflowName
	integration := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Logic/integrationAccounts/" + accountName
	definition := map[string]any{"$schema": "https://schema.management.azure.com/providers/Microsoft.Logic/schemas/2016-06-01/workflowdefinition.json#", "contentVersion": "1.0.0.0", "triggers": map[string]any{"manual": map[string]any{"type": "Request", "kind": "Http"}}, "actions": map[string]any{"Response": map[string]any{"type": "Response", "inputs": map[string]any{"statusCode": 200}}}}
	q := "?api-version=" + logicAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateWorkflow", method: http.MethodPut, path: workflow + q, payload: map[string]any{"location": region, "properties": map[string]any{"definition": definition}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "DisableWorkflow", method: http.MethodPost, path: workflow + "/disable" + q, statuses: []int{http.StatusOK}},
		{name: "EnableWorkflow", method: http.MethodPost, path: workflow + "/enable" + q, statuses: []int{http.StatusOK}},
		{name: "ListWorkflows", method: http.MethodGet, path: base + "/providers/Microsoft.Logic/workflows" + q, statuses: []int{http.StatusOK}},
		{name: "CreateIntegrationAccount", method: http.MethodPut, path: integration + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "Basic"}, "properties": map[string]any{}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListIntegrationAccountCallbackUrl", method: http.MethodPost, path: integration + "/listCallbackUrl" + q, payload: map[string]any{"keyType": "Primary"}, statuses: []int{http.StatusOK}},
		{name: "CreateSchema", method: http.MethodPut, path: integration + "/schemas/order" + q, payload: map[string]any{"properties": map[string]any{"schemaType": "Xml", "contentType": "application/xml", "content": "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"/>"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateMap", method: http.MethodPut, path: integration + "/maps/toInvoice" + q, payload: map[string]any{"properties": map[string]any{"mapType": "Liquid", "contentType": "text/plain", "content": "{ \"id\": \"{{ content.id }}\" }"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListSchemas", method: http.MethodGet, path: integration + "/schemas" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteIntegrationAccount", method: http.MethodDelete, path: integration + q, statuses: []int{http.StatusOK}},
		{name: "DeleteWorkflow", method: http.MethodDelete, path: workflow + q, statuses: []int{http.StatusOK}},
		{name: "GetWorkflowAfterDelete", method: http.MethodGet, path: workflow + q, statuses: []int{http.StatusNotFound}},
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
