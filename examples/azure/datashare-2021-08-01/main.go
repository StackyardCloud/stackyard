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

const dataShareAPIVersion = "2021-08-01"

type dataShareClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataShareSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataShareSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataShareClient(endpoint, account, subscriptionKey string) *dataShareClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"datashare-2021-08-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dataShareSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &dataShareClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataShareClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_DATASHARE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATASHARE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-datashare")
	location := getenv("STACKYARD_AZURE_LOCATION", "eastus2")
	accountName := getenv("STACKYARD_AZURE_DATASHARE_ACCOUNT_NAME", "datashare-account-example")
	shareName := getenv("STACKYARD_AZURE_DATASHARE_SHARE_NAME", "datashare-share-example")
	invitationName := getenv("STACKYARD_AZURE_DATASHARE_INVITATION_NAME", "datashare-invitation-example")
	dataSetName := getenv("STACKYARD_AZURE_DATASHARE_DATA_SET_NAME", "datashare-dataset-example")
	syncSettingName := getenv("STACKYARD_AZURE_DATASHARE_SYNC_SETTING_NAME", "datashare-sync-setting-example")
	shareSubscriptionName := getenv("STACKYARD_AZURE_DATASHARE_SHARE_SUBSCRIPTION_NAME", "datashare-subscription-example")
	dataSetMappingName := getenv("STACKYARD_AZURE_DATASHARE_DATA_SET_MAPPING_NAME", "datashare-mapping-example")
	triggerName := getenv("STACKYARD_AZURE_DATASHARE_TRIGGER_NAME", "datashare-trigger-example")
	providerShareSubscriptionID := getenv("STACKYARD_AZURE_DATASHARE_PROVIDER_SHARE_SUBSCRIPTION_ID", "11111111-1111-1111-1111-111111111111")
	consumerInvitationID := getenv("STACKYARD_AZURE_DATASHARE_CONSUMER_INVITATION_ID", "22222222-2222-2222-2222-222222222222")
	synchronizationID := getenv("STACKYARD_AZURE_DATASHARE_SYNCHRONIZATION_ID", "33333333-3333-3333-3333-333333333333")
	sourceDataSetID := getenv("STACKYARD_AZURE_DATASHARE_SOURCE_DATA_SET_ID", "44444444-4444-4444-4444-444444444444")

	client := newDataShareClient(endpoint, account, subscriptionKey)

	accountBase := fmt.Sprintf(
		"/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataShare/accounts/%s",
		subscriptionID,
		resourceGroup,
		accountName,
	)
	shareBase := accountBase + "/shares/" + shareName
	invitationBase := shareBase + "/invitations/" + invitationName
	dataSetBase := shareBase + "/dataSets/" + dataSetName
	syncSettingBase := shareBase + "/synchronizationSettings/" + syncSettingName
	providerShareSubscriptionBase := shareBase + "/providerShareSubscriptions/" + providerShareSubscriptionID
	shareSubscriptionBase := accountBase + "/shareSubscriptions/" + shareSubscriptionName
	dataSetMappingBase := shareSubscriptionBase + "/dataSetMappings/" + dataSetMappingName
	triggerBase := shareSubscriptionBase + "/triggers/" + triggerName
	consumerInvitationBase := fmt.Sprintf(
		"/azure/providers/Microsoft.DataShare/locations/%s/consumerInvitations/%s",
		location,
		consumerInvitationID,
	)

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviderOperations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DataShare/operations?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "RegisterEmail",
			method:   http.MethodPost,
			path:     fmt.Sprintf("/azure/providers/Microsoft.DataShare/locations/%s/registerEmail?api-version=%s", location, dataShareAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAccount",
			method: http.MethodPut,
			path:   accountBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"location": location,
				"identity": map[string]any{
					"type": "SystemAssigned",
				},
				"tags": map[string]any{
					"environment": "local",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetAccount",
			method:   http.MethodGet,
			path:     accountBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccountsByResourceGroup",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/resourceGroups/%s/providers/Microsoft.DataShare/accounts?api-version=%s", subscriptionID, resourceGroup, dataShareAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListAccountsBySubscription",
			method:   http.MethodGet,
			path:     fmt.Sprintf("/azure/subscriptions/%s/providers/Microsoft.DataShare/accounts?api-version=%s", subscriptionID, dataShareAPIVersion),
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateAccount",
			method: http.MethodPatch,
			path:   accountBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"environment": "local-updated",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateShare",
			method: http.MethodPut,
			path:   shareBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"description": "share description",
					"terms":       "Confidential",
					"shareKind":   "CopyBased",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetShare",
			method:   http.MethodGet,
			path:     shareBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSharesByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/shares?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListShareSynchronizations",
			method:   http.MethodGet,
			path:     shareBase + "/synchronizations?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListShareSynchronizationDetails",
			method:   http.MethodGet,
			path:     shareBase + "/synchronizationDetails?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateInvitation",
			method: http.MethodPut,
			path:   invitationBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"targetEmail":    "consumer@example.com",
					"expirationDate": "2030-01-01T00:00:00Z",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetInvitation",
			method:   http.MethodGet,
			path:     invitationBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListInvitationsByShare",
			method:   http.MethodGet,
			path:     shareBase + "/invitations?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDataSet",
			method: http.MethodPut,
			path:   dataSetBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"kind": "Blob",
				"properties": map[string]any{
					"resourceGroup":      resourceGroup,
					"storageAccountName": "storage1",
					"containerName":      "container1",
					"filePath":           "folder/file.csv",
					"subscriptionId":     subscriptionID,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetDataSet",
			method:   http.MethodGet,
			path:     dataSetBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDataSetsByShare",
			method:   http.MethodGet,
			path:     shareBase + "/dataSets?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSynchronizationSetting",
			method: http.MethodPut,
			path:   syncSettingBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"kind": "ScheduleBased",
				"properties": map[string]any{
					"recurrenceInterval":  "Day",
					"synchronizationTime": "2026-01-01T00:00:00Z",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetSynchronizationSetting",
			method:   http.MethodGet,
			path:     syncSettingBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSynchronizationSettingsByShare",
			method:   http.MethodGet,
			path:     shareBase + "/synchronizationSettings?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateShareSubscription",
			method: http.MethodPut,
			path:   shareSubscriptionBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"invitationId":        consumerInvitationID,
					"sourceShareLocation": location,
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetShareSubscription",
			method:   http.MethodGet,
			path:     shareSubscriptionBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListShareSubscriptionsByAccount",
			method:   http.MethodGet,
			path:     accountBase + "/shareSubscriptions?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateDataSetMapping",
			method: http.MethodPut,
			path:   dataSetMappingBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"kind": "Blob",
				"properties": map[string]any{
					"dataSetId":          sourceDataSetID,
					"resourceGroup":      resourceGroup,
					"storageAccountName": "storage2",
					"containerName":      "sink-container",
					"filePath":           "landing/file.csv",
					"subscriptionId":     subscriptionID,
					"outputType":         "Csv",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
		{
			name:     "GetDataSetMapping",
			method:   http.MethodGet,
			path:     dataSetMappingBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListDataSetMappingsByShareSubscription",
			method:   http.MethodGet,
			path:     shareSubscriptionBase + "/dataSetMappings?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateTrigger",
			method: http.MethodPut,
			path:   triggerBase + "?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"kind": "ScheduleBased",
				"properties": map[string]any{
					"recurrenceInterval":  "Day",
					"synchronizationTime": "2026-01-01T00:00:00Z",
					"synchronizationMode": "Incremental",
				},
			},
			statuses: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
		},
		{
			name:     "GetTrigger",
			method:   http.MethodGet,
			path:     triggerBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTriggersByShareSubscription",
			method:   http.MethodGet,
			path:     shareSubscriptionBase + "/triggers?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListConsumerSourceDataSetsByShareSubscription",
			method:   http.MethodGet,
			path:     shareSubscriptionBase + "/consumerSourceDataSets?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListShareSubscriptionSynchronizations",
			method:   http.MethodGet,
			path:     shareSubscriptionBase + "/synchronizations?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListShareSubscriptionSynchronizationDetails",
			method:   http.MethodGet,
			path:     shareSubscriptionBase + "/synchronizationDetails?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSourceShareSynchronizationSettings",
			method:   http.MethodPost,
			path:     shareSubscriptionBase + "/listSourceShareSynchronizationSettings?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "SynchronizeShareSubscription",
			method: http.MethodPost,
			path:   shareSubscriptionBase + "/synchronize?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"synchronizationMode": "Incremental",
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:   "CancelShareSubscriptionSynchronization",
			method: http.MethodPost,
			path:   shareSubscriptionBase + "/cancelSynchronization?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"synchronizationId": synchronizationID,
			},
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetProviderShareSubscription",
			method:   http.MethodGet,
			path:     providerShareSubscriptionBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListProviderShareSubscriptionsByShare",
			method:   http.MethodGet,
			path:     shareBase + "/providerShareSubscriptions?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "AdjustProviderShareSubscription",
			method: http.MethodPost,
			path:   providerShareSubscriptionBase + "/adjust?api-version=" + dataShareAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"expirationDate": "2030-01-01T00:00:00Z",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ReinstateProviderShareSubscription",
			method:   http.MethodPost,
			path:     providerShareSubscriptionBase + "/reinstate?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "RevokeProviderShareSubscription",
			method:   http.MethodPost,
			path:     providerShareSubscriptionBase + "/revoke?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted},
		},
		{
			name:     "GetConsumerInvitation",
			method:   http.MethodGet,
			path:     consumerInvitationBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListConsumerInvitations",
			method:   http.MethodGet,
			path:     "/azure/providers/Microsoft.DataShare/listInvitations?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "RejectConsumerInvitation",
			method: http.MethodPost,
			path:   fmt.Sprintf("/azure/providers/Microsoft.DataShare/locations/%s/rejectInvitation?api-version=%s", location, dataShareAPIVersion),
			payload: map[string]any{
				"invitationId": consumerInvitationID,
			},
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	cleanup := []struct {
		name     string
		method   string
		path     string
		statuses []int
	}{
		{
			name:     "DeleteTrigger",
			method:   http.MethodDelete,
			path:     triggerBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteDataSetMapping",
			method:   http.MethodDelete,
			path:     dataSetMappingBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteShareSubscription",
			method:   http.MethodDelete,
			path:     shareSubscriptionBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteSynchronizationSetting",
			method:   http.MethodDelete,
			path:     syncSettingBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteDataSet",
			method:   http.MethodDelete,
			path:     dataSetBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteInvitation",
			method:   http.MethodDelete,
			path:     invitationBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
		{
			name:     "DeleteShare",
			method:   http.MethodDelete,
			path:     shareBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
		{
			name:     "DeleteAccount",
			method:   http.MethodDelete,
			path:     accountBase + "?api-version=" + dataShareAPIVersion,
			statuses: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
		},
	}

	for _, call := range cleanup {
		payload, status, err := client.doJSON(ctx, call.method, call.path, nil, call.statuses...)
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
