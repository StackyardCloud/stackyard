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

const devCenterAdministratorAPIVersion = "2024-02-01"

type devCenterAdministratorClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type devCenterAdministratorSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *devCenterAdministratorSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDevCenterAdministratorClient(endpoint, account, subscriptionKey string) *devCenterAdministratorClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"devcenter-administrator-2024-02-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&devCenterAdministratorSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &devCenterAdministratorClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *devCenterAdministratorClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DEVCENTER_ADMINISTRATOR_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEVCENTER_ADMINISTRATOR_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-devcenter")
	location := getenv("STACKYARD_AZURE_LOCATION", "centralus")

	devCenterName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_DEV_CENTER_NAME", "devcenter-example")
	projectName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_PROJECT_NAME", "project-example")
	networkConnectionName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_NETWORK_CONNECTION_NAME", "network-example")
	attachedNetworkConnectionName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_ATTACHED_NETWORK_NAME", "attached-network-example")
	catalogName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_CATALOG_NAME", "catalog-example")
	projectCatalogName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_PROJECT_CATALOG_NAME", "project-catalog-example")
	devBoxDefinitionName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_DEVBOX_DEFINITION_NAME", "devbox-definition-example")
	environmentTypeName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_ENVIRONMENT_TYPE_NAME", "envtype-example")
	projectEnvironmentTypeName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_PROJECT_ENVIRONMENT_TYPE_NAME", "project-envtype-example")
	environmentDefinitionName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_ENVIRONMENT_DEFINITION_NAME", "environment-definition-example")
	galleryName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_GALLERY_NAME", "gallery-example")
	imageName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_IMAGE_NAME", "image-example")
	imageVersionName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_IMAGE_VERSION_NAME", "1.0.0")
	poolName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_POOL_NAME", "pool-example")
	scheduleName := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_SCHEDULE_NAME", "weekday-shutdown")
	operationID := getenv("STACKYARD_AZURE_DEVCENTER_ADMIN_OPERATION_ID", "3fa1a29d-e807-488d-81d1-f1c5456a08cd")

	devCenterBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevCenter/devcenters/%s",
		subscriptionID,
		resourceGroup,
		devCenterName,
	)
	projectsBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevCenter/projects/%s",
		subscriptionID,
		resourceGroup,
		projectName,
	)
	networkConnectionBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevCenter/networkConnections/%s",
		subscriptionID,
		resourceGroup,
		networkConnectionName,
	)
	catalogBase := devCenterBase + "/catalogs/" + catalogName
	projectCatalogBase := projectsBase + "/catalogs/" + projectCatalogName
	attachedNetworkByDevCenterBase := devCenterBase + "/attachednetworks/" + attachedNetworkConnectionName
	attachedNetworkByProjectBase := projectsBase + "/attachednetworks/" + attachedNetworkConnectionName
	devBoxDefinitionBase := devCenterBase + "/devboxdefinitions/" + devBoxDefinitionName
	devBoxDefinitionByProjectBase := projectsBase + "/devboxdefinitions/" + devBoxDefinitionName
	environmentTypeBase := devCenterBase + "/environmentTypes/" + environmentTypeName
	projectEnvironmentTypeBase := projectsBase + "/environmentTypes/" + projectEnvironmentTypeName
	projectAllowedEnvironmentTypeBase := projectsBase + "/allowedEnvironmentTypes/" + projectEnvironmentTypeName
	environmentDefinitionBase := catalogBase + "/environmentDefinitions/" + environmentDefinitionName
	projectEnvironmentDefinitionBase := projectCatalogBase + "/environmentDefinitions/" + environmentDefinitionName
	galleryBase := devCenterBase + "/galleries/" + galleryName
	imageBase := galleryBase + "/images/" + imageName
	imageVersionBase := imageBase + "/versions/" + imageVersionName
	poolBase := projectsBase + "/pools/" + poolName
	scheduleBase := poolBase + "/schedules/" + scheduleName
	operationStatusBase := fmt.Sprintf(
		"/azure/subscriptions/%s/providers/Microsoft.DevCenter/locations/%s/operationStatuses/%s",
		subscriptionID,
		location,
		operationID,
	)

	devCenterCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"CostCode": "12345",
		},
		"properties": map[string]any{
			"displayName": "Stackyard Dev Center",
		},
	}

	devCenterUpdatePayload := map[string]any{
		"tags": map[string]any{
			"CostCode": "67890",
		},
		"properties": map[string]any{
			"displayName": "Stackyard Dev Center Updated",
		},
	}

	projectCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"CostCenter": "R&D",
		},
		"properties": map[string]any{
			"displayName":        "Stackyard Project",
			"description":        "Project for Dev Center admin API example",
			"devCenterId":        strings.TrimPrefix(devCenterBase, "/azure"),
			"maxDevBoxesPerUser": 3,
		},
	}

	projectUpdatePayload := map[string]any{
		"tags": map[string]any{
			"CostCenter": "Engineering",
		},
		"properties": map[string]any{
			"displayName": "Stackyard Project Updated",
			"description": "Updated project metadata",
			"catalogSettings": map[string]any{
				"catalogItemSyncTypes": []any{"EnvironmentDefinition"},
			},
		},
	}

	networkConnectionCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"domainName":                  "contoso.local",
			"domainUsername":              "devbox-admin@contoso.local",
			"domainPassword":              "SuperSecretPassword!123",
			"subnetId":                    "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.Network/virtualNetworks/example-vnet/subnets/default",
			"domainJoinType":              "HybridAzureADJoin",
			"networkingResourceGroupName": "NetworkInterfaces",
		},
	}

	networkConnectionUpdatePayload := map[string]any{
		"properties": map[string]any{
			"domainPassword": "UpdatedSecretPassword!456",
		},
	}

	attachedNetworkCreatePayload := map[string]any{
		"properties": map[string]any{
			"networkConnectionId": strings.TrimPrefix(networkConnectionBase, "/azure"),
		},
	}

	catalogCreatePayload := map[string]any{
		"properties": map[string]any{
			"gitHub": map[string]any{
				"uri":    "https://github.com/contoso/devcenter-catalog",
				"branch": "main",
				"path":   "/environments",
			},
			"syncType": "Manual",
		},
	}

	catalogUpdatePayload := map[string]any{
		"properties": map[string]any{
			"gitHub": map[string]any{
				"path": "/environments/updated",
			},
		},
	}

	projectCatalogCreatePayload := map[string]any{
		"properties": map[string]any{
			"adoGit": map[string]any{
				"uri":    "https://dev.azure.com/contoso/DevCenter/_git/project-catalog",
				"branch": "main",
				"path":   "/catalog",
			},
			"syncType": "Manual",
		},
	}

	projectCatalogPatchPayload := map[string]any{
		"properties": map[string]any{
			"gitHub": map[string]any{
				"path": "/catalog/updated",
			},
		},
	}

	devBoxDefinitionCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"imageReference": map[string]any{
				"id": strings.TrimPrefix(imageBase, "/azure"),
			},
			"sku": map[string]any{
				"name": "general_i_8c32gb256ssd_v2",
			},
		},
	}

	devBoxDefinitionUpdatePayload := map[string]any{
		"properties": map[string]any{
			"hibernateSupport": "Enabled",
		},
	}

	environmentTypeCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"displayName": "Dev/Test",
			"status":      "Enabled",
		},
	}

	environmentTypeUpdatePayload := map[string]any{
		"properties": map[string]any{
			"displayName": "Dev/Test Updated",
			"status":      "Enabled",
		},
	}

	projectEnvironmentTypeCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"displayName":        "Project Dev/Test",
			"status":             "Enabled",
			"deploymentTargetId": fmt.Sprintf("/subscriptions/%s", subscriptionID),
		},
	}

	projectEnvironmentTypeUpdatePayload := map[string]any{
		"properties": map[string]any{
			"displayName": "Project Dev/Test Updated",
			"status":      "Enabled",
		},
	}

	galleryCreatePayload := map[string]any{
		"properties": map[string]any{
			"galleryResourceId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/image-rg/providers/Microsoft.Compute/galleries/gallery-example",
		},
	}

	poolCreatePayload := map[string]any{
		"location": location,
		"properties": map[string]any{
			"devBoxDefinitionName":  devBoxDefinitionName,
			"networkConnectionName": networkConnectionName,
			"licenseType":           "Windows_Client",
			"localAdministrator":    "Enabled",
			"singleSignOnStatus":    "Disabled",
			"virtualNetworkType":    "Unmanaged",
			"displayName":           "Developer Pool",
			"stopOnDisconnect": map[string]any{
				"status":             "Enabled",
				"gracePeriodMinutes": 60,
			},
		},
	}

	poolUpdatePayload := map[string]any{
		"properties": map[string]any{
			"stopOnDisconnect": map[string]any{
				"status":             "Enabled",
				"gracePeriodMinutes": 30,
			},
		},
	}

	scheduleCreatePayload := map[string]any{
		"properties": map[string]any{
			"type":      "StopDevBox",
			"state":     "Enabled",
			"time":      "19:00",
			"timezone":  "UTC",
			"frequency": "Daily",
		},
	}

	scheduleUpdatePayload := map[string]any{
		"properties": map[string]any{
			"state": "Enabled",
			"time":  "20:00",
		},
	}

	checkNameAvailabilityPayload := map[string]any{
		"name": "devcenter-check-example",
		"type": "Microsoft.DevCenter/devcenters",
	}

	checkScopedNameAvailabilityPayload := map[string]any{
		"name":  "catalog-check-example",
		"type":  "Microsoft.DevCenter/projects/catalogs",
		"scope": strings.TrimPrefix(projectsBase, "/azure"),
	}

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DevCenter/operations?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSKUsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/skus?api-version=%s", subscriptionID, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListUsagesByLocation",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/locations/%s/usages?api-version=%s", subscriptionID, location, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckNameAvailabilityExecute",
			method:   http.MethodPost,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/checkNameAvailability?api-version=%s", subscriptionID, devCenterAdministratorAPIVersion),
			payload:  checkNameAvailabilityPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckScopedNameAvailabilityExecute",
			method:   http.MethodPost,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/checkScopedNameAvailability?api-version=%s", subscriptionID, devCenterAdministratorAPIVersion),
			payload:  checkScopedNameAvailabilityPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateDevCenter",
			method:   http.MethodPut,
			path:     devCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  devCenterCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevCentersByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevCenter/devcenters?api-version=%s", subscriptionID, resourceGroup, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevCentersBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/devcenters?api-version=%s", subscriptionID, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateDevCenter",
			method:   http.MethodPatch,
			path:     devCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  devCenterUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateNetworkConnection",
			method:   http.MethodPut,
			path:     networkConnectionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  networkConnectionCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetNetworkConnection",
			method:   http.MethodGet,
			path:     networkConnectionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListNetworkConnectionsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevCenter/networkConnections?api-version=%s", subscriptionID, resourceGroup, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListNetworkConnectionsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/networkConnections?api-version=%s", subscriptionID, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetNetworkConnectionHealthDetails",
			method:   http.MethodGet,
			path:     networkConnectionBase + "/healthChecks/latest?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListNetworkConnectionHealthDetails",
			method:   http.MethodGet,
			path:     networkConnectionBase + "/healthChecks?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListNetworkConnectionOutboundDependencies",
			method:   http.MethodGet,
			path:     networkConnectionBase + "/outboundNetworkDependenciesEndpoints?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "RunNetworkConnectionHealthChecks",
			method:   http.MethodPost,
			path:     networkConnectionBase + "/runHealthChecks?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "UpdateNetworkConnection",
			method:   http.MethodPatch,
			path:     networkConnectionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  networkConnectionUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateAttachedNetworkByDevCenter",
			method:   http.MethodPut,
			path:     attachedNetworkByDevCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  attachedNetworkCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetAttachedNetworkByDevCenter",
			method:   http.MethodGet,
			path:     attachedNetworkByDevCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAttachedNetworksByDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "/attachednetworks?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateProject",
			method:   http.MethodPut,
			path:     projectsBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  projectCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetProject",
			method:   http.MethodGet,
			path:     projectsBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProjectsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevCenter/projects?api-version=%s", subscriptionID, resourceGroup, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProjectsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevCenter/projects?api-version=%s", subscriptionID, devCenterAdministratorAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateProject",
			method:   http.MethodPatch,
			path:     projectsBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  projectUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetAttachedNetworkByProject",
			method:   http.MethodGet,
			path:     attachedNetworkByProjectBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAttachedNetworksByProject",
			method:   http.MethodGet,
			path:     projectsBase + "/attachednetworks?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateEnvironmentType",
			method:   http.MethodPut,
			path:     environmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  environmentTypeCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetEnvironmentType",
			method:   http.MethodGet,
			path:     environmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironmentTypesByDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "/environmentTypes?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateEnvironmentType",
			method:   http.MethodPatch,
			path:     environmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  environmentTypeUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateProjectEnvironmentType",
			method:   http.MethodPut,
			path:     projectEnvironmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  projectEnvironmentTypeCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetProjectEnvironmentType",
			method:   http.MethodGet,
			path:     projectEnvironmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProjectEnvironmentTypes",
			method:   http.MethodGet,
			path:     projectsBase + "/environmentTypes?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateProjectEnvironmentType",
			method:   http.MethodPatch,
			path:     projectEnvironmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  projectEnvironmentTypeUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProjectAllowedEnvironmentType",
			method:   http.MethodGet,
			path:     projectAllowedEnvironmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProjectAllowedEnvironmentTypes",
			method:   http.MethodGet,
			path:     projectsBase + "/allowedEnvironmentTypes?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateCatalog",
			method:   http.MethodPut,
			path:     catalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  catalogCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetCatalog",
			method:   http.MethodGet,
			path:     catalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListCatalogsByDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "/catalogs?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ConnectCatalog",
			method:   http.MethodPost,
			path:     catalogBase + "/connect?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "SyncCatalog",
			method:   http.MethodPost,
			path:     catalogBase + "/sync?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetCatalogSyncErrorDetails",
			method:   http.MethodPost,
			path:     catalogBase + "/getSyncErrorDetails?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateCatalog",
			method:   http.MethodPatch,
			path:     catalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  catalogUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateProjectCatalog",
			method:   http.MethodPut,
			path:     projectCatalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  projectCatalogCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetProjectCatalog",
			method:   http.MethodGet,
			path:     projectCatalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProjectCatalogs",
			method:   http.MethodGet,
			path:     projectsBase + "/catalogs?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ConnectProjectCatalog",
			method:   http.MethodPost,
			path:     projectCatalogBase + "/connect?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "SyncProjectCatalog",
			method:   http.MethodPost,
			path:     projectCatalogBase + "/sync?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetProjectCatalogSyncErrorDetails",
			method:   http.MethodPost,
			path:     projectCatalogBase + "/getSyncErrorDetails?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PatchProjectCatalog",
			method:   http.MethodPatch,
			path:     projectCatalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  projectCatalogPatchPayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateDevBoxDefinition",
			method:   http.MethodPut,
			path:     devBoxDefinitionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  devBoxDefinitionCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetDevBoxDefinition",
			method:   http.MethodGet,
			path:     devBoxDefinitionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevBoxDefinitionsByDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "/devboxdefinitions?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetDevBoxDefinitionByProject",
			method:   http.MethodGet,
			path:     devBoxDefinitionByProjectBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDevBoxDefinitionsByProject",
			method:   http.MethodGet,
			path:     projectsBase + "/devboxdefinitions?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateDevBoxDefinition",
			method:   http.MethodPatch,
			path:     devBoxDefinitionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  devBoxDefinitionUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateGallery",
			method:   http.MethodPut,
			path:     galleryBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  galleryCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetGallery",
			method:   http.MethodGet,
			path:     galleryBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListGalleriesByDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "/galleries?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetImage",
			method:   http.MethodGet,
			path:     imageBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListImagesByDevCenter",
			method:   http.MethodGet,
			path:     devCenterBase + "/images?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListImagesByGallery",
			method:   http.MethodGet,
			path:     galleryBase + "/images?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetImageVersion",
			method:   http.MethodGet,
			path:     imageVersionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListImageVersionsByImage",
			method:   http.MethodGet,
			path:     imageBase + "/versions?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEnvironmentDefinition",
			method:   http.MethodGet,
			path:     environmentDefinitionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironmentDefinitionsByCatalog",
			method:   http.MethodGet,
			path:     catalogBase + "/environmentDefinitions?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEnvironmentDefinitionErrorDetails",
			method:   http.MethodPost,
			path:     environmentDefinitionBase + "/getErrorDetails?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEnvironmentDefinitionByProjectCatalog",
			method:   http.MethodGet,
			path:     projectEnvironmentDefinitionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEnvironmentDefinitionsByProjectCatalog",
			method:   http.MethodGet,
			path:     projectCatalogBase + "/environmentDefinitions?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetProjectCatalogEnvironmentDefinitionErrorDetails",
			method:   http.MethodPost,
			path:     projectEnvironmentDefinitionBase + "/getErrorDetails?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdatePool",
			method:   http.MethodPut,
			path:     poolBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  poolCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetPool",
			method:   http.MethodGet,
			path:     poolBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPoolsByProject",
			method:   http.MethodGet,
			path:     projectsBase + "/pools?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "RunPoolHealthChecks",
			method:   http.MethodPost,
			path:     poolBase + "/runHealthChecks?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "UpdatePool",
			method:   http.MethodPatch,
			path:     poolBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  poolUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "CreateOrUpdateSchedule",
			method:   http.MethodPut,
			path:     scheduleBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  scheduleCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetSchedule",
			method:   http.MethodGet,
			path:     scheduleBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSchedulesByPool",
			method:   http.MethodGet,
			path:     poolBase + "/schedules?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateSchedule",
			method:   http.MethodPatch,
			path:     scheduleBase + "?api-version=" + devCenterAdministratorAPIVersion,
			payload:  scheduleUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetOperationStatus",
			method:   http.MethodGet,
			path:     operationStatusBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
	}

	client := newDevCenterAdministratorClient(endpoint, account, subscriptionKey)
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
			name:     "DeleteSchedule",
			method:   http.MethodDelete,
			path:     scheduleBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeletePool",
			method:   http.MethodDelete,
			path:     poolBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteProjectEnvironmentType",
			method:   http.MethodDelete,
			path:     projectEnvironmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteEnvironmentType",
			method:   http.MethodDelete,
			path:     environmentTypeBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteDevBoxDefinition",
			method:   http.MethodDelete,
			path:     devBoxDefinitionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteProjectCatalog",
			method:   http.MethodDelete,
			path:     projectCatalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteCatalog",
			method:   http.MethodDelete,
			path:     catalogBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteGallery",
			method:   http.MethodDelete,
			path:     galleryBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteProject",
			method:   http.MethodDelete,
			path:     projectsBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteAttachedNetworkByDevCenter",
			method:   http.MethodDelete,
			path:     attachedNetworkByDevCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteNetworkConnection",
			method:   http.MethodDelete,
			path:     networkConnectionBase + "?api-version=" + devCenterAdministratorAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteDevCenter",
			method:   http.MethodDelete,
			path:     devCenterBase + "?api-version=" + devCenterAdministratorAPIVersion,
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
