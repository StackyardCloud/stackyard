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

const textToSpeechAPIVersion = "2024-04-01"

type textToSpeechClient struct {
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

func newTextToSpeechClient(endpoint, account, subscriptionKey string) *textToSpeechClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"text-to-speech-2024-04-01",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &textToSpeechClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *textToSpeechClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	jobID := getenv("STACKYARD_AZURE_TEXT_TO_SPEECH_JOB_ID", "synth-1")
	operationID := getenv("STACKYARD_AZURE_TEXT_TO_SPEECH_OPERATION_ID", "op-123")

	fmt.Printf("Stackyard Azure Text to Speech (text-to-speech-2024-04-01) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newTextToSpeechClient(endpoint, account, subscriptionKey)

	createPath := "/azure/batchtexttospeech/2024-04-01/batchsyntheses/" + jobID + "?api-version=" + textToSpeechAPIVersion
	_, status, err := client.doJSON(ctx, http.MethodPut, createPath, map[string]any{
		"displayName": "stackyard-batch-synthesis",
		"description": "local synthesis job",
		"inputKind":   "PlainText",
		"inputs": []map[string]any{
			{"content": "Hello from Stackyard"},
		},
		"properties": map[string]any{
			"outputFormat": "riff-24khz-16bit-mono-pcm",
			"timeToLive":   "PT1H",
		},
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("CreateBatchSynthesis failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(createPath)
		return
	}

	getPath := "/azure/batchtexttospeech/2024-04-01/batchsyntheses/" + jobID + "?api-version=" + textToSpeechAPIVersion
	_, status, err = client.doJSON(ctx, http.MethodGet, getPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetBatchSynthesis failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getPath)
		return
	}

	listPath := "/azure/batchtexttospeech/2024-04-01/batchsyntheses?api-version=" + textToSpeechAPIVersion + "&maxpagesize=10&skip=0"
	_, status, err = client.doJSON(ctx, http.MethodGet, listPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("ListBatchSyntheses failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(listPath)
		return
	}

	opPath := "/azure/batchtexttospeech/2024-04-01/operations/" + operationID + "?api-version=" + textToSpeechAPIVersion
	_, status, err = client.doJSON(ctx, http.MethodGet, opPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("GetOperation failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(opPath)
		return
	}

	deletePath := "/azure/batchtexttospeech/2024-04-01/batchsyntheses/" + jobID + "?api-version=" + textToSpeechAPIVersion
	_, status, err = client.doJSON(ctx, http.MethodDelete, deletePath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("DeleteBatchSynthesis failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(deletePath)
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
