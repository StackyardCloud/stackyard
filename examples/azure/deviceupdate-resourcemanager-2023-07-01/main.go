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

const deviceUpdateResourceManagerAPIVersion = "2023-07-01"

type deviceUpdateResourceManagerClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type deviceUpdateResourceManagerSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type deviceUpdateResourceManagerCall struct {
	name     string
	method   string
	path     string
	payload  any
	statuses []int
}

func (p *deviceUpdateResourceManagerSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDeviceUpdateResourceManagerClient(endpoint, account, subscriptionKey string) *deviceUpdateResourceManagerClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"deviceupdate-resourcemanager-2023-07-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&deviceUpdateResourceManagerSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &deviceUpdateResourceManagerClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *deviceUpdateResourceManagerClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-deviceupdate")
	location := getenv("STACKYARD_AZURE_LOCATION", "westus2")
	accountName := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_ACCOUNT_NAME", "contoso")
	instanceName := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_INSTANCE_NAME", "blue")
	privateEndpointConnectionProxyID := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_PRIVATE_ENDPOINT_CONNECTION_PROXY_ID", "peproxy01")
	privateEndpointConnectionName := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_PRIVATE_ENDPOINT_CONNECTION_NAME", "peconnection01")
	privateLinkGroupID := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_PRIVATE_LINK_GROUP_ID", "DeviceUpdate")
	ioTHubName := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_IOTHUB_NAME", "blue-contoso-hub")
	privateEndpointName := getenv("STACKYARD_AZURE_DEVICEUPDATE_RM_PRIVATE_ENDPOINT_NAME", "pe-deviceupdate")

	client := newDeviceUpdateResourceManagerClient(endpoint, account, subscriptionKey)

	accountBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DeviceUpdate/accounts/%s",
		subscriptionID,
		resourceGroup,
		accountName,
	)
	instanceBase := accountBase + "/instances/" + instanceName
	privateEndpointConnectionProxyBase := accountBase + "/privateEndpointConnectionProxies/" + privateEndpointConnectionProxyID
	privateEndpointConnectionBase := accountBase + "/privateEndpointConnections/" + privateEndpointConnectionName
	privateLinkResourceBase := accountBase + "/privateLinkResources/" + privateLinkGroupID

	remotePrivateEndpointID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/%s",
		subscriptionID,
		resourceGroup,
		privateEndpointName,
	)
	iotHubResourceID := fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Devices/IotHubs/%s",
		subscriptionID,
		resourceGroup,
		ioTHubName,
	)

	checkNameAvailabilityPayload := map[string]any{
		"name": accountName,
		"type": "Microsoft.DeviceUpdate/accounts",
	}

	accountCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
			"service":     "deviceupdate-resourcemanager",
		},
		"properties": map[string]any{
			"publicNetworkAccess": "Enabled",
		},
	}

	accountUpdatePayload := map[string]any{
		"tags": map[string]any{
			"environment": "local-updated",
			"service":     "deviceupdate-resourcemanager",
		},
	}

	instanceCreatePayload := map[string]any{
		"location": location,
		"tags": map[string]any{
			"environment": "local",
			"service":     "deviceupdate-instance",
		},
		"properties": map[string]any{
			"enableDiagnostics": false,
			"iotHubs": []any{
				map[string]any{
					"resourceId": iotHubResourceID,
				},
			},
		},
	}

	instanceUpdatePayload := map[string]any{
		"tags": map[string]any{
			"environment": "local-updated",
			"service":     "deviceupdate-instance",
		},
	}

	privateEndpointConnectionProxyPayload := map[string]any{
		"remotePrivateEndpoint": map[string]any{
			"id":                      remotePrivateEndpointID,
			"location":                location,
			"immutableSubscriptionId": subscriptionID,
			"immutableResourceId":     remotePrivateEndpointID,
			"manualPrivateLinkServiceConnections": []any{
				map[string]any{
					"name":           privateEndpointConnectionProxyID,
					"groupIds":       []any{"DeviceUpdate"},
					"requestMessage": "Please approve my connection, thanks.",
				},
			},
			"privateLinkServiceProxies": []any{
				map[string]any{
					"id":                           remotePrivateEndpointID + "/privateLinkServiceProxies/" + privateEndpointConnectionProxyID,
					"groupConnectivityInformation": []any{},
				},
			},
		},
		"status": "Approved",
	}

	privateEndpointUpdatePayload := map[string]any{
		"id":                      remotePrivateEndpointID,
		"immutableResourceId":     remotePrivateEndpointID,
		"immutableSubscriptionId": subscriptionID,
		"location":                location,
		"vnetTrafficTag":          "12345678",
	}

	privateEndpointConnectionPayload := map[string]any{
		"properties": map[string]any{
			"privateLinkServiceConnectionState": map[string]any{
				"status":      "Approved",
				"description": "Auto-Approved",
			},
			"groupIds": []any{"DeviceUpdate"},
			"privateEndpoint": map[string]any{
				"id": remotePrivateEndpointID,
			},
		},
	}

	calls := []deviceUpdateResourceManagerCall{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DeviceUpdate/operations?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckNameAvailability",
			method:   http.MethodPost,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DeviceUpdate/checknameavailability?api-version=%s", subscriptionID, deviceUpdateResourceManagerAPIVersion),
			payload:  checkNameAvailabilityPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdateAccount",
			method:   http.MethodPut,
			path:     accountBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  accountCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "HeadAccount",
			method:   http.MethodHead,
			path:     accountBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccountsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DeviceUpdate/accounts?api-version=%s", subscriptionID, resourceGroup, deviceUpdateResourceManagerAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccountsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DeviceUpdate/accounts?api-version=%s", subscriptionID, deviceUpdateResourceManagerAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateAccount",
			method:   http.MethodPatch,
			path:     accountBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  accountUpdatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "CreateOrUpdateInstance",
			method:   http.MethodPut,
			path:     instanceBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  instanceCreatePayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "HeadInstance",
			method:   http.MethodHead,
			path:     instanceBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetInstance",
			method:   http.MethodGet,
			path:     instanceBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListInstancesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/instances?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdateInstance",
			method:   http.MethodPatch,
			path:     instanceBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  instanceUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdatePrivateEndpointConnectionProxy",
			method:   http.MethodPut,
			path:     privateEndpointConnectionProxyBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  privateEndpointConnectionProxyPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetPrivateEndpointConnectionProxy",
			method:   http.MethodGet,
			path:     privateEndpointConnectionProxyBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnectionProxiesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/privateEndpointConnectionProxies?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ValidatePrivateEndpointConnectionProxy",
			method:   http.MethodPost,
			path:     privateEndpointConnectionProxyBase + "/validate?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  privateEndpointConnectionProxyPayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "UpdatePrivateEndpointProperties",
			method:   http.MethodPost,
			path:     privateEndpointConnectionProxyBase + "/updatePrivateEndpointProperties?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  privateEndpointUpdatePayload,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CreateOrUpdatePrivateEndpointConnection",
			method:   http.MethodPut,
			path:     privateEndpointConnectionBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			payload:  privateEndpointConnectionPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetPrivateEndpointConnection",
			method:   http.MethodGet,
			path:     privateEndpointConnectionBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateEndpointConnectionsByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/privateEndpointConnections?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListPrivateLinkResourcesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/privateLinkResources?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetPrivateLinkResource",
			method:   http.MethodGet,
			path:     privateLinkResourceBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeletePrivateEndpointConnection",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeletePrivateEndpointConnectionProxy",
			method:   http.MethodDelete,
			path:     privateEndpointConnectionProxyBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteInstance",
			method:   http.MethodDelete,
			path:     instanceBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteAccount",
			method:   http.MethodDelete,
			path:     accountBase + "?api-version=" + deviceUpdateResourceManagerAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
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
