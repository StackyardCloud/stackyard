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

const loadTestingDataPlaneAPIVersion = "2022-11-01"

type loadTestingDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type loadTestingDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *loadTestingDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newLoadTestingDataPlaneClient(endpoint, account, subscriptionKey string) *loadTestingDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"loadtesting-dataplane-2022-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&loadTestingDataPlaneSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &loadTestingDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *loadTestingDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, readErr)
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

func (c *loadTestingDataPlaneClient) doRaw(ctx context.Context, method, path, contentType string, body []byte, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Raw().Header.Set("Content-Type", contentType)
		req.Raw().Body = io.NopCloser(strings.NewReader(string(body)))
		req.Raw().ContentLength = int64(len(body))
	}
	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, readErr)
	}
	if err := ensureStatus(resp.StatusCode, respBody, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	var decoded map[string]any
	if len(strings.TrimSpace(string(respBody))) > 0 {
		if err := json.Unmarshal(respBody, &decoded); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
		}
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) > 0 {
		for _, expected := range expectedStatuses {
			if status == expected {
				return nil
			}
		}
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_LOADTESTING_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_LOADTESTING_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	testID := getenv("STACKYARD_AZURE_LOADTESTING_TEST_ID", "test-a")
	runID := getenv("STACKYARD_AZURE_LOADTESTING_TEST_RUN_ID", "run-a")

	fmt.Printf("Stackyard Azure Load Testing Data Plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newLoadTestingDataPlaneClient(endpoint, account, subscriptionKey)
	base := "/azure/loadtesting-data-plane/2022-11-01"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "CreateTest",
			method: http.MethodPatch,
			path:   base + "/tests/" + testID + "?api-version=" + loadTestingDataPlaneAPIVersion,
			payload: map[string]any{
				"displayName": "Stackyard Smoke Test",
				"description": "Stackyard load test definition",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTests",
			method:   http.MethodGet,
			path:     base + "/tests?api-version=" + loadTestingDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchAppComponents",
			method: http.MethodPatch,
			path:   base + "/tests/" + testID + "/app-components?api-version=" + loadTestingDataPlaneAPIVersion,
			payload: map[string]any{
				"components": map[string]any{
					"/subscriptions/000/resourceGroups/rg/providers/Microsoft.Web/sites/site-a": map[string]any{
						"resourceId": "site-a",
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateRun",
			method: http.MethodPatch,
			path:   base + "/test-runs/" + runID + "?api-version=" + loadTestingDataPlaneAPIVersion,
			payload: map[string]any{
				"testId":      testID,
				"displayName": "Stackyard Run",
				"description": "run triggered from example",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetRun",
			method:   http.MethodGet,
			path:     base + "/test-runs/" + runID + "?api-version=" + loadTestingDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "MetricNamespaces",
			method:   http.MethodGet,
			path:     base + "/test-runs/" + runID + "/metric-namespaces?api-version=" + loadTestingDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "MetricDefinitions",
			method:   http.MethodGet,
			path:     base + "/test-runs/" + runID + "/metric-definitions?api-version=" + loadTestingDataPlaneAPIVersion + "&metricNamespace=LoadTestRunMetrics",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "MetricDimensions",
			method:   http.MethodGet,
			path:     base + "/test-runs/" + runID + "/metric-dimensions/result?api-version=" + loadTestingDataPlaneAPIVersion + "&metricName=response_time_ms&metricNamespace=LoadTestRunMetrics",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "Metrics",
			method:   http.MethodPost,
			path:     base + "/test-runs/" + runID + "/metrics?api-version=" + loadTestingDataPlaneAPIVersion + "&metricname=response_time_ms&metricNamespace=LoadTestRunMetrics&timespan=PT5M&aggregation=Average&interval=PT1M",
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	filePayload, status, err := client.doRaw(
		ctx,
		http.MethodPut,
		base+"/tests/"+testID+"/files/test-plan.jmx?api-version="+loadTestingDataPlaneAPIVersion+"&fileType=JMX_FILE",
		"application/octet-stream",
		[]byte("jmeter test plan"),
		http.StatusOK,
	)
	if err != nil {
		fatalf("UploadTestFile failed: %v", err)
	}
	printJSON("UploadTestFile", status, filePayload)

	stopPayload, status, err := client.doJSON(ctx, http.MethodPost, base+"/test-runs/"+runID+":stop?api-version="+loadTestingDataPlaneAPIVersion, nil, http.StatusOK)
	if err != nil {
		fatalf("StopRun failed: %v", err)
	}
	printJSON("StopRun", status, stopPayload)
}

func printJSON(name string, status int, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("marshal %s response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(body))
}

func asString(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
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
