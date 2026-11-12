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

const translatorAPIVersion = "3.0"

type translatorClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type sharedKeyPolicy struct {
	account         string
	subscriptionKey string
	region          string
}

func (p *sharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	if strings.TrimSpace(p.region) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Region", p.region)
	}
	return req.Next()
}

func newTranslatorClient(endpoint, account, subscriptionKey, region string) *translatorClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"translator-v3.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyPolicy{account: account, subscriptionKey: subscriptionKey, region: region}},
		},
		&policy.ClientOptions{},
	)
	return &translatorClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *translatorClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json; charset=UTF-8")
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
	region := getenv("STACKYARD_AZURE_TRANSLATOR_REGION", "global")

	fmt.Printf("Stackyard Azure Translator (translator-v3.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newTranslatorClient(endpoint, account, subscriptionKey, region)
	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "Translate",
			method: http.MethodPost,
			path:   "/azure/translator/translate?api-version=" + translatorAPIVersion + "&from=en&to=es",
			payload: []map[string]any{
				{"Text": "Hello world"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "Detect",
			method: http.MethodPost,
			path:   "/azure/translator/detect?api-version=" + translatorAPIVersion,
			payload: []map[string]any{
				{"Text": "Hola mundo"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "BreakSentence",
			method: http.MethodPost,
			path:   "/azure/translator/breaksentence?api-version=" + translatorAPIVersion + "&language=en",
			payload: []map[string]any{
				{"Text": "Hello world. This is Stackyard."},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "DictionaryLookup",
			method: http.MethodPost,
			path:   "/azure/translator/dictionary/lookup?api-version=" + translatorAPIVersion + "&from=en&to=es",
			payload: []map[string]any{
				{"Text": "work"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "DictionaryExamples",
			method: http.MethodPost,
			path:   "/azure/translator/dictionary/examples?api-version=" + translatorAPIVersion + "&from=en&to=es",
			payload: []map[string]any{
				{"Text": "work", "Translation": "trabajo"},
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "Languages",
			method:   http.MethodGet,
			path:     "/azure/translator/languages?api-version=" + translatorAPIVersion + "&scope=translation,transliteration,dictionary",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "Transliterate",
			method: http.MethodPost,
			path:   "/azure/translator/transliterate?api-version=" + translatorAPIVersion + "&language=ja&fromScript=Jpan&toScript=Latn",
			payload: []map[string]any{
				{"Text": "こんにちは"},
			},
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
		fmt.Println("All translator routes are staged in this Stackyard build.")
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
