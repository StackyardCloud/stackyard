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

const apiCenterDataPlaneAPIVersion = "2024-02-01-preview"

type apiCenterClient struct {
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

func newAPICenterClient(endpoint, account, subscriptionKey string) *apiCenterClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"api-center-data-plane-2024-02-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &apiCenterClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *apiCenterClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
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
	account := getenv("STACKYARD_AZURE_APICENTER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APICENTER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	workspaceName := getenv("STACKYARD_AZURE_APICENTER_WORKSPACE", "default")
	apiName := getenv("STACKYARD_AZURE_APICENTER_API", "echo-api")
	versionName := getenv("STACKYARD_AZURE_APICENTER_VERSION", "2023-01-01")
	definitionName := getenv("STACKYARD_AZURE_APICENTER_DEFINITION", "default")
	deploymentName := getenv("STACKYARD_AZURE_APICENTER_DEPLOYMENT", "production")
	environmentName := getenv("STACKYARD_AZURE_APICENTER_ENVIRONMENT", "production")
	operationID := getenv("STACKYARD_AZURE_APICENTER_OPERATION_ID", "00000000-0000-0000-0000-000000000001")

	fmt.Printf("Stackyard Azure API Center Data Plane (data-plane-2024-02-01-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAPICenterClient(endpoint, account, subscriptionKey)
	base := "/azure/apicenter"
	workspace := base + "/workspaces/" + workspaceName
	apiScope := workspace + "/apis/" + apiName
	versionScope := apiScope + "/versions/" + versionName
	definitionScope := versionScope + "/definitions/" + definitionName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListAllApis",
			method:   http.MethodGet,
			path:     base + "/apis?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspaceApis",
			method:   http.MethodGet,
			path:     workspace + "/apis?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetApi",
			method:   http.MethodGet,
			path:     apiScope + "?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListVersions",
			method:   http.MethodGet,
			path:     apiScope + "/versions?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetVersion",
			method:   http.MethodGet,
			path:     versionScope + "?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDefinitions",
			method:   http.MethodGet,
			path:     versionScope + "/definitions?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDefinition",
			method:   http.MethodGet,
			path:     definitionScope + "?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ExportSpecification",
			method: http.MethodPost,
			path:   definitionScope + ":exportSpecification?api-version=" + apiCenterDataPlaneAPIVersion,
			payload: map[string]any{
				"format": "openapi",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetExportSpecificationOperationStatus",
			method:   http.MethodGet,
			path:     definitionScope + "/operations/" + operationID + "?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDeployments",
			method:   http.MethodGet,
			path:     apiScope + "/deployments?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDeployment",
			method:   http.MethodGet,
			path:     apiScope + "/deployments/" + deploymentName + "?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAllEnvironments",
			method:   http.MethodGet,
			path:     base + "/environments?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspaceEnvironments",
			method:   http.MethodGet,
			path:     workspace + "/environments?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEnvironment",
			method:   http.MethodGet,
			path:     workspace + "/environments/" + environmentName + "?api-version=" + apiCenterDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}

	fmt.Println("Azure API Center Data Plane staged routes completed successfully.")
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
