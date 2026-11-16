package main

import (
	"context"
	"encoding/base64"
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
	nginxAPIVersion     = "2025-11-01"
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
		"nginx-2025-11-01",
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
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-nginx-example")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	deploymentName := getenv("STACKYARD_AZURE_NGINX_DEPLOYMENT", "edge")

	fmt.Printf("Stackyard NGINXaaS for Azure (nginx-2025-11-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newARMClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	groupPath := base + "/resourcegroups/" + resourceGroup + "?api-version=" + resourcesAPIVersion
	ids := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers"
	deployment := base + "/resourceGroups/" + resourceGroup + "/providers/Nginx.NginxPlus/nginxDeployments/" + deploymentName
	subnet := ids + "/Microsoft.Network/virtualNetworks/edge/subnets/nginx"
	conf := base64.StdEncoding.EncodeToString([]byte("http {\n  server {\n    listen 80;\n    location / { return 200 \"ok\"; }\n  }\n}\n"))
	q := "?api-version=" + nginxAPIVersion

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{name: "CreateResourceGroup", method: http.MethodPut, path: groupPath, payload: map[string]any{"location": region}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "CreateDeployment", method: http.MethodPut, path: deployment + q, payload: map[string]any{"location": region, "sku": map[string]any{"name": "standardv3_Monthly"}, "identity": map[string]any{"type": "SystemAssigned"}, "properties": map[string]any{"networkProfile": map[string]any{"frontEndIPConfiguration": map[string]any{"privateIPAddresses": []any{map[string]any{"privateIPAddress": "10.0.1.10", "privateIPAllocationMethod": "Static", "subnetId": subnet}}}, "networkInterfaceConfiguration": map[string]any{"subnetId": subnet}}, "scalingProperties": map[string]any{"capacity": 10}, "autoUpgradeProfile": map[string]any{"upgradeChannel": "stable"}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "UploadConfiguration", method: http.MethodPut, path: deployment + "/configurations/default" + q, payload: map[string]any{"properties": map[string]any{"rootFile": "/etc/nginx/nginx.conf", "files": []any{map[string]any{"virtualPath": "/etc/nginx/nginx.conf", "content": conf}}}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "AnalyzeConfiguration", method: http.MethodPost, path: deployment + "/configurations/default/analyze" + q, payload: map[string]any{"config": map[string]any{"rootFile": "/etc/nginx/nginx.conf", "files": []any{map[string]any{"virtualPath": "/etc/nginx/nginx.conf", "content": conf}}}}, statuses: []int{http.StatusOK}},
		{name: "AddCertificate", method: http.MethodPut, path: deployment + "/certificates/site" + q, payload: map[string]any{"properties": map[string]any{"keyVirtualPath": "/etc/nginx/ssl/site.key", "certificateVirtualPath": "/etc/nginx/ssl/site.crt", "keyVaultSecretId": "https://contoso.vault.azure.net/secrets/site"}}, statuses: []int{http.StatusCreated, http.StatusOK}},
		{name: "ListDefaultWafPolicies", method: http.MethodPost, path: deployment + "/listDefaultWafPolicies" + q, statuses: []int{http.StatusOK}},
		{name: "ScaleDeployment", method: http.MethodPatch, path: deployment + q, payload: map[string]any{"properties": map[string]any{"scalingProperties": map[string]any{"capacity": 20}}}, statuses: []int{http.StatusOK}},
		{name: "DeleteDeployment", method: http.MethodDelete, path: deployment + q, statuses: []int{http.StatusOK}},
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
