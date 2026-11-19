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
	resourcesAPIVersion     = "2021-04-01"
	postgresqlhscAPIVersion = "2023-03-02-preview"
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
		"postgresqlhsc-2023-03-02-preview",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-postgresqlhsc-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	clusterName := getenv("STACKYARD_AZURE_PG_CLUSTER", "stackyard-citus")

	fmt.Printf("Stackyard Azure Cosmos DB for PostgreSQL (postgresqlhsc-2023-03-02-preview) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	cluster := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DBforPostgreSQL/serverGroupsv2/" + clusterName
	q := "?api-version=" + postgresqlhscAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckNameAvailability", method: http.MethodPost, path: base + "/providers/Microsoft.DBforPostgreSQL/checkNameAvailability" + q, payload: map[string]any{"name": clusterName, "type": "Microsoft.DBforPostgreSQL/serverGroupsv2"}, statuses: []int{http.StatusOK}},
		{name: "CreateCluster", method: http.MethodPut, path: cluster + q, payload: map[string]any{"location": region, "tags": map[string]any{"owner": "stackyard"}, "properties": map[string]any{"administratorLoginPassword": "Pa55word!", "postgresqlVersion": "16", "coordinatorServerEdition": "GeneralPurpose", "coordinatorStorageQuotaInMb": 131072, "coordinatorVCores": 4, "nodeCount": 0, "enableShardsOnCoordinator": true}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ScaleOutCluster", method: http.MethodPatch, path: cluster + q, payload: map[string]any{"properties": map[string]any{"nodeCount": 2, "nodeVCores": 4, "nodeStorageQuotaInMb": 524288, "maintenanceWindow": map[string]any{"customWindow": "Enabled", "dayOfWeek": 0, "startHour": 8, "startMinute": 0}}}, statuses: []int{http.StatusOK}},
		{name: "ListServers", method: http.MethodGet, path: cluster + "/servers" + q, statuses: []int{http.StatusOK}},
		{name: "GetCoordinatorServer", method: http.MethodGet, path: cluster + "/servers/" + clusterName + "-c" + q, statuses: []int{http.StatusOK}},
		{name: "CreateFirewallRule", method: http.MethodPut, path: cluster + "/firewallRules/office" + q, payload: map[string]any{"properties": map[string]any{"startIpAddress": "203.0.113.0", "endIpAddress": "203.0.113.255"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListFirewallRules", method: http.MethodGet, path: cluster + "/firewallRules" + q, statuses: []int{http.StatusOK}},
		{name: "CreateRole", method: http.MethodPut, path: cluster + "/roles/reporting" + q, payload: map[string]any{"properties": map[string]any{"password": "Pa55word!"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListRoles", method: http.MethodGet, path: cluster + "/roles" + q, statuses: []int{http.StatusOK}},
		{name: "UpdateCoordinatorConfiguration", method: http.MethodPut, path: cluster + "/coordinatorConfigurations/array_nulls" + q, payload: map[string]any{"properties": map[string]any{"value": "off"}}, statuses: []int{http.StatusOK, http.StatusCreated}},
		{name: "GetNodeConfiguration", method: http.MethodGet, path: cluster + "/nodeConfigurations/array_nulls" + q, statuses: []int{http.StatusOK}},
		{name: "GetClusterConfiguration", method: http.MethodGet, path: cluster + "/configurations/array_nulls" + q, statuses: []int{http.StatusOK}},
		{name: "ListServerConfigurations", method: http.MethodGet, path: cluster + "/servers/" + clusterName + "-w0/configurations" + q, statuses: []int{http.StatusOK}},
		{name: "ListPrivateLinkResources", method: http.MethodGet, path: cluster + "/privateLinkResources" + q, statuses: []int{http.StatusOK}},
		{name: "ApprovePrivateEndpointConnection", method: http.MethodPut, path: cluster + "/privateEndpointConnections/pec-1" + q, payload: map[string]any{"properties": map[string]any{"privateLinkServiceConnectionState": map[string]any{"status": "Approved", "description": "Approved by admin"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListPrivateEndpointConnections", method: http.MethodGet, path: cluster + "/privateEndpointConnections" + q, statuses: []int{http.StatusOK}},
		{name: "StopCluster", method: http.MethodPost, path: cluster + "/stop" + q, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "StartCluster", method: http.MethodPost, path: cluster + "/start" + q, statuses: []int{http.StatusOK, http.StatusAccepted}},
		{name: "RestartCluster", method: http.MethodPost, path: cluster + "/restart" + q, statuses: []int{http.StatusAccepted}},
		{name: "ListClusters", method: http.MethodGet, path: base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DBforPostgreSQL/serverGroupsv2" + q, statuses: []int{http.StatusOK}},
		{name: "DeletePrivateEndpointConnection", method: http.MethodDelete, path: cluster + "/privateEndpointConnections/pec-1" + q, statuses: []int{http.StatusNoContent, http.StatusAccepted}},
		{name: "DeleteRole", method: http.MethodDelete, path: cluster + "/roles/reporting" + q, statuses: []int{http.StatusNoContent, http.StatusAccepted}},
		{name: "DeleteFirewallRule", method: http.MethodDelete, path: cluster + "/firewallRules/office" + q, statuses: []int{http.StatusNoContent, http.StatusAccepted}},
		{name: "DeleteCluster", method: http.MethodDelete, path: cluster + q, statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent}},
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
