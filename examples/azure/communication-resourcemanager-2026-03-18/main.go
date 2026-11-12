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

const communicationAPIVersion = "2026-03-18"

type communicationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationClient(endpoint, account, subscriptionKey string) *communicationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-resourcemanager-2026-03-18",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_COMMUNICATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-communication")
	location := getenv("STACKYARD_AZURE_LOCATION", "global")
	communicationName := getenv("STACKYARD_AZURE_COMMUNICATION_NAME", "stackyardcomm")
	emailServiceName := getenv("STACKYARD_AZURE_EMAIL_SERVICE_NAME", "stackyardemail")

	fmt.Printf("Stackyard Azure Communication ARM example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	communicationBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Communication/communicationServices/" + communicationName
	emailServiceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Communication/emailServices/" + emailServiceName
	domainBase := emailServiceBase + "/domains/mail.contoso.com"

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
			path:     "/azure/providers/Microsoft.Communication/operations?api-version=" + communicationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   subscriptionBase + "/providers/Microsoft.Communication/checkNameAvailability?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"name": communicationName,
				"type": "Microsoft.Communication/CommunicationServices",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateCommunicationService",
			method: http.MethodPut,
			path:   communicationBase + "?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{"type": "SystemAssigned"},
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"dataLocation":        "United States",
					"disableLocalAuth":    false,
					"publicNetworkAccess": "Enabled",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListKeys",
			method:   http.MethodPost,
			path:     communicationBase + "/listKeys?api-version=" + communicationAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RegenerateKey",
			method: http.MethodPost,
			path:   communicationBase + "/regenerateKey?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"keyType": "Primary",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateEmailService",
			method: http.MethodPut,
			path:   emailServiceBase + "?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"dataLocation": "United States",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListVerifiedExchangeOnlineDomains",
			method:   http.MethodPost,
			path:     subscriptionBase + "/providers/Microsoft.Communication/listVerifiedExchangeOnlineDomains?api-version=" + communicationAPIVersion,
			payload:  map[string]any{},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDomain",
			method: http.MethodPut,
			path:   domainBase + "?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"domainManagement":       "CustomerManaged",
					"userEngagementTracking": "Enabled",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateSenderUsername",
			method: http.MethodPut,
			path:   domainBase + "/senderUsernames/sender-a?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"username":    "sender-a",
					"displayName": "Sender A",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateSuppressionList",
			method: http.MethodPut,
			path:   domainBase + "/suppressionLists/list-a?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"listName": "List A",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateSuppressionListAddress",
			method: http.MethodPut,
			path:   domainBase + "/suppressionLists/list-a/suppressionListAddresses/address-a?api-version=" + communicationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"email":     "user@example.com",
					"firstName": "User",
					"lastName":  "Example",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
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
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(body))
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
