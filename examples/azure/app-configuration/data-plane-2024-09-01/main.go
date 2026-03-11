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

const appConfigurationDataPlaneAPIVersion = "2024-09-01"

type appConfigurationDataPlaneClient struct {
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

func newAppConfigurationDataPlaneClient(endpoint, account, subscriptionKey string) *appConfigurationDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"app-configuration-data-plane-2024-09-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)
	return &appConfigurationDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *appConfigurationDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

	if len(expectedStatuses) == 0 {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
	} else {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
		}
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
	account := getenv("STACKYARD_AZURE_APPCONFIG_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPCONFIG_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	key := getenv("STACKYARD_AZURE_APPCONFIG_DATA_PLANE_KEY", "Message")
	snapshot := getenv("STACKYARD_AZURE_APPCONFIG_DATA_PLANE_SNAPSHOT", "Prod-2022-08-01")

	fmt.Printf("Stackyard Azure App Configuration Data Plane (data-plane-2024-09-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAppConfigurationDataPlaneClient(endpoint, account, subscriptionKey)

	keyScope := "/azure/appconfiguration/kv/" + key
	snapshotScope := "/azure/appconfiguration/snapshots/" + snapshot
	lockScope := "/azure/appconfiguration/locks/" + key

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "CheckKeyValue",
			method:   http.MethodHead,
			path:     keyScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetKeyValue",
			method:   http.MethodGet,
			path:     keyScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PutKeyValue",
			method: http.MethodPut,
			path:   keyScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			payload: map[string]any{
				"key":   key,
				"value": "hello",
				"label": "prod",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteKeyValue",
			method:   http.MethodDelete,
			path:     keyScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckKeyValues",
			method:   http.MethodHead,
			path:     "/azure/appconfiguration/kv?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetKeyValues",
			method:   http.MethodGet,
			path:     "/azure/appconfiguration/kv?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckKeys",
			method:   http.MethodHead,
			path:     "/azure/appconfiguration/keys?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetKeys",
			method:   http.MethodGet,
			path:     "/azure/appconfiguration/keys?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckLabels",
			method:   http.MethodHead,
			path:     "/azure/appconfiguration/labels?api-version=" + appConfigurationDataPlaneAPIVersion + "&$select=name",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetLabels",
			method:   http.MethodGet,
			path:     "/azure/appconfiguration/labels?api-version=" + appConfigurationDataPlaneAPIVersion + "&$select=name",
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckRevisions",
			method:   http.MethodHead,
			path:     "/azure/appconfiguration/revisions?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetRevisions",
			method:   http.MethodGet,
			path:     "/azure/appconfiguration/revisions?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSnapshot",
			method: http.MethodPut,
			path:   snapshotScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			payload: map[string]any{
				"name": snapshot,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateSnapshot",
			method: http.MethodPatch,
			path:   snapshotScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			payload: map[string]any{
				"status": "ready",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckSnapshot",
			method:   http.MethodHead,
			path:     snapshotScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSnapshot",
			method:   http.MethodGet,
			path:     snapshotScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CheckSnapshots",
			method:   http.MethodHead,
			path:     "/azure/appconfiguration/snapshots?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSnapshots",
			method:   http.MethodGet,
			path:     "/azure/appconfiguration/snapshots?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "PutLock",
			method:   http.MethodPut,
			path:     lockScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteLock",
			method:   http.MethodDelete,
			path:     lockScope + "?api-version=" + appConfigurationDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetOperationDetails",
			method:   http.MethodGet,
			path:     "/azure/appconfiguration/operations?api-version=" + appConfigurationDataPlaneAPIVersion + "&snapshot=" + snapshot,
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		_, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: status=%d\n", call.name, status)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
