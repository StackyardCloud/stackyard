package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

type dataPlaneListManagementClient struct {
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

func newDataPlaneListManagementClient(endpoint, account, subscriptionKey string) *dataPlaneListManagementClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-data-plane-list-management-v1.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &dataPlaneListManagementClient{endpoint: strings.TrimRight(endpoint, "/"), pipeline: pipeline}
}

func (c *dataPlaneListManagementClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (any, int, error) {
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body %s %s: %w", method, path, err)
	}

	if len(expectedStatuses) > 0 {
		matched := false
		for _, status := range expectedStatuses {
			if resp.StatusCode == status {
				matched = true
				break
			}
		}
		if !matched {
			return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
		}
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if len(strings.TrimSpace(string(respBody))) == 0 {
		return map[string]any{}, resp.StatusCode, nil
	}

	if strings.HasPrefix(strings.TrimSpace(string(respBody)), "[") {
		var rows []map[string]any
		if err := json.Unmarshal(respBody, &rows); err != nil {
			return nil, resp.StatusCode, fmt.Errorf("decode JSON array for %s %s: %w", method, path, err)
		}
		return rows, resp.StatusCode, nil
	}

	var payloadMap map[string]any
	if err := json.Unmarshal(respBody, &payloadMap); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode JSON object for %s %s: %w", method, path, err)
	}
	return payloadMap, resp.StatusCode, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DATA_PLANE_ACCOUNT", getenv("STACKYARD_AZURE_CONTENTMODERATOR_ACCOUNT", "devstoreaccount1"))
	subscriptionKey := getenv("STACKYARD_AZURE_DATA_PLANE_SUBSCRIPTION_KEY", getenv("STACKYARD_AZURE_CONTENTMODERATOR_SUBSCRIPTION_KEY", "stackyard-local-subscription-key"))

	fmt.Printf("Stackyard Azure AI Services Data Plane - List Management example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataPlaneListManagementClient(endpoint, account, subscriptionKey)

	createdAny, _, err := client.doJSON(ctx, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists", map[string]any{
		"Name":        "sdk-list",
		"Description": "stackyard list",
		"Metadata": map[string]string{
			"owner": "sdk",
		},
	}, http.StatusOK)
	if err != nil {
		exitf("CreateImageList failed: %v", err)
	}
	created := asMap(createdAny)
	listID := asInt64(created["Id"])
	if listID <= 0 {
		exitf("CreateImageList expected positive Id, got %#v", created)
	}

	allAny, _, err := client.doJSON(ctx, http.MethodGet, "/azure/contentmoderator/lists/v1.0/imagelists", nil, http.StatusOK)
	if err != nil {
		exitf("GetAllImageLists failed: %v", err)
	}
	allRows := asList(allAny)
	if len(allRows) == 0 {
		exitf("GetAllImageLists expected non-empty list")
	}

	_, _, err = client.doJSON(ctx, http.MethodGet, "/azure/contentmoderator/lists/v1.0/imagelists/"+strconv.FormatInt(listID, 10), nil, http.StatusOK)
	if err != nil {
		exitf("GetImageListDetails failed: %v", err)
	}

	updatedAny, _, err := client.doJSON(ctx, http.MethodPut, "/azure/contentmoderator/lists/v1.0/imagelists/"+strconv.FormatInt(listID, 10), map[string]any{
		"Name":        "sdk-list-updated",
		"Description": "stackyard list updated",
		"Metadata": map[string]string{
			"owner": "sdk",
			"env":   "local",
		},
	}, http.StatusOK)
	if err != nil {
		exitf("UpdateImageList failed: %v", err)
	}
	updated := asMap(updatedAny)
	if asString(updated["Name"]) != "sdk-list-updated" {
		exitf("UpdateImageList expected updated name, got %#v", updated)
	}

	refreshAny, _, err := client.doJSON(ctx, http.MethodPost, "/azure/contentmoderator/lists/v1.0/imagelists/"+strconv.FormatInt(listID, 10)+"/RefreshIndex", nil, http.StatusOK)
	if err != nil {
		exitf("RefreshImageListIndex failed: %v", err)
	}
	refresh := asMap(refreshAny)
	if ok, _ := refresh["IsUpdateSuccess"].(bool); !ok {
		exitf("RefreshImageListIndex expected IsUpdateSuccess=true, got %#v", refresh)
	}

	_, _, err = client.doJSON(ctx, http.MethodDelete, "/azure/contentmoderator/lists/v1.0/imagelists/"+strconv.FormatInt(listID, 10), nil, http.StatusOK)
	if err != nil {
		exitf("DeleteImageList failed: %v", err)
	}

	fmt.Println("Done.")
}

func asMap(value any) map[string]any {
	row, _ := value.(map[string]any)
	if row == nil {
		return map[string]any{}
	}
	return row
}

func asList(value any) []map[string]any {
	if rows, ok := value.([]map[string]any); ok {
		return rows
	}
	generic, _ := value.([]any)
	rows := make([]map[string]any, 0, len(generic))
	for _, row := range generic {
		if item, ok := row.(map[string]any); ok {
			rows = append(rows, item)
		}
	}
	return rows
}

func asInt64(value any) int64 {
	v, _ := value.(float64)
	return int64(v)
}

func asString(value any) string {
	v, _ := value.(string)
	return strings.TrimSpace(v)
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
