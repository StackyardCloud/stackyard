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

const analyzeTextAPIVersion = "2024-11-01"

type analyzeTextClient struct {
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

func newAnalyzeTextClient(endpoint, account, subscriptionKey string) *analyzeTextClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-language-analyze-text-2024-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &analyzeTextClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *analyzeTextClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	taskKind := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_TEXT_KIND", "SentimentAnalysis")
	documentLanguage := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_TEXT_LANGUAGE", "en")
	documentText := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_TEXT_DOCUMENT", "The food was delicious and the staff was friendly.")

	fmt.Printf("Stackyard Azure AI Services - Language - Analyze Text (ai-services-language-analyze-text-2024-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAnalyzeTextClient(endpoint, account, subscriptionKey)

	path := "/azure/language/:analyze-text?api-version=" + analyzeTextAPIVersion + "&showStats=true"
	payload := map[string]any{
		"kind": taskKind,
		"analysisInput": map[string]any{
			"documents": []map[string]any{
				{
					"id":       "1",
					"language": documentLanguage,
					"text":     documentText,
				},
			},
		},
		"parameters": map[string]any{
			"modelVersion": "latest",
		},
	}

	_, status, err := client.doJSON(ctx, http.MethodPost, path, payload, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("AnalyzeText failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		fmt.Printf("Route is recognized but not implemented yet: %s\n", path)
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
