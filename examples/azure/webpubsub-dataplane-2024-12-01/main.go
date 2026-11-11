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

const webPubSubDataPlaneAPIVersion = "2024-12-01"

type webPubSubDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type webPubSubDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *webPubSubDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newWebPubSubDataPlaneClient(endpoint, account, subscriptionKey string) *webPubSubDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"webpubsub-dataplane-2024-12-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&webPubSubDataPlaneSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &webPubSubDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *webPubSubDataPlaneClient) do(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, readErr)
	}
	if err := ensureWebPubSubStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
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

func ensureWebPubSubStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) > 0 {
		for _, expected := range expectedStatuses {
			if status == expected {
				return nil
			}
		}
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_WEBPUBSUB_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_WEBPUBSUB_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	hub := getenv("STACKYARD_AZURE_WEBPUBSUB_DATA_PLANE_HUB", "chat")
	connectionID := getenv("STACKYARD_AZURE_WEBPUBSUB_DATA_PLANE_CONNECTION_ID", "conn-a")
	userID := getenv("STACKYARD_AZURE_WEBPUBSUB_DATA_PLANE_USER_ID", "user-a")

	fmt.Printf("Stackyard Azure Web PubSub Data Plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newWebPubSubDataPlaneClient(endpoint, account, subscriptionKey)
	base := "/azure/webpubsub-data-plane/2024-12-01/api/hubs/" + hub

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "Health",
			method:   http.MethodHead,
			path:     "/azure/webpubsub-data-plane/2024-12-01/api/health?api-version=" + webPubSubDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GenerateToken",
			method:   http.MethodPost,
			path:     base + "/:generateToken?api-version=" + webPubSubDataPlaneAPIVersion + "&userId=" + userID + "&roles=webpubsub.joinLeaveGroup&groups=group-a&minutesToExpire=30",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AddConnectionToGroup",
			method:   http.MethodPut,
			path:     base + "/groups/group-a/connections/" + connectionID + "?api-version=" + webPubSubDataPlaneAPIVersion + "&userId=" + userID,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "AddUserToGroup",
			method:   http.MethodPut,
			path:     base + "/users/" + userID + "/groups/group-a?api-version=" + webPubSubDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GrantPermission",
			method:   http.MethodPut,
			path:     base + "/permissions/sendToGroup/connections/" + connectionID + "?api-version=" + webPubSubDataPlaneAPIVersion + "&targetName=group-a",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckPermission",
			method:   http.MethodHead,
			path:     base + "/permissions/sendToGroup/connections/" + connectionID + "?api-version=" + webPubSubDataPlaneAPIVersion + "&targetName=group-a",
			statuses: []int{http.StatusOK},
		},
		{
			name:   "BulkAddToGroups",
			method: http.MethodPost,
			path:   base + "/:addToGroups?api-version=" + webPubSubDataPlaneAPIVersion,
			payload: map[string]any{
				"groups": []string{"group-b", "group-c"},
				"filter": "connectionId eq '" + connectionID + "'",
			},
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "ListConnectionsInGroup",
			method:   http.MethodGet,
			path:     base + "/groups/group-b/connections?api-version=" + webPubSubDataPlaneAPIVersion + "&maxpagesize=10",
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SendToGroup",
			method: http.MethodPost,
			path:   base + "/groups/group-a/:send?api-version=" + webPubSubDataPlaneAPIVersion,
			payload: map[string]any{
				"message": "hello from stackyard",
			},
			statuses: []int{http.StatusNoContent},
		},
		{
			name:     "CloseUserConnections",
			method:   http.MethodPost,
			path:     base + "/users/" + userID + "/:closeConnections?api-version=" + webPubSubDataPlaneAPIVersion,
			statuses: []int{http.StatusNoContent},
		},
	}

	for _, call := range calls {
		payload, status, err := client.do(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		encoded, _ := json.MarshalIndent(payload, "", "  ")
		if len(encoded) == 0 || string(encoded) == "null" {
			encoded = []byte("{}")
		}
		fmt.Printf("%s -> %d\n%s\n", call.name, status, string(encoded))
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
