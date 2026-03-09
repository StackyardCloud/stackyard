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

type dataPlaneImageModerationClient struct {
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

func newDataPlaneImageModerationClient(endpoint, account, subscriptionKey string) *dataPlaneImageModerationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-data-plane-image-moderation-v1.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &dataPlaneImageModerationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataPlaneImageModerationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	if len(expectedStatuses) > 0 {
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
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, resp.StatusCode, nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DATA_PLANE_ACCOUNT", getenv("STACKYARD_AZURE_CONTENTMODERATOR_ACCOUNT", "devstoreaccount1"))
	subscriptionKey := getenv("STACKYARD_AZURE_DATA_PLANE_SUBSCRIPTION_KEY", getenv("STACKYARD_AZURE_CONTENTMODERATOR_SUBSCRIPTION_KEY", "stackyard-local-subscription-key"))
	listID := getenv("STACKYARD_AZURE_DATA_PLANE_LIST_ID", getenv("STACKYARD_AZURE_CONTENTMODERATOR_LIST_ID", "12345"))
	language := getenv("STACKYARD_AZURE_DATA_PLANE_LANGUAGE", getenv("STACKYARD_AZURE_CONTENTMODERATOR_LANGUAGE", "eng"))
	imageURL := getenv("STACKYARD_AZURE_DATA_PLANE_IMAGE_URL", getenv("STACKYARD_AZURE_CONTENTMODERATOR_IMAGE_URL", "https://example.com/safe-image.jpg"))

	fmt.Printf("Stackyard Azure AI Services Data Plane - Image Moderation example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataPlaneImageModerationClient(endpoint, account, subscriptionKey)
	urlPayload := map[string]any{
		"DataRepresentation": "URL",
		"Value":              imageURL,
	}

	evaluateResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=url&CacheImage=true", urlPayload, http.StatusOK)
	if err != nil {
		exitf("Evaluate failed: %v", err)
	}
	if status := statusCode(evaluateResp); status != 3000 {
		exitf("Evaluate expected status code 3000, got %d payload=%#v", status, evaluateResp)
	}

	findFacesResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/FindFaces?overload=url&CacheImage=true", urlPayload, http.StatusOK)
	if err != nil {
		exitf("FindFaces failed: %v", err)
	}
	if _, ok := findFacesResp["Count"].(float64); !ok {
		exitf("FindFaces expected Count field, got payload=%#v", findFacesResp)
	}

	matchResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/Match?overload=url&listId="+listID+"&CacheImage=true", urlPayload, http.StatusOK)
	if err != nil {
		exitf("Match failed: %v", err)
	}
	if _, ok := matchResp["IsMatch"].(bool); !ok {
		exitf("Match expected IsMatch field, got payload=%#v", matchResp)
	}

	ocrResp, _, err := client.doJSON(ctx, http.MethodPost, "/azure/contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=url&language="+language+"&enhanced=true&CacheImage=true", urlPayload, http.StatusOK)
	if err != nil {
		exitf("OCR failed: %v", err)
	}
	text, _ := ocrResp["Text"].(string)
	if strings.TrimSpace(text) == "" {
		exitf("OCR expected non-empty Text, got payload=%#v", ocrResp)
	}

	fmt.Println("Done.")
}

func statusCode(payload map[string]any) int {
	status, ok := payload["Status"].(map[string]any)
	if !ok {
		return 0
	}
	code, ok := status["Code"].(float64)
	if !ok {
		return 0
	}
	return int(code)
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
