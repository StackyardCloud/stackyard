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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const fleetAPIVersion = "2025-03-01"
const fleetPrefix = "/azure"

type fleetClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type fleetSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

type fleetCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
}

type fleetPayloadContext struct {
	location        string
	subscriptionID  string
	resourceGroup   string
	fleetName       string
	updateStrategy  string
	fleetMemberName string
	clusterResource string
}

func (p *fleetSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newFleetClient(endpoint, account, subscriptionKey string) *fleetClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"fleet-2025-03-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&fleetSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &fleetClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *fleetClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func buildPath(template string, vars map[string]string) string {
	path := template
	for key, value := range vars {
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
	}
	return fleetPrefix + path
}

func payloadForMethod(methodPath string, ctx fleetPayloadContext) any {
	switch methodPath {
	case "auto-upgrade-profiles/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"channel": "Stable",
			},
		}
	case "fleet-members/create":
		return map[string]any{
			"properties": map[string]any{
				"clusterResourceId": ctx.clusterResource,
			},
		}
	case "fleet-members/update":
		return map[string]any{
			"properties": map[string]any{
				"group": "staging",
			},
		}
	case "fleet-update-strategies/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"strategy": map[string]any{
					"stages": []any{
						map[string]any{
							"name": "stage1",
							"groups": []any{
								map[string]any{"name": "group-a"},
							},
							"afterStageWaitInSeconds": 60,
						},
					},
				},
			},
		}
	case "fleets/create-or-update":
		return map[string]any{
			"location": ctx.location,
			"tags": map[string]any{
				"tier": "production",
				"env":  "local",
			},
			"properties": map[string]any{
				"hubProfile": map[string]any{
					"dnsPrefix": "dnsprefix1",
					"agentProfile": map[string]any{
						"vmSize": "Standard_DS1",
					},
				},
			},
		}
	case "fleets/update":
		return map[string]any{
			"tags": map[string]any{
				"tier": "secure",
				"env":  "prod",
			},
		}
	case "update-runs/create-or-update":
		return map[string]any{
			"properties": map[string]any{
				"strategy": map[string]any{
					"stages": []any{
						map[string]any{
							"name": "stage1",
							"groups": []any{
								map[string]any{"name": "group-a"},
							},
							"afterStageWaitInSeconds": 120,
						},
					},
				},
				"managedClusterUpdate": map[string]any{
					"upgrade": map[string]any{
						"type":              "Full",
						"kubernetesVersion": "1.26.1",
					},
					"nodeImageSelection": map[string]any{
						"type": "Latest",
					},
				},
			},
		}
	case "update-runs/skip":
		return map[string]any{
			"targets": []any{
				map[string]any{
					"type": "Member",
					"name": ctx.fleetMemberName,
				},
				map[string]any{
					"type": "AfterStageWait",
					"name": "stage1",
				},
			},
		}
	}
	return nil
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_FLEET_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_FLEET_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-fleet")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus")
	fleetName := getenv("STACKYARD_AZURE_FLEET_NAME", "fleet1")
	autoUpgradeProfileName := getenv("STACKYARD_AZURE_FLEET_AUTO_UPGRADE_PROFILE_NAME", "autoupgradeprofile1")
	fleetMemberName := getenv("STACKYARD_AZURE_FLEET_MEMBER_NAME", "member-1")
	updateStrategyName := getenv("STACKYARD_AZURE_FLEET_UPDATE_STRATEGY_NAME", "strategy1")
	updateRunName := getenv("STACKYARD_AZURE_FLEET_UPDATE_RUN_NAME", "run1")
	managedClusterName := getenv("STACKYARD_AZURE_FLEET_MANAGED_CLUSTER_NAME", "cluster-1")
	clusterResourceID := getenv(
		"STACKYARD_AZURE_FLEET_CLUSTER_RESOURCE_ID",
		fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerService/managedClusters/%s",
			subscriptionID,
			resourceGroup,
			managedClusterName,
		),
	)

	client := newFleetClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"subscriptionId":         subscriptionID,
		"resourceGroupName":      resourceGroup,
		"fleetName":              fleetName,
		"autoUpgradeProfileName": autoUpgradeProfileName,
		"fleetMemberName":        fleetMemberName,
		"updateStrategyName":     updateStrategyName,
		"updateRunName":          updateRunName,
	}

	payloadCtx := fleetPayloadContext{
		location:        location,
		subscriptionID:  subscriptionID,
		resourceGroup:   resourceGroup,
		fleetName:       fleetName,
		updateStrategy:  updateStrategyName,
		fleetMemberName: fleetMemberName,
		clusterResource: clusterResourceID,
	}

	calls := []fleetCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.ContainerService/operations?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleets_list_by_subscription",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/providers/Microsoft.ContainerService/fleets?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleets_list_by_resource_group",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleets_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}?api-version=2025-03-01",
			payload:  payloadForMethod("fleets/create-or-update", payloadCtx),
			statuses: []int{200, 201},
		},
		{
			name:     "fleets_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleets_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}?api-version=2025-03-01",
			payload:  payloadForMethod("fleets/update", payloadCtx),
			statuses: []int{200, 202},
		},
		{
			name:     "fleets_list_credentials",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/listCredentials?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleet_members_list_by_fleet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/members?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleet_members_create",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/members/{fleetMemberName}?api-version=2025-03-01",
			payload:  payloadForMethod("fleet-members/create", payloadCtx),
			statuses: []int{200, 201},
		},
		{
			name:     "fleet_members_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/members/{fleetMemberName}?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleet_members_update",
			method:   http.MethodPatch,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/members/{fleetMemberName}?api-version=2025-03-01",
			payload:  payloadForMethod("fleet-members/update", payloadCtx),
			statuses: []int{200, 202},
		},
		{
			name:     "fleet_update_strategies_list_by_fleet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateStrategies?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "fleet_update_strategies_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateStrategies/{updateStrategyName}?api-version=2025-03-01",
			payload:  payloadForMethod("fleet-update-strategies/create-or-update", payloadCtx),
			statuses: []int{200, 201},
		},
		{
			name:     "fleet_update_strategies_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateStrategies/{updateStrategyName}?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "auto_upgrade_profiles_list_by_fleet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "auto_upgrade_profiles_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles/{autoUpgradeProfileName}?api-version=2025-03-01",
			payload:  payloadForMethod("auto-upgrade-profiles/create-or-update", payloadCtx),
			statuses: []int{200, 201},
		},
		{
			name:     "auto_upgrade_profiles_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles/{autoUpgradeProfileName}?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "auto_upgrade_profile_operations_generate_update_run",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles/{autoUpgradeProfileName}/generateUpdateRun?api-version=2025-03-01",
			statuses: []int{200, 202},
		},
		{
			name:     "update_runs_list_by_fleet",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "update_runs_create_or_update",
			method:   http.MethodPut,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}?api-version=2025-03-01",
			payload:  payloadForMethod("update-runs/create-or-update", payloadCtx),
			statuses: []int{200, 201},
		},
		{
			name:     "update_runs_get",
			method:   http.MethodGet,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}?api-version=2025-03-01",
			statuses: []int{200},
		},
		{
			name:     "update_runs_start",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}/start?api-version=2025-03-01",
			statuses: []int{200, 202},
		},
		{
			name:     "update_runs_skip",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}/skip?api-version=2025-03-01",
			payload:  payloadForMethod("update-runs/skip", payloadCtx),
			statuses: []int{200, 202},
		},
		{
			name:     "update_runs_stop",
			method:   http.MethodPost,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}/stop?api-version=2025-03-01",
			statuses: []int{200, 202},
		},
		{
			name:     "update_runs_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}?api-version=2025-03-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "auto_upgrade_profiles_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles/{autoUpgradeProfileName}?api-version=2025-03-01",
			statuses: []int{202, 204},
		},
		{
			name:     "fleet_update_strategies_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateStrategies/{updateStrategyName}?api-version=2025-03-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "fleet_members_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/members/{fleetMemberName}?api-version=2025-03-01",
			statuses: []int{200, 202, 204},
		},
		{
			name:     "fleets_delete",
			method:   http.MethodDelete,
			template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}?api-version=2025-03-01",
			statuses: []int{200, 202, 204},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
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
