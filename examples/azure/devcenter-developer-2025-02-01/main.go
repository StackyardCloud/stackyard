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

const devCenterDeveloperAPIVersion = "2025-02-01"

type devCenterDeveloperClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type devCenterDeveloperSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *devCenterDeveloperSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDevCenterDeveloperClient(endpoint, account, subscriptionKey string) *devCenterDeveloperClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"devcenter-developer-2025-02-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&devCenterDeveloperSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &devCenterDeveloperClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *devCenterDeveloperClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	projectName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_PROJECT_NAME", "myProject")
	userID := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_USER_ID", "me")
	environmentName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_ENVIRONMENT_NAME", "myEnvironment")
	environmentType := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_ENVIRONMENT_TYPE", "DevTest")
	catalogName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_CATALOG_NAME", "main")
	environmentDefinitionName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_ENVIRONMENT_DEFINITION_NAME", "helloworld")
	devBoxName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_DEVBOX_NAME", "myDevBox")
	poolName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_POOL_NAME", "LargeDevWorkStationPool")
	customizationGroupName := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_CUSTOMIZATION_GROUP_NAME", "Provisioning")
	customizationTaskID := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_CUSTOMIZATION_TASK_ID", "91835dc0-ef5a-4f58-9e3a-099aea8481f4")
	imageBuildLogID := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_IMAGE_BUILD_LOG_ID", "91835dc0-ef5a-4f58-9e3a-099aea8481f4")
	environmentOperationID := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_ENV_OPERATION_ID", "f5dbdfab-fa0e-4831-8d13-25359aa5e680")
	devBoxOperationID := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_DEVBOX_OPERATION_ID", "f5dbdfab-fa0e-4831-8d13-25359aa5e680")
	operationStatusID := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_OPERATION_STATUS_ID", "fa067167-e49d-41bd-8dd8-de719b9de3b3")
	delayUntil := getenv("STACKYARD_AZURE_DEVCENTER_DEVELOPER_DELAY_UNTIL", "2026-01-01T18:00:00Z")

	client := newDevCenterDeveloperClient(endpoint, account, subscriptionKey)

	base := "/azure/devcenter-developer/2025-02-01"
	projectBase := base + "/projects/" + projectName
	userProjectBase := projectBase + "/users/" + userID
	environmentBase := userProjectBase + "/environments/" + environmentName
	devBoxBase := userProjectBase + "/devboxes/" + devBoxName
	operationStatusBase := projectBase + "/operationstatuses/" + operationStatusID

	createEnvironmentPayload := map[string]any{
		"environmentType":           environmentType,
		"catalogName":               catalogName,
		"environmentDefinitionName": environmentDefinitionName,
		"parameters": map[string]any{
			"functionAppRuntime": "node",
			"storageAccountType": "Standard_LRS",
		},
		"expirationDate": "2026-12-31T23:59:00Z",
	}

	patchEnvironmentPayload := map[string]any{
		"expirationDate": "2027-01-31T23:59:00Z",
	}

	createDevBoxPayload := map[string]any{
		"poolName": poolName,
	}

	validateCustomizationPayload := map[string]any{
		"tasks": []any{
			map[string]any{
				"name": catalogName + "/choco",
				"parameters": map[string]any{
					"packageName":    "vscode",
					"packageVersion": "1.0.0",
				},
			},
			map[string]any{
				"name": catalogName + "/write-to-disk",
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
			name:     "ListProjects",
			method:   http.MethodGet,
			path:     base + "/projects?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProject",
			method:   http.MethodGet,
			path:     projectBase + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProjectAbilities",
			method:   http.MethodGet,
			path:     userProjectBase + "/abilities?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCatalogsByProject",
			method:   http.MethodGet,
			path:     projectBase + "/catalogs?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironmentTypes",
			method:   http.MethodGet,
			path:     projectBase + "/environmentTypes?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrReplaceEnvironment",
			method:   http.MethodPut,
			path:     environmentBase + "?api-version=" + devCenterDeveloperAPIVersion,
			payload:  createEnvironmentPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetEnvironmentByUser",
			method:   http.MethodGet,
			path:     environmentBase + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PatchEnvironment",
			method:   http.MethodPatch,
			path:     environmentBase + "?api-version=" + devCenterDeveloperAPIVersion,
			payload:  patchEnvironmentPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironments",
			method:   http.MethodGet,
			path:     projectBase + "/environments?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironmentsByUser",
			method:   http.MethodGet,
			path:     userProjectBase + "/environments?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironmentOperations",
			method:   http.MethodGet,
			path:     environmentBase + "/operations?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEnvironmentLogsByOperation",
			method:   http.MethodGet,
			path:     environmentBase + "/operations/" + environmentOperationID + "/logs?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAllDevBoxes",
			method:   http.MethodGet,
			path:     base + "/devboxes?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAllDevBoxesByUser",
			method:   http.MethodGet,
			path:     base + "/users/" + userID + "/devboxes?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevBoxesByUser",
			method:   http.MethodGet,
			path:     userProjectBase + "/devboxes?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPools",
			method:   http.MethodGet,
			path:     projectBase + "/pools?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateDevBox",
			method:   http.MethodPut,
			path:     devBoxBase + "?api-version=" + devCenterDeveloperAPIVersion,
			payload:  createDevBoxPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetDevBoxByUser",
			method:   http.MethodGet,
			path:     devBoxBase + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevBoxActions",
			method:   http.MethodGet,
			path:     devBoxBase + "/actions?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DelayAllDevBoxActions",
			method:   http.MethodPost,
			path:     devBoxBase + "/actions:delay?api-version=" + devCenterDeveloperAPIVersion + "&until=" + url.QueryEscape(delayUntil),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevBoxOperations",
			method:   http.MethodGet,
			path:     devBoxBase + "/operations?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDevBoxOperation",
			method:   http.MethodGet,
			path:     devBoxBase + "/operations/" + devBoxOperationID + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetRemoteConnection",
			method:   http.MethodGet,
			path:     devBoxBase + "/remoteConnection?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCustomizationTaskDefinitionsByProject",
			method:   http.MethodGet,
			path:     projectBase + "/customizationTasks?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ValidateCustomizationTasksAction",
			method:   http.MethodPost,
			path:     projectBase + "/customizationTasks:validateGroup?api-version=" + devCenterDeveloperAPIVersion,
			payload:  validateCustomizationPayload,
			statuses: []int{http.StatusAccepted},
		},
		{
			name:     "GetCustomizationTaskLog",
			method:   http.MethodGet,
			path:     devBoxBase + "/customizationGroups/" + customizationGroupName + "/logs/" + customizationTaskID + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetImagingTaskLog",
			method:   http.MethodGet,
			path:     projectBase + "/imageBuildLogs/" + imageBuildLogID + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetOperationStatus",
			method:   http.MethodGet,
			path:     operationStatusBase + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeleteDevBox",
			method:   http.MethodDelete,
			path:     devBoxBase + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteEnvironment",
			method:   http.MethodDelete,
			path:     environmentBase + "?api-version=" + devCenterDeveloperAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		payload, status, err := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(formatted))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
