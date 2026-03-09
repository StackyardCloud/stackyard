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

type dataPlaneTextModerationClient struct {
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

func newDataPlaneTextModerationClient(endpoint, account, subscriptionKey string) *dataPlaneTextModerationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-data-plane-text-moderation-v1.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &dataPlaneTextModerationClient{endpoint: strings.TrimRight(endpoint, "/"), pipeline: pipeline}
}

func (c *dataPlaneTextModerationClient) doJSON(ctx context.Context, method, path, contentType string, body []byte, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if strings.TrimSpace(contentType) != "" {
		req.Raw().Header.Set("Content-Type", contentType)
	}
	if body != nil {
		req.Raw().Body = io.NopCloser(strings.NewReader(string(body)))
		req.Raw().ContentLength = int64(len(body))
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, err)
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
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	decoded := map[string]any{}
	if len(strings.TrimSpace(string(respBody))) == 0 {
		return decoded, resp.StatusCode, nil
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DATA_PLANE_ACCOUNT", getenv("STACKYARD_AZURE_CONTENTMODERATOR_ACCOUNT", "devstoreaccount1"))
	subscriptionKey := getenv("STACKYARD_AZURE_DATA_PLANE_SUBSCRIPTION_KEY", getenv("STACKYARD_AZURE_CONTENTMODERATOR_SUBSCRIPTION_KEY", "stackyard-local-subscription-key"))
	listID := getenv("STACKYARD_AZURE_DATA_PLANE_TEXT_LIST_ID", getenv("STACKYARD_AZURE_CONTENTMODERATOR_TEXT_LIST_ID", "42"))

	fmt.Printf("Stackyard Azure AI Services Data Plane - Text Moderation example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataPlaneTextModerationClient(endpoint, account, subscriptionKey)

	detectResp, _, err := client.doJSON(
		ctx,
		http.MethodPost,
		"/azure/contentmoderator/moderate/v1.0/ProcessText/DetectLanguage",
		"text/plain",
		[]byte("hola equipo gracias"),
		http.StatusOK,
	)
	if err != nil {
		exitf("DetectLanguage failed: %v", err)
	}
	if got := asString(detectResp["DetectedLanguage"]); got != "spa" {
		exitf("DetectLanguage expected spa, got payload=%#v", detectResp)
	}

	screenPath := "/azure/contentmoderator/moderate/v1.0/ProcessText/Screen/?language=eng&autocorrect=true&PII=true&classify=true&listId=" + listID
	screenResp, _, err := client.doJSON(
		ctx,
		http.MethodPost,
		screenPath,
		"text/plain",
		[]byte("teh damn message from ops@example.com"),
		http.StatusOK,
	)
	if err != nil {
		exitf("Screen failed: %v", err)
	}
	if asString(screenResp["NormalizedText"]) == "" {
		exitf("Screen expected NormalizedText payload=%#v", screenResp)
	}
	if _, ok := screenResp["Terms"].([]any); !ok {
		exitf("Screen expected Terms array payload=%#v", screenResp)
	}
	if _, ok := screenResp["Classification"].(map[string]any); !ok {
		exitf("Screen expected Classification object payload=%#v", screenResp)
	}

	fmt.Println("Done.")
}

func asString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
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
