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

const defenderForEASMDataPlaneAPIVersion = "2024-10-01-preview"
const defenderForEASMDataPlanePrefix = "/azure/defenderforeasm-dataplanepreview/2024-10-01-preview"

type defenderForEASMDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type defenderForEASMDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *defenderForEASMDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDefenderForEASMDataPlaneClient(endpoint, account, subscriptionKey string) *defenderForEASMDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"defenderforeasm-dataplanepreview-2024-10-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&defenderForEASMDataPlaneSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &defenderForEASMDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *defenderForEASMDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DEFENDERFOREASM_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DEFENDERFOREASM_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	assetID := getenv("STACKYARD_AZURE_DEFENDERFOREASM_ASSET_ID", "aG9zdCQkY29udG9zby5jb20")
	cveID := getenv("STACKYARD_AZURE_DEFENDERFOREASM_CVE_ID", "CVE-2023-12345")
	dataConnectionName := getenv("STACKYARD_AZURE_DEFENDERFOREASM_DATA_CONNECTION_NAME", "ThisisaDataConnection")
	discoGroupName := getenv("STACKYARD_AZURE_DEFENDERFOREASM_DISCO_GROUP_NAME", "ThisisaDiscoGroup")
	discoTemplateID := getenv("STACKYARD_AZURE_DEFENDERFOREASM_DISCO_TEMPLATE_ID", "12345")
	policyName := getenv("STACKYARD_AZURE_DEFENDERFOREASM_POLICY_NAME", "ThisisaPolicy")
	savedFilterName := getenv("STACKYARD_AZURE_DEFENDERFOREASM_SAVED_FILTER_NAME", "ThisisaSavedFilter")
	taskID := getenv("STACKYARD_AZURE_DEFENDERFOREASM_TASK_ID", "ThisisaTaskId")

	filterConfirmed := url.QueryEscape(`state = "confirmed"`)
	filterConfirmedDomain := url.QueryEscape(`state = "confirmed" AND kind = "domain"`)
	filterAssetUpdate := url.QueryEscape(`state = "confirmed" AND name ^= "contoso.com"`)

	dataConnectionPayload := map[string]any{
		"kind": "logAnalytics",
		"properties": map[string]any{
			"apiKey":      "ThisisApiKey",
			"workspaceId": "ThisisWorkspaceId",
		},
		"frequency":       "weekly",
		"frequencyOffset": 1,
	}

	discoGroupPayload := map[string]any{
		"description":           "This is a disco group",
		"frequencyMilliseconds": 604800000,
		"tier":                  "advanced",
		"seeds": []any{
			map[string]any{
				"name": "thisisatest.microsoft.com",
				"kind": "host",
			},
		},
	}

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "Assets_ListAssetResource",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/assets?api-version=%s&filter=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion, filterConfirmedDomain),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "Assets_GetAssetResource",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/assets/%s?api-version=%s", defenderForEASMDataPlanePrefix, assetID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:   "Assets_GetAssetsExport",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/assets:export?api-version=%s&filter=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion, filterConfirmed),
			payload: map[string]any{
				"fileName": "assets-export.csv",
				"columns":  []any{"asset", "name"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "Assets_GetDeltaDetails",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/assets:getDeltaDetails?api-version=%s&skip=0&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"date":      "2024-10-01",
				"priorDays": 7,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "Assets_GetDeltaSummary",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/assets:getDeltaSummary?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"date":      "2024-10-01",
				"priorDays": 7,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "Assets_GetObservations",
			method:   http.MethodPost,
			path:     fmt.Sprintf("%s/assets/%s:getObservations?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, assetID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:   "Assets_UpdateAssets",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/assets?api-version=%s&filter=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion, filterAssetUpdate),
			payload: map[string]any{
				"externalId": "external-contoso-123",
				"labels": map[string]any{
					"triage": "confirmed",
				},
				"state": "confirmed",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CisaCves_GetCisaCves",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/cisaCves?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CisaCves_GetCisaCve",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/cisaCves/%s?api-version=%s", defenderForEASMDataPlanePrefix, cveID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "DataConnections_ListDataConnection",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/dataConnections?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DataConnections_CreateOrReplaceDataConnection",
			method:   http.MethodPut,
			path:     fmt.Sprintf("%s/dataConnections/%s?api-version=%s", defenderForEASMDataPlanePrefix, dataConnectionName, defenderForEASMDataPlaneAPIVersion),
			payload:  dataConnectionPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "DataConnections_GetDataConnection",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/dataConnections/%s?api-version=%s", defenderForEASMDataPlanePrefix, dataConnectionName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:   "DataConnections_ValidateDataConnection",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/dataConnections:validate?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"kind": "logAnalytics",
				"properties": map[string]any{
					"apiKey":      "ThisisApiKey",
					"workspaceId": "ThisisWorkspaceId",
				},
				"frequency":       "weekly",
				"frequencyOffset": 1,
				"name":            dataConnectionName,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DiscoveryGroups_ListDiscoGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/discoGroups?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DiscoveryGroups_CreateOrReplaceDiscoGroup",
			method:   http.MethodPut,
			path:     fmt.Sprintf("%s/discoGroups/%s?api-version=%s", defenderForEASMDataPlanePrefix, discoGroupName, defenderForEASMDataPlaneAPIVersion),
			payload:  discoGroupPayload,
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "DiscoveryGroups_GetDiscoGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/discoGroups/%s?api-version=%s", defenderForEASMDataPlanePrefix, discoGroupName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:   "DiscoveryGroups_ValidateDiscoGroup",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/discoGroups:validate?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"name":                  discoGroupName,
				"description":           "This is a disco group",
				"frequencyMilliseconds": 604800000,
				"tier":                  "advanced",
				"seeds": []any{
					map[string]any{
						"name": "thisisatest.microsoft.com",
						"kind": "host",
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DiscoveryGroups_ListRuns",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/discoGroups/%s/runs?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, discoGroupName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "DiscoveryGroups_RunDiscoGroup",
			method:   http.MethodPost,
			path:     fmt.Sprintf("%s/discoGroups/%s:run?api-version=%s", defenderForEASMDataPlanePrefix, discoGroupName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusNoContent, http.StatusNotFound},
		},
		{
			name:   "DiscoveryGroups_GetAssetChainSummary",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/discoGroups:getAssetChainSummary?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"assetChainSource": "ASSET",
				"sourceIds": []any{
					"DOMAIN$$contoso.com",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "DiscoveryGroups_DismissAssetChain",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/discoGroups:dismissAssetChain?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"assetChainSource": "ASSET",
				"sourceIds": []any{
					"DOMAIN$$contoso.com",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DiscoveryTemplates_ListDiscoTemplate",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/discoTemplates?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DiscoveryTemplates_GetDiscoTemplate",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/discoTemplates/%s?api-version=%s", defenderForEASMDataPlanePrefix, discoTemplateID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:   "SavedFilters_CreateOrReplaceSavedFilter",
			method: http.MethodPut,
			path:   fmt.Sprintf("%s/savedFilters/%s?api-version=%s", defenderForEASMDataPlanePrefix, savedFilterName, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"filter":      `state IN ("confirmed")`,
				"description": "This is a saved filter",
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "SavedFilters_ListSavedFilter",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/savedFilters?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "SavedFilters_GetSavedFilter",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/savedFilters/%s?api-version=%s", defenderForEASMDataPlanePrefix, savedFilterName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:   "Policies_CreateOrReplacePolicy",
			method: http.MethodPut,
			path:   fmt.Sprintf("%s/policies/%s?api-version=%s", defenderForEASMDataPlanePrefix, policyName, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"filterName":       savedFilterName,
				"description":      "This is a policy",
				"action":           "setState",
				"actionParameters": map[string]any{"value": "confirmed"},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "Policies_ListPolicy",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/policies?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "Reports_GetBillable",
			method:   http.MethodPost,
			path:     fmt.Sprintf("%s/reports/assets:getBillable?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "Reports_GetSummary",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/reports/assets:getSummary?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"metricCategories": nil,
				"metrics":          nil,
				"filters": []any{
					`state = "confirmed" AND kind = "domain"`,
					`state = "confirmed" AND kind = "host"`,
				},
				"groupBy": nil,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "Reports_GetSnapshot",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/reports/assets:getSnapshot?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"metric": "site_status_active",
				"page":   0,
				"size":   5,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "Reports_GetSnapshotExport",
			method: http.MethodPost,
			path:   fmt.Sprintf("%s/reports/assets:getSnapshotExport?api-version=%s", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			payload: map[string]any{
				"metric":   "site_status_active",
				"fileName": "snapshot-export.csv",
				"columns": []any{
					"asset",
					"name",
					"kind",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "Tasks_ListTask",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/tasks?api-version=%s&maxpagesize=5", defenderForEASMDataPlanePrefix, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "Tasks_GetTask",
			method:   http.MethodGet,
			path:     fmt.Sprintf("%s/tasks/%s?api-version=%s", defenderForEASMDataPlanePrefix, taskID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "Tasks_RunTask",
			method:   http.MethodPost,
			path:     fmt.Sprintf("%s/tasks/%s:run?api-version=%s", defenderForEASMDataPlanePrefix, taskID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "Tasks_DownloadTask",
			method:   http.MethodPost,
			path:     fmt.Sprintf("%s/tasks/%s:download?api-version=%s", defenderForEASMDataPlanePrefix, taskID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "Tasks_CancelTask",
			method:   http.MethodPost,
			path:     fmt.Sprintf("%s/tasks/%s:cancel?api-version=%s", defenderForEASMDataPlanePrefix, taskID, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusOK, http.StatusNotFound},
		},
		{
			name:     "DataConnections_DeleteDataConnection",
			method:   http.MethodDelete,
			path:     fmt.Sprintf("%s/dataConnections/%s?api-version=%s", defenderForEASMDataPlanePrefix, dataConnectionName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusNoContent, http.StatusNotFound},
		},
		{
			name:     "SavedFilters_DeleteSavedFilter",
			method:   http.MethodDelete,
			path:     fmt.Sprintf("%s/savedFilters/%s?api-version=%s", defenderForEASMDataPlanePrefix, savedFilterName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusNoContent, http.StatusNotFound},
		},
		{
			name:     "DiscoveryGroups_DeleteDiscoGroup",
			method:   http.MethodDelete,
			path:     fmt.Sprintf("%s/discoGroups/%s?api-version=%s", defenderForEASMDataPlanePrefix, discoGroupName, defenderForEASMDataPlaneAPIVersion),
			statuses: []int{http.StatusNoContent, http.StatusNotFound},
		},
	}

	client := newDefenderForEASMDataPlaneClient(endpoint, account, subscriptionKey)
	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
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
