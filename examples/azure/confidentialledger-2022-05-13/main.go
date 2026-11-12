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

const confidentialLedgerAPIVersion = "2022-05-13"

type confidentialLedgerClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type confidentialLedgerSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *confidentialLedgerSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newConfidentialLedgerClient(endpoint, account, subscriptionKey string) *confidentialLedgerClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"confidentialledger-2022-05-13",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&confidentialLedgerSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &confidentialLedgerClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *confidentialLedgerClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	account := getenv("STACKYARD_AZURE_CONFIDENTIAL_LEDGER_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONFIDENTIAL_LEDGER_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-confidentialledger")
	region := getenv("STACKYARD_AZURE_REGION", "eastus")
	ledgerName := getenv("STACKYARD_AZURE_CONFIDENTIAL_LEDGER_NAME", "ledger-a")

	fmt.Printf("Stackyard Azure Confidential Ledger example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newConfidentialLedgerClient(endpoint, account, subscriptionKey)
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	resourceBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConfidentialLedger/ledgers/" + ledgerName

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
			path:     "/azure/providers/Microsoft.ConfidentialLedger/operations?api-version=" + confidentialLedgerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateLedger",
			method: http.MethodPut,
			path:   resourceBase + "?api-version=" + confidentialLedgerAPIVersion,
			payload: map[string]any{
				"location": region,
				"tags":     map[string]any{"env": "dev"},
				"properties": map[string]any{
					"ledgerType": "Public",
					"aadBasedSecurityPrincipals": []any{
						map[string]any{
							"principalId":    "11111111-1111-1111-1111-111111111111",
							"tenantId":       "22222222-2222-2222-2222-222222222222",
							"ledgerRoleName": "Administrator",
						},
					},
				},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "GetLedger",
			method:   http.MethodGet,
			path:     resourceBase + "?api-version=" + confidentialLedgerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListLedgersByResourceGroup",
			method:   http.MethodGet,
			path:     subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConfidentialLedger/ledgers?api-version=" + confidentialLedgerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListLedgersBySubscription",
			method:   http.MethodGet,
			path:     subscriptionBase + "/providers/Microsoft.ConfidentialLedger/ledgers?api-version=" + confidentialLedgerAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "PatchLedger",
			method: http.MethodPatch,
			path:   resourceBase + "?api-version=" + confidentialLedgerAPIVersion,
			payload: map[string]any{
				"tags": map[string]any{
					"env":   "stage",
					"owner": "stackyard",
				},
				"properties": map[string]any{
					"certBasedSecurityPrincipals": []any{
						map[string]any{
							"cert":           "-----BEGIN CERT-----",
							"ledgerRoleName": "Reader",
						},
					},
				},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "DeleteLedger",
			method:   http.MethodDelete,
			path:     resourceBase + "?api-version=" + confidentialLedgerAPIVersion,
			statuses: []int{http.StatusOK, http.StatusNoContent},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		printJSON(call.name, status, payload)
	}
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(formatted))
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
