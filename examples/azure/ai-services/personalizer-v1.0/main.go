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

type personalizerClient struct {
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

func newPersonalizerClient(endpoint, account, subscriptionKey string) *personalizerClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"personalizer-v1.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &personalizerClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *personalizerClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	eventID := getenv("STACKYARD_AZURE_PERSONALIZER_EVENT_ID", "event-1")

	fmt.Printf("Stackyard Azure Personalizer (personalizer-v1.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newPersonalizerClient(endpoint, account, subscriptionKey)

	rankPath := "/azure/personalizer/v1.0/rank"
	_, status, err := client.doJSON(ctx, http.MethodPost, rankPath, map[string]any{
		"contextFeatures": []map[string]any{
			{"timeOfDay": "morning", "deviceType": "mobile"},
		},
		"actions": []map[string]any{
			{"id": "action-1", "features": []map[string]any{{"topic": "news"}}},
			{"id": "action-2", "features": []map[string]any{{"topic": "sports"}}},
		},
		"eventId":         eventID,
		"excludedActions": []string{},
		"deferActivation": false,
	}, http.StatusOK, http.StatusCreated, http.StatusNotImplemented)
	if err != nil {
		exitf("Rank failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(rankPath)
		return
	}

	rewardPath := "/azure/personalizer/v1.0/events/" + eventID + "/reward"
	_, status, err = client.doJSON(ctx, http.MethodPost, rewardPath, map[string]any{
		"value": 1.0,
	}, http.StatusOK, http.StatusNoContent, http.StatusNotImplemented)
	if err != nil {
		exitf("Reward failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(rewardPath)
		return
	}

	activatePath := "/azure/personalizer/v1.0/events/" + eventID + "/activate"
	_, status, err = client.doJSON(ctx, http.MethodPost, activatePath, nil, http.StatusOK, http.StatusNoContent, http.StatusNotImplemented)
	if err != nil {
		exitf("Activate failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(activatePath)
		return
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
