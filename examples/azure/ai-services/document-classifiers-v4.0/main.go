package main

import (
	"bytes"
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

const documentClassifiersAPIVersion = "2024-11-30"

type documentClassifiersClient struct {
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

func newDocumentClassifiersClient(endpoint, account, subscriptionKey string) *documentClassifiersClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"ai-services-data-plane-document-classifiers-v4.0",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{&sharedKeyAndSubscriptionPolicy{account: account, subscriptionKey: subscriptionKey}},
		},
		&policy.ClientOptions{},
	)
	return &documentClassifiersClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *documentClassifiersClient) doRequest(ctx context.Context, method, path string, payload any, contentType string, expectedStatuses ...int) ([]byte, int, http.Header, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")

	if payload != nil {
		if body, ok := payload.([]byte); ok {
			contentType = strings.TrimSpace(contentType)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			req.Raw().Header.Set("Content-Type", contentType)
			req.Raw().Body = io.NopCloser(bytes.NewReader(body))
			req.Raw().ContentLength = int64(len(body))
		} else {
			if err := runtime.MarshalAsJSON(req, payload); err != nil {
				return nil, 0, nil, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
			}
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, resp.Header, fmt.Errorf("read response body %s %s: %w", method, path, readErr)
	}

	if len(expectedStatuses) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, resp.StatusCode, resp.Header, nil
		}
		return body, resp.StatusCode, resp.Header, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	for _, expected := range expectedStatuses {
		if resp.StatusCode == expected {
			return body, resp.StatusCode, resp.Header, nil
		}
	}
	return body, resp.StatusCode, resp.Header, fmt.Errorf("unexpected status %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func (c *documentClassifiersClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, http.Header, error) {
	body, status, headers, err := c.doRequest(ctx, method, path, payload, "application/json", expectedStatuses...)
	if err != nil {
		return nil, status, headers, err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return map[string]any{}, status, headers, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, status, headers, fmt.Errorf("decode JSON body for %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, status, headers, nil
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_AISERVICES_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_AISERVICES_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	classifierID := getenv("STACKYARD_AZURE_AISERVICES_CLASSIFIER_ID", "invoice-classifier")
	targetClassifierID := getenv("STACKYARD_AZURE_AISERVICES_TARGET_CLASSIFIER_ID", classifierID+"-copy")
	resultID := getenv("STACKYARD_AZURE_AISERVICES_RESULT_ID", "result-1")
	documentURL := getenv("STACKYARD_AZURE_AISERVICES_DOCUMENT_URL", "https://example.com/invoice.pdf")

	fmt.Printf("Stackyard Azure AI Services - Data Plane - Document Classifiers (ai-services-data-plane-document-classifiers-v4.0) example using %s\n", strings.TrimRight(endpoint, "/"))

	client := newDocumentClassifiersClient(endpoint, account, subscriptionKey)

	authorizeCopyPath := "/azure/aiservices/documentintelligence/documentClassifiers:authorizeCopy?api-version=" + documentClassifiersAPIVersion
	authorizeCopyResp, status, _, err := client.doJSON(ctx, http.MethodPost, authorizeCopyPath, map[string]any{
		"classifierId": targetClassifierID,
		"description":  "copy authorization for stackyard local workflow",
		"tags": map[string]string{
			"env": "local",
		},
	}, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("AuthorizeClassifierCopy failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(authorizeCopyPath)
		return
	}

	buildPath := "/azure/aiservices/documentintelligence/documentClassifiers:build?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodPost, buildPath, map[string]any{
		"classifierId": classifierID,
		"description":  "stackyard local document classifier",
		"docTypes": map[string]any{
			"invoice": map[string]any{
				"azureBlobSource": map[string]any{
					"containerUrl": "https://example.blob.core.windows.net/document-classifier-data",
					"prefix":       "invoices",
				},
			},
		},
	}, http.StatusAccepted, http.StatusNotImplemented)
	if err != nil {
		exitf("BuildClassifier failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(buildPath)
		return
	}

	analyzePath := "/azure/aiservices/documentintelligence/documentClassifiers/" + classifierID + ":analyze?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodPost, analyzePath, map[string]any{
		"urlSource": documentURL,
	}, http.StatusAccepted, http.StatusNotImplemented)
	if err != nil {
		exitf("ClassifyDocument failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(analyzePath)
		return
	}

	_, status, _, err = client.doRequest(ctx, http.MethodPost, analyzePath, []byte("%PDF-1.7 stackyard document stream"), "application/pdf", http.StatusAccepted, http.StatusNotImplemented)
	if err != nil {
		exitf("ClassifyDocumentFromStream failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(analyzePath)
		return
	}

	copyToPath := "/azure/aiservices/documentintelligence/documentClassifiers/" + classifierID + ":copyTo?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodPost, copyToPath, map[string]any{
		"targetResourceId":     "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/example-rg/providers/Microsoft.CognitiveServices/accounts/stackyard-target",
		"targetResourceRegion": "eastus",
		"targetClassifierId":   targetClassifierID,
		"accessToken":          fallbackString(authorizeCopyResp["accessToken"], "stackyard-copy-token"),
		"expirationDateTime":   "2099-01-01T00:00:00Z",
	}, http.StatusAccepted, http.StatusNotImplemented)
	if err != nil {
		exitf("CopyClassifierTo failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(copyToPath)
		return
	}

	getClassifierPath := "/azure/aiservices/documentintelligence/documentClassifiers/" + classifierID + "?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodGet, getClassifierPath, nil, http.StatusOK, http.StatusNotFound, http.StatusNotImplemented)
	if err != nil {
		exitf("GetClassifier failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getClassifierPath)
		return
	}

	listClassifiersPath := "/azure/aiservices/documentintelligence/documentClassifiers?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodGet, listClassifiersPath, nil, http.StatusOK, http.StatusNotImplemented)
	if err != nil {
		exitf("ListClassifiers failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(listClassifiersPath)
		return
	}

	getResultPath := "/azure/aiservices/documentintelligence/documentClassifiers/" + classifierID + "/analyzeResults/" + resultID + "?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doJSON(ctx, http.MethodGet, getResultPath, nil, http.StatusOK, http.StatusNotFound, http.StatusNotImplemented)
	if err != nil {
		exitf("GetClassifyResult failed: %v", err)
	}
	if status == http.StatusNotImplemented {
		notImplemented(getResultPath)
		return
	}

	deletePath := "/azure/aiservices/documentintelligence/documentClassifiers/" + classifierID + "?api-version=" + documentClassifiersAPIVersion
	_, status, _, err = client.doRequest(ctx, http.MethodDelete, deletePath, nil, "", http.StatusNoContent, http.StatusNotFound, http.StatusNotImplemented)
	if err != nil {
		exitf("DeleteClassifier failed: %v", err)
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

func fallbackString(value any, fallback string) string {
	text, _ := value.(string)
	if strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	return fallback
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
