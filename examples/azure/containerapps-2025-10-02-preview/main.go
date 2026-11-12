package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const containerAppsDataPlaneAPIVersion = "2025-10-02-preview"

type containerAppsDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type containerAppsDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *containerAppsDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newContainerAppsDataPlaneClient(endpoint, account, subscriptionKey string) *containerAppsDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"containerapps-2025-10-02-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&containerAppsDataPlaneSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &containerAppsDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *containerAppsDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Raw().Header.Set(key, value)
	}
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

func (c *containerAppsDataPlaneClient) doMultipart(ctx context.Context, path, filename, contentType string, content []byte, expectedStatuses ...int) (map[string]any, int, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, 0, fmt.Errorf("create multipart form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return nil, 0, fmt.Errorf("write multipart content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, 0, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := runtime.NewRequest(ctx, http.MethodPost, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create multipart request: %w", err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	req.Raw().Header.Set("Content-Type", writer.FormDataContentType())
	req.Raw().Body = io.NopCloser(bytes.NewReader(buffer.Bytes()))
	req.Raw().ContentLength = int64(buffer.Len())

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute multipart request: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read multipart response: %w", readErr)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode multipart response: %w", err)
	}
	_ = contentType
	return decoded, resp.StatusCode, nil
}

func (c *containerAppsDataPlaneClient) doRaw(ctx context.Context, method, path string, expectedStatuses ...int) ([]byte, int, string, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, "", fmt.Errorf("create raw request %s %s: %w", method, path, err)
	}
	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("execute raw request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, "", fmt.Errorf("read raw response %s %s: %w", method, path, readErr)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, "", err
	}
	return body, resp.StatusCode, resp.Header.Get("Content-Type"), nil
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
	account := getenv("STACKYARD_AZURE_CONTAINER_APPS_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_CONTAINER_APPS_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	identifier := getenv("STACKYARD_AZURE_CONTAINER_APPS_DATA_PLANE_IDENTIFIER", "session-a")

	fmt.Printf("Stackyard Azure Container Apps Data Plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newContainerAppsDataPlaneClient(endpoint, account, subscriptionKey)
	base := "/azure/containerapps-data-plane/2025-10-02-preview"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:     "GetMetadata",
			method:   http.MethodGet,
			path:     base + "/metadata?api-version=" + containerAppsDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ExecuteCode",
			method: http.MethodPost,
			path:   base + "/executions?api-version=" + containerAppsDataPlaneAPIVersion + "&identifier=" + identifier,
			payload: map[string]any{
				"timeoutInSeconds": 30,
				"code":             "print('hello from stackyard')",
				"codeInputType":    "inline",
				"executionType":    "synchronous",
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:     "GetSession",
			method:   http.MethodGet,
			path:     base + "/session?api-version=" + containerAppsDataPlaneAPIVersion + "&identifier=" + identifier,
			statuses: []int{http.StatusOK},
		},
		{
			name:     "ListSessions",
			method:   http.MethodGet,
			path:     base + "/listSessions?api-version=" + containerAppsDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
	}

	var executionID string
	for _, call := range calls {
		payload, status, err := client.doJSON(ctx, call.method, call.path, call.payload, nil, call.statuses...)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		if call.name == "ExecuteCode" {
			executionID = strings.TrimSpace(asString(payload["id"]))
		}
		printJSON(call.name, status, payload)
	}

	if executionID == "" {
		fatalf("ExecuteCode did not return an execution id")
	}

	payload, status, err := client.doJSON(ctx, http.MethodGet, base+"/executions/"+executionID+"?api-version="+containerAppsDataPlaneAPIVersion+"&identifier="+identifier, nil, nil, http.StatusOK)
	if err != nil {
		fatalf("GetExecution failed: %v", err)
	}
	printJSON("GetExecution", status, payload)

	uploadPayload, uploadStatus, err := client.doMultipart(ctx, base+"/files?api-version="+containerAppsDataPlaneAPIVersion+"&identifier="+identifier+"&path=workspace", "notes.txt", "text/plain", []byte("stackyard data plane file"), http.StatusOK)
	if err != nil {
		fatalf("UploadFile failed: %v", err)
	}
	printJSON("UploadFile", uploadStatus, uploadPayload)

	for _, path := range []string{
		base + "/files?api-version=" + containerAppsDataPlaneAPIVersion + "&identifier=" + identifier + "&path=workspace",
		base + "/files/notes.txt?api-version=" + containerAppsDataPlaneAPIVersion + "&identifier=" + identifier + "&path=workspace",
	} {
		payload, status, err := client.doJSON(ctx, http.MethodGet, path, nil, nil, http.StatusOK)
		if err != nil {
			fatalf("File metadata call failed for %s: %v", path, err)
		}
		printJSON(path, status, payload)
	}

	contentBody, contentStatus, contentType, err := client.doRaw(ctx, http.MethodGet, base+"/files/notes.txt/content?api-version="+containerAppsDataPlaneAPIVersion+"&identifier="+identifier+"&path=workspace", http.StatusOK)
	if err != nil {
		fatalf("GetFileContent failed: %v", err)
	}
	fmt.Printf("\n== GetFileContent (%d) ==\ncontent-type=%s\n%s\n", contentStatus, contentType, string(contentBody))

	for _, call := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"DeleteFile", http.MethodDelete, base + "/files/notes.txt?api-version=" + containerAppsDataPlaneAPIVersion + "&identifier=" + identifier + "&path=workspace", http.StatusNoContent},
		{"DeleteSession", http.MethodDelete, base + "/session?api-version=" + containerAppsDataPlaneAPIVersion + "&identifier=" + identifier, http.StatusNoContent},
	} {
		_, status, err := client.doJSON(ctx, call.method, call.path, nil, nil, call.status)
		if err != nil {
			fatalf("%s failed: %v", call.name, err)
		}
		fmt.Printf("\n== %s (%d) ==\n{}\n", call.name, status)
	}
}

func printJSON(name string, status int, payload map[string]any) {
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("%s: marshal response: %v", name, err)
	}
	fmt.Printf("\n== %s (%d) ==\n%s\n", name, status, string(formatted))
}

func asString(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
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
