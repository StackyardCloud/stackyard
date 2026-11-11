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

const dataMigrationAPIVersion = "2022-03-30-preview"
const dataMigrationOperationsAPIVersion = "2022-01-30-preview"

type dataMigrationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataMigrationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataMigrationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataMigrationClient(endpoint, account, subscriptionKey string) *dataMigrationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datamigration-2022-03-30-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataMigrationSharedKeyPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &dataMigrationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataMigrationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATAMIGRATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATAMIGRATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datamigration")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	serviceName := getenv("STACKYARD_AZURE_DATAMIGRATION_SERVICE_NAME", "dms-example")
	projectName := getenv("STACKYARD_AZURE_DATAMIGRATION_PROJECT_NAME", "project-example")
	serviceTaskName := getenv("STACKYARD_AZURE_DATAMIGRATION_SERVICE_TASK_NAME", "service-task-example")
	taskName := getenv("STACKYARD_AZURE_DATAMIGRATION_TASK_NAME", "task-example")
	fileName := getenv("STACKYARD_AZURE_DATAMIGRATION_FILE_NAME", "payload.json")

	client := newDataMigrationClient(endpoint, account, subscriptionKey)

	serviceBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataMigration/services/%s",
		subscriptionID,
		resourceGroup,
		serviceName,
	)
	projectBase := serviceBase + "/projects/" + projectName
	serviceTaskBase := serviceBase + "/serviceTasks/" + serviceTaskName
	taskBase := projectBase + "/tasks/" + taskName
	fileBase := projectBase + "/files/" + fileName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DataMigration/operations?api-version=" + dataMigrationOperationsAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckServiceNameAvailability",
			method: http.MethodPost,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DataMigration/locations/%s/checkNameAvailability?api-version=%s",
				subscriptionID,
				location,
				dataMigrationAPIVersion,
			),
			payload: map[string]any{
				"name": serviceName,
				"type": "Microsoft.DataMigration/services",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckChildResourceNameAvailability",
			method: http.MethodPost,
			path: serviceBase +
				"/checkNameAvailability?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"name": projectName,
				"type": "projects",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateService",
			method: http.MethodPut,
			path:   serviceBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"location": location,
				"sku": map[string]any{
					"name": "Standard_1vCore",
				},
				"properties": map[string]any{
					"virtualSubnetId": fmt.Sprintf(
						"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/default",
						subscriptionID,
						resourceGroup,
					),
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetService",
			method:   http.MethodGet,
			path:     serviceBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListServices",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataMigration/services?api-version=%s", subscriptionID, dataMigrationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListServicesByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataMigration/services?api-version=%s", subscriptionID, resourceGroup, dataMigrationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSkus",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataMigration/skus?api-version=%s", subscriptionID, dataMigrationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StartService",
			method:   http.MethodPost,
			path:     serviceBase + "/start?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CheckServiceStatus",
			method:   http.MethodPost,
			path:     serviceBase + "/checkStatus?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "StopService",
			method:   http.MethodPost,
			path:     serviceBase + "/stop?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "UpdateService",
			method: http.MethodPatch,
			path:   serviceBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local",
					"owner":       "stackyard",
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateOrUpdateProject",
			method: http.MethodPut,
			path:   projectBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"sourcePlatform": "SQL",
					"targetPlatform": "SQLDB",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetProject",
			method:   http.MethodGet,
			path:     projectBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProjects",
			method:   http.MethodGet,
			path:     serviceBase + "/projects?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateProject",
			method: http.MethodPatch,
			path:   projectBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"stage": "dev",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateServiceTask",
			method: http.MethodPut,
			path:   serviceTaskBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"taskType": "ConnectToTarget.SqlServer",
					"input": map[string]any{
						"targetConnectionInfo": map[string]any{"type": "SqlConnectionInfo"},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "ListServiceTasks",
			method:   http.MethodGet,
			path:     serviceBase + "/serviceTasks?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetServiceTask",
			method:   http.MethodGet,
			path:     serviceTaskBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateServiceTask",
			method: http.MethodPatch,
			path:   serviceTaskBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"input": map[string]any{"retryCount": 1},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CommandServiceTask",
			method: http.MethodPost,
			path:   serviceTaskBase + "/command?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"commandType": "Cancel",
				"properties":  map[string]any{},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CancelServiceTask",
			method:   http.MethodPost,
			path:     serviceTaskBase + "/cancel?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CreateOrUpdateTask",
			method: http.MethodPut,
			path:   taskBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"taskType": "Migrate.SqlServer.SqlDb",
					"input": map[string]any{
						"sourceConnectionInfo": map[string]any{"type": "SqlConnectionInfo"},
						"targetConnectionInfo": map[string]any{"type": "SqlConnectionInfo"},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetTask",
			method:   http.MethodGet,
			path:     taskBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTasks",
			method:   http.MethodGet,
			path:     projectBase + "/tasks?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CommandTask",
			method: http.MethodPost,
			path:   taskBase + "/command?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"commandType": "Cancel",
				"properties":  map[string]any{},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CancelTask",
			method:   http.MethodPost,
			path:     taskBase + "/cancel?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "UpdateTask",
			method: http.MethodPatch,
			path:   taskBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"input": map[string]any{"continueOnError": true},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateOrUpdateFile",
			method: http.MethodPut,
			path:   fileBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"extension": "json",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetFile",
			method:   http.MethodGet,
			path:     fileBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListFiles",
			method:   http.MethodGet,
			path:     projectBase + "/files?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ReadFile",
			method: http.MethodPost,
			path:   fileBase + "/read?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"offset": 0,
				"length": 1024,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ReadWriteFile",
			method: http.MethodPost,
			path:   fileBase + "/readwrite?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"content": "ewoJIm1pZ3JhdGlvbiI6ICJkZW1vIgp9",
				"offset":  0,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateFile",
			method: http.MethodPatch,
			path:   fileBase + "?api-version=" + dataMigrationAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"readOnly": false,
				},
			},
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
			name:     "DeleteFile",
			method:   http.MethodDelete,
			path:     fileBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteTask",
			method:   http.MethodDelete,
			path:     taskBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteServiceTask",
			method:   http.MethodDelete,
			path:     serviceTaskBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteProject",
			method:   http.MethodDelete,
			path:     projectBase + "?api-version=" + dataMigrationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteService",
			method:   http.MethodDelete,
			path:     serviceBase + "?api-version=" + dataMigrationAPIVersion,
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
