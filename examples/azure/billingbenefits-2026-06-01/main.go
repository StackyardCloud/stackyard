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
	resourcesAPIVersion       = "2021-04-01"
	billingBenefitsAPIVersion = "2026-06-01"
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
		"billingbenefits-2026-06-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-billingbenefits-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	aliasName := getenv("STACKYARD_AZURE_SAVINGS_PLAN_ALIAS", "stackyard-savings-plan")

	fmt.Printf("Stackyard Azure Billing Benefits (billingbenefits-2026-06-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	root := "/azure/providers/Microsoft.BillingBenefits/"
	alias := root + "savingsPlanOrderAliases/" + aliasName
	v := "?api-version=" + billingBenefitsAPIVersion
	sub := "/subscriptions/" + subscriptionID
	purchase := map[string]any{
		"sku": map[string]any{"name": "Compute_Savings_Plan"},
		"properties": map[string]any{
			"displayName": "Compute_SavingsPlan_example", "appliedScopeType": "Shared", "billingPlan": "P1M", "billingScopeId": sub,
			"commitment": map[string]any{"amount": 0.001, "currencyCode": "USD", "grain": "Hourly"}, "term": "P3Y",
		},
	}
	// An alias purchases once, so a rerun reads the earlier purchase.
	if _, status, err := client.doJSON(ctx, http.MethodPut, alias+v, purchase, http.StatusCreated, http.StatusBadRequest); err != nil {
		fmt.Fprintf(os.Stderr, "CreateSavingsPlanOrderAlias failed: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("CreateSavingsPlanOrderAlias -> HTTP %d\n", status)
	}
	created, _, err := client.doJSON(ctx, http.MethodGet, alias+v, nil, http.StatusOK)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSavingsPlanOrderAlias failed: %v\n", err)
		os.Exit(1)
	}
	order := "/azure" + created["properties"].(map[string]any)["savingsPlanOrderId"].(string)
	orderBody, _, err := client.doJSON(ctx, http.MethodGet, order+v, nil, http.StatusOK)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetSavingsPlanOrder failed: %v\n", err)
		os.Exit(1)
	}
	plan := "/azure" + orderBody["properties"].(map[string]any)["savingsPlans"].([]any)[0].(string)
	reservation := map[string]any{
		"location": region,
		"sku":      map[string]any{"name": "Standard_D2s_v5"},
		"properties": map[string]any{
			"displayName": "ReservationOrder_example", "reservedResourceType": "VirtualMachines", "quantity": 2, "term": "P1Y",
			"billingScopeId": sub, "appliedScopeType": "Shared", "reservedResourceProperties": map[string]any{"instanceFlexibility": "On"},
		},
	}
	q := "?api-version=" + billingBenefitsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "GetSavingsPlanOrderWithSchedule", method: http.MethodGet, path: order + q + "&$expand=schedule", statuses: []int{http.StatusOK}},
		{name: "ListSavingsPlanOrders", method: http.MethodGet, path: root + "savingsPlanOrders" + q, statuses: []int{http.StatusOK}},
		{name: "ElevateSavingsPlanOrder", method: http.MethodPost, path: order + "/elevate" + q, statuses: []int{http.StatusOK}},
		{name: "ListSavingsPlansInOrder", method: http.MethodGet, path: order + "/savingsPlans" + q, statuses: []int{http.StatusOK}},
		{name: "GetSavingsPlan", method: http.MethodGet, path: plan + q, statuses: []int{http.StatusOK}},
		{name: "ValidateSavingsPlanUpdate", method: http.MethodPost, path: plan + "/validate" + q, payload: map[string]any{"benefits": []any{
			map[string]any{"appliedScopeType": "Single", "appliedScopeProperties": map[string]any{"subscriptionId": sub}},
			map[string]any{"appliedScopeType": "Single"},
		}}, statuses: []int{http.StatusOK}},
		{name: "UpdateSavingsPlan", method: http.MethodPatch, path: plan + q, payload: map[string]any{"properties": map[string]any{
			"displayName": "Compute_SavingsPlan_renamed", "appliedScopeType": "Single", "appliedScopeProperties": map[string]any{"subscriptionId": sub},
			"renew": true, "renewProperties": map[string]any{"purchaseProperties": map[string]any{"properties": map[string]any{"term": "P1Y"}}},
		}}, statuses: []int{http.StatusOK}},
		{name: "ListAllSavingsPlans", method: http.MethodGet, path: root + "savingsPlans" + q + "&$filter=properties%2fdisplayProvisioningState+eq+'Succeeded'&$orderby=properties/displayName&take=10", statuses: []int{http.StatusOK}},
		{name: "ValidateSavingsPlanPurchase", method: http.MethodPost, path: root + "validate" + q, payload: map[string]any{"benefits": []any{purchase}}, statuses: []int{http.StatusOK}},
		{name: "CreateReservationOrderAlias", method: http.MethodPut, path: root + "reservationOrderAliases/" + aliasName + "-ri" + q, payload: reservation, statuses: []int{http.StatusCreated, http.StatusBadRequest}},
		{name: "GetReservationOrderAlias", method: http.MethodGet, path: root + "reservationOrderAliases/" + aliasName + "-ri" + q, statuses: []int{http.StatusOK}},
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
