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

const appComplianceAPIVersion = "2024-06-27"

type appComplianceClient struct {
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

func newAppComplianceClient(endpoint, account, subscriptionKey string) *appComplianceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"app-compliance-2024-06-27",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{
				account:         account,
				subscriptionKey: subscriptionKey,
			}},
		},
		&policy.ClientOptions{},
	)

	return &appComplianceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *appComplianceClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_APPCOMPLIANCE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_APPCOMPLIANCE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	reportName := getenv("STACKYARD_AZURE_APPCOMPLIANCE_REPORT", "report-a")
	evidenceName := getenv("STACKYARD_AZURE_APPCOMPLIANCE_EVIDENCE", "evidence-a")
	scopingConfigurationName := getenv("STACKYARD_AZURE_APPCOMPLIANCE_SCOPING_CONFIGURATION", "scoping-a")
	snapshotName := getenv("STACKYARD_AZURE_APPCOMPLIANCE_SNAPSHOT", "snapshot-a")
	webhookName := getenv("STACKYARD_AZURE_APPCOMPLIANCE_WEBHOOK", "webhook-a")

	fmt.Printf("Stackyard Azure App Compliance (app-compliance-2024-06-27) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newAppComplianceClient(endpoint, account, subscriptionKey)

	providerScope := "/azure/providers/Microsoft.AppComplianceAutomation"
	reportScope := providerScope + "/reports/" + reportName
	evidenceScope := reportScope + "/evidences/" + evidenceName
	scopingConfigurationScope := reportScope + "/scopingConfigurations/" + scopingConfigurationName
	snapshotScope := reportScope + "/snapshots/" + snapshotName
	webhookScope := reportScope + "/webhooks/" + webhookName

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "ListOperations",
			method:   http.MethodGet,
			path:     providerScope + "/operations?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CheckNameAvailability",
			method: http.MethodPost,
			path:   providerScope + "/checkNameAvailability?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"name": reportName,
				"type": "Microsoft.AppComplianceAutomation/reports",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GetCollectionCount",
			method: http.MethodPost,
			path:   providerScope + "/getCollectionCount?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"reportName": reportName,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GetOverviewStatus",
			method: http.MethodPost,
			path:   providerScope + "/getOverviewStatus?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"reportName": reportName,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListInUseStorageAccounts",
			method:   http.MethodPost,
			path:     providerScope + "/listInUseStorageAccounts?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "Onboard",
			method: http.MethodPost,
			path:   providerScope + "/onboard?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"subscriptionIds": []string{"00000000-0000-0000-0000-000000000000"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateReport",
			method: http.MethodPut,
			path:   reportScope + "?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"offerGuid": "11111111-1111-1111-1111-111111111111",
					"timeZone":  "UTC",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetReport",
			method:   http.MethodGet,
			path:     reportScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListReports",
			method:   http.MethodGet,
			path:     providerScope + "/reports?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateReport",
			method: http.MethodPatch,
			path:   reportScope + "?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"timeZone": "UTC",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetScopingQuestions",
			method:   http.MethodPost,
			path:     reportScope + "/getScopingQuestions?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "TriggerEvaluation",
			method:   http.MethodPost,
			path:     providerScope + "/triggerEvaluation?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "VerifyReport",
			method:   http.MethodPost,
			path:     reportScope + "/verify?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateEvidence",
			method: http.MethodPut,
			path:   evidenceScope + "?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"displayName": "Evidence A",
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetEvidence",
			method:   http.MethodGet,
			path:     evidenceScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListEvidencesByReport",
			method:   http.MethodGet,
			path:     reportScope + "/evidences?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DownloadEvidence",
			method:   http.MethodPost,
			path:     evidenceScope + "/download?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateScopingConfiguration",
			method: http.MethodPut,
			path:   scopingConfigurationScope + "?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"answers": []map[string]any{
						{"questionId": "q1", "answers": []string{"a1"}},
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetScopingConfiguration",
			method:   http.MethodGet,
			path:     scopingConfigurationScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListScopingConfigurations",
			method:   http.MethodGet,
			path:     reportScope + "/scopingConfigurations?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSnapshot",
			method:   http.MethodGet,
			path:     snapshotScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSnapshots",
			method:   http.MethodGet,
			path:     reportScope + "/snapshots?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DownloadSnapshot",
			method:   http.MethodPost,
			path:     snapshotScope + "/download?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateWebhook",
			method: http.MethodPut,
			path:   webhookScope + "?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"uri":    "https://example.com/hook",
					"events": []string{"ReportUpdated"},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpdateWebhook",
			method: http.MethodPatch,
			path:   webhookScope + "?api-version=" + appComplianceAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{
					"events": []string{"ReportVerified"},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetWebhook",
			method:   http.MethodGet,
			path:     webhookScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListWebhooks",
			method:   http.MethodGet,
			path:     reportScope + "/webhooks?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteWebhook",
			method:   http.MethodDelete,
			path:     webhookScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteScopingConfiguration",
			method:   http.MethodDelete,
			path:     scopingConfigurationScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteEvidence",
			method:   http.MethodDelete,
			path:     evidenceScope + "?api-version=" + appComplianceAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteReport",
			method:   http.MethodDelete,
			path:     reportScope + "?api-version=" + appComplianceAPIVersion,
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
