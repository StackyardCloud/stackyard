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

const dataManagerForAgriDataPlaneAPIVersion = "2023-11-01-preview"

type dataManagerForAgriDataPlaneClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataManagerForAgriDataPlaneSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataManagerForAgriDataPlaneSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataManagerForAgriDataPlaneClient(endpoint, account, subscriptionKey string) *dataManagerForAgriDataPlaneClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"data-manager-for-agri-dataplane-2023-11-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataManagerForAgriDataPlaneSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &dataManagerForAgriDataPlaneClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataManagerForAgriDataPlaneClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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

func (c *dataManagerForAgriDataPlaneClient) doMultipart(ctx context.Context, path, filename string, content []byte, expectedStatuses ...int) (map[string]any, int, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
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

	req, err := runtime.NewRequest(ctx, http.MethodPatch, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create multipart request: %w", err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	req.Raw().Header.Set("Content-Type", writer.FormDataContentType())
	req.Raw().Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
	req.Raw().ContentLength = int64(buf.Len())

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
	return decoded, resp.StatusCode, nil
}

func (c *dataManagerForAgriDataPlaneClient) doRaw(ctx context.Context, method, path string, expectedStatuses ...int) ([]byte, int, string, error) {
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
	account := getenv("STACKYARD_AZURE_AGRI_DATA_PLANE_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AGRI_DATA_PLANE_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	partyID := getenv("STACKYARD_AZURE_AGRI_PARTY_ID", "party-a")

	fmt.Printf("Stackyard Azure Data Manager for Agriculture Data Plane example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataManagerForAgriDataPlaneClient(endpoint, account, subscriptionKey)
	base := "/azure/data-manager-for-agri/dataplane/2023-11-01-preview"

	calls := []struct {
		name     string
		method   string
		path     string
		payload  any
		statuses []int
	}{
		{
			name:   "UpsertParty",
			method: http.MethodPatch,
			path:   base + "/parties/" + partyID + "?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"name":        "Primary Party",
				"description": "Stackyard agriculture data-plane example",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "UpsertFarm",
			method: http.MethodPatch,
			path:   base + "/parties/" + partyID + "/farms/farm-a?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"name":   "Farm A",
				"status": "Active",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:     "ListFarms",
			method:   http.MethodGet,
			path:     base + "/farms?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpsertField",
			method: http.MethodPatch,
			path:   base + "/parties/" + partyID + "/fields/field-a?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"name":   "Field A",
				"status": "Active",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "SearchFields",
			method: http.MethodPost,
			path:   base + "/fields:search?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"partyId": partyID,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "UpsertDataset",
			method: http.MethodPatch,
			path:   base + "/datasets/dataset-a?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"name":   "Agronomy Dataset",
				"status": "Active",
			},
			statuses: []int{http.StatusCreated, http.StatusOK},
		},
		{
			name:   "GrantDatasetAccess",
			method: http.MethodPost,
			path:   base + "/datasets/dataset-a/access:grant?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"grantee":     "grower@example.com",
				"permissions": []string{"read", "write"},
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "FetchWeather",
			method: http.MethodPost,
			path:   base + "/weather-data:fetch?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"partyId": partyID,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "ConnectOAuth",
			method: http.MethodPost,
			path:   base + "/oauth/tokens/:connect?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"providerName": "demo-provider",
				"partyId":      partyID,
			},
			statuses: []int{http.StatusOK},
		},
		{
			name:   "CreateSolutionJob",
			method: http.MethodPost,
			path:   base + "/solutions/crop-nitrogen:create?api-version=" + dataManagerForAgriDataPlaneAPIVersion,
			payload: map[string]any{
				"partyId": partyID,
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

	attachment, status, err := client.doMultipart(
		ctx,
		base+"/parties/"+partyID+"/attachments/attachment-a?api-version="+dataManagerForAgriDataPlaneAPIVersion,
		"field-notes.txt",
		[]byte("field notes from stackyard"),
		http.StatusCreated, http.StatusOK,
	)
	if err != nil {
		fatalf("UploadAttachment failed: %v", err)
	}
	printJSON("UploadAttachment", status, attachment)

	body, status, contentType, err := client.doRaw(
		ctx,
		http.MethodGet,
		base+"/parties/"+partyID+"/attachments/attachment-a/file?api-version="+dataManagerForAgriDataPlaneAPIVersion,
		http.StatusOK,
	)
	if err != nil {
		fatalf("DownloadAttachment failed: %v", err)
	}
	fmt.Printf("\n== DownloadAttachment (%d) ==\nContent-Type: %s\n%s\n", status, contentType, string(body))
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
