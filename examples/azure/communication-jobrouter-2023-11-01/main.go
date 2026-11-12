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

const communicationJobRouterAPIVersion = "2023-11-01"

type communicationJobRouterClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type communicationJobRouterSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *communicationJobRouterSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newCommunicationJobRouterClient(endpoint, account, subscriptionKey string) *communicationJobRouterClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"communication-jobrouter-2023-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&communicationJobRouterSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &communicationJobRouterClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *communicationJobRouterClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Raw().Header.Set(key, value)
	}
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/merge-patch+json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, resp.Header, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, resp.Header, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, resp.Header, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_COMMUNICATION_JOBROUTER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_COMMUNICATION_JOBROUTER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Communication Job Router example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newCommunicationJobRouterClient(endpoint, account, subscriptionKey)
	base := "/azure/communication-jobrouter/2023-11-01/routing"

	_, _, _, err := client.doJSON(ctx, http.MethodPatch, base+"/distributionPolicies/dist-a?api-version="+communicationJobRouterAPIVersion, map[string]any{
		"name": "dist-a",
		"mode": map[string]any{"kind": "roundRobin"},
	}, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		fatalf("UpsertDistributionPolicy failed: %v", err)
	}

	queuePayload, status, _, err := client.doJSON(ctx, http.MethodPatch, base+"/queues/queue-a?api-version="+communicationJobRouterAPIVersion, map[string]any{
		"name":                 "Main",
		"distributionPolicyId": "dist-a",
		"labels":               map[string]any{"env": "example"},
	}, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		fatalf("UpsertQueue failed: %v", err)
	}
	printJSON("UpsertQueue", status, queuePayload)

	workerPayload, status, _, err := client.doJSON(ctx, http.MethodPatch, base+"/workers/worker-a?api-version="+communicationJobRouterAPIVersion, map[string]any{
		"queueIds": []string{"queue-a"},
		"capacity": 1,
		"labels":   map[string]any{"lang": "en"},
	}, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		fatalf("UpsertWorker failed: %v", err)
	}
	printJSON("UpsertWorker", status, workerPayload)

	jobPayload, status, _, err := client.doJSON(ctx, http.MethodPatch, base+"/jobs/job-a?api-version="+communicationJobRouterAPIVersion, map[string]any{
		"channelId": "chat",
		"queueId":   "queue-a",
		"priority":  10,
		"labels":    map[string]any{"topic": "sales"},
	}, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		fatalf("UpsertJob failed: %v", err)
	}
	printJSON("UpsertJob", status, jobPayload)

	workerGetPayload, status, _, err := client.doJSON(ctx, http.MethodGet, base+"/workers/worker-a?api-version="+communicationJobRouterAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetWorker failed: %v", err)
	}
	printJSON("GetWorker", status, workerGetPayload)

	offerID := ""
	if offers, ok := workerGetPayload["offers"].([]any); ok && len(offers) > 0 {
		if offer, ok := offers[0].(map[string]any); ok {
			offerID = fmt.Sprintf("%v", offer["id"])
		}
	}
	if strings.TrimSpace(offerID) == "" {
		fatalf("no offer id returned from worker payload")
	}

	acceptPayload, status, _, err := client.doJSON(ctx, http.MethodPost, base+"/workers/worker-a/offers/"+offerID+":accept?api-version="+communicationJobRouterAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("AcceptJobOffer failed: %v", err)
	}
	printJSON("AcceptJobOffer", status, acceptPayload)

	statsPayload, status, _, err := client.doJSON(ctx, http.MethodGet, base+"/queues/queue-a/statistics?api-version="+communicationJobRouterAPIVersion, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetQueueStatistics failed: %v", err)
	}
	printJSON("GetQueueStatistics", status, statsPayload)
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
		fatalf("format %s response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(formatted))
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
