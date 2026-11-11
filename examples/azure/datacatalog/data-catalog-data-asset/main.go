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

const dataCatalogDataAssetAPIVersion = "2016-03-30"

type dataCatalogDataAssetClient struct {
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

func newDataCatalogDataAssetClient(endpoint, account, subscriptionKey string) *dataCatalogDataAssetClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"data-catalog-data-asset",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&dataCatalogSharedKeyPolicy{account: account, subscriptionKey: subscriptionKey},
			},
		},
		&policy.ClientOptions{},
	)
	return &dataCatalogDataAssetClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *dataCatalogDataAssetClient) doJSON(ctx context.Context, method, path string, payload any, headers map[string]string, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	if _, ok := headers["Accept"]; !ok {
		req.Raw().Header.Set("Accept", "application/json")
	}
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
	viewName := getenv("STACKYARD_AZURE_DATACATALOG_VIEW_NAME", "tables")
	assetName := getenv("STACKYARD_AZURE_DATACATALOG_ASSET_NAME", "Orders")

	fmt.Printf("Stackyard Azure Data Catalog Data Asset example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDataCatalogDataAssetClient(endpoint, account, subscriptionKey)
	registerPath := "/azure/catalogs/" + catalogName + "/views/" + viewName + "?api-version=" + dataCatalogDataAssetAPIVersion

	registerPayload := map[string]any{
		"roles": []map[string]any{
			{
				"role": "Contributor",
				"members": []map[string]any{
					{"objectId": "00000000-0000-0000-0000-000000000201"},
				},
			},
		},
		"properties": map[string]any{
			"fromSourceSystem": true,
			"name":             assetName,
			"dataSource": map[string]any{
				"sourceType": "SQL Server",
				"objectType": "Table",
			},
			"dsl": map[string]any{
				"protocol":       "tds",
				"authentication": "windows",
				"address": map[string]any{
					"server":   "MyServer.contoso.com",
					"database": "NORTHWND",
					"schema":   "dbo",
					"object":   assetName,
				},
			},
		},
		"annotations": map[string]any{
			"schema": map[string]any{
				"properties": map[string]any{
					"fromSourceSystem": true,
					"columns": []map[string]any{
						{"name": "OrderID", "type": "int", "isNullable": false},
					},
				},
			},
		},
	}

	payload, status, err := client.doJSON(ctx, http.MethodPost, registerPath, registerPayload, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		exitf("RegisterOrUpdate failed: %v", err)
	}
	printJSON("RegisterOrUpdate", status, payload)

	assetID := strings.TrimSpace(asString(payload["id"]))
	if assetID == "" {
		exitf("register response missing id: %#v", payload)
	}
	assetPath := toLocalPath(assetID) + "?api-version=" + dataCatalogDataAssetAPIVersion

	payload, status, err = client.doJSON(ctx, http.MethodGet, assetPath, nil, map[string]string{
		"Accept": "application/json;adc.metadata=full",
	}, http.StatusOK)
	if err != nil {
		exitf("GetWithAnnotations failed: %v", err)
	}
	printJSON("GetWithAnnotations", status, payload)

	assetETag := strings.TrimSpace(asString(payload["etag"]))
	if assetETag == "" {
		exitf("get response missing etag")
	}

	annotatePath := strings.TrimSuffix(assetPath, "?api-version="+dataCatalogDataAssetAPIVersion) + "/experts?api-version=" + dataCatalogDataAssetAPIVersion
	annotatePayload := map[string]any{
		"properties": map[string]any{
			"fromSourceSystem": false,
			"key":              "22c3fa019b3945dc97143ebc3ad74cbf--1111fa019b3945dc97143ebc3ad74cbf",
			"expert": map[string]any{
				"upn":      "expert1@contoso.com",
				"objectId": "1111fa019b3945dc97143ebc3ad74cbf",
			},
		},
	}

	payload, status, err = client.doJSON(ctx, http.MethodPost, annotatePath, annotatePayload, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		exitf("Annotate failed: %v", err)
	}
	printJSON("Annotate", status, payload)

	payload, status, err = client.doJSON(ctx, http.MethodGet, assetPath, nil, map[string]string{
		"Accept": "application/json;adc.metadata=full",
	}, http.StatusOK)
	if err != nil {
		exitf("GetAfterAnnotate failed: %v", err)
	}
	printJSON("GetAfterAnnotate", status, payload)

	expertID, expertETag := extractFirstExpert(payload)
	if expertID == "" || expertETag == "" {
		exitf("unable to locate expert annotation id/etag in get response")
	}
	updateExpertPath := toLocalPath(expertID) + "?api-version=" + dataCatalogDataAssetAPIVersion
	updatePayload := map[string]any{
		"etag": expertETag,
		"properties": map[string]any{
			"fromSourceSystem": false,
			"key":              "22c3fa019b3945dc97143ebc3ad74cbf--1111fa019b3945dc97143ebc3ad74cbf",
			"expert": map[string]any{
				"upn":      "expert2@contoso.com",
				"objectId": "1111fa019b3945dc97143ebc3ad74cbf",
			},
		},
	}

	payload, status, err = client.doJSON(ctx, http.MethodPut, updateExpertPath, updatePayload, nil, http.StatusOK)
	if err != nil {
		exitf("UpdateAnnotation failed: %v", err)
	}
	printJSON("UpdateAnnotation", status, payload)

	searchPath := "/azure/catalogs/" + catalogName + "/search/search?searchTerms=" + assetName + "&count=10&startPage=1&api-version=" + dataCatalogDataAssetAPIVersion
	payload, status, err = client.doJSON(ctx, http.MethodGet, searchPath, nil, nil, http.StatusOK)
	if err != nil {
		exitf("Search failed: %v", err)
	}
	printJSON("Search", status, payload)

	payload, status, err = client.doJSON(ctx, http.MethodDelete, updateExpertPath, nil, map[string]string{
		"If-Match": `W/"` + expertETag + `"`,
	}, http.StatusNoContent)
	if err != nil {
		exitf("DeleteAnnotation failed: %v", err)
	}
	printJSON("DeleteAnnotation", status, payload)

	payload, status, err = client.doJSON(ctx, http.MethodDelete, assetPath, nil, map[string]string{
		"If-Match": `W/"` + assetETag + `"`,
	}, http.StatusNoContent)
	if err != nil {
		exitf("DeleteAsset failed: %v", err)
	}
	printJSON("DeleteAsset", status, payload)
}

func extractFirstExpert(payload map[string]any) (string, string) {
	annotations, ok := payload["annotations"].(map[string]any)
	if !ok {
		return "", ""
	}
	experts, ok := annotations["experts"].([]any)
	if !ok || len(experts) == 0 {
		return "", ""
	}
	item, ok := experts[0].(map[string]any)
	if !ok {
		return "", ""
	}
	return strings.TrimSpace(asString(item["id"])), strings.TrimSpace(asString(item["etag"]))
}

func toLocalPath(id string) string {
	path := strings.TrimSpace(id)
	path = strings.TrimPrefix(path, "https://stackyard.local")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
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
