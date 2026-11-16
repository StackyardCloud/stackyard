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
	resourcesAPIVersion = "2021-04-01"
	networkAPIVersion = "2024-05-01"
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
		"network-2024-05-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-network-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	vnetName := getenv("STACKYARD_AZURE_VNET", "spoke")

	fmt.Printf("Stackyard Azure Virtual Network (network-2024-05-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	net := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network"
	ids := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network"
	vnet := net + "/virtualNetworks/" + vnetName
	lb := ids + "/loadBalancers/web-lb"
	q := "?api-version=" + networkAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateRouteTable", method: http.MethodPut, path: net + "/routeTables/spoke-routes" + q, payload: map[string]any{"location": region, "properties": map[string]any{"disableBgpRoutePropagation": true, "routes": []any{map[string]any{"name": "default-to-firewall", "properties": map[string]any{"addressPrefix": "0.0.0.0/0", "nextHopType": "VirtualAppliance", "nextHopIpAddress": "10.10.0.4"}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateSecurityGroup", method: http.MethodPut, path: net + "/networkSecurityGroups/web-nsg" + q, payload: map[string]any{"location": region, "properties": map[string]any{"securityRules": []any{map[string]any{"name": "allow-https", "properties": map[string]any{"protocol": "Tcp", "access": "Allow", "direction": "Inbound", "priority": 100, "sourceAddressPrefix": "Internet", "sourcePortRange": "*", "destinationAddressPrefix": "*", "destinationPortRange": "443"}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AddSecurityRule", method: http.MethodPut, path: net + "/networkSecurityGroups/web-nsg/securityRules/allow-ssh-from-hub" + q, payload: map[string]any{"properties": map[string]any{"protocol": "Tcp", "access": "Allow", "direction": "Inbound", "priority": 110, "sourceAddressPrefix": "10.10.0.0/16", "sourcePortRange": "*", "destinationAddressPrefix": "*", "destinationPortRange": "22"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		// Inline subnets become child resources the network keeps listing.
		{name: "CreateVirtualNetwork", method: http.MethodPut, path: vnet + q, payload: map[string]any{"location": region, "properties": map[string]any{"addressSpace": map[string]any{"addressPrefixes": []any{"10.20.0.0/16"}}, "subnets": []any{map[string]any{"name": "web", "properties": map[string]any{"addressPrefix": "10.20.1.0/24", "networkSecurityGroup": map[string]any{"id": ids + "/networkSecurityGroups/web-nsg"}, "routeTable": map[string]any{"id": ids + "/routeTables/spoke-routes"}}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AddSubnet", method: http.MethodPut, path: vnet + "/subnets/data" + q, payload: map[string]any{"properties": map[string]any{"addressPrefix": "10.20.2.0/24", "serviceEndpoints": []any{map[string]any{"service": "Microsoft.Storage"}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListSubnets", method: http.MethodGet, path: vnet + "/subnets" + q, statuses: []int{http.StatusOK}},
		{name: "PeerToHub", method: http.MethodPut, path: vnet + "/virtualNetworkPeerings/to-hub" + q, payload: map[string]any{"properties": map[string]any{"remoteVirtualNetwork": map[string]any{"id": ids + "/virtualNetworks/hub"}, "allowForwardedTraffic": true}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreatePublicIP", method: http.MethodPut, path: net + "/publicIPAddresses/web-ip" + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "Standard"}, "properties": map[string]any{"publicIPAllocationMethod": "Static", "dnsSettings": map[string]any{"domainNameLabel": "stackyard-web-example"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateLoadBalancer", method: http.MethodPut, path: net + "/loadBalancers/web-lb" + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "Standard"}, "properties": map[string]any{"frontendIPConfigurations": []any{map[string]any{"name": "public", "properties": map[string]any{"publicIPAddress": map[string]any{"id": ids + "/publicIPAddresses/web-ip"}}}}, "backendAddressPools": []any{map[string]any{"name": "web"}}, "probes": []any{map[string]any{"name": "https", "properties": map[string]any{"protocol": "Https", "port": 443, "requestPath": "/healthz"}}}, "loadBalancingRules": []any{map[string]any{"name": "https", "properties": map[string]any{"protocol": "Tcp", "frontendPort": 443, "backendPort": 443, "frontendIPConfiguration": map[string]any{"id": lb + "/frontendIPConfigurations/public"}, "backendAddressPool": map[string]any{"id": lb + "/backendAddressPools/web"}, "probe": map[string]any{"id": lb + "/probes/https"}}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateNetworkInterface", method: http.MethodPut, path: net + "/networkInterfaces/web-01-nic" + q, payload: map[string]any{"location": region, "properties": map[string]any{"ipConfigurations": []any{map[string]any{"name": "ipconfig1", "properties": map[string]any{"subnet": map[string]any{"id": ids + "/virtualNetworks/" + vnetName + "/subnets/web"}, "loadBalancerBackendAddressPools": []any{map[string]any{"id": lb + "/backendAddressPools/web"}}}}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "DeleteNetworkInterface", method: http.MethodDelete, path: net + "/networkInterfaces/web-01-nic" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteLoadBalancer", method: http.MethodDelete, path: net + "/loadBalancers/web-lb" + q, statuses: []int{http.StatusOK}},
		{name: "DeletePublicIP", method: http.MethodDelete, path: net + "/publicIPAddresses/web-ip" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteSubnet", method: http.MethodDelete, path: vnet + "/subnets/data" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteVirtualNetwork", method: http.MethodDelete, path: vnet + q, statuses: []int{http.StatusOK}},
		{name: "DeleteSecurityGroup", method: http.MethodDelete, path: net + "/networkSecurityGroups/web-nsg" + q, statuses: []int{http.StatusOK}},
		{name: "DeleteRouteTable", method: http.MethodDelete, path: net + "/routeTables/spoke-routes" + q, statuses: []int{http.StatusOK}},
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
