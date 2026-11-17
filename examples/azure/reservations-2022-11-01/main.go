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
	"github.com/google/uuid"
)

const (
	resourcesAPIVersion    = "2021-04-01"
	reservationsAPIVersion = "2022-11-01"
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
		"reservations-2022-11-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-reservations-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	orderID := getenv("STACKYARD_AZURE_RESERVATION_ORDER_ID", "1f14354c-dc12-4c8d-8090-6f295a3a34aa")

	fmt.Printf("Stackyard Azure Reservations (reservations-2022-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	capacity := "/azure/providers/Microsoft.Capacity"
	order := capacity + "/reservationOrders/" + orderID
	reservation := order + "/reservations/" + uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.ToLower(strings.TrimPrefix(order, "/azure")))).String()
	purchase := map[string]any{"location": region, "sku": map[string]any{"name": "Standard_D2s_v5"}, "properties": map[string]any{
		"reservedResourceType": "VirtualMachines", "billingScopeId": strings.TrimPrefix(base, "/azure"), "term": "P1Y",
		"billingPlan": "Monthly", "quantity": 4, "displayName": "web tier", "appliedScopeType": "Shared",
	}}
	q := "?api-version=" + reservationsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListCatalog", method: http.MethodGet, path: base + "/providers/Microsoft.Capacity/catalogs" + q, statuses: []int{http.StatusOK}},
		{name: "CalculatePrice", method: http.MethodPost, path: capacity + "/calculatePrice" + q, payload: purchase, statuses: []int{http.StatusOK}},
		{name: "PurchaseReservationOrder", method: http.MethodPut, path: order + q, payload: purchase, statuses: []int{http.StatusOK}},
		{name: "GetReservation", method: http.MethodGet, path: reservation + q, statuses: []int{http.StatusOK}},
		{name: "ScopeToSubscription", method: http.MethodPatch, path: reservation + q, payload: map[string]any{"properties": map[string]any{
			"name": "web tier (prod)", "appliedScopeType": "Single", "appliedScopes": []string{strings.TrimPrefix(base, "/azure")},
		}}, statuses: []int{http.StatusOK}},
		{name: "SplitReservation", method: http.MethodPost, path: order + "/split" + q, payload: map[string]any{"properties": map[string]any{"quantities": []int{1, 3}, "reservationId": strings.TrimPrefix(reservation, "/azure")}}, statuses: []int{http.StatusOK}},
		{name: "ListOrderReservations", method: http.MethodGet, path: order + "/reservations" + q, statuses: []int{http.StatusOK}},
		{name: "ArchiveSplitSource", method: http.MethodPost, path: reservation + "/archive" + q, statuses: []int{http.StatusOK}},
		{name: "ListAllReservations", method: http.MethodGet, path: capacity + "/reservations" + q, statuses: []int{http.StatusOK}},
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
