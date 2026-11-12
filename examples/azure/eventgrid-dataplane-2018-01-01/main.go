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

const eventGridDataPlaneAPIVersion = "2018-01-01"
const eventGridDataPlanePrefix = "/azure/eventgrid-dataplane/2018-01-01"

type eventGridDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type eventGridDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type eventGridDataPlaneCall struct {
	name        string
	method      string
	template    string
	payload     any
	contentType string
	headers     map[string]string
	statuses    []int
}

func (p *eventGridDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEventGridDataPlaneClient(endpoint, account, subscriptionKey string) *eventGridDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"eventgrid-dataplane-2018-01-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&eventGridDataPlaneSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &eventGridDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *eventGridDataPlaneClient) doJSON(
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
	return eventGridDataPlanePrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EVENTGRID_DATAPLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EVENTGRID_DATAPLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	topicHostname := getenv("STACKYARD_AZURE_EVENTGRID_TOPIC_HOSTNAME", "mytopic.westcentralus-1.eventgrid.azure.net")
	channelNameHeader := getenv("STACKYARD_AZURE_EVENTGRID_CHANNEL_NAME", "channel-a")

	client := newEventGridDataPlaneClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"topicHostname": topicHostname,
	}

	eventGridEventsPayload := []any{
		map[string]any{
			"id":          "b3ccc7e3-c1cb-49bf-b7c8-0d4e60980616",
			"subject":     "TestSubject",
			"eventType":   "Microsoft.Contoso.TestEvent",
			"eventTime":   "2017-12-04T22:06:09.147165Z",
			"dataVersion": "1.0",
			"data": map[string]any{
				"Property1": "Value1",
				"Property2": "Value2",
			},
		},
	}

	cloudEventsPayload := []any{
		map[string]any{
			"id":          "b3ccc7e3-c1cb-49bf-b7c8-0d4e60980617",
			"source":      "/microsoft/autorest/examples/eventgrid/cloud-events/publish",
			"specversion": "1.0",
			"type":        "Microsoft.Contoso.TestEvent",
			"time":        "2017-12-04T22:06:09.147165Z",
			"data": map[string]any{
				"Property1": "Value1",
				"Property2": "Value2",
			},
		},
	}

	customEventsPayload := []any{
		map[string]any{
			"sub":  "Test event",
			"type": "Microsoft.Contoso.TestEvent",
			"ver":  "1.0",
			"data": map[string]any{
				"Property1": "Value1",
				"Property2": "Value2",
			},
		},
	}

	calls := []eventGridDataPlaneCall{
		{
			name:        "publish_event_grid_events",
			method:      http.MethodPost,
			template:    "/{topicHostname}?overload=EventGridEvent&api-version=2018-01-01",
			payload:     eventGridEventsPayload,
			contentType: "application/json",
			statuses:    []int{200},
		},
		{
			name:        "publish_cloud_event_events",
			method:      http.MethodPost,
			template:    "/{topicHostname}?overload=cloudEvent&api-version=2018-01-01",
			payload:     cloudEventsPayload,
			contentType: "application/cloudevents-batch+json; charset=utf-8",
			headers: map[string]string{
				"aeg-channel-name": channelNameHeader,
			},
			statuses: []int{200},
		},
		{
			name:        "publish_custom_event_events",
			method:      http.MethodPost,
			template:    "/{topicHostname}?overload=customEvent&api-version=2018-01-01",
			payload:     customEventsPayload,
			contentType: "application/json",
			statuses:    []int{200},
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
