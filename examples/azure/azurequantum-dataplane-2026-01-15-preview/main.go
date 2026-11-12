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

const quantumAPIVersion = "2026-01-15-preview"

type quantumClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type quantumSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *quantumSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newQuantumClient(endpoint, account, subscriptionKey string) *quantumClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"azurequantum-dataplane-2026-01-15-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&quantumSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &quantumClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *quantumClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func ensureStatus(status int, body []byte, expectedStatuses ...int) error {
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
	account := getenv("STACKYARD_AZURE_QUANTUM_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_QUANTUM_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")

	fmt.Printf("Stackyard Azure Quantum data-plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newQuantumClient(endpoint, account, subscriptionKey)
	base := "/azure/quantum-data-plane/2026-01-15-preview"

	sessionID := ""
	sessionJobID := ""
	jobID := ""

	calls := []struct {
		name     string
		method   string
		path     func() string
		payload  any
		statuses []int
	}{
		{
			name:     "ListProviders",
			method:   http.MethodGet,
			path:     func() string { return base + "/providers?api-version=" + quantumAPIVersion },
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListQuotas",
			method:   http.MethodGet,
			path:     func() string { return base + "/quotas?api-version=" + quantumAPIVersion },
			statuses: []int{http.StatusOK},
		},
		{
			name:   "OpenSession",
			method: http.MethodPost,
			path:   func() string { return base + "/sessions?api-version=" + quantumAPIVersion },
			payload: map[string]any{
				"name":             "session-a",
				"providerId":       "ionq",
				"target":           "ionq.simulator",
				"jobFailurePolicy": "Continue",
			},
			statuses: []int{http.StatusCreated},
		},
		{
			name:   "CreateSessionJob",
			method: http.MethodPost,
			path:   func() string { return base + "/sessions/" + sessionID + "/jobs?api-version=" + quantumAPIVersion },
			payload: map[string]any{
				"name":       "session-job-a",
				"providerId": "ionq",
				"target":     "ionq.simulator",
				"inputParams": map[string]any{
					"shots": 100,
				},
			},
			statuses: []int{http.StatusCreated},
		},
		{
			name:   "GetSessionJob",
			method: http.MethodGet,
			path: func() string {
				return base + "/sessions/" + sessionID + "/jobs/" + sessionJobID + "?api-version=" + quantumAPIVersion
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSessionJobs",
			method:   http.MethodGet,
			path:     func() string { return base + "/sessions/" + sessionID + "/jobs?api-version=" + quantumAPIVersion },
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateJob",
			method: http.MethodPost,
			path:   func() string { return base + "/jobs?api-version=" + quantumAPIVersion },
			payload: map[string]any{
				"name":       "job-a",
				"providerId": "microsoft-qc",
				"target":     "microsoft.estimator",
			},
			statuses: []int{http.StatusCreated},
		},
		{
			name:     "CancelJob",
			method:   http.MethodPost,
			path:     func() string { return base + "/jobs/" + jobID + ":cancel?api-version=" + quantumAPIVersion },
			statuses: []int{http.StatusOK},
		},
		{
			name:   "GetStorageSAS",
			method: http.MethodPost,
			path:   func() string { return base + "/storage/sasUri?api-version=" + quantumAPIVersion },
			payload: map[string]any{
				"containerName": "quantum-data",
				"blobName":      "input.json",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListTopLevelItems",
			method:   http.MethodGet,
			path:     func() string { return base + "/topLevelItems?api-version=" + quantumAPIVersion },
			statuses: []int{http.StatusOK},
		},
		{
			name:     "CloseSession",
			method:   http.MethodPost,
			path:     func() string { return base + "/sessions/" + sessionID + ":close?api-version=" + quantumAPIVersion },
			statuses: []int{http.StatusOK},
		},
	}

	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path(), call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		if call.name == "OpenSession" {
			sessionID = stringValue(payload["id"])
		}
		if call.name == "CreateSessionJob" {
			sessionJobID = stringValue(payload["id"])
		}
		if call.name == "CreateJob" {
			jobID = stringValue(payload["id"])
		}
		fmt.Printf("%s -> %d %s\n", call.name, status, summarizePayload(payload))
	}
}

func summarizePayload(payload map[string]any) string {
	if len(payload) == 0 {
		return "{}"
	}
	if id := stringValue(payload["id"]); id != "" {
		return "id=" + id
	}
	if values, ok := payload["value"].([]any); ok {
		return fmt.Sprintf("value_count=%d", len(values))
	}
	if sas := stringValue(payload["sasUri"]); sas != "" {
		return "sasUri=" + sas
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%#v", payload)
	}
	return string(body)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
