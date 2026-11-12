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

const guestConfigurationAPIVersion = "2024-04-05"

type guestConfigurationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type guestConfigurationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *guestConfigurationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newGuestConfigurationClient(endpoint, account, subscriptionKey string) *guestConfigurationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"guestconfiguration-2024-04-05",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&guestConfigurationSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey},
			},
		},
		&policy.ClientOptions{},
	)
	return &guestConfigurationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *guestConfigurationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-guestconfiguration")
	location := getenv("STACKYARD_AZURE_LOCATION", "westcentralus")

	vmName := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_VM_NAME", "myVMName")
	vmssName := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_VMSS_NAME", "myVMSSName")
	connectedVMName := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_CONNECTED_VM_NAME", "myConnectedVMName")
	machineName := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_HCRP_MACHINE_NAME", "myArcMachine")
	assignmentName := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_ASSIGNMENT_NAME", "NotInstalledApplicationForWindows")
	reportID := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_REPORT_ID", "7367cbb8-ae99-47d0-a33b-a283564d2cb1")

	assignmentVersion := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_VERSION", "1.0.0.3")
	assignmentContentURI := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_CONTENT_URI", "https://thisisfake/package")
	assignmentContentHash := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_CONTENT_HASH", "123contenthash")
	assignmentParameterName := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_PARAMETER_NAME", "[InstalledApplication]NotInstalledApplicationResource1;Name")
	assignmentParameterValue := getenv("STACKYARD_AZURE_GUESTCONFIGURATION_PARAMETER_VALUE", "NotePad,sql")

	client := newGuestConfigurationClient(endpoint, account, subscriptionKey)

	providerBase := "/azure/providers/Microsoft.GuestConfiguration"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceGroupBase := subscriptionBase + "/resourceGroups/" + resourceGroup

	vmAssignmentsBase := resourceGroupBase + "/providers/Microsoft.Compute/virtualMachines/" + vmName + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"
	vmAssignmentBase := vmAssignmentsBase + "/" + assignmentName
	vmReportsBase := vmAssignmentBase + "/reports"
	vmReportBase := vmReportsBase + "/" + reportID

	vmssAssignmentsBase := resourceGroupBase + "/providers/Microsoft.Compute/virtualMachineScaleSets/" + vmssName + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"
	vmssAssignmentBase := vmssAssignmentsBase + "/" + assignmentName
	vmssReportsBase := vmssAssignmentBase + "/reports"
	vmssReportBase := vmssReportsBase + "/" + reportID

	connectedAssignmentsBase := resourceGroupBase + "/providers/Microsoft.ConnectedVMwarevSphere/virtualmachines/" + connectedVMName + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"
	connectedAssignmentBase := connectedAssignmentsBase + "/" + assignmentName
	connectedReportsBase := connectedAssignmentBase + "/reports"
	connectedReportBase := connectedReportsBase + "/" + reportID

	hcrpAssignmentsBase := resourceGroupBase + "/providers/Microsoft.HybridCompute/machines/" + machineName + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"
	hcrpAssignmentBase := hcrpAssignmentsBase + "/" + assignmentName
	hcrpReportsBase := hcrpAssignmentBase + "/reports"
	hcrpReportBase := hcrpReportsBase + "/" + reportID

	rgAssignmentsBase := resourceGroupBase + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"
	subscriptionAssignmentsBase := subscriptionBase + "/providers/Microsoft.GuestConfiguration/guestConfigurationAssignments"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "OperationsList",
			method:   http.MethodGet,
			path:     providerBase + "/operations?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentsCreateOrUpdateVM",
			method:   http.MethodPut,
			path:     vmAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			payload:  assignmentPayload(assignmentName, location, assignmentVersion, assignmentContentURI, assignmentContentHash, assignmentParameterName, assignmentParameterValue),
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GuestConfigurationAssignmentsGetVM",
			method:   http.MethodGet,
			path:     vmAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentsListVM",
			method:   http.MethodGet,
			path:     vmAssignmentsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentReportsListVM",
			method:   http.MethodGet,
			path:     vmReportsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentReportsGetVM",
			method:   http.MethodGet,
			path:     vmReportBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentsCreateOrUpdateVMSS",
			method:   http.MethodPut,
			path:     vmssAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			payload:  assignmentPayload(assignmentName, location, assignmentVersion, assignmentContentURI, assignmentContentHash, assignmentParameterName, assignmentParameterValue),
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GuestConfigurationAssignmentsGetVMSS",
			method:   http.MethodGet,
			path:     vmssAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentsListVMSS",
			method:   http.MethodGet,
			path:     vmssAssignmentsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentReportsVMSSList",
			method:   http.MethodGet,
			path:     vmssReportsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentReportsVMSSGet",
			method:   http.MethodGet,
			path:     vmssReportBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationConnectedVMwarevSphereAssignmentsCreateOrUpdate",
			method:   http.MethodPut,
			path:     connectedAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			payload:  assignmentPayload(assignmentName, location, assignmentVersion, assignmentContentURI, assignmentContentHash, assignmentParameterName, assignmentParameterValue),
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GuestConfigurationConnectedVMwarevSphereAssignmentsGet",
			method:   http.MethodGet,
			path:     connectedAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationConnectedVMwarevSphereAssignmentsList",
			method:   http.MethodGet,
			path:     connectedAssignmentsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationConnectedVMwarevSphereAssignmentsReportsList",
			method:   http.MethodGet,
			path:     connectedReportsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationConnectedVMwarevSphereAssignmentsReportsGet",
			method:   http.MethodGet,
			path:     connectedReportBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationHCRPAssignmentsCreateOrUpdate",
			method:   http.MethodPut,
			path:     hcrpAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			payload:  assignmentPayload(assignmentName, location, assignmentVersion, assignmentContentURI, assignmentContentHash, assignmentParameterName, assignmentParameterValue),
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GuestConfigurationHCRPAssignmentsGet",
			method:   http.MethodGet,
			path:     hcrpAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationHCRPAssignmentsList",
			method:   http.MethodGet,
			path:     hcrpAssignmentsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationHCRPAssignmentReportsList",
			method:   http.MethodGet,
			path:     hcrpReportsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationHCRPAssignmentReportsGet",
			method:   http.MethodGet,
			path:     hcrpReportBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentsRGList",
			method:   http.MethodGet,
			path:     rgAssignmentsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "GuestConfigurationAssignmentsSubscriptionList",
			method:   http.MethodGet,
			path:     subscriptionAssignmentsBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "GuestConfigurationAssignmentsDeleteVM",
			method:   http.MethodDelete,
			path:     vmAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GuestConfigurationAssignmentsDeleteVMSS",
			method:   http.MethodDelete,
			path:     vmssAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "GuestConfigurationConnectedVMwarevSphereAssignmentsDelete",
			method:   http.MethodDelete,
			path:     connectedAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "GuestConfigurationHCRPAssignmentsDelete",
			method:   http.MethodDelete,
			path:     hcrpAssignmentBase + "?api-version=" + guestConfigurationAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		response, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, response)
	}
}

func assignmentPayload(name, location, version, contentURI, contentHash, parameterName, parameterValue string) map[string]any {
	return map[string]any{
		"name":     name,
		"location": location,
		"properties": map[string]any{
			"context": "Azure policy",
			"guestConfiguration": map[string]any{
				"name":           name,
				"assignmentType": "ApplyAndAutoCorrect",
				"configurationParameter": []map[string]any{
					{
						"name":  parameterName,
						"value": parameterValue,
					},
				},
				"contentHash": contentHash,
				"contentUri":  contentURI,
				"version":     version,
			},
		},
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
