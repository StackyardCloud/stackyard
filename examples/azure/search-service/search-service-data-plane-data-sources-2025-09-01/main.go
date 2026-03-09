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

const searchServiceDataSourcesAPIVersion = "2025-09-01"

type searchServiceDataSourcesClient struct {
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

func newSearchServiceDataSourcesClient(endpoint, account, subscriptionKey string) *searchServiceDataSourcesClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"search-service-data-plane-data-sources-2025-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &searchServiceDataSourcesClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *searchServiceDataSourcesClient) doJSON(ctx context.Context, method, path string, payload any, extraHeaders map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Raw().Header.Set(k, v)
	}

	if payload != nil {
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

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
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
	account := getenv("STACKYARD_AZURE_SEARCH_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SEARCH_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	dataSourceName := getenv("STACKYARD_AZURE_SEARCH_DATASOURCE_NAME", "hotels-ds")
	containerName := getenv("STACKYARD_AZURE_SEARCH_DATASOURCE_CONTAINER", "hotels")

	fmt.Printf("Stackyard Azure Search Service - Data Plane - Data Sources (search-service-data-plane-data-sources-2025-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newSearchServiceDataSourcesClient(endpoint, account, subscriptionKey)

	collectionPath := "/azure/datasources"
	entityPath := "/azure/datasources('" + dataSourceName + "')"

	basePayload := map[string]any{
		"name": dataSourceName,
		"type": "azuresql",
		"credentials": map[string]any{
			"connectionString": "Server=tcp:local,1433;Database=searchdb;User Id=sa;Password=P@ssw0rd!",
		},
		"container": map[string]any{
			"name": containerName,
		},
	}

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		headers  map[string]string
		statuses []int
	}{
		{
			name:     "Create",
			method:   http.MethodPost,
			path:     collectionPath,
			payload:  basePayload,
			statuses: []int{http.StatusCreated, http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:    "CreateOrUpdate",
			method:  http.MethodPut,
			path:    entityPath + "?allowIndexDowntime=true",
			payload: basePayload,
			headers: map[string]string{
				"If-Match": "*",
			},
			statuses: []int{http.StatusCreated, http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "Get",
			method:   http.MethodGet,
			path:     entityPath,
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:     "List",
			method:   http.MethodGet,
			path:     collectionPath + "?$select=name&$top=10",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "Delete",
			method: http.MethodDelete,
			path:   entityPath,
			headers: map[string]string{
				"If-Match": "*",
			},
			statuses: []int{http.StatusNoContent, http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		fullPath := withAPIVersion(call.path, searchServiceDataSourcesAPIVersion)
		_, status, err := client.doJSON(ctx, call.method, fullPath, call.payload, call.headers, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			notImplemented(fullPath)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All search-service data-plane data-sources routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func withAPIVersion(path, apiVersion string) string {
	if strings.Contains(path, "?") {
		return path + "&api-version=" + apiVersion
	}
	return path + "?api-version=" + apiVersion
}

func notImplemented(path string) {
	fmt.Printf("Route is recognized but not implemented yet: %s\n", path)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
