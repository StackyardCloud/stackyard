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

const dtlAPIVersion = "2018-09-15"

type dtlClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dtlSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dtlSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDtlClient(endpoint, account, subscriptionKey string) *dtlClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"dtl-2018-09-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dtlSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &dtlClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dtlClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DTL_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DTL_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-dtl")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus2")

	labName := getenv("STACKYARD_AZURE_DTL_LAB_NAME", "lab-a")
	userName := getenv("STACKYARD_AZURE_DTL_USER_NAME", "user-a")
	artifactSourceName := getenv("STACKYARD_AZURE_DTL_ARTIFACT_SOURCE_NAME", "public-repo")
	armTemplateName := getenv("STACKYARD_AZURE_DTL_ARM_TEMPLATE_NAME", "template-a")
	artifactName := getenv("STACKYARD_AZURE_DTL_ARTIFACT_NAME", "linux-install-nodejs")
	costName := getenv("STACKYARD_AZURE_DTL_COST_NAME", "lab-cost")
	customImageName := getenv("STACKYARD_AZURE_DTL_CUSTOM_IMAGE_NAME", "custom-image-a")
	formulaName := getenv("STACKYARD_AZURE_DTL_FORMULA_NAME", "formula-a")
	notificationChannelName := getenv("STACKYARD_AZURE_DTL_NOTIFICATION_CHANNEL_NAME", "channel-a")
	policySetName := getenv("STACKYARD_AZURE_DTL_POLICY_SET_NAME", "default")
	policyName := getenv("STACKYARD_AZURE_DTL_POLICY_NAME", "labVmCount")
	scheduleName := getenv("STACKYARD_AZURE_DTL_SCHEDULE_NAME", "schedule-a")
	serviceRunnerName := getenv("STACKYARD_AZURE_DTL_SERVICE_RUNNER_NAME", "runner-a")
	diskName := getenv("STACKYARD_AZURE_DTL_DISK_NAME", "disk-a")
	environmentName := getenv("STACKYARD_AZURE_DTL_ENVIRONMENT_NAME", "env-a")
	secretName := getenv("STACKYARD_AZURE_DTL_SECRET_NAME", "secret-a")
	serviceFabricName := getenv("STACKYARD_AZURE_DTL_SERVICE_FABRIC_NAME", "sf-a")
	serviceFabricScheduleName := getenv("STACKYARD_AZURE_DTL_SERVICE_FABRIC_SCHEDULE_NAME", "sf-schedule-a")
	virtualMachineName := getenv("STACKYARD_AZURE_DTL_VM_NAME", "vm-a")
	vmScheduleName := getenv("STACKYARD_AZURE_DTL_VM_SCHEDULE_NAME", "vm-schedule-a")
	virtualNetworkName := getenv("STACKYARD_AZURE_DTL_VNET_NAME", "vnet-a")
	operationName := getenv("STACKYARD_AZURE_DTL_OPERATION_NAME", "operation-a")

	client := newDtlClient(endpoint, account, subscriptionKey)

	labBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs/%s",
		subscriptionID,
		resourceGroup,
		labName,
	)
	userBase := labBase + "/users/" + userName
	artifactSourceBase := labBase + "/artifactsources/" + artifactSourceName
	armTemplateBase := artifactSourceBase + "/armtemplates/" + armTemplateName
	artifactBase := artifactSourceBase + "/artifacts/" + artifactName
	costBase := labBase + "/costs/" + costName
	customImageBase := labBase + "/customimages/" + customImageName
	formulaBase := labBase + "/formulas/" + formulaName
	notificationChannelBase := labBase + "/notificationchannels/" + notificationChannelName
	policyBase := labBase + "/policysets/" + policySetName + "/policies/" + policyName
	scheduleBase := labBase + "/schedules/" + scheduleName
	serviceRunnerBase := labBase + "/servicerunners/" + serviceRunnerName
	diskBase := userBase + "/disks/" + diskName
	environmentBase := userBase + "/environments/" + environmentName
	secretBase := userBase + "/secrets/" + secretName
	serviceFabricBase := userBase + "/servicefabrics/" + serviceFabricName
	serviceFabricScheduleBase := serviceFabricBase + "/schedules/" + serviceFabricScheduleName
	vmBase := labBase + "/virtualmachines/" + virtualMachineName
	vmScheduleBase := vmBase + "/schedules/" + vmScheduleName
	vnetBase := labBase + "/virtualnetworks/" + virtualNetworkName
	globalScheduleBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/schedules/%s",
		subscriptionID,
		resourceGroup,
		scheduleName,
	)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ProviderOperationsList",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DevTestLab/operations?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "OperationsGet",
			method: http.MethodGet,
			path: fmt.Sprintf(
				"/azure/subscriptions/%s/providers/Microsoft.DevTestLab/locations/%s/operations/%s?api-version=%s",
				subscriptionID,
				location,
				operationName,
				dtlAPIVersion,
			),
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "LabsCreateOrUpdate",
			method: http.MethodPut,
			path:   labBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"tags": map[string]any{
					"environment": "local",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "LabsGet",
			method:   http.MethodGet,
			path:     labBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "LabsUpdate",
			method: http.MethodPatch,
			path:   labBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{"environment": "local-updated"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "LabsListByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs?api-version=%s", subscriptionID, resourceGroup, dtlAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "LabsListBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevTestLab/labs?api-version=%s", subscriptionID, dtlAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "LabsClaimAnyVm",
			method:   http.MethodPost,
			path:     labBase + "/claimAnyVm?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "LabsCreateEnvironment",
			method: http.MethodPost,
			path:   labBase + "/createEnvironment?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"name":     environmentName,
				"location": location,
				"properties": map[string]any{
					"deploymentProperties": map[string]any{},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "LabsExportResourceUsage",
			method: http.MethodPost,
			path:   labBase + "/exportResourceUsage?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"storageType": "Standard",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "LabsGenerateUploadUri",
			method:   http.MethodPost,
			path:     labBase + "/generateUploadUri?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "LabsImportVirtualMachine",
			method: http.MethodPost,
			path:   labBase + "/importVirtualMachine?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"sourceVirtualMachineResourceId": fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/labs/%s/virtualmachines/%s", subscriptionID, resourceGroup, labName, virtualMachineName),
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "LabsListVhds",
			method:   http.MethodPost,
			path:     labBase + "/listVhds?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ArtifactSourcesCreateOrUpdate",
			method: http.MethodPut,
			path:   artifactSourceBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"sourceType": "VsoGit",
					"uri":        "https://github.com/Azure/azure-devtestlab.git",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "ArtifactSourcesList",
			method:   http.MethodGet,
			path:     labBase + "/artifactsources?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ArmTemplatesList",
			method:   http.MethodGet,
			path:     artifactSourceBase + "/armtemplates?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ArmTemplatesGet",
			method:   http.MethodGet,
			path:     armTemplateBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ArtifactsList",
			method:   http.MethodGet,
			path:     artifactSourceBase + "/artifacts?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ArtifactsGet",
			method:   http.MethodGet,
			path:     artifactBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ArtifactsGenerateArmTemplate",
			method:   http.MethodPost,
			path:     artifactBase + "/generateArmTemplate?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CostsCreateOrUpdate",
			method: http.MethodPut,
			path:   costBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "CostsGet",
			method:   http.MethodGet,
			path:     costBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CustomImagesCreateOrUpdate",
			method: http.MethodPut,
			path:   customImageBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"osType": "Windows",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "CustomImagesGet",
			method:   http.MethodGet,
			path:     customImageBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "FormulasCreateOrUpdate",
			method: http.MethodPut,
			path:   formulaBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"description": "sample formula",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "FormulasGet",
			method:   http.MethodGet,
			path:     formulaBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GalleryImagesList",
			method:   http.MethodGet,
			path:     labBase + "/galleryimages?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "NotificationChannelsCreateOrUpdate",
			method: http.MethodPut,
			path:   notificationChannelBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"description":        "Auto-shutdown notification",
					"emailRecipient":     "devtest@example.com",
					"notificationLocale": "en",
					"webHookUrl":         "https://example.com/hooks/devtestlabs",
					"events": []any{
						map[string]any{"eventName": "AutoShutdown"},
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "NotificationChannelsGet",
			method:   http.MethodGet,
			path:     notificationChannelBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "NotificationChannelsNotify",
			method: http.MethodPost,
			path:   notificationChannelBase + "/notify?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"eventName": "AutoShutdown",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PoliciesCreateOrUpdate",
			method: http.MethodPut,
			path:   policyBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"factName":      "LabVmCount",
					"evaluatorType": "AllowedValuesPolicy",
					"threshold":     "5",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "PoliciesGet",
			method:   http.MethodGet,
			path:     policyBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PolicySetsEvaluatePolicies",
			method: http.MethodPost,
			path:   labBase + "/policysets/" + policySetName + "/evaluatePolicies?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"factName": "LabVmCount",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SchedulesCreateOrUpdate",
			method: http.MethodPut,
			path:   scheduleBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"status":     "Enabled",
					"taskType":   "LabVmsShutdownTask",
					"timeZoneId": "Pacific Standard Time",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "SchedulesGet",
			method:   http.MethodGet,
			path:     scheduleBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SchedulesListApplicable",
			method:   http.MethodPost,
			path:     scheduleBase + "/listApplicable?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ServiceRunnersCreateOrUpdate",
			method: http.MethodPut,
			path:   serviceRunnerBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"identityRef": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/runner",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "ServiceRunnersGet",
			method:   http.MethodGet,
			path:     serviceRunnerBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UsersCreateOrUpdate",
			method: http.MethodPut,
			path:   userBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "UsersGet",
			method:   http.MethodGet,
			path:     userBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "DisksCreateOrUpdate",
			method: http.MethodPut,
			path:   diskBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"diskType":    "Standard",
					"diskSizeGiB": 32,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "DisksGet",
			method:   http.MethodGet,
			path:     diskBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "DisksAttach",
			method: http.MethodPost,
			path:   diskBase + "/attach?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"leasedByLabVmId": vmBase,
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "DisksDetach",
			method:   http.MethodPost,
			path:     diskBase + "/detach?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "EnvironmentsCreateOrUpdate",
			method: http.MethodPut,
			path:   environmentBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"deploymentProperties": map[string]any{},
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "EnvironmentsGet",
			method:   http.MethodGet,
			path:     environmentBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SecretsCreateOrUpdate",
			method: http.MethodPut,
			path:   secretBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"value": "super-secret-value",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "SecretsGet",
			method:   http.MethodGet,
			path:     secretBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ServiceFabricsCreateOrUpdate",
			method: http.MethodPut,
			path:   serviceFabricBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"environmentId": environmentBase,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "ServiceFabricsGet",
			method:   http.MethodGet,
			path:     serviceFabricBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ServiceFabricsListApplicableSchedules",
			method:   http.MethodPost,
			path:     serviceFabricBase + "/listApplicableSchedules?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ServiceFabricsStart",
			method:   http.MethodPost,
			path:     serviceFabricBase + "/start?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "ServiceFabricsStop",
			method:   http.MethodPost,
			path:     serviceFabricBase + "/stop?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "ServiceFabricSchedulesCreateOrUpdate",
			method: http.MethodPut,
			path:   serviceFabricScheduleBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"status": "Enabled",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "ServiceFabricSchedulesGet",
			method:   http.MethodGet,
			path:     serviceFabricScheduleBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ServiceFabricSchedulesExecute",
			method:   http.MethodPost,
			path:     serviceFabricScheduleBase + "/execute?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "VirtualMachinesCreateOrUpdate",
			method: http.MethodPut,
			path:   vmBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
				"properties": map[string]any{
					"size":     "Standard_B1ms",
					"userName": "devuser",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "VirtualMachinesGet",
			method:   http.MethodGet,
			path:     vmBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "VirtualMachinesAddDataDisk",
			method: http.MethodPost,
			path:   vmBase + "/addDataDisk?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"name":        diskName,
				"diskSizeGiB": 32,
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "VirtualMachinesApplyArtifacts",
			method: http.MethodPost,
			path:   vmBase + "/applyArtifacts?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"artifacts": []any{
					map[string]any{
						"artifactId": "/artifactsources/" + artifactSourceName + "/artifacts/" + artifactName,
					},
				},
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "VirtualMachinesGetRdpFileContents",
			method:   http.MethodPost,
			path:     vmBase + "/getRdpFileContents?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "VirtualMachinesListApplicableSchedules",
			method:   http.MethodPost,
			path:     vmBase + "/listApplicableSchedules?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "VirtualMachinesStart",
			method:   http.MethodPost,
			path:     vmBase + "/start?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "VirtualMachinesStop",
			method:   http.MethodPost,
			path:     vmBase + "/stop?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "VirtualMachineSchedulesCreateOrUpdate",
			method: http.MethodPut,
			path:   vmScheduleBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location":   location,
				"properties": map[string]any{"status": "Enabled"},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "VirtualMachineSchedulesGet",
			method:   http.MethodGet,
			path:     vmScheduleBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "VirtualMachineSchedulesExecute",
			method:   http.MethodPost,
			path:     vmScheduleBase + "/execute?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "VirtualNetworksCreateOrUpdate",
			method: http.MethodPut,
			path:   vnetBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location": location,
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "VirtualNetworksGet",
			method:   http.MethodGet,
			path:     vnetBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "VirtualNetworksList",
			method:   http.MethodGet,
			path:     labBase + "/virtualnetworks?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GlobalSchedulesCreateOrUpdate",
			method: http.MethodPut,
			path:   globalScheduleBase + "?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"location":   location,
				"properties": map[string]any{"status": "Enabled"},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GlobalSchedulesGet",
			method:   http.MethodGet,
			path:     globalScheduleBase + "?api-version=" + dtlAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GlobalSchedulesRetarget",
			method: http.MethodPost,
			path:   globalScheduleBase + "/retarget?api-version=" + dtlAPIVersion,
			payload: map[string]any{
				"targetResourceId": labBase,
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GlobalSchedulesListByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DevTestLab/schedules?api-version=%s", subscriptionID, resourceGroup, dtlAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GlobalSchedulesListBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DevTestLab/schedules?api-version=%s", subscriptionID, dtlAPIVersion),
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
}

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) == 0 {
		if status < 200 || status >= 300 {
			return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
		}
		return nil
	}
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func printJSON(name string, status int, payload map[string]any) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(body))
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
