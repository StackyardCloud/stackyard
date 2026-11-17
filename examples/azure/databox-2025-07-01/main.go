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
	dataBoxAPIVersion   = "2025-07-01"
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
		"databox-2025-07-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-databox-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")

	fmt.Printf("Stackyard Azure Data Box (databox-2025-07-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	job := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DataBox/jobs/import-photos"
	locations := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DataBox/locations/" + region
	storageAccountID := strings.TrimPrefix(base, "/azure") + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/photos"
	order := map[string]any{"location": region, "sku": map[string]any{"name": "DataBox", "model": "DataBox"}, "properties": map[string]any{
		"transferType": "ImportToAzure",
		"details": map[string]any{
			"jobDetailsType":    "DataBox",
			"contactDetails":    map[string]any{"contactName": "Avery Chen", "phone": "5550100", "emailList": []any{"ops@contoso.com"}},
			"shippingAddress":   map[string]any{"streetAddress1": "1 Main St", "city": "Redmond", "stateOrProvince": "WA", "country": "US", "postalCode": "98052", "addressType": "Commercial"},
			"dataImportDetails": []any{map[string]any{"accountDetails": map[string]any{"dataAccountType": "StorageAccount", "storageAccountId": storageAccountID}}},
		},
	}}
	q := "?api-version=" + dataBoxAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListAvailableSkus", method: http.MethodPost, path: locations + "/availableSkus" + q, payload: map[string]any{"transferType": "ImportToAzure", "country": "US", "location": region}, statuses: []int{http.StatusOK}},
		{name: "ValidateInputs", method: http.MethodPost, path: locations + "/validateInputs" + q, payload: map[string]any{"validationCategory": "JobCreationValidation", "individualRequestDetails": []any{
			map[string]any{"validationType": "ValidateAddress", "deviceType": "DataBox", "shippingAddress": map[string]any{"streetAddress1": "1 Main St", "country": "US", "postalCode": "98052"}},
			map[string]any{"validationType": "ValidateSkuAvailability", "deviceType": "DataBox", "transferType": "ImportToAzure", "country": "US", "location": region},
		}}, statuses: []int{http.StatusOK}},
		{name: "CreateJob", method: http.MethodPut, path: job + q, payload: order, statuses: []int{http.StatusOK}},
		{name: "UpdateContact", method: http.MethodPatch, path: job + q, payload: map[string]any{"properties": map[string]any{"details": map[string]any{"contactDetails": map[string]any{"phone": "5550199"}}}}, statuses: []int{http.StatusOK}},
		{name: "ListCredentials", method: http.MethodPost, path: job + "/listCredentials" + q, statuses: []int{http.StatusOK}},
		{name: "CancelJob", method: http.MethodPost, path: job + "/cancel" + q, payload: map[string]any{"reason": "Example finished"}, statuses: []int{http.StatusNoContent}},
		{name: "GetCancelledJob", method: http.MethodGet, path: job + q, statuses: []int{http.StatusOK}},
		{name: "DeleteJob", method: http.MethodDelete, path: job + q, statuses: []int{http.StatusOK}},
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
