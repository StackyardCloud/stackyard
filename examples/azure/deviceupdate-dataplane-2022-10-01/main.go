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

const deviceUpdateDataPlaneAPIVersion = "2022-10-01"
const deviceUpdateDataPlanePrefix = "/azure/deviceupdate-dataplane/2022-10-01"

type deviceUpdateDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type deviceUpdateDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type deviceUpdateCall struct {
	name       string
	method     string
	template   string
	extraQuery string
	payload    any
	statuses   []int
}

func (p *deviceUpdateDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDeviceUpdateDataPlaneClient(endpoint, account, subscriptionKey string) *deviceUpdateDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"deviceupdate-dataplane-2022-10-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&deviceUpdateDataPlaneSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &deviceUpdateDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *deviceUpdateDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func buildPath(template string, vars map[string]string, extraQuery string) string {
	path := template
	for key, value := range vars {
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
	}
	query := "api-version=" + deviceUpdateDataPlaneAPIVersion
	if strings.TrimSpace(extraQuery) != "" {
		query += "&" + extraQuery
	}
	return deviceUpdateDataPlanePrefix + path + "?" + query
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DEVICEUPDATE_DATAPLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEVICEUPDATE_DATAPLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	instanceID := getenv("STACKYARD_AZURE_DEVICEUPDATE_INSTANCE_ID", "blue")
	provider := getenv("STACKYARD_AZURE_DEVICEUPDATE_PROVIDER", "microsoft")
	updateName := getenv("STACKYARD_AZURE_DEVICEUPDATE_NAME", "adu")
	version := getenv("STACKYARD_AZURE_DEVICEUPDATE_VERSION", "1.2.3.4")
	fileID := getenv("STACKYARD_AZURE_DEVICEUPDATE_FILE_ID", "file1.bin")
	groupID := getenv("STACKYARD_AZURE_DEVICEUPDATE_GROUP_ID", "group-a")
	deploymentID := getenv("STACKYARD_AZURE_DEVICEUPDATE_DEPLOYMENT_ID", "deployment-a")
	deviceClassID := getenv("STACKYARD_AZURE_DEVICEUPDATE_DEVICE_CLASS_ID", "device-class-a")
	deviceID := getenv("STACKYARD_AZURE_DEVICEUPDATE_DEVICE_ID", "device-a")
	moduleID := getenv("STACKYARD_AZURE_DEVICEUPDATE_MODULE_ID", "module-a")
	operationID := getenv("STACKYARD_AZURE_DEVICEUPDATE_OPERATION_ID", "op-a")
	healthFilter := getenv("STACKYARD_AZURE_DEVICEUPDATE_HEALTH_FILTER", "state eq 'unhealthy'")

	client := newDeviceUpdateDataPlaneClient(endpoint, account, subscriptionKey)

	routeVars := map[string]string{
		"instanceId":    instanceID,
		"provider":      provider,
		"name":          updateName,
		"version":       version,
		"fileId":        fileID,
		"groupId":       groupID,
		"deploymentId":  deploymentID,
		"deviceClassId": deviceClassID,
		"deviceId":      deviceID,
		"moduleId":      moduleID,
		"operationId":   operationID,
	}
	healthFilterQuery := "filter=" + url.QueryEscape(healthFilter)

	createOrUpdateDeploymentPayload := map[string]any{
		"deploymentId":  deploymentID,
		"groupId":       groupID,
		"startDateTime": "2020-04-22T05:12:12-07:00",
		"update": map[string]any{
			"updateId": map[string]any{
				"provider": provider,
				"name":     updateName,
				"version":  version,
			},
		},
	}

	startLogCollectionPayload := map[string]any{
		"deviceList": []any{
			map[string]any{"deviceId": deviceID},
			map[string]any{"deviceId": deviceID, "moduleId": moduleID},
		},
		"description": "Stackyard diagnostics collection",
	}

	updateDeviceClassPayload := map[string]any{
		"friendlyName": "GA Webcams",
	}

	importUpdatePayload := []any{
		map[string]any{
			"importManifest": map[string]any{
				"url":         "http://test.blob.core.windows.net/test/uploadimportMan.json",
				"sizeInBytes": 816,
				"hashes": map[string]any{
					"sha256": "O19LyyncPe1AGstOdkcmozLV8pSbBdqrE18HdYVohRc=",
				},
			},
			"files": []any{
				map[string]any{"filename": "file1.bin", "url": "http://test.blob.core.windows.net/test/upload1v5uww1q"},
				map[string]any{"filename": "file2.bin", "url": "http://test.blob.core.windows.net/test/uploadkrmn5yw0"},
			},
		},
	}

	calls := []deviceUpdateCall{
		{
			name:     "DeviceManagementCreateOrUpdateDeployment",
			method:   http.MethodPut,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments/{deploymentId}",
			payload:  createOrUpdateDeploymentPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementDeleteDeploymentForDeviceClassSubgroup",
			method:   http.MethodDelete,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments/{deploymentId}",
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "DeviceManagementDeleteDeployment",
			method:   http.MethodDelete,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments/{deploymentId}",
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "DeviceManagementDeleteDeviceClassSubgroup",
			method:   http.MethodDelete,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}",
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "DeviceManagementDeleteDeviceClass",
			method:   http.MethodDelete,
			template: "/deviceUpdate/{instanceId}/management/deviceClasses/{deviceClassId}",
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "DeviceManagementDeleteGroup",
			method:   http.MethodDelete,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}",
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "DeviceManagementGetBestUpdatesForDeviceClassSubgroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/bestUpdates",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeploymentForDeviceClassSubgroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments/{deploymentId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeploymentStatus",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments/{deploymentId}/status",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeployment",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments/{deploymentId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeviceClassSubgroupDeploymentStatus",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments/{deploymentId}/status",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeviceClassSubgroupUpdateCompliance",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/updateCompliance",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeviceClassSubgroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeviceClass",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/deviceClasses/{deviceClassId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDeviceModule",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/devices/{deviceId}/modules/{moduleId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetDevice",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/devices/{deviceId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetGroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetLogCollectionDetailedStatus",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/deviceDiagnostics/logCollections/{operationId}/detailedStatus",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetLogCollection",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/deviceDiagnostics/logCollections/{operationId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetOperationStatus",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/operations/{operationId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetUpdateComplianceForGroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/updateCompliance",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementGetUpdateCompliance",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/updateCompliance",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementImportDevices",
			method:   http.MethodPost,
			template: "/deviceUpdate/{instanceId}/management/devices:import",
			statuses: []int{http.StatusAccepted},
		},
		{
			name:     "DeviceManagementListBestUpdatesForGroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/bestUpdates",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListDeploymentsForDeviceClassSubgroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListDeploymentsForGroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deployments",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListDeviceClassSubgroupsForGroup",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListDeviceClasses",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/deviceClasses",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListDeviceStatesForDeviceClassSubgroupDeployment",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments/{deploymentId}/devicestates",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListDevices",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/devices",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListGroups",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/groups",
			statuses: []int{http.StatusOK},
		},
		{
			name:       "DeviceManagementListHealthOfDevices",
			method:     http.MethodGet,
			template:   "/deviceUpdate/{instanceId}/management/deviceDiagnostics/deviceHealth",
			extraQuery: healthFilterQuery,
			statuses:   []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListInstallableUpdatesForDeviceClass",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/deviceClasses/{deviceClassId}/installableUpdates",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListLogCollections",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/deviceDiagnostics/logCollections",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementListOperationStatuses",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/management/operations",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementRetryDeployment",
			method:   http.MethodPost,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments/{deploymentId}:retry",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementStartLogCollection",
			method:   http.MethodPut,
			template: "/deviceUpdate/{instanceId}/management/deviceDiagnostics/logCollections/{operationId}",
			payload:  startLogCollectionPayload,
			statuses: []int{http.StatusCreated},
		},
		{
			name:     "DeviceManagementStopDeployment",
			method:   http.MethodPost,
			template: "/deviceUpdate/{instanceId}/management/groups/{groupId}/deviceClassSubgroups/{deviceClassId}/deployments/{deploymentId}:cancel",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceManagementUpdateDeviceClass",
			method:   http.MethodPatch,
			template: "/deviceUpdate/{instanceId}/management/deviceClasses/{deviceClassId}",
			payload:  updateDeviceClassPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateDeleteUpdate",
			method:   http.MethodDelete,
			template: "/deviceUpdate/{instanceId}/updates/providers/{provider}/names/{name}/versions/{version}",
			statuses: []int{http.StatusAccepted},
		},
		{
			name:     "DeviceUpdateGetFile",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/providers/{provider}/names/{name}/versions/{version}/files/{fileId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateGetOperationStatus",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/operations/{operationId}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateGetUpdate",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/providers/{provider}/names/{name}/versions/{version}",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateImportUpdate",
			method:   http.MethodPost,
			template: "/deviceUpdate/{instanceId}/updates:import",
			payload:  importUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "DeviceUpdateListFiles",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/providers/{provider}/names/{name}/versions/{version}/files",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateListNames",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/providers/{provider}/names",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateListOperationStatuses",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/operations",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateListProviders",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/providers",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateListUpdates",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeviceUpdateListVersions",
			method:   http.MethodGet,
			template: "/deviceUpdate/{instanceId}/updates/providers/{provider}/names/{name}/versions",
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, routeVars, call.extraQuery)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
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
