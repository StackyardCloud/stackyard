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

const edgeHardwareCenterAPIVersion = "2024-02-01"
const edgeHardwareCenterPrefix = "/azure"

type edgeHardwareCenterClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type edgeHardwareCenterSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type edgeHardwareCenterCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

func (p *edgeHardwareCenterSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEdgeHardwareCenterClient(endpoint, account, subscriptionKey string) *edgeHardwareCenterClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"edgehardwarecenter-2024-02-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&edgeHardwareCenterSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &edgeHardwareCenterClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *edgeHardwareCenterClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return edgeHardwareCenterPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EDGEHARDWARECENTER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EDGEHARDWARECENTER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroupName := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-edgehardwarecenter")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	addressName := getenv("STACKYARD_AZURE_EDGEHARDWARECENTER_ADDRESS_NAME", "TestAddressName2")
	orderItemName := getenv("STACKYARD_AZURE_EDGEHARDWARECENTER_ORDER_ITEM_NAME", "TestOrderItemName2")
	orderName := getenv("STACKYARD_AZURE_EDGEHARDWARECENTER_ORDER_NAME", "TestOrderName2")

	client := newEdgeHardwareCenterClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":    subscriptionID,
		"resourceGroupName": resourceGroupName,
		"location":          location,
		"addressName":       addressName,
		"orderItemName":     orderItemName,
		"orderName":         orderName,
	}

	addressCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"addressClassification": "Shipping",
			"contactDetails": map[string]any{
				"contactName":    "XXXX XXXX",
				"emailList":      []any{"xxxx@xxxx.xxx"},
				"phone":          "0000000000",
				"phoneExtension": "",
			},
			"shippingAddress": map[string]any{
				"addressType":     "None",
				"city":            "San Francisco",
				"companyName":     "Microsoft",
				"country":         "US",
				"postalCode":      "94107",
				"stateOrProvince": "CA",
				"streetAddress1":  "16 TOWNSEND ST",
				"streetAddress2":  "UNIT 1",
			},
		},
		"tags": map[string]any{
			"environment": "local",
			"service":     "edgehardwarecenter",
		},
	}

	addressUpdatePayload := map[string]any{
		"properties": map[string]any{
			"contactDetails": map[string]any{
				"contactName":    "YYYY YYYY",
				"emailList":      []any{"xxxx@xxxx.xxx"},
				"phone":          "0000000000",
				"phoneExtension": "",
			},
			"shippingAddress": map[string]any{
				"addressType":     "None",
				"city":            "San Francisco",
				"companyName":     "Microsoft",
				"country":         "US",
				"postalCode":      "94107",
				"stateOrProvince": "CA",
				"streetAddress1":  "16 TOWNSEND ST",
				"streetAddress2":  "UNIT 1",
			},
		},
		"tags": map[string]any{
			"tag1": "value1",
			"tag2": "value2",
		},
	}

	orderID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.EdgeOrder/locations/%s/orders/%s",
		subscriptionID,
		resourceGroupName,
		location,
		orderName,
	)

	orderItemCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"addressDetails": map[string]any{
				"forwardAddress": map[string]any{
					"contactDetails": map[string]any{
						"contactName":    "XXXX XXXX",
						"emailList":      []any{"xxxx@xxxx.xxx"},
						"phone":          "0000000000",
						"phoneExtension": "",
					},
					"shippingAddress": map[string]any{
						"addressType":     "None",
						"city":            "San Francisco",
						"companyName":     "Microsoft",
						"country":         "US",
						"postalCode":      "94107",
						"stateOrProvince": "CA",
						"streetAddress1":  "16 TOWNSEND ST",
						"streetAddress2":  "UNIT 1",
					},
				},
			},
			"orderId": orderID,
			"orderItemDetails": map[string]any{
				"orderItemType": "Purchase",
				"preferences": map[string]any{
					"transportPreferences": map[string]any{
						"preferredShipmentType": "MicrosoftManaged",
					},
				},
				"productDetails": map[string]any{
					"hierarchyInformation": map[string]any{
						"configurationName": "edgep_base",
						"productFamilyName": "azurestackedge",
						"productLineName":   "azurestackedge",
						"productName":       "azurestackedgegpu",
					},
				},
			},
		},
		"tags": map[string]any{
			"environment": "local",
		},
	}

	orderItemUpdatePayload := map[string]any{
		"properties": map[string]any{
			"preferences": map[string]any{
				"transportPreferences": map[string]any{
					"preferredShipmentType": "CustomerManaged",
				},
			},
			"notificationEmailList": []any{"alerts@example.com"},
		},
	}

	orderItemCancelPayload := map[string]any{
		"reason": "Order cancelled",
	}

	orderItemReturnPayload := map[string]any{
		"returnReason":        "Order returned",
		"serviceTag":          "SRV-123456",
		"shippingBoxRequired": true,
	}

	listConfigurationsPayload := map[string]any{
		"configurationFilter": map[string]any{
			"filterableProperty": []any{
				map[string]any{
					"type":            "ShipToCountries",
					"supportedValues": []any{"US"},
				},
			},
			"hierarchyInformation": map[string]any{
				"productFamilyName": "azurestackedge",
				"productLineName":   "azurestackedge",
				"productName":       "azurestackedgegpu",
			},
		},
	}

	listProductFamiliesPayload := map[string]any{
		"filterableProperties": map[string]any{
			"azurestackedge": []any{
				map[string]any{
					"type":            "ShipToCountries",
					"supportedValues": []any{"US"},
				},
			},
		},
	}

	calls := []edgeHardwareCenterCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.EdgeOrder/operations?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "addresses_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/addresses?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "addresses_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "addresses_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01",
			payload:  addressCreatePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "addresses_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "addresses_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01",
			payload:  addressUpdatePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "order_items_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/orderItems?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "order_items_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "order_items_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}?api-version=2024-02-01",
			payload:  orderItemCreatePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "order_items_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "order_items_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}?api-version=2024-02-01",
			payload:  orderItemUpdatePayload,
			statuses: []int{200, 202},
		},
		{
			name:     "order_items_cancel",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}/cancel?api-version=2024-02-01",
			payload:  orderItemCancelPayload,
			statuses: []int{200, 204},
		},
		{
			name:     "order_items_return",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}/return?api-version=2024-02-01",
			payload:  orderItemReturnPayload,
			statuses: []int{200, 202},
		},
		{
			name:     "orders_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/orders?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "orders_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orders?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "orders_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/locations/{location}/orders/{orderName}?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "products_and_configurations_list_configurations",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/listConfigurations?api-version=2024-02-01",
			payload:  listConfigurationsPayload,
			statuses: []int{200},
		},
		{
			name:     "products_and_configurations_list_product_families",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/listProductFamilies?api-version=2024-02-01&%24expand=configurations",
			payload:  listProductFamiliesPayload,
			statuses: []int{200},
		},
		{
			name:     "products_and_configurations_list_product_families_metadata",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.EdgeOrder/productFamiliesMetadata?api-version=2024-02-01",
			statuses: []int{200},
		},
		{
			name:     "order_items_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/orderItems/{orderItemName}?api-version=2024-02-01",
			statuses: []int{202, 204},
		},
		{
			name:     "addresses_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.EdgeOrder/addresses/{addressName}?api-version=2024-02-01",
			statuses: []int{202, 204},
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
