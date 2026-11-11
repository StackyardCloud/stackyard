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

const desktopVirtualizationAPIVersion = "2025-10-10"

type desktopVirtualizationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type desktopVirtualizationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *desktopVirtualizationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDesktopVirtualizationClient(endpoint, account, subscriptionKey string) *desktopVirtualizationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"desktopvirtualization-2025-10-10",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&desktopVirtualizationSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &desktopVirtualizationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *desktopVirtualizationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DESKTOPVIRTUALIZATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DESKTOPVIRTUALIZATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-desktopvirtualization")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")

	hostPoolName := getenv("STACKYARD_AZURE_DV_HOST_POOL_NAME", "dv-hostpool-example")
	applicationGroupName := getenv("STACKYARD_AZURE_DV_APPLICATION_GROUP_NAME", "dv-appgroup-example")
	applicationName := getenv("STACKYARD_AZURE_DV_APPLICATION_NAME", "dv-app-example")
	desktopName := getenv("STACKYARD_AZURE_DV_DESKTOP_NAME", "SessionDesktop")
	workspaceName := getenv("STACKYARD_AZURE_DV_WORKSPACE_NAME", "dv-workspace-example")
	sessionHostName := getenv("STACKYARD_AZURE_DV_SESSION_HOST_NAME", "sessionhost1.contoso.com")
	userSessionID := getenv("STACKYARD_AZURE_DV_USER_SESSION_ID", "1")
	appAttachPackageName := getenv("STACKYARD_AZURE_DV_APP_ATTACH_PACKAGE_NAME", "dv-appattach-package")
	msixPackageFullName := getenv("STACKYARD_AZURE_DV_MSIX_PACKAGE_FULL_NAME", "dv-msix-package-fullname")
	scalingPlanName := getenv("STACKYARD_AZURE_DV_SCALING_PLAN_NAME", "dv-scaling-plan")
	personalScheduleName := getenv("STACKYARD_AZURE_DV_PERSONAL_SCHEDULE_NAME", "dv-personal-weekday")
	pooledScheduleName := getenv("STACKYARD_AZURE_DV_POOLED_SCHEDULE_NAME", "dv-pooled-weekday")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_DV_PRIVATE_ENDPOINT_CONNECTION_NAME", hostPoolName+".377103f1-5179-4bdf-8556-4cdd3207cc5b")

	hostPoolBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/hostPools/%s",
		subscriptionID,
		resourceGroup,
		hostPoolName,
	)
	applicationGroupBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/applicationGroups/%s",
		subscriptionID,
		resourceGroup,
		applicationGroupName,
	)
	applicationBase := applicationGroupBase + "/applications/" + applicationName
	desktopBase := applicationGroupBase + "/desktops/" + desktopName
	workspaceBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/workspaces/%s",
		subscriptionID,
		resourceGroup,
		workspaceName,
	)
	appAttachPackageBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/appAttachPackages/%s",
		subscriptionID,
		resourceGroup,
		appAttachPackageName,
	)
	msixPackageBase := hostPoolBase + "/msixPackages/" + msixPackageFullName
	scalingPlanBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/scalingPlans/%s",
		subscriptionID,
		resourceGroup,
		scalingPlanName,
	)
	personalScheduleBase := scalingPlanBase + "/personalSchedules/" + personalScheduleName
	pooledScheduleBase := scalingPlanBase + "/pooledSchedules/" + pooledScheduleName
	sessionHostBase := hostPoolBase + "/sessionHosts/" + sessionHostName
	userSessionBase := sessionHostBase + "/userSessions/" + userSessionID
	privateEndpointConnectionByHostPoolBase := hostPoolBase + "/privateEndpointConnections/" + privateEndpointConnectionName
	privateEndpointConnectionByWorkspaceBase := workspaceBase + "/privateEndpointConnections/" + privateEndpointConnectionName

	hostPoolCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
			"service":     "desktopvirtualization",
		},
		"properties": map[string]any{
			"friendlyName":                "Desktop Virtualization Host Pool",
			"description":                 "Local Desktop Virtualization host pool",
			"hostPoolType":                "Pooled",
			"loadBalancerType":            "BreadthFirst",
			"preferredAppGroupType":       "Desktop",
			"maxSessionLimit":             16,
			"startVMOnConnect":            true,
			"customRdpProperty":           "audiocapturemode:i:1;redirectclipboard:i:1",
			"publicNetworkAccess":         "Enabled",
			"validationEnvironment":       false,
			"ssoadfsAuthority":            "https://login.microsoftonline.com/",
			"ssoClientId":                 "00000000-0000-0000-0000-000000000000",
			"ssoClientSecretKeyVaultPath": "https://example-keyvault.vault.azure.net/secrets/desktopvirtualization",
			"ssoSecretType":               "SharedKey",
		},
	}

	hostPoolUpdatePayload := map[string]any{
		"tags": map[string]any{
			"environment": "local-updated",
			"service":     "desktopvirtualization",
		},
		"properties": map[string]any{
			"friendlyName":     "Desktop Virtualization Host Pool Updated",
			"maxSessionLimit":  32,
			"startVMOnConnect": true,
		},
	}

	applicationGroupCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
		},
		"properties": map[string]any{
			"applicationGroupType": "RemoteApp",
			"hostPoolArmPath":      strings.TrimPrefix(hostPoolBase, "/azure"),
			"friendlyName":         "Desktop Virtualization Application Group",
			"description":          "Local application group",
		},
	}

	applicationGroupUpdatePayload := map[string]any{
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization Application Group Updated",
			"description":  "Local application group updated",
		},
	}

	applicationCreatePayload := map[string]any{
		"properties": map[string]any{
			"friendlyName":         "Desktop Virtualization App",
			"description":          "Local application",
			"filePath":             "C:\\Windows\\System32\\notepad.exe",
			"commandLineSetting":   "Allow",
			"commandLineArguments": "",
			"showInPortal":         true,
			"applicationType":      "InBuilt",
		},
	}

	applicationUpdatePayload := map[string]any{
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization App Updated",
			"description":  "Local application updated",
		},
	}

	desktopUpdatePayload := map[string]any{
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization Desktop Updated",
			"description":  "Local desktop updated",
		},
	}

	workspaceCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
		},
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization Workspace",
			"description":  "Local workspace",
			"applicationGroupReferences": []any{
				strings.TrimPrefix(applicationGroupBase, "/azure"),
			},
		},
	}

	workspaceUpdatePayload := map[string]any{
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization Workspace Updated",
			"description":  "Local workspace updated",
		},
	}

	appAttachPackageCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"image": map[string]any{
				"imagePath":             "\\\\fileserver\\msix\\desktopvirtualization.vhdx",
				"packageAlias":          "desktopvirtualization",
				"packageName":           "DesktopVirtualizationPackage",
				"packageFamilyName":     "DesktopVirtualizationPackage_8wekyb3d8bbwe",
				"packageFullName":       "DesktopVirtualizationPackage_1.0.0.0_x64__8wekyb3d8bbwe",
				"displayName":           "Desktop Virtualization Package",
				"packageRelativePath":   "desktopvirtualization.msix",
				"version":               "1.0.0.0",
				"isRegularRegistration": false,
				"isActive":              true,
			},
			"hostPoolReferences": []any{
				map[string]any{
					"hostPoolArmPath": strings.TrimPrefix(hostPoolBase, "/azure"),
				},
			},
			"keyVaultURL": "",
		},
	}

	appAttachPackageUpdatePayload := map[string]any{
		"properties": map[string]any{
			"keyVaultURL": "https://example-keyvault.vault.azure.net/",
		},
	}

	appAttachImportPayload := map[string]any{
		"path":                "\\\\fileserver\\msix\\desktopvirtualization.vhdx",
		"packageArchitecture": "x64",
	}

	msixPackageCreatePayload := map[string]any{
		"properties": map[string]any{
			"displayName":           "Desktop Virtualization MSIX",
			"imagePath":             "\\\\fileserver\\msix\\desktopvirtualization.vhdx",
			"packageName":           "DesktopVirtualizationPackage",
			"packageFamilyName":     "DesktopVirtualizationPackage_8wekyb3d8bbwe",
			"packageRelativePath":   "desktopvirtualization.msix",
			"version":               "1.0.0.0",
			"isRegularRegistration": false,
			"isActive":              true,
		},
	}

	msixPackageUpdatePayload := map[string]any{
		"properties": map[string]any{
			"isActive": true,
		},
	}

	scalingPlanCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
		},
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization Scaling Plan",
			"description":  "Local scaling plan",
			"timeZone":     "UTC",
			"hostPoolType": "Pooled",
			"hostPoolReferences": []any{
				map[string]any{
					"hostPoolArmPath":    strings.TrimPrefix(hostPoolBase, "/azure"),
					"scalingPlanEnabled": true,
				},
			},
		},
	}

	scalingPlanUpdatePayload := map[string]any{
		"properties": map[string]any{
			"friendlyName": "Desktop Virtualization Scaling Plan Updated",
			"description":  "Local scaling plan updated",
		},
	}

	pooledScheduleCreatePayload := map[string]any{
		"properties": map[string]any{
			"daysOfWeek":                     []any{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			"rampUpStartTime":                map[string]any{"hour": 6, "minute": 0},
			"rampUpLoadBalancingAlgorithm":   "DepthFirst",
			"rampUpMinimumHostsPct":          20,
			"rampUpCapacityThresholdPct":     80,
			"peakStartTime":                  map[string]any{"hour": 8, "minute": 0},
			"peakLoadBalancingAlgorithm":     "BreadthFirst",
			"rampDownStartTime":              map[string]any{"hour": 18, "minute": 0},
			"rampDownLoadBalancingAlgorithm": "DepthFirst",
			"rampDownMinimumHostsPct":        20,
			"rampDownCapacityThresholdPct":   50,
			"rampDownForceLogoffUsers":       true,
			"rampDownWaitTimeMinutes":        30,
			"offPeakStartTime":               map[string]any{"hour": 20, "minute": 0},
			"offPeakLoadBalancingAlgorithm":  "DepthFirst",
		},
	}

	pooledScheduleUpdatePayload := map[string]any{
		"properties": map[string]any{
			"rampDownMinimumHostsPct": 25,
			"rampDownWaitTimeMinutes": 20,
		},
	}

	personalScheduleCreatePayload := map[string]any{
		"properties": map[string]any{
			"daysOfWeek":                        []any{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
			"rampUpStartTime":                   map[string]any{"hour": 6, "minute": 0},
			"rampUpStartVMOnConnect":            "Enable",
			"rampUpActionOnDisconnect":          "None",
			"rampUpMinutesToWaitOnDisconnect":   10,
			"peakStartTime":                     map[string]any{"hour": 8, "minute": 0},
			"peakStartVMOnConnect":              "Enable",
			"peakActionOnDisconnect":            "None",
			"peakMinutesToWaitOnDisconnect":     10,
			"rampDownStartTime":                 map[string]any{"hour": 18, "minute": 0},
			"rampDownStartVMOnConnect":          "Enable",
			"rampDownActionOnDisconnect":        "Deallocate",
			"rampDownMinutesToWaitOnDisconnect": 10,
			"offPeakStartTime":                  map[string]any{"hour": 20, "minute": 0},
			"offPeakStartVMOnConnect":           "Enable",
			"offPeakActionOnDisconnect":         "None",
			"offPeakMinutesToWaitOnDisconnect":  10,
		},
	}

	personalScheduleUpdatePayload := map[string]any{
		"properties": map[string]any{
			"offPeakMinutesToWaitOnDisconnect": 5,
			"offPeakActionOnDisconnect":        "Deallocate",
		},
	}

	privateEndpointUpdatePayload := map[string]any{
		"properties": map[string]any{
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Approved",
				"description":     "Auto-approved in local emulator",
				"actionsRequired": "None",
			},
		},
	}

	sessionHostUpdatePayload := map[string]any{
		"properties": map[string]any{
			"allowNewSession": true,
			"assignedUser":    "user1@contoso.com",
			"friendlyName":    "session-host-01",
		},
	}

	expandMsixImagePayload := map[string]any{
		"uri": "\\\\fileserver\\msix\\desktopvirtualization.vhdx",
	}

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
			path:     "/azure/providers/Microsoft.DesktopVirtualization/operations?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateHostPool",
			method:   http.MethodPut,
			path:     hostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  hostPoolCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetHostPool",
			method:   http.MethodGet,
			path:     hostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListHostPoolsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/hostPools?api-version=%s", subscriptionID, resourceGroup, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListHostPoolsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DesktopVirtualization/hostPools?api-version=%s", subscriptionID, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListHostPoolRegistrationTokens",
			method:   http.MethodPost,
			path:     hostPoolBase + "/listRegistrationTokens?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "RetrieveHostPoolRegistrationToken",
			method:   http.MethodPost,
			path:     hostPoolBase + "/retrieveRegistrationToken?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateHostPool",
			method:   http.MethodPatch,
			path:     hostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  hostPoolUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateApplicationGroup",
			method:   http.MethodPut,
			path:     applicationGroupBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  applicationGroupCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetApplicationGroup",
			method:   http.MethodGet,
			path:     applicationGroupBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListApplicationGroupsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/applicationGroups?api-version=%s", subscriptionID, resourceGroup, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListApplicationGroupsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DesktopVirtualization/applicationGroups?api-version=%s", subscriptionID, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateApplicationGroup",
			method:   http.MethodPatch,
			path:     applicationGroupBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  applicationGroupUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateApplication",
			method:   http.MethodPut,
			path:     applicationBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  applicationCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetApplication",
			method:   http.MethodGet,
			path:     applicationBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListApplications",
			method:   http.MethodGet,
			path:     applicationGroupBase + "/applications?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateApplication",
			method:   http.MethodPatch,
			path:     applicationBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  applicationUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDesktop",
			method:   http.MethodGet,
			path:     desktopBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDesktops",
			method:   http.MethodGet,
			path:     applicationGroupBase + "/desktops?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateDesktop",
			method:   http.MethodPatch,
			path:     desktopBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  desktopUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListStartMenuItems",
			method:   http.MethodGet,
			path:     applicationGroupBase + "/startMenuItems?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateWorkspace",
			method:   http.MethodPut,
			path:     workspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  workspaceCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspacesByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/workspaces?api-version=%s", subscriptionID, resourceGroup, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWorkspacesBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DesktopVirtualization/workspaces?api-version=%s", subscriptionID, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateWorkspace",
			method:   http.MethodPatch,
			path:     workspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  workspaceUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateScalingPlan",
			method:   http.MethodPut,
			path:     scalingPlanBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  scalingPlanCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetScalingPlan",
			method:   http.MethodGet,
			path:     scalingPlanBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListScalingPlansByHostPool",
			method:   http.MethodGet,
			path:     hostPoolBase + "/scalingPlans?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListScalingPlansByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/scalingPlans?api-version=%s", subscriptionID, resourceGroup, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListScalingPlansBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DesktopVirtualization/scalingPlans?api-version=%s", subscriptionID, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateScalingPlan",
			method:   http.MethodPatch,
			path:     scalingPlanBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  scalingPlanUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateScalingPlanPooledSchedule",
			method:   http.MethodPut,
			path:     pooledScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  pooledScheduleCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetScalingPlanPooledSchedule",
			method:   http.MethodGet,
			path:     pooledScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListScalingPlanPooledSchedules",
			method:   http.MethodGet,
			path:     scalingPlanBase + "/pooledSchedules?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateScalingPlanPooledSchedule",
			method:   http.MethodPatch,
			path:     pooledScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  pooledScheduleUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateScalingPlanPersonalSchedule",
			method:   http.MethodPut,
			path:     personalScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  personalScheduleCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetScalingPlanPersonalSchedule",
			method:   http.MethodGet,
			path:     personalScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListScalingPlanPersonalSchedules",
			method:   http.MethodGet,
			path:     scalingPlanBase + "/personalSchedules?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateScalingPlanPersonalSchedule",
			method:   http.MethodPatch,
			path:     personalScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  personalScheduleUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateAppAttachPackage",
			method:   http.MethodPut,
			path:     appAttachPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  appAttachPackageCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetAppAttachPackage",
			method:   http.MethodGet,
			path:     appAttachPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAppAttachPackagesByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DesktopVirtualization/appAttachPackages?api-version=%s", subscriptionID, resourceGroup, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAppAttachPackagesBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DesktopVirtualization/appAttachPackages?api-version=%s", subscriptionID, desktopVirtualizationAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateAppAttachPackage",
			method:   http.MethodPatch,
			path:     appAttachPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  appAttachPackageUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ImportAppAttachPackageInfo",
			method:   http.MethodPost,
			path:     hostPoolBase + "/importAppAttachPackageInfo?api-version=" + desktopVirtualizationAPIVersion,
			payload:  appAttachImportPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateMSIXPackage",
			method:   http.MethodPut,
			path:     msixPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  msixPackageCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetMSIXPackage",
			method:   http.MethodGet,
			path:     msixPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListMSIXPackages",
			method:   http.MethodGet,
			path:     hostPoolBase + "/msixPackages?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateMSIXPackage",
			method:   http.MethodPatch,
			path:     msixPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  msixPackageUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ExpandMSIXImage",
			method:   http.MethodPost,
			path:     hostPoolBase + "/expandMsixImage?api-version=" + desktopVirtualizationAPIVersion,
			payload:  expandMsixImagePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResourcesByHostPool",
			method:   http.MethodGet,
			path:     hostPoolBase + "/privateLinkResources?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResourcesByWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "/privateLinkResources?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnectionsByHostPool",
			method:   http.MethodGet,
			path:     hostPoolBase + "/privateEndpointConnections?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateEndpointConnectionByHostPool",
			method:   http.MethodGet,
			path:     privateEndpointConnectionByHostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdatePrivateEndpointConnectionByHostPool",
			method:   http.MethodPut,
			path:     privateEndpointConnectionByHostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  privateEndpointUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnectionsByWorkspace",
			method:   http.MethodGet,
			path:     workspaceBase + "/privateEndpointConnections?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateEndpointConnectionByWorkspace",
			method:   http.MethodGet,
			path:     privateEndpointConnectionByWorkspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdatePrivateEndpointConnectionByWorkspace",
			method:   http.MethodPut,
			path:     privateEndpointConnectionByWorkspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			payload:  privateEndpointUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSessionHost",
			method:   http.MethodGet,
			path:     sessionHostBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSessionHosts",
			method:   http.MethodGet,
			path:     hostPoolBase + "/sessionHosts?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateSessionHost",
			method:   http.MethodPatch,
			path:     sessionHostBase + "?api-version=" + desktopVirtualizationAPIVersion + "&force=true",
			payload:  sessionHostUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetUserSession",
			method:   http.MethodGet,
			path:     userSessionBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListUserSessions",
			method:   http.MethodGet,
			path:     sessionHostBase + "/userSessions?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListUserSessionsByHostPool",
			method:   http.MethodGet,
			path:     hostPoolBase + "/userSessions?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DisconnectUserSession",
			method:   http.MethodPost,
			path:     userSessionBase + "/disconnect?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SendUserSessionMessage",
			method: http.MethodPost,
			path:   userSessionBase + "/sendMessage?api-version=" + desktopVirtualizationAPIVersion,
			payload: map[string]any{
				"messageTitle": "Desktop Virtualization Local Message",
				"messageBody":  "Sent from Stackyard Desktop Virtualization example",
			},
			statuses: []int{http.StatusOK},
		},
	}

	client := newDesktopVirtualizationClient(endpoint, account, subscriptionKey)
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
			name:     "DeletePrivateEndpointConnectionByHostPool",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionByHostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeletePrivateEndpointConnectionByWorkspace",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionByWorkspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteUserSession",
			method:   http.MethodDelete,
			path:     userSessionBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteSessionHost",
			method:   http.MethodDelete,
			path:     sessionHostBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteMSIXPackage",
			method:   http.MethodDelete,
			path:     msixPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteAppAttachPackage",
			method:   http.MethodDelete,
			path:     appAttachPackageBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteScalingPlanPooledSchedule",
			method:   http.MethodDelete,
			path:     pooledScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteScalingPlanPersonalSchedule",
			method:   http.MethodDelete,
			path:     personalScheduleBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteScalingPlan",
			method:   http.MethodDelete,
			path:     scalingPlanBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteApplication",
			method:   http.MethodDelete,
			path:     applicationBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteApplicationGroup",
			method:   http.MethodDelete,
			path:     applicationGroupBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteWorkspace",
			method:   http.MethodDelete,
			path:     workspaceBase + "?api-version=" + desktopVirtualizationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteHostPool",
			method:   http.MethodDelete,
			path:     hostPoolBase + "?api-version=" + desktopVirtualizationAPIVersion,
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
