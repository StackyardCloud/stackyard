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

const analyzeConversationsAPIVersion = "2024-11-01"

type analyzeConversationsClient struct {
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

func newAnalyzeConversationsClient(endpoint, account, subscriptionKey string) *analyzeConversationsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-language-analyze-conversations-2024-11-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &analyzeConversationsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *analyzeConversationsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, resp.Header, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
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
			return nil, resp.StatusCode, resp.Header, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, resp.Header, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, resp.Header, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	projectName := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_CONVERSATIONS_PROJECT", "customer-support")
	deploymentName := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_CONVERSATIONS_DEPLOYMENT", "production")
	conversationText := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_CONVERSATIONS_TEXT", "Hello, I need help with my billing issue.")
	jobID := getenv("STACKYARD_AZURE_LANGUAGE_ANALYZE_CONVERSATIONS_JOB_ID", "job-123")

	fmt.Printf("Stackyard Azure AI Services - Language - Analyze Conversations (ai-services-language-analyze-conversations-2024-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAnalyzeConversationsClient(endpoint, account, subscriptionKey)

	notImplementedCount := 0
	totalCalls := 4

	analyzePath := "/azure/language/:analyze-conversations?api-version=" + analyzeConversationsAPIVersion + "&showStats=true"
	_, status, _, err := client.doJSON(ctx, http.MethodPost, analyzePath, map[string]any{
		"kind": "Conversation",
		"analysisInput": map[string]any{
			"conversationItem": map[string]any{
				"id":            "1",
				"participantId": "user-1",
				"modality":      "text",
				"language":      "en",
				"text":          conversationText,
			},
		},
		"parameters": map[string]any{
			"projectName":    projectName,
			"deploymentName": deploymentName,
			"verbose":        true,
		},
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("AnalyzeConversations failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplementedCount++
		notImplemented(analyzePath)
	} else {
		fmt.Printf("AnalyzeConversations: status=%d\n", status)
	}

	submitPath := "/azure/language/analyze-conversations/jobs?api-version=" + analyzeConversationsAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodPost, submitPath, map[string]any{
		"displayName": "stackyard-conversation-job",
		"analysisInput": map[string]any{
			"conversations": []map[string]any{
				{
					"id":       "conversation-1",
					"language": "en",
					"conversationItems": []map[string]any{
						{
							"id":            "1",
							"participantId": "user-1",
							"modality":      "text",
							"text":          conversationText,
						},
					},
				},
			},
		},
		"tasks": []map[string]any{
			{
				"kind":     "ConversationalSummarizationTask",
				"taskName": "summarize-conversation",
				"parameters": map[string]any{
					"summaryAspects": []string{"Issue", "Resolution"},
				},
			},
		},
	}, http.StatusAccepted, http.StatusNotImplemented)
	if err != nil {
		exitf("SubmitAnalysisJob failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplementedCount++
		notImplemented(submitPath)
	} else {
		fmt.Printf("SubmitAnalysisJob: status=%d\n", status)
	}

	statusPath := "/azure/language/analyze-conversations/jobs/" + jobID + "?api-version=" + analyzeConversationsAPIVersion + "&showStats=true"
	_, status, _, err = client.doJSON(ctx, http.MethodGet, statusPath, nil, http.StatusOK, http.StatusNotFound, http.StatusNotImplemented)
	if err != nil {
		exitf("GetAnalysisStatus failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplementedCount++
		notImplemented(statusPath)
	} else {
		fmt.Printf("GetAnalysisStatus: status=%d\n", status)
	}

	cancelPath := "/azure/language/analyze-conversations/jobs/" + jobID + ":cancel?api-version=" + analyzeConversationsAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodPost, cancelPath, nil, http.StatusAccepted, http.StatusNotFound, http.StatusNotImplemented)
	if err != nil {
		exitf("CancelAnalysisJob failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplementedCount++
		notImplemented(cancelPath)
	} else {
		fmt.Printf("CancelAnalysisJob: status=%d\n", status)
	}

	if notImplementedCount == totalCalls {
		fmt.Println("All analyze-conversations routes are staged in this Stackyard build.")
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
