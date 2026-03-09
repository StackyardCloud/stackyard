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

const analyzeTextAuthoringAPIVersion = "2023-04-01"

type analyzeTextAuthoringClient struct {
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

func newAnalyzeTextAuthoringClient(endpoint, account, subscriptionKey string) *analyzeTextAuthoringClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-language-analyze-text-authoring-2023-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &analyzeTextAuthoringClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *analyzeTextAuthoringClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	projectName := getenv("STACKYARD_AZURE_LANGUAGE_AUTHORING_PROJECT", "proj-a")
	modelLabel := getenv("STACKYARD_AZURE_LANGUAGE_AUTHORING_MODEL_LABEL", "model-v1")
	deploymentName := getenv("STACKYARD_AZURE_LANGUAGE_AUTHORING_DEPLOYMENT", "prod")
	jobID := getenv("STACKYARD_AZURE_LANGUAGE_AUTHORING_JOB_ID", "job-123")
	resourceName := getenv("STACKYARD_AZURE_LANGUAGE_AUTHORING_RESOURCE", "authoring-resource-a")

	fmt.Printf("Stackyard Azure AI Services - Language - Analyze Text Authoring (ai-services-language-analyze-text-authoring-2023-04-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAnalyzeTextAuthoringClient(endpoint, account, subscriptionKey)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "GetSupportedLanguages",
			method:   http.MethodGet,
			path:     "/azure/language/authoring/analyze-text/projects/" + projectName + "/supportedLanguages",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "CreateProject",
			method: http.MethodPut,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName,
			payload: map[string]any{
				"projectKind":  "customSingleLabelClassification",
				"language":     "en",
				"multilingual": false,
			},
			statuses: []int{http.StatusCreated, http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "UpdateProjectMetadata",
			method: http.MethodPatch,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + "/metadata",
			payload: map[string]any{
				"description": "stackyard local project metadata",
			},
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "ExportProject",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + ":export",
			payload: map[string]any{
				"assetKind":       "EntireProject",
				"stringIndexType": "Utf16CodeUnit",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "ImportProject",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + ":import",
			payload: map[string]any{
				"projectFileVersion": "2023-04-01",
				"stringIndexType":    "Utf16CodeUnit",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "Train",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + "/train/jobs",
			payload: map[string]any{
				"modelLabel":   modelLabel,
				"trainingMode": "standard",
			},
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:     "CancelTrainingJob",
			method:   http.MethodPost,
			path:     "/azure/language/authoring/analyze-text/projects/" + projectName + "/train/jobs/" + jobID + ":cancel",
			statuses: []int{http.StatusAccepted, http.StatusNotImplemented},
		},
		{
			name:   "DeployModel",
			method: http.MethodPut,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + "/deployments/" + deploymentName,
			payload: map[string]any{
				"trainedModelLabel": modelLabel,
			},
			statuses: []int{http.StatusAccepted, http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "SwapDeployments",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + "/deployments:swap",
			payload: map[string]any{
				"firstDeploymentName":  "staging",
				"secondDeploymentName": deploymentName,
			},
			statuses: []int{http.StatusAccepted, http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTrainedModels",
			method:   http.MethodGet,
			path:     "/azure/language/authoring/analyze-text/projects/" + projectName + "/models",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "LoadSnapshot",
			method: http.MethodPost,
			path:   "/azure/language/authoring/analyze-text/projects/" + projectName + "/models:loadSnapshot",
			payload: map[string]any{
				"trainedModelLabel": modelLabel + "-snapshot",
				"modelId":           "source-model-id",
			},
			statuses: []int{http.StatusAccepted, http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:     "ListTextAuthoringResources",
			method:   http.MethodGet,
			path:     "/azure/language/authoring/analyze-text/resources",
			statuses: []int{http.StatusOK, http.StatusNotImplemented},
		},
		{
			name:   "NewTextAuthoringResource",
			method: http.MethodPut,
			path:   "/azure/language/authoring/analyze-text/resources/" + resourceName,
			payload: map[string]any{
				"resourceId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.CognitiveServices/accounts/authoring",
			},
			statuses: []int{http.StatusCreated, http.StatusOK, http.StatusAccepted, http.StatusNotImplemented},
		},
	}

	notImplementedCount := 0
	for _, call := range calls {
		fullPath := withAPIVersion(call.path, analyzeTextAuthoringAPIVersion)
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
		fmt.Println("All analyze-text-authoring routes are staged in this Stackyard build.")
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
