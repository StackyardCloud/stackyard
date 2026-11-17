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
	peeringAPIVersion   = "2025-05-01"
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
		"peering-2025-05-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-peering-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")

	fmt.Printf("Stackyard Azure Peering (peering-2025-05-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	asn := base + "/providers/Microsoft.Peering/peerAsns/contoso"
	peering := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Peering/peerings/seattle-direct"
	service := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Peering/peeringServices/branch-office"
	q := "?api-version=" + peeringAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListDirectLocations", method: http.MethodGet, path: base + "/providers/Microsoft.Peering/peeringLocations" + q + "&kind=Direct", statuses: []int{http.StatusOK}},
		{name: "AssociatePeerAsn", method: http.MethodPut, path: asn + q, payload: map[string]any{"properties": map[string]any{
			"peerAsn": 65000, "peerName": "Contoso",
			"peerContactDetail": []any{map[string]any{"role": "Noc", "email": "noc@contoso.com", "phone": "+1 (234) 567-8999"}},
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDirectPeering", method: http.MethodPut, path: peering + q, payload: map[string]any{"kind": "Direct", "location": region, "sku": map[string]any{"name": "Basic_Direct_Free"}, "properties": map[string]any{
			"peeringLocation": "Seattle",
			"direct": map[string]any{"directPeeringType": "Edge", "peerAsn": map[string]any{"id": strings.TrimPrefix(asn, "/azure")}, "connections": []any{map[string]any{
				"peeringDBFacilityId": 71, "bandwidthInMbps": 10000, "sessionAddressProvider": "Peer",
				"bgpSession": map[string]any{"sessionPrefixV4": "10.21.31.0/31", "sessionPrefixV6": "fe01::3e:0/127", "maxPrefixesAdvertisedV4": 1000, "maxPrefixesAdvertisedV6": 100},
			}}},
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "TagPeering", method: http.MethodPatch, path: peering + q, payload: map[string]any{"tags": map[string]any{"env": "example"}}, statuses: []int{http.StatusOK}},
		{name: "RegisterPrefix", method: http.MethodPut, path: peering + "/registeredPrefixes/customer1" + q, payload: map[string]any{"properties": map[string]any{"prefix": "192.168.1.0/24"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CheckProviderAvailability", method: http.MethodPost, path: base + "/providers/Microsoft.Peering/checkServiceProviderAvailability" + q, payload: map[string]any{"peeringServiceLocation": "Washington", "peeringServiceProvider": "Contoso Networks"}, statuses: []int{http.StatusOK}},
		{name: "CreatePeeringService", method: http.MethodPut, path: service + q, payload: map[string]any{"location": region, "properties": map[string]any{
			"peeringServiceLocation": "Washington", "peeringServiceProvider": "Contoso Networks", "providerPrimaryPeeringLocation": "Seattle",
		}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AddServicePrefix", method: http.MethodPut, path: service + "/prefixes/office" + q, payload: map[string]any{"properties": map[string]any{"prefix": "192.168.2.0/24"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "DeletePeeringService", method: http.MethodDelete, path: service + q, statuses: []int{http.StatusOK}},
		{name: "DeletePeering", method: http.MethodDelete, path: peering + q, statuses: []int{http.StatusOK}},
		{name: "DeletePeerAsn", method: http.MethodDelete, path: asn + q, statuses: []int{http.StatusOK}},
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
