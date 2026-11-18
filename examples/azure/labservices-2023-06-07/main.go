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

const (
	resourcesAPIVersion        = "2021-04-01"
	labServicesAPIVersion = "2023-06-07"
)

type armClient struct {
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

func newARMClient(endpoint, account, subscriptionKey string) *armClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"labservices-2023-06-07",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &armClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (any, int, error) {
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
	if len(expectedStatuses) > 0 {
		ok := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				ok = true
				break
			}
		}
		if !ok {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if strings.TrimSpace(string(body)) == "" {
		return nil, resp.StatusCode, nil
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-labs-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	labName := getenv("STACKYARD_AZURE_LAB_NAME", "stackyard-lab")

	fmt.Printf("Stackyard Azure Lab Services (labservices-2023-06-07) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	ns := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LabServices"
	plan := ns + "/labPlans/" + labName + "-plan"
	lab := ns + "/labs/" + labName
	planID := strings.TrimPrefix(plan, "/azure")
	q := "?api-version=" + labServicesAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListSkus", method: http.MethodGet, path: base + "/providers/Microsoft.LabServices/skus" + q, statuses: []int{http.StatusOK}},
		{name: "ListUsages", method: http.MethodGet, path: base + "/providers/Microsoft.LabServices/locations/" + region + "/usages" + q, statuses: []int{http.StatusOK}},
		{name: "CreateLabPlan", method: http.MethodPut, path: plan + q, payload: map[string]any{"location": region, "properties": map[string]any{"allowedRegions": []any{region}, "defaultConnectionProfile": map[string]any{"clientRdpAccess": "Public"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListImages", method: http.MethodGet, path: plan + "/images" + q, statuses: []int{http.StatusOK}},
		{name: "EnableImage", method: http.MethodPatch, path: plan + "/images/microsoftwindowsserver.windowsserver.2022-datacenter-g2" + q, payload: map[string]any{"properties": map[string]any{"enabledState": "Enabled"}}, statuses: []int{http.StatusOK}},
		{name: "CreateLab", method: http.MethodPut, path: lab + q, payload: map[string]any{"location": region, "properties": map[string]any{"labPlanId": planID, "title": "Stackyard lab", "autoShutdownProfile": map[string]any{"shutdownOnDisconnect": "Enabled", "disconnectDelay": "PT15M"}, "connectionProfile": map[string]any{"clientRdpAccess": "Public"}, "securityProfile": map[string]any{"openAccess": "Disabled"}, "virtualMachineProfile": map[string]any{"createOption": "Image", "usageQuota": "PT10H", "imageReference": map[string]any{"publisher": "MicrosoftWindowsServer", "offer": "WindowsServer", "sku": "2022-datacenter-g2"}, "sku": map[string]any{"name": "Standard_Fv2_2_4GB_64_S_SSD", "capacity": 1}, "adminUser": map[string]any{"username": "labadmin"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateUser", method: http.MethodPut, path: lab + "/users/student1" + q, payload: map[string]any{"properties": map[string]any{"email": "student1@contoso.com"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "InviteUser", method: http.MethodPost, path: lab + "/users/student1/invite" + q, payload: map[string]any{"text": "Welcome to the lab"}, statuses: []int{http.StatusOK}},
		{name: "CreateSchedule", method: http.MethodPut, path: lab + "/schedules/weekdays" + q, payload: map[string]any{"properties": map[string]any{"startAt": "2026-12-01T09:00:00Z", "stopAt": "2026-12-01T17:00:00Z", "timeZoneId": "America/Los_Angeles", "recurrencePattern": map[string]any{"frequency": "Weekly", "weekDays": []any{"Monday", "Wednesday"}, "expirationDate": "2027-01-31T23:59:59Z"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "PublishLab", method: http.MethodPost, path: lab + "/publish" + q, statuses: []int{http.StatusOK}},
		{name: "ListVirtualMachines", method: http.MethodGet, path: lab + "/virtualMachines" + q, statuses: []int{http.StatusOK}},
		{name: "StartVirtualMachine", method: http.MethodPost, path: lab + "/virtualMachines/0/start" + q, statuses: []int{http.StatusOK}},
		{name: "StopVirtualMachine", method: http.MethodPost, path: lab + "/virtualMachines/0/stop" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteLab", method: http.MethodDelete, path: lab + q, statuses: []int{http.StatusOK}},
		{name: "DeleteLabPlan", method: http.MethodDelete, path: plan + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if resp != nil {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
