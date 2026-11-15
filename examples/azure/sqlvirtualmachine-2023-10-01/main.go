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
	resourcesAPIVersion         = "2021-04-01"
	sqlVirtualMachineAPIVersion = "2023-10-01"
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
		"sqlvirtualmachine-2023-10-01",
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

func (c *armClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-sqlvm-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	vmName := getenv("STACKYARD_AZURE_SQL_VM", "stackyard-sql01")

	fmt.Printf("Stackyard SQL Server on Azure VMs (sqlvirtualmachine-2023-10-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	sqlvm := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.SqlVirtualMachine"
	vmID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/" + vmName
	groupID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.SqlVirtualMachine/sqlVirtualMachineGroups/stackyard-ag"
	q := "?api-version=" + sqlVirtualMachineAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		// The witness storage key and SQL password are accepted but never returned.
		{name: "CreateSqlVirtualMachineGroup", method: http.MethodPut, path: sqlvm + "/sqlVirtualMachineGroups/stackyard-ag" + q, payload: map[string]any{"location": region, "properties": map[string]any{"sqlImageOffer": "SQL2022-WS2022", "sqlImageSku": "Enterprise", "wsfcDomainProfile": map[string]any{"domainFqdn": "corp.example.com", "storageAccountUrl": "https://witness.blob.core.windows.net/", "storageAccountPrimaryKey": "stackyard-example"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateSqlVirtualMachine", method: http.MethodPut, path: sqlvm + "/sqlVirtualMachines/" + vmName + q, payload: map[string]any{"location": region, "properties": map[string]any{"virtualMachineResourceId": vmID, "sqlServerLicenseType": "AHUB", "sqlManagement": "Full", "sqlVirtualMachineGroupResourceId": groupID, "serverConfigurationsManagementSettings": map[string]any{"sqlConnectivityUpdateSettings": map[string]any{"connectivityType": "PRIVATE", "port": 1433, "sqlAuthUpdateUserName": "sqladmin", "sqlAuthUpdatePassword": "stackyard-example"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateListener", method: http.MethodPut, path: sqlvm + "/sqlVirtualMachineGroups/stackyard-ag/availabilityGroupListeners/stackyard-agl" + q, payload: map[string]any{"properties": map[string]any{"availabilityGroupName": "ag1", "port": 1433, "loadBalancerConfigurations": []any{map[string]any{"privateIpAddress": map[string]any{"ipAddress": "10.0.0.10"}, "probePort": 59999}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "StartAssessment", method: http.MethodPost, path: sqlvm + "/sqlVirtualMachines/" + vmName + "/startAssessment" + q, statuses: []int{http.StatusAccepted}},
		{name: "ListSqlVirtualMachines", method: http.MethodGet, path: base + "/providers/Microsoft.SqlVirtualMachine/sqlVirtualMachines" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteSqlVirtualMachine", method: http.MethodDelete, path: sqlvm + "/sqlVirtualMachines/" + vmName + q, statuses: []int{http.StatusOK}},
		{name: "DeleteSqlVirtualMachineGroup", method: http.MethodDelete, path: sqlvm + "/sqlVirtualMachineGroups/stackyard-ag" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteResourceGroup", method: http.MethodDelete, path: groupPath, statuses: []int{http.StatusOK}},
	}

	for _, call := range calls {
		resp, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", call.name, status)
		if len(resp) != 0 {
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
