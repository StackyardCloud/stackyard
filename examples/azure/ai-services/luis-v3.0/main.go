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

type luisClient struct {
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

func newLuisClient(endpoint, account, subscriptionKey string) *luisClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"luis-v3.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &luisClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *luisClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	appID := getenv("STACKYARD_AZURE_LUIS_APP_ID", "4fdbf3f0-d5de-4a6d-af4b-1213d1e6e123")
	slotName := getenv("STACKYARD_AZURE_LUIS_SLOT_NAME", "staging")
	versionID := getenv("STACKYARD_AZURE_LUIS_VERSION_ID", "0.1")
	query := getenv("STACKYARD_AZURE_LUIS_QUERY", "forward to frank 30 dollars through HSBC")

	fmt.Printf("Stackyard Azure LUIS (luis-v3.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newLuisClient(endpoint, account, subscriptionKey)

	queryEscaped := url.QueryEscape(query)
	postBody := map[string]any{
		"query": query,
		"options": map[string]any{
			"datetimeReference": "2015-02-13T13:15:00.000Z",
		},
		"externalEntities": []map[string]any{
			{
				"entityName":   "Bank",
				"startIndex":   36,
				"entityLength": 4,
				"resolution": map[string]any{
					"text": "International Bank",
				},
			},
		},
		"dynamicLists": []map[string]any{
			{
				"listEntityName": "Employees",
				"requestLists": []map[string]any{
					{
						"name":          "Management",
						"canonicalForm": "Frank",
						"synonyms":      []string{},
					},
				},
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
			name:     "GetSlotPrediction",
			method:   http.MethodPost,
			path:     "/azure/luis/prediction/v3.0/apps/" + appID + "/slots/" + slotName + "/predict?verbose=true&show-all-intents=true&log=true",
			payload:  postBody,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetSlotPredictionGET",
			method:   http.MethodGet,
			path:     "/azure/luis/prediction/v3.0/apps/" + appID + "/slots/" + slotName + "/predict?query=" + queryEscaped + "&verbose=true&show-all-intents=true&log=true",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetVersionPrediction",
			method:   http.MethodPost,
			path:     "/azure/luis/prediction/v3.0/apps/" + appID + "/versions/" + versionID + "/predict?verbose=true&show-all-intents=true&log=true",
			payload:  postBody,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetVersionPredictionGET",
			method:   http.MethodGet,
			path:     "/azure/luis/prediction/v3.0/apps/" + appID + "/versions/" + versionID + "/predict?query=" + queryEscaped + "&verbose=true&show-all-intents=true&log=true",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			fmt.Printf("Route is recognized but not implemented yet: %s\n", call.path)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All luis routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
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
