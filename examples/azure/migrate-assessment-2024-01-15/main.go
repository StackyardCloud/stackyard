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

const migrateAssessmentAPIVersion = "2024-01-15"

type migrateAssessmentClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type migrateAssessmentSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *migrateAssessmentSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newMigrateAssessmentClient(endpoint, account, subscriptionKey string) *migrateAssessmentClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"migrate-assessment-2024-01-15",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&migrateAssessmentSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &migrateAssessmentClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *migrateAssessmentClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	return decoded, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_MIGRATE_ASSESSMENT_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_MIGRATE_ASSESSMENT_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	subscriptionID := getenv("STACKYARD_AZURE_SUBSCRIPTION_ID", "00000000-0000-0000-0000-000000000000")
	resourceGroup := getenv("STACKYARD_AZURE_RESOURCE_GROUP", "rg-migrate")
	projectName := getenv("STACKYARD_AZURE_MIGRATE_ASSESSMENT_PROJECT_NAME", "project-a")
	groupName := getenv("STACKYARD_AZURE_MIGRATE_ASSESSMENT_GROUP_NAME", "group-a")
	assessmentName := getenv("STACKYARD_AZURE_MIGRATE_ASSESSMENT_NAME", "assessment-a")

	fmt.Printf("Stackyard Azure Migrate Assessment example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newMigrateAssessmentClient(endpoint, account, subscriptionKey)
	base := "/azure/subscriptions/" + subscriptionID
	projectBase := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Migrate/assessmentProjects/" + projectName

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
			path:     "/azure/providers/Microsoft.Migrate/operations?api-version=" + migrateAssessmentAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateProject",
			method: http.MethodPut,
			path:   projectBase + "?api-version=" + migrateAssessmentAPIVersion,
			payload: map[string]any{
				"location": "eastus",
				"tags":     map[string]any{"env": "dev"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "CreateGroup",
			method: http.MethodPut,
			path:   projectBase + "/groups/" + groupName + "?api-version=" + migrateAssessmentAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"groupType": "Custom"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "UpdateGroupMachines",
			method: http.MethodPost,
			path:   projectBase + "/groups/" + groupName + "/updateMachines?api-version=" + migrateAssessmentAPIVersion,
			payload: map[string]any{
				"machineNames": []any{"machine-1", "machine-2"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateAssessment",
			method: http.MethodPut,
			path:   projectBase + "/groups/" + groupName + "/assessments/" + assessmentName + "?api-version=" + migrateAssessmentAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"assessmentType": "Machine"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListAssessedMachines",
			method:   http.MethodGet,
			path:     projectBase + "/groups/" + groupName + "/assessments/" + assessmentName + "/assessedMachines?api-version=" + migrateAssessmentAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateVmwareCollector",
			method: http.MethodPut,
			path:   projectBase + "/vmwarecollectors/collector-a?api-version=" + migrateAssessmentAPIVersion,
			payload: map[string]any{
				"properties": map[string]any{"discoverySiteId": "site-a"},
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
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
