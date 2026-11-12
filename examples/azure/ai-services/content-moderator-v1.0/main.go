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

type contentModeratorClient struct {
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

func newContentModeratorClient(endpoint, account, subscriptionKey string) *contentModeratorClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"content-moderator-v1.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &contentModeratorClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *contentModeratorClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return c.execute(req, method, path, expectedStatuses...)
}

func (c *contentModeratorClient) doRaw(ctx context.Context, method, path, contentType string, body []byte, expectedStatuses ...int) (map[string]any, int, error) {
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
	return c.execute(req, method, path, expectedStatuses...)
}

func (c *contentModeratorClient) execute(req *policy.Request, method, path string, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	teamName := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_TEAM", "local-team")
	reviewID := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_REVIEW_ID", "review-1")
	jobID := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_JOB_ID", "job-1")
	termListID := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_TERM_LIST_ID", "123")
	imageURL := getenv("STACKYARD_AZURE_CONTENT_MODERATOR_IMAGE_URL", "https://example.com/safe-image.jpg")

	fmt.Printf("Stackyard Azure Content Moderator (content-moderator-v1.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newContentModeratorClient(endpoint, account, subscriptionKey)

	okStatuses := []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent, http.StatusNotImplemented}

	calls := []struct {
		name        string
		method      string
		path        string
		payload     any
		rawBody     []byte
		rawBodyType string
		useRawBody  bool
	}{
		{
			name:   "EvaluateImageURL",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=url&CacheImage=true",
			payload: map[string]any{
				"DataRepresentation": "URL",
				"Value":              imageURL,
			},
		},
		{
			name:        "DetectLanguage",
			method:      http.MethodPost,
			path:        "/azure/contentmoderator/moderate/v1.0/ProcessText/DetectLanguage",
			rawBody:     []byte("hola equipo gracias"),
			rawBodyType: "text/plain",
			useRawBody:  true,
		},
		{
			name:   "CreateImageList",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/lists/v1.0/imagelists",
			payload: map[string]any{
				"Name":        "sdk-list",
				"Description": "stackyard list",
			},
		},
		{
			name:   "CreateTermList",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/lists/v1.0/termlists",
			payload: map[string]any{
				"Name":        "sdk-term-list",
				"Description": "stackyard term list",
			},
		},
		{
			name:   "AddTerm",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/lists/v1.0/termlists/" + termListID + "/terms?language=eng",
			payload: map[string]any{
				"Term": "forbidden",
			},
		},
		{
			name:   "CreateReview",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/review/v1.0/teams/" + teamName + "/reviews",
			payload: map[string]any{
				"Type":    "Image",
				"Content": imageURL,
			},
		},
		{
			name:   "GetReview",
			method: http.MethodGet,
			path:   "/azure/contentmoderator/review/v1.0/teams/" + teamName + "/reviews/" + reviewID,
		},
		{
			name:   "CreateJob",
			method: http.MethodPost,
			path:   "/azure/contentmoderator/review/v1.0/teams/" + teamName + "/jobs",
			payload: map[string]any{
				"Type":    "Image",
				"Content": imageURL,
			},
		},
		{
			name:   "GetJob",
			method: http.MethodGet,
			path:   "/azure/contentmoderator/review/v1.0/teams/" + teamName + "/jobs/" + jobID,
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		var (
			status int
			err    error
		)
		if call.useRawBody {
			_, status, err = client.doRaw(ctx, call.method, call.path, call.rawBodyType, call.rawBody, okStatuses...)
		} else {
			_, status, err = client.doJSON(ctx, call.method, call.path, call.payload, okStatuses...)
		}
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
		fmt.Println("All content moderator routes are staged in this Stackyard build.")
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
