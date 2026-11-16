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
	resourcesAPIVersion  = "2021-04-01"
	lighthouseAPIVersion = "2022-10-01"
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
		"managedservices-2022-10-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-lighthouse-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	definitionID := getenv("STACKYARD_AZURE_LIGHTHOUSE_DEFINITION", "f1c3f8a1-2a6b-4c3e-9d6b-0b1c2d3e4f50")
	assignmentID := getenv("STACKYARD_AZURE_LIGHTHOUSE_ASSIGNMENT", "0b4c8d2e-5f6a-4b7c-8d9e-0f1a2b3c4d5e")

	fmt.Printf("Stackyard Azure Lighthouse (managedservices-2022-10-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	definition := base + "/providers/Microsoft.ManagedServices/registrationDefinitions/" + definitionID
	assignment := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedServices/registrationAssignments/" + assignmentID
	q := "?api-version=" + lighthouseAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateRegistrationDefinition", method: http.MethodPut, path: definition + q, payload: map[string]any{"properties": map[string]any{"registrationDefinitionName": "Contoso managed monitoring", "description": "Monitoring Reader for the Contoso ops team", "managedByTenantId": "2c1f5d3e-8a7b-4c6d-9e0f-1a2b3c4d5e6f", "authorizations": []any{map[string]any{"principalId": "6f3e2d1c-0b9a-4876-a543-210fedcba987", "principalIdDisplayName": "Contoso ops", "roleDefinitionId": "43d0d8ad-25c7-4714-9337-8ba259a9fe05"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AssignToResourceGroup", method: http.MethodPut, path: assignment + q, payload: map[string]any{"properties": map[string]any{"registrationDefinitionId": "/subscriptions/" + subscriptionID + "/providers/Microsoft.ManagedServices/registrationDefinitions/" + definitionID}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListAssignments", method: http.MethodGet, path: base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedServices/registrationAssignments" + q, statuses: []int{http.StatusOK}},
		{name: "ListDefinitions", method: http.MethodGet, path: base + "/providers/Microsoft.ManagedServices/registrationDefinitions" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteAssignment", method: http.MethodDelete, path: assignment + q, statuses: []int{http.StatusOK}},
		{name: "DeleteDefinition", method: http.MethodDelete, path: definition + q, statuses: []int{http.StatusOK}},
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
