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

const eventGridNamespaceAPIVersion = "2024-06-01"
const eventGridNamespacePrefix = "/azure/eventgrid-namespace/2024-06-01"

type eventGridNamespaceClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type eventGridNamespaceSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type eventGridNamespaceCall struct {
	name        string
	method      string
	template    string
	payload     any
	contentType string
	headers     map[string]string
	statuses    []int
}

func (p *eventGridNamespaceSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEventGridNamespaceClient(endpoint, account, subscriptionKey string) *eventGridNamespaceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"eventgrid-namespace-2024-06-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&eventGridNamespaceSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &eventGridNamespaceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *eventGridNamespaceClient) doJSON(
	ctx context.Context,
	method string,
	path string,
	payload any,
	contentType string,
	headers map[string]string,
	expectedStatuses ...int,
) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			req.Raw().Header.Set(key, value)
		}
	}

	if payload != nil {
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/json"
		}
		req.Raw().Header.Set("Content-Type", contentType)
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
	return eventGridNamespacePrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	namespaceEndpoint := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_ENDPOINT", "myNamespaceName.westus-1.eventgrid.azure.net")
	topicName := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_TOPIC_NAME", "myTopic")
	eventSubscriptionName := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_EVENT_SUBSCRIPTION_NAME", "myEventSubscription")
	lockToken := getenv("STACKYARD_AZURE_EVENTGRID_NAMESPACE_LOCK_TOKEN", "CgMKATESCQoHdG9rZW4tMQ==")

	client := newEventGridNamespaceClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"endpoint":              namespaceEndpoint,
		"topicName":             topicName,
		"eventSubscriptionName": eventSubscriptionName,
	}

	singleCloudEventPayload := map[string]any{
		"id":          "b3ccc7e3-c1cb-49bf-b7c8-0d4e60980616",
		"source":      "/microsoft/autorest/examples/eventgrid/cloud-events/publish",
		"specversion": "1.0",
		"type":        "Microsoft.Contoso.TestEvent",
		"time":        "2023-05-04T23:06:09.147165Z",
		"data": map[string]any{
			"Property1": "Value1",
			"Property2": "Value2",
		},
	}

	batchCloudEventsPayload := []any{
		map[string]any{
			"id":          "b3ccc7e3-c1cb-49bf-b7c8-0d4e60980617",
			"source":      "/microsoft/autorest/examples/eventgrid/cloud-events/publish",
			"specversion": "1.0",
			"type":        "Microsoft.Contoso.TestEvent",
			"time":        "2023-05-04T23:06:09.147165Z",
			"data": map[string]any{
				"Property1": "Value1",
				"Property2": "Value2",
			},
		},
	}

	lockTokensPayload := map[string]any{
		"lockTokens": []any{lockToken},
	}

	calls := []eventGridNamespaceCall{
		{
			name:        "publish_cloud_event",
			method:      http.MethodPost,
			template:    "/{endpoint}/topics/{topicName}:publish?api-version=2024-06-01",
			payload:     singleCloudEventPayload,
			contentType: "application/cloudevents+json; charset=utf-8",
			statuses:    []int{200},
		},
		{
			name:        "publish_cloud_events",
			method:      http.MethodPost,
			template:    "/{endpoint}/topics/{topicName}:publish?_overload=publishCloudEvents&api-version=2024-06-01",
			payload:     batchCloudEventsPayload,
			contentType: "application/cloudevents-batch+json; charset=utf-8",
			statuses:    []int{200},
		},
		{
			name:     "receive_cloud_events",
			method:   http.MethodPost,
			template: "/{endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:receive?api-version=2024-06-01&maxEvents=1&maxWaitTime=60",
			payload:  nil,
			statuses: []int{200},
		},
		{
			name:     "acknowledge_cloud_events",
			method:   http.MethodPost,
			template: "/{endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:acknowledge?api-version=2024-06-01",
			payload:  lockTokensPayload,
			statuses: []int{200},
		},
		{
			name:     "reject_cloud_events",
			method:   http.MethodPost,
			template: "/{endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:reject?api-version=2024-06-01",
			payload:  lockTokensPayload,
			statuses: []int{200},
		},
		{
			name:     "release_cloud_events",
			method:   http.MethodPost,
			template: "/{endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:release?api-version=2024-06-01",
			payload:  lockTokensPayload,
			statuses: []int{200},
		},
		{
			name:     "renew_cloud_event_locks",
			method:   http.MethodPost,
			template: "/{endpoint}/topics/{topicName}/eventsubscriptions/{eventSubscriptionName}:renewLock?api-version=2024-06-01",
			payload:  lockTokensPayload,
			statuses: []int{200},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.contentType, call.headers, call.statuses...)
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
