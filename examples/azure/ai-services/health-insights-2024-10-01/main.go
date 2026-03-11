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

const healthInsightsAPIVersion = "2024-10-01"

type healthInsightsClient struct {
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

func newHealthInsightsClient(endpoint, account, subscriptionKey string) *healthInsightsClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"health-insights-2024-10-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &healthInsightsClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *healthInsightsClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
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
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	jobID := getenv("STACKYARD_AZURE_HEALTH_INSIGHTS_JOB_ID", "job-123")

	fmt.Printf("Stackyard Azure Health Insights (health-insights-2024-10-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newHealthInsightsClient(endpoint, account, subscriptionKey)

	createPath := "/azure/health-insights/radiology-insights/jobs/" + jobID + "?api-version=" + healthInsightsAPIVersion
	_, status, err := client.doJSON(ctx, http.MethodPut, createPath, map[string]any{
		"jobData": map[string]any{
			"configuration": map[string]any{
				"locale": "en-US",
			},
			"patients": []map[string]any{
				{
					"id": "patient-1",
					"patientDocuments": []map[string]any{
						{
							"type":         "note",
							"clinicalType": "radiologyReport",
							"id":           "doc-1",
							"language":     "en",
							"content": map[string]any{
								"sourceType": "inline",
								"value":      "Chest CT shows no acute cardiopulmonary process.",
							},
						},
					},
				},
			},
		},
	}, http.StatusOK, http.StatusCreated, http.StatusNotImplemented)
	if err != nil {
		exitf("CreateRadiologyInsightsJob failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(createPath)
		return
	}

	getPath := "/azure/health-insights/radiology-insights/jobs/" + jobID + "?api-version=" + healthInsightsAPIVersion + "&expand=patients"
	_, status, err = client.doJSON(ctx, http.MethodGet, getPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetRadiologyInsightsJob failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getPath)
		return
	}

	fmt.Println("Done.")
}

func notImplemented(path string) {
	fmt.Printf("Route is recognized but not implemented yet: %s\n", path)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
