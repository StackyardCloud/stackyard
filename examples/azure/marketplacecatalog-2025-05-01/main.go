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

const (
	resourcesAPIVersion = "2021-04-01"
	productsAPIVersion  = "2025-05-01"
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
		"marketplacecatalog-2025-05-01",
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
	// Search and the API key operations require the caller's tenant.
	req.Raw().Header.Set("x-ms-client-tenant-id", getenv("STACKYARD_AZURE_TENANT_ID", "00000000-0000-0000-0000-000000000000"))
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-marketplacecatalog-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	keyAlias := getenv("STACKYARD_AZURE_MARKETPLACE_KEY_ALIAS", "stackyard-key")
	billingAccount := getenv("STACKYARD_AZURE_BILLING_ACCOUNT", "12345678:12345678_2019-05-31")

	fmt.Printf("Stackyard Azure Marketplace Catalog (marketplacecatalog-2025-05-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	products := base + "/providers/Microsoft.Marketplace/products"
	billingProducts := "/azure/providers/Microsoft.Billing/billingAccounts/" + billingAccount + "/providers/Microsoft.Marketplace/products"
	keys := base + "/providers/Microsoft.Marketplace/keys"
	kq := "?api-version=2023-01-01-preview"
	filter := url.Values{"$filter": {"productType eq 'SaaS'"}, "$select": {"displayName,uniqueProductId,productType,publisherDisplayName"}, "language": {"en"}}.Encode()
	q := "?api-version=" + productsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListProductsBySubscription", method: http.MethodGet, path: products + q + "&" + filter, statuses: []int{http.StatusOK}},
		{name: "ListProductsByTenant", method: http.MethodGet, path: "/azure/providers/Microsoft.Marketplace/products" + q + "&market=US", statuses: []int{http.StatusOK}},
		{name: "ListProductsByBillingAccount", method: http.MethodGet, path: billingProducts + q + "&locations=eastus", statuses: []int{http.StatusOK}},
		{name: "ListProductsByBillingProfile", method: http.MethodGet, path: "/azure/providers/Microsoft.Billing/billingAccounts/" + billingAccount + "/billingProfiles/ABCD-EFGH-1234-5678/providers/Microsoft.Marketplace/products" + q, statuses: []int{http.StatusOK}},
		{name: "GetProductBySubscription", method: http.MethodGet, path: products + "/securescan-enterprise" + q + "&includeHiddenPlans=true", statuses: []int{http.StatusOK}},
		{name: "GetProductByTenant", method: http.MethodGet, path: "/azure/providers/Microsoft.Marketplace/products/securescan-enterprise" + q + "&market=US", statuses: []int{http.StatusOK}},
		{name: "GetProductByBillingAccount", method: http.MethodGet, path: billingProducts + "/canonical.ubuntu-24_04-lts" + q, statuses: []int{http.StatusOK}},
		{name: "Search", method: http.MethodPost, path: "/azure/providers/Microsoft.Marketplace/search?api-version=2022-02-02", payload: map[string]any{"searchQuery": "a", "take": 2, "languages": []string{"en"}, "stores": []string{"Ibiza"}, "select": []string{"legacyId", "offerType"}, "privateStoreAllowances": []string{"AllowedByDefault"}}, statuses: []int{http.StatusOK}},
		{name: "CreateApiKey", method: http.MethodPost, path: keys + "/" + keyAlias + "/create" + kq, statuses: []int{http.StatusOK}},
		{name: "ListApiKeys", method: http.MethodGet, path: keys + kq, statuses: []int{http.StatusOK}},
		{name: "RevokeApiKey", method: http.MethodPost, path: keys + "/" + keyAlias + "/revoke" + kq, statuses: []int{http.StatusOK}},
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
