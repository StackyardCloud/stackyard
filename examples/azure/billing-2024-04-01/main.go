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

const billingAPIVersion = "2024-04-01"

type billingClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type billingSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *billingSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newBillingClient(endpoint, account, subscriptionKey string) *billingClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"billing-2024-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&billingSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &billingClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *billingClient) doJSON(ctx context.Context, method, path string, payload any, expected ...int) (map[string]any, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}
	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expected...); err != nil {
		return nil, err
	}
	if len(body) == 0 || strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, nil
}

func ensureStatus(status int, body []byte, expected ...int) error {
	for _, candidate := range expected {
		if status == candidate {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_BILLING_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_BILLING_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	billingAccountName := getenv("STACKYARD_AZURE_BILLING_ACCOUNT_NAME", "mca-account")

	fmt.Printf("Stackyard Azure Billing example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newBillingClient(endpoint, account, subscriptionKey)
	base := "/azure/providers/Microsoft.Billing"
	accountBase := base + "/billingAccounts/" + billingAccountName
	subscriptionBase := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Billing"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "ValidateAddress",
			method: http.MethodPost,
			path:   base + "/validateAddress?api-version=" + billingAPIVersion,
			payload: map[string]any{
				"address": map[string]any{
					"line1":   "1 Microsoft Way",
					"city":    "Redmond",
					"country": "US",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListBillingAccounts",
			method:   http.MethodGet,
			path:     base + "/billingAccounts?api-version=" + billingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetBillingAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + billingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateBillingProfile",
			method: http.MethodPut,
			path:   accountBase + "/billingProfiles/profile-example?api-version=" + billingAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Example Profile",
					"currency":    "USD",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateInvoiceSection",
			method: http.MethodPut,
			path:   accountBase + "/billingProfiles/profile-example/invoiceSections/section-example?api-version=" + billingAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Example Section",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetBillingProperty",
			method:   http.MethodGet,
			path:     subscriptionBase + "/billingProperty/default?api-version=" + billingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPaymentMethods",
			method:   http.MethodGet,
			path:     accountBase + "/paymentMethods?api-version=" + billingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListInvoices",
			method:   http.MethodGet,
			path:     accountBase + "/invoices?api-version=" + billingAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateBillingRequest",
			method: http.MethodPut,
			path:   base + "/billingRequests/request-example?api-version=" + billingAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"status": "Submitted",
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s ok\n", call.name)
		if len(payload) > 0 {
			if pretty, err := json.MarshalIndent(payload, "", "  "); err == nil {
				fmt.Println(string(pretty))
			}
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
