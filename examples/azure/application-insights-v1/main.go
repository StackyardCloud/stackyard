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

type applicationInsightsClient struct {
	endpoint string
	pipeline runtime.Pipeline
	appID    string
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

func newApplicationInsightsClient(endpoint, account, subscriptionKey, appID string) *applicationInsightsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"application-insights-v1",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &applicationInsightsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
		appID:    appID,
	}
}

func (c *applicationInsightsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

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

func (c *applicationInsightsClient) doRaw(ctx context.Context, method, path string, expectedStatuses ...int) (string, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return "", 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	resp, err := c.pipeline.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}
	if len(expectedStatuses) > 0 {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return "", resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}
	return string(body), resp.StatusCode, nil
}

func (c *applicationInsightsClient) basePath() string {
	return "/azure/application-insights/v1/apps/" + c.appID
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_APPLICATION_INSIGHTS_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPLICATION_INSIGHTS_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	appID := getenv("STACKYARD_AZURE_APPLICATION_INSIGHTS_APP_ID", "app-123")

	fmt.Printf("Stackyard Azure Application Insights (application-insights-v1) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newApplicationInsightsClient(endpoint, account, subscriptionKey, appID)

	paths := []struct {
		name    string
		method  string
		path    string
		payload any
		raw     bool
	}{
		{
			name:   "QueryGet",
			method: http.MethodGet,
			path:   client.basePath() + "/query?query=requests%20%7C%20take%201&timespan=PT30M",
		},
		{
			name:   "QueryPost",
			method: http.MethodPost,
			path:   client.basePath() + "/query",
			payload: map[string]any{
				"query":    "requests | summarize count()",
				"timespan": "PT1H",
			},
		},
		{
			name:   "EventsByType",
			method: http.MethodGet,
			path:   client.basePath() + "/events/requests?$count=true",
		},
		{
			name:   "EventGet",
			method: http.MethodGet,
			path:   client.basePath() + "/events/requests/event-123",
		},
		{
			name:   "EventsMetadata",
			method: http.MethodGet,
			path:   client.basePath() + "/events/$metadata",
			raw:    true,
		},
		{
			name:   "MetricsMetadata",
			method: http.MethodGet,
			path:   client.basePath() + "/metrics/metadata",
		},
		{
			name:   "MetricGet",
			method: http.MethodGet,
			path:   client.basePath() + "/metrics/requests/count?aggregation=sum&segment=request/name",
		},
		{
			name:   "MetricGetMultiple",
			method: http.MethodPost,
			path:   client.basePath() + "/metrics",
			payload: []map[string]any{
				{"id": "requests/count"},
				{"id": "exceptions/count"},
			},
		},
		{
			name:   "MetadataGet",
			method: http.MethodGet,
			path:   client.basePath() + "/metadata",
		},
		{
			name:    "MetadataPost",
			method:  http.MethodPost,
			path:    client.basePath() + "/metadata",
			payload: map[string]any{"includeTables": []string{"requests"}},
		},
	}

	for _, step := range paths {
		if step.raw {
			body, status, err := client.doRaw(ctx, step.method, step.path, http.StatusOK, http.StatusNotImplemented)
			if err != nil {
				exitf("%s failed: %v", step.name, err)
			}
			if status == http.StatusNotImplemented {
				notImplemented(step.path)
				return
			}
			if !strings.Contains(body, "<edmx:Edmx") {
				exitf("%s returned unexpected XML body: %s", step.name, body)
			}
			continue
		}
		_, status, err := client.doJSON(ctx, step.method, step.path, step.payload, http.StatusOK, http.StatusNotImplemented)
		if err != nil {
			exitf("%s failed: %v", step.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplemented(step.path)
			return
		}
	}

	fmt.Println("Done.")
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
