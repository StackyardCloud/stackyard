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

const dellStorageAPIVersion = "2025-03-21"

type dellStorageClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dellStorageSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dellStorageSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDellStorageClient(endpoint, account, subscriptionKey string) *dellStorageClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"dellstorage-2025-03-21",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dellStorageSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &dellStorageClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dellStorageClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DELLSTORAGE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DELLSTORAGE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-dellstorage")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	fileSystemName := getenv("STACKYARD_AZURE_DELLSTORAGE_FILESYSTEM_NAME", "dell-fs-example")
	delegatedSubnetID := getenv(
		"STACKYARD_AZURE_DELLSTORAGE_DELEGATED_SUBNET_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/dell-vnet/subnets/dell-delegated-subnet",
			subscriptionID,
			resourceGroup,
		),
	)
	delegatedSubnetCIDR := getenv("STACKYARD_AZURE_DELLSTORAGE_DELEGATED_SUBNET_CIDR", "10.42.0.0/24")
	userEmail := getenv("STACKYARD_AZURE_DELLSTORAGE_USER_EMAIL", "ops@example.com")
	dellReferenceNumber := getenv("STACKYARD_AZURE_DELLSTORAGE_REFERENCE_NUMBER", "DELL-REF-0001")
	marketplacePlanID := getenv("STACKYARD_AZURE_DELLSTORAGE_MARKETPLACE_PLAN_ID", "plan-standard")
	marketplaceOfferID := getenv("STACKYARD_AZURE_DELLSTORAGE_MARKETPLACE_OFFER_ID", "offer-dell-filesystem")
	marketplacePlanName := getenv("STACKYARD_AZURE_DELLSTORAGE_MARKETPLACE_PLAN_NAME", "dell-storage-plan")

	client := newDellStorageClient(endpoint, account, subscriptionKey)

	fileSystemBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Dell.Storage/filesystems/%s",
		subscriptionID,
		resourceGroup,
		fileSystemName,
	)

	createPayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
			"service":     "dellstorage",
		},
		"properties": map[string]any{
			"marketplace": map[string]any{
				"planId":   marketplacePlanID,
				"offerId":  marketplaceOfferID,
				"planName": marketplacePlanName,
			},
			"delegatedSubnetId":   delegatedSubnetID,
			"delegatedSubnetCidr": delegatedSubnetCIDR,
			"user": map[string]any{
				"email": userEmail,
			},
			"dellReferenceNumber": dellReferenceNumber,
			"encryption": map[string]any{
				"encryptionType": "Microsoft-managed keys (MMK)",
			},
		},
	}

	updatePayload := map[string]any{
		"tags": map[string]any{
			"environment": "local-updated",
			"service":     "dellstorage",
		},
		"properties": map[string]any{
			"delegatedSubnetId": delegatedSubnetID,
			"capacity": map[string]any{
				"current": "10",
			},
			"encryption": map[string]any{
				"encryptionType": "Microsoft-managed keys (MMK)",
			},
		},
	}

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Dell.Storage/operations?api-version=" + dellStorageAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFileSystemsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Dell.Storage/filesystems?api-version=%s", subscriptionID, dellStorageAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateFileSystem",
			method: http.MethodPut,
			path:   fileSystemBase + "?api-version=" + dellStorageAPIVersion,
			payload: map[string]any{
				"location":   createPayload["location"],
				"tags":       createPayload["tags"],
				"properties": createPayload["properties"],
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetFileSystem",
			method:   http.MethodGet,
			path:     fileSystemBase + "?api-version=" + dellStorageAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFileSystemsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Dell.Storage/filesystems?api-version=%s", subscriptionID, resourceGroup, dellStorageAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateFileSystem",
			method:   http.MethodPatch,
			path:     fileSystemBase + "?api-version=" + dellStorageAPIVersion,
			payload:  updatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteFileSystem",
			method:   http.MethodDelete,
			path:     fileSystemBase + "?api-version=" + dellStorageAPIVersion,
			statuses: []int{http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		response, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
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
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
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
