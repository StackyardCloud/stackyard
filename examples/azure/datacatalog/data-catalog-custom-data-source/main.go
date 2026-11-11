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

const dataCatalogCustomDataSourceAPIVersion = "2016-03-30"

type dataCatalogCustomDataSourceClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type dataCatalogSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *dataCatalogSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newDataCatalogCustomDataSourceClient(endpoint, account, subscriptionKey string) *dataCatalogCustomDataSourceClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"data-catalog-custom-data-source",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&dataCatalogSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &dataCatalogCustomDataSourceClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataCatalogCustomDataSourceClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
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
	if err := ensureDataCatalogStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
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

func ensureDataCatalogStatus(status int, body []byte, expectedStatuses ...int) error {
	if len(expectedStatuses) == 0 {
		if status < 200 || status >= 300 {
			return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
		}
		return nil
	}
	for _, expected := range expectedStatuses {
		if status == expected {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d body=%s", status, strings.TrimSpace(string(body)))
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_DATACATALOG_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_DATACATALOG_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	catalogName := getenv("STACKYARD_AZURE_DATACATALOG_CATALOG_NAME", "DefaultCatalog")
	protocolNamespace := getenv("STACKYARD_AZURE_DATACATALOG_PROTOCOL_NAMESPACE", "Test.MyTestProtocols")
	protocolName := getenv("STACKYARD_AZURE_DATACATALOG_PROTOCOL_NAME", "abc-def")

	fmt.Printf("Stackyard Azure Data Catalog Custom Data Source example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataCatalogCustomDataSourceClient(endpoint, account, subscriptionKey)
	path := "/azure/catalogs/" + catalogName + "/dataSourceProtocols?api-version=" + dataCatalogCustomDataSourceAPIVersion

	payload, status, err := client.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		exitf("EnumerateInitial failed: %v", err)
	}
	printJSON("EnumerateInitial", status, payload)

	registerPayload := map[string]any{
		"namespace": protocolNamespace,
		"name":      protocolName,
		"identityProperties": []map[string]any{
			{"name": "prop1", "type": "string", "ignoreCase": true},
			{"name": "prop2", "type": "int"},
		},
		"identitySets": []map[string]any{
			{"name": "by-prop1", "properties": []string{"prop1"}},
			{"name": "by-prop1-prop2", "properties": []string{"prop1", "prop2"}},
		},
	}

	payload, status, err = client.doJSON(ctx, http.MethodPost, path, registerPayload, http.StatusOK)
	if err != nil {
		exitf("Register failed: %v", err)
	}
	printJSON("Register", status, payload)

	payload, status, err = client.doJSON(ctx, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		exitf("EnumerateAfterRegister failed: %v", err)
	}
	printJSON("EnumerateAfterRegister", status, payload)

	payload, status, err = client.doJSON(ctx, http.MethodPost, path, registerPayload, http.StatusBadRequest)
	if err != nil {
		exitf("RegisterDuplicate failed: %v", err)
	}
	errorObject, _ := payload["error"].(map[string]any)
	if errorObject["code"] != "ImmutableViewItem" {
		exitf("RegisterDuplicate returned unexpected error payload: %#v", payload)
	}
	printJSON("RegisterDuplicate", status, payload)
}

func printJSON(name string, status int, payload map[string]any) {
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	if len(encoded) == 0 || string(encoded) == "null" {
		encoded = []byte("{}")
	}
	fmt.Printf("%s -> %d\n%s\n", name, status, string(encoded))
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
