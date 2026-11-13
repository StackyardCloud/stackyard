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
	streamAnalyticsAPIVersion = "2020-03-01"
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
		"streamanalytics-2020-03-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-streamanalytics-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	jobName := getenv("STACKYARD_AZURE_STREAM_ANALYTICS_JOB", "stackyard-clicks")

	fmt.Printf("Stackyard Azure Stream Analytics (streamanalytics-2020-03-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	job := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.StreamAnalytics/streamingjobs/" + jobName
	q := "?api-version=" + streamAnalyticsAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateJob", method: http.MethodPut, path: job + q, payload: map[string]any{"location": region, "properties": map[string]any{"sku": map[string]any{"name": "StandardV2"}, "eventsOutOfOrderPolicy": "Adjust", "outputErrorPolicy": "Drop"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		// The shared access key is accepted but never returned.
		{name: "CreateInput", method: http.MethodPut, path: job + "/inputs/clicks" + q, payload: map[string]any{"properties": map[string]any{"type": "Stream", "datasource": map[string]any{"type": "Microsoft.ServiceBus/EventHub", "properties": map[string]any{"serviceBusNamespace": "stackyard", "eventHubName": "clicks", "sharedAccessPolicyName": "listen", "sharedAccessPolicyKey": "example-key"}}, "serialization": map[string]any{"type": "Json", "properties": map[string]any{"encoding": "UTF8"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateOutput", method: http.MethodPut, path: job + "/outputs/lake" + q, payload: map[string]any{"properties": map[string]any{"datasource": map[string]any{"type": "Microsoft.Storage/Blob", "properties": map[string]any{"container": "clicks", "pathPattern": "{date}"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateTransformation", method: http.MethodPut, path: job + "/transformations/main" + q, payload: map[string]any{"properties": map[string]any{"query": "SELECT * INTO lake FROM clicks", "streamingUnits": 3}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "StartJob", method: http.MethodPost, path: job + "/start" + q, payload: map[string]any{"outputStartMode": "JobStartTime"}, statuses: []int{http.StatusOK}},
		{name: "GetJob", method: http.MethodGet, path: job + q, statuses: []int{http.StatusOK}},
		{name: "StopJob", method: http.MethodPost, path: job + "/stop" + q, statuses: []int{http.StatusOK}},
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
