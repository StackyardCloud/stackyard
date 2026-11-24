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
	shareSubscriptionBase := accountBase + "/shareSubscriptions/" + shareSubscriptionName
	dataSetMappingBase := shareSubscriptionBase + "/dataSetMappings/" + dataSetMappingName
	triggerBase := shareSubscriptionBase + "/triggers/" + triggerName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "CreateResourceGroup",
			method:   http.MethodPut,
			path:     "/azure/subscriptions/" + subscriptionID + "/resourcegroups/" + resourceGroup + "?api-version=2021-04-01",
			payload:  map[string]any{"location": location},
			statuses: []int{http.StatusOK, http.StatusCreated},
		},
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
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}

	// The provider-to-consumer flow, which needs IDs the service issues: the
	// invitation's invitationId, the source data set's dataSetId, the
	// synchronization's ID and the provider share subscription's object ID.
	q := "?api-version=" + dataShareAPIVersion
	run := func(name, method, path string, payload any, statuses ...int) map[string]any {
		body, status, err := client.doJSON(ctx, method, path, payload, statuses...)
		if err != nil {
			fatalf("%s failed: %v", name, err)
		}
		printJSON(name, status, body)
		return body
	}
	prop := func(body map[string]any, key string) string {
		props, _ := body["properties"].(map[string]any)
		value, _ := props[key].(string)
		if value == "" {
			fatalf("response has no properties.%s: %v", key, body)
		}
		return value
	}
	firstProps := func(body map[string]any) map[string]any {
		items, _ := body["value"].([]any)
		if len(items) == 0 {
			fatalf("expected at least one item in %v", body)
		}
		return items[0].(map[string]any)
	}
	providerLocations := "/azure/providers/Microsoft.DataShare/locations/" + location

	invitationID := prop(run("GetInvitationID", http.MethodGet, invitationBase+q, nil, http.StatusOK), "invitationId")
	run("ListConsumerInvitations", http.MethodGet, "/azure/providers/Microsoft.DataShare/listInvitations"+q, nil, http.StatusOK)
	run("GetConsumerInvitation", http.MethodGet, providerLocations+"/consumerInvitations/"+invitationID+q, nil, http.StatusOK)

	subscription := run("CreateShareSubscription", http.MethodPut, shareSubscriptionBase+q, map[string]any{
		"properties": map[string]any{"invitationId": invitationID, "sourceShareLocation": location},
	}, http.StatusOK, http.StatusCreated)
	objectID := prop(subscription, "shareSubscriptionObjectId")
	run("GetShareSubscription", http.MethodGet, shareSubscriptionBase+q, nil, http.StatusOK)
	run("ListShareSubscriptionsByAccount", http.MethodGet, accountBase+"/shareSubscriptions"+q, nil, http.StatusOK)

	sourceDataSetID := prop(firstProps(run("ListConsumerSourceDataSetsByShareSubscription", http.MethodGet, shareSubscriptionBase+"/consumerSourceDataSets"+q, nil, http.StatusOK)), "dataSetId")
	run("CreateDataSetMapping", http.MethodPut, dataSetMappingBase+q, map[string]any{
		"kind": "Blob",
		"properties": map[string]any{
			"dataSetId": sourceDataSetID, "resourceGroup": resourceGroup, "storageAccountName": "storage2",
			"containerName": "sink-container", "filePath": "landing/file.csv", "subscriptionId": subscriptionID, "outputType": "Csv",
		},
	}, http.StatusOK, http.StatusCreated)
	run("GetDataSetMapping", http.MethodGet, dataSetMappingBase+q, nil, http.StatusOK)
	run("ListDataSetMappingsByShareSubscription", http.MethodGet, shareSubscriptionBase+"/dataSetMappings"+q, nil, http.StatusOK)

	run("CreateTrigger", http.MethodPut, triggerBase+q, map[string]any{
		"kind": "ScheduleBased",
		"properties": map[string]any{
			"recurrenceInterval": "Day", "synchronizationTime": "2026-01-01T00:00:00Z", "synchronizationMode": "Incremental",
		},
	}, http.StatusOK, http.StatusCreated, http.StatusAccepted)
	run("GetTrigger", http.MethodGet, triggerBase+q, nil, http.StatusOK)
	run("ListTriggersByShareSubscription", http.MethodGet, shareSubscriptionBase+"/triggers"+q, nil, http.StatusOK)
	run("ListSourceShareSynchronizationSettings", http.MethodPost, shareSubscriptionBase+"/listSourceShareSynchronizationSettings"+q, nil, http.StatusOK)

	sync := run("SynchronizeShareSubscription", http.MethodPost, shareSubscriptionBase+"/synchronize"+q,
		map[string]any{"synchronizationMode": "Incremental"}, http.StatusOK, http.StatusAccepted)
	syncRef := map[string]any{"synchronizationId": sync["synchronizationId"]}
	run("ListShareSubscriptionSynchronizations", http.MethodPost, shareSubscriptionBase+"/listSynchronizations"+q, nil, http.StatusOK)
	run("ListShareSubscriptionSynchronizationDetails", http.MethodPost, shareSubscriptionBase+"/listSynchronizationDetails"+q, syncRef, http.StatusOK)
	run("CancelShareSubscriptionSynchronization", http.MethodPost, shareSubscriptionBase+"/cancelSynchronization"+q, syncRef, http.StatusOK, http.StatusAccepted)
	run("ListShareSynchronizations", http.MethodPost, shareBase+"/listSynchronizations"+q, nil, http.StatusOK)
	run("ListShareSynchronizationDetails", http.MethodPost, shareBase+"/listSynchronizationDetails"+q, syncRef, http.StatusOK)

	providerShareSubscription := shareBase + "/providerShareSubscriptions/" + objectID
	run("ListProviderShareSubscriptionsByShare", http.MethodGet, shareBase+"/providerShareSubscriptions"+q, nil, http.StatusOK)
	run("GetProviderShareSubscription", http.MethodGet, providerShareSubscription+q, nil, http.StatusOK)
	run("AdjustProviderShareSubscription", http.MethodPost, providerShareSubscription+"/adjust"+q,
		map[string]any{"properties": map[string]any{"expirationDate": "2031-01-01T00:00:00Z"}}, http.StatusOK)
	run("RevokeProviderShareSubscription", http.MethodPost, providerShareSubscription+"/revoke"+q, nil, http.StatusOK, http.StatusAccepted)
	run("ReinstateProviderShareSubscription", http.MethodPost, providerShareSubscription+"/reinstate"+q, nil, http.StatusOK, http.StatusAccepted)

	// A second invitation for the consumer to reject.
	rejectBase := shareBase + "/invitations/" + invitationName + "-reject"
	rejectID := prop(run("CreateInvitationToReject", http.MethodPut, rejectBase+q, map[string]any{
		"properties": map[string]any{"targetEmail": "consumer@example.com"},
	}, http.StatusOK, http.StatusCreated), "invitationId")
	run("RejectConsumerInvitation", http.MethodPost, providerLocations+"/rejectInvitation"+q,
		map[string]any{"properties": map[string]any{"invitationId": rejectID}}, http.StatusOK)

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
