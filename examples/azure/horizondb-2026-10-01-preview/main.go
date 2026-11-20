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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const (
	resourcesAPIVersion = "2021-04-01"
	horizonDBAPIVersion = "2026-10-01-preview"
)

type armClient struct {
	endpoint string
	pipeline runtime.Pipeline
	// ifMatch, when set, is sent as the If-Match header.
	ifMatch string
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
		"horizondb-2026-10-01-preview",
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
	if c.ifMatch != "" {
		req.Raw().Header.Set("If-Match", c.ifMatch)
	}
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-horizondb-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	clusterName := getenv("STACKYARD_AZURE_HORIZONDB_CLUSTER", "stackyard-hzdb")
	poolName := getenv("STACKYARD_AZURE_HORIZONDB_POOL", "primary")

	fmt.Printf("Stackyard Azure HorizonDB (horizondb-2026-10-01-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	providers := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.HorizonDb/"
	armProviders := strings.TrimPrefix(providers, "/azure")
	location := base + "/providers/Microsoft.HorizonDb/locations/" + region + "/"
	cluster := providers + "clusters/" + clusterName
	pool := cluster + "/pools/" + poolName
	group := providers + "parameterGroups/stackyard-params"
	q := "?api-version=" + horizonDBAPIVersion

	run := func(name, method, path string, payload any, statuses ...int) map[string]any {
		resp, status, err := client.doJSON(ctx, method, path, payload, statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> HTTP %d\n", name, status)
		if len(resp) != 0 {
			encoded, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Println(string(encoded))
		}
		return resp
	}
	created := []int{http.StatusCreated, http.StatusOK}
	deleted := []int{http.StatusOK, http.StatusNoContent}

	run("CreateResourceGroup", http.MethodPut, groupPath, map[string]any{"location": region}, created...)
	run("ListOperations", http.MethodGet, "/azure/providers/Microsoft.HorizonDb/operations"+q, nil, http.StatusOK)
	run("GetCapabilities", http.MethodGet, location+"capabilities/default"+q, nil, http.StatusOK)
	run("ListDefaultParameterGroups", http.MethodGet, location+"parameterGroups"+q, nil, http.StatusOK)
	run("CreateCluster", http.MethodPut, cluster+q, map[string]any{"location": region, "tags": map[string]any{"env": "dev"}, "properties": map[string]any{
		"postgreSqlVersion": "17", "network": map[string]any{"publicAccess": "Enabled"}, "backup": map[string]any{"geoRedundancy": "Disabled", "retentionDays": 14},
	}}, created...)
	run("UpdateCluster", http.MethodPatch, cluster+q, map[string]any{"properties": map[string]any{"aiModelManagement": "Enabled"}}, http.StatusOK)
	run("EnablePasswordAuthentication", http.MethodPut, cluster+"/authentications/passwordMethod"+q, map[string]any{"properties": map[string]any{
		"state": "Enabled", "administratorLogin": "hzadmin", "administratorLoginPassword": "Stackyard-Admin-2026",
	}}, created...)
	run("EnableMicrosoftEntraAuthentication", http.MethodPatch, cluster+"/authentications/microsoftEntra"+q, map[string]any{"properties": map[string]any{"state": "Enabled"}}, http.StatusOK)
	run("ListAuthentications", http.MethodGet, cluster+"/authentications"+q, nil, http.StatusOK)
	run("AddEntraAdministrator", http.MethodPut, cluster+"/microsoftEntraAdministrators/11111111-1111-1111-1111-111111111111"+q, map[string]any{"properties": map[string]any{
		"principalName": "Database Administrators", "principalType": "Group",
	}}, created...)
	run("CreatePool", http.MethodPut, pool+q, map[string]any{"properties": map[string]any{
		"computeModel": map[string]any{"type": "Provisioned", "computeProcessor": "AMD", "computeGeneration": 3, "vCores": 4},
		"nodeCount":    2, "highAvailability": map[string]any{"mode": "AnyZone"},
	}}, created...)
	run("GetCluster", http.MethodGet, cluster+q, nil, http.StatusOK)
	run("ListEndpoints", http.MethodGet, cluster+"/endpoints"+q, nil, http.StatusOK)
	run("ListNodes", http.MethodGet, pool+"/nodes"+q, nil, http.StatusOK)
	run("CreateFirewallRule", http.MethodPut, pool+"/firewallRules/office"+q, map[string]any{"properties": map[string]any{
		"startIpAddress": "203.0.113.1", "endIpAddress": "203.0.113.20", "description": "office network",
	}}, created...)
	run("ApprovePrivateEndpointConnection", http.MethodPut, cluster+"/privateEndpointConnections/pe-app"+q, map[string]any{"properties": map[string]any{
		"privateLinkServiceConnectionState": map[string]any{"status": "Approved", "description": "approved for the app network"},
	}}, created...)
	run("ListPrivateLinkResources", http.MethodGet, cluster+"/privateLinkResources"+q, nil, http.StatusOK)
	run("CreateParameterGroup", http.MethodPut, group+q, map[string]any{"location": region, "properties": map[string]any{
		"postgreSqlVersion": "17", "description": "tuned for the example",
		"parameters": []any{map[string]any{"name": "work_mem", "value": "8192"}, map[string]any{"name": "application_name", "value": "stackyard"}},
	}}, created...)
	run("UsePoolParameterGroup", http.MethodPatch, pool+q, map[string]any{"properties": map[string]any{
		"parameterGroup": map[string]any{"parameterGroupResourceId": armProviders + "parameterGroups/stackyard-params", "applyMethod": "PendingReboot"},
	}}, http.StatusOK)
	run("ListParameterGroupConnections", http.MethodGet, group+"/connections"+q, nil, http.StatusOK)
	run("RestartPool", http.MethodPost, pool+"/restart"+q, nil, http.StatusOK)
	run("Failover", http.MethodPost, cluster+"/failover"+q, nil, http.StatusOK)
	events := run("ListMaintenanceEvents", http.MethodGet, pool+"/maintenanceEvents"+q+"&$filter="+url.QueryEscape("properties/status eq 'Planned'"), nil, http.StatusOK)
	if items, _ := events["value"].([]any); len(items) > 0 {
		event, _ := items[0].(map[string]any)
		props, _ := event["properties"].(map[string]any)
		start, _ := time.Parse(time.RFC3339, fmt.Sprint(props["startTimeUtc"]))
		eventPath := pool + "/maintenanceEvents/" + fmt.Sprint(event["name"])
		run("RescheduleMaintenance", http.MethodPost, eventPath+"/reschedule"+q, map[string]any{"postponeToDateTimeUtc": start.Add(24 * time.Hour).Format(time.RFC3339)}, http.StatusOK)
	}
	logs := run("ListLogFiles", http.MethodGet, cluster+"/logFiles"+q, nil, http.StatusOK)
	if items, _ := logs["value"].([]any); len(items) > 0 {
		file, _ := items[0].(map[string]any)
		run("ListLogDownloadUrls", http.MethodPost, cluster+"/logFiles/"+fmt.Sprint(file["name"])+"/listDownloadUrls"+q, nil, http.StatusOK)
	}
	run("UpdateLogCapture", http.MethodPatch, cluster+"/logCaptureConfigurations/PostgreSqlServer"+q, map[string]any{"properties": map[string]any{"retentionDays": 14}}, http.StatusOK)
	// Change data capture: Fabric Mirroring for one database.
	identity := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/hzdb-cdc"
	destination := cluster + "/changeDataCaptureDestinations/fabric"
	run("AttachClusterIdentity", http.MethodPatch, cluster+q, map[string]any{"identity": map[string]any{
		"type": "UserAssigned", "userAssignedIdentities": map[string]any{identity: map[string]any{}},
	}}, http.StatusOK)
	run("EnableChangeDataCapture", http.MethodPut, cluster+"/changeDataCapture/default"+q, map[string]any{"properties": map[string]any{"desiredState": "Enabled", "autoUpgradeExtension": true}}, created...)
	run("CreateCDCDestination", http.MethodPut, destination+q, map[string]any{"properties": map[string]any{
		"desiredState": "Enabled", "destination": map[string]any{"destinationType": "Fabric", "protocol": "Native", "workspaceId": "stackyard-workspace"},
	}}, created...)
	run("AssignCDCIdentity", http.MethodPut, destination+"/identityAssignments/primary"+q, map[string]any{"properties": map[string]any{"identityResourceId": identity}}, created...)
	current := run("GetCDCDestination", http.MethodGet, destination+q, nil, http.StatusOK)
	client.ifMatch = fmt.Sprint(current["etag"])
	run("ActivateCDCIdentity", http.MethodPost, destination+"/activateIdentity"+q, map[string]any{
		"identityAssignmentResourceId": strings.TrimPrefix(destination, "/azure") + "/identityAssignments/primary",
	}, http.StatusOK)
	client.ifMatch = ""
	run("CreateCDCConfiguration", http.MethodPut, cluster+"/changeDataCaptureConfigurations/app"+q, map[string]any{"properties": map[string]any{
		"desiredState": "Enabled", "databaseName": "app", "usageProfile": "Mirroring", "destinationResourceId": strings.TrimPrefix(destination, "/azure"),
	}}, created...)
	run("DisableCDCConfiguration", http.MethodPatch, cluster+"/changeDataCaptureConfigurations/app"+q, map[string]any{"properties": map[string]any{"desiredState": "Disabled"}}, http.StatusOK)
	run("DeleteCDCConfiguration", http.MethodDelete, cluster+"/changeDataCaptureConfigurations/app"+q, nil, deleted...)
	run("StopPool", http.MethodPost, pool+"/stop"+q, nil, http.StatusOK)
	run("StartPool", http.MethodPost, pool+"/start"+q, nil, http.StatusOK)
	run("DeleteFirewallRule", http.MethodDelete, pool+"/firewallRules/office"+q, nil, deleted...)
	run("DeletePool", http.MethodDelete, pool+q, nil, deleted...)
	run("DeleteParameterGroup", http.MethodDelete, group+q, nil, deleted...)
	run("DeleteCluster", http.MethodDelete, cluster+q, nil, deleted...)
	run("DeleteResourceGroup", http.MethodDelete, groupPath, nil, http.StatusOK)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
