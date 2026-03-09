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

const searchServiceDocumentsAPIVersion = "2025-09-01"

type searchServiceDocumentsClient struct {
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

func newSearchServiceDocumentsClient(endpoint, account, subscriptionKey string) *searchServiceDocumentsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"search-service-data-plane-documents-2025-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &searchServiceDocumentsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *searchServiceDocumentsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_SEARCH_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SEARCH_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	indexName := getenv("STACKYARD_AZURE_SEARCH_INDEX", "hotels")
	docID := getenv("STACKYARD_AZURE_SEARCH_DOC_ID", "1")
	suggesterName := getenv("STACKYARD_AZURE_SEARCH_SUGGESTER", "sg")
	searchText := getenv("STACKYARD_AZURE_SEARCH_TEXT", "stackyard")

	fmt.Printf("Stackyard Azure Search Service - Data Plane - Documents (search-service-data-plane-documents-2025-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newSearchServiceDocumentsClient(endpoint, account, subscriptionKey)
	indexBase := "/azure/indexes('" + indexName + "')/docs"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "AutocompleteGet",
			method:   http.MethodGet,
			path:     indexBase + "/search.autocomplete?search=" + searchText + "&suggesterName=" + suggesterName,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "AutocompletePost",
			method: http.MethodPost,
			path:   indexBase + "/search.post.autocomplete",
			payload: map[string]any{
				"search":           searchText,
				"suggesterName":    suggesterName,
				"autocompleteMode": "oneTermWithContext",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "Count",
			method:   http.MethodGet,
			path:     indexBase + "/$count",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "GetDocument",
			method:   http.MethodGet,
			path:     indexBase + "('" + docID + "')?select=hotelName",
			statuses: []int{http.StatusOK, http.StatusNotFound, http.StatusNotImplemented},
		},
		{
			name:   "Index",
			method: http.MethodPost,
			path:   indexBase + "/search.index",
			payload: map[string]any{
				"value": []map[string]any{
					{
						"@search.action": "upload",
						"id":             docID,
						"hotelName":      "Stackyard Inn",
						"description":    "Local cloud development hotel",
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusMultiStatus, http.StatusNotImplemented},
		},
		{
			name:     "SearchGet",
			method:   http.MethodGet,
			path:     indexBase + "?search=" + searchText + "&$top=5",
			statuses: []int{http.StatusOK, http.StatusPartialContent, http.StatusNotImplemented},
		},
		{
			name:   "SearchPost",
			method: http.MethodPost,
			path:   indexBase + "/search.post.search",
			payload: map[string]any{
				"search": searchText,
				"top":    5,
			},
			statuses: []int{http.StatusOK, http.StatusPartialContent, http.StatusNotImplemented},
		},
		{
			name:     "SuggestGet",
			method:   http.MethodGet,
			path:     indexBase + "/search.suggest?search=" + searchText + "&suggesterName=" + suggesterName,
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "SuggestPost",
			method: http.MethodPost,
			path:   indexBase + "/search.post.suggest",
			payload: map[string]any{
				"search":        searchText,
				"suggesterName": suggesterName,
				"top":           5,
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		fullPath := withAPIVersion(call.path, searchServiceDocumentsAPIVersion)
		_, status, err := client.doJSON(ctx, call.method, fullPath, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		if status == http.StatusNotImplemented {
			notImplementedCount++
			notImplemented(fullPath)
			continue
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	if notImplementedCount == len(calls) {
		fmt.Println("All search-service data-plane documents routes are staged in this Stackyard build.")
		return
	}
	fmt.Println("Done.")
}

func withAPIVersion(path, apiVersion string) string {
	if strings.Contains(path, "?") {
		return path + "&api-version=" + apiVersion
	}
	return path + "?api-version=" + apiVersion
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
