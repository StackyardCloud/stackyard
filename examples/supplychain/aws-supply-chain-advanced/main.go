package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
	Name   string
	Method string
	Path   string
	Body   []byte
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AWS Supply Chain advanced client using %s\n", endpoint)

	instanceID := "scn-instance-000001"
	namespace := "example-namespace"
	dataset := "example-dataset"
	flow := "example-flow"
	resourceArn := url.PathEscape("arn:aws:scn:us-east-1:123456789012:instance/" + instanceID)

	calls := []apiCall{
		{Name: "CreateInstance", Method: http.MethodPost, Path: "/api/instance", Body: []byte(`{"instanceName":"example-supply-chain","clientToken":"example-token-000001"}`)},
		{Name: "GetInstance", Method: http.MethodGet, Path: "/api/instance/" + url.PathEscape(instanceID)},
		{Name: "ListInstances", Method: http.MethodGet, Path: "/api/instance?instanceNameFilter=stackyard&instanceStateFilter=Active&maxResults=10"},
		{Name: "UpdateInstance", Method: http.MethodPatch, Path: "/api/instance/" + url.PathEscape(instanceID), Body: []byte(`{"instanceDescription":"updated by advanced example"}`)},
		{Name: "CreateDataLakeNamespace", Method: http.MethodPut, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace), Body: []byte(`{}`)},
		{Name: "GetDataLakeNamespace", Method: http.MethodGet, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace)},
		{Name: "ListDataLakeNamespaces", Method: http.MethodGet, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces?maxResults=10"},
		{Name: "UpdateDataLakeNamespace", Method: http.MethodPatch, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace), Body: []byte(`{}`)},
		{Name: "CreateDataLakeDataset", Method: http.MethodPut, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace) + "/datasets/" + url.PathEscape(dataset), Body: []byte(`{}`)},
		{Name: "GetDataLakeDataset", Method: http.MethodGet, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace) + "/datasets/" + url.PathEscape(dataset)},
		{Name: "ListDataLakeDatasets", Method: http.MethodGet, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace) + "/datasets?maxResults=10"},
		{Name: "UpdateDataLakeDataset", Method: http.MethodPatch, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace) + "/datasets/" + url.PathEscape(dataset), Body: []byte(`{}`)},
		{Name: "CreateDataIntegrationFlow", Method: http.MethodPut, Path: "/api/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows/" + url.PathEscape(flow), Body: []byte(`{}`)},
		{Name: "GetDataIntegrationFlow", Method: http.MethodGet, Path: "/api/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows/" + url.PathEscape(flow)},
		{Name: "ListDataIntegrationFlows", Method: http.MethodGet, Path: "/api/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows?maxResults=10"},
		{Name: "UpdateDataIntegrationFlow", Method: http.MethodPatch, Path: "/api/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows/" + url.PathEscape(flow), Body: []byte(`{}`)},
		{Name: "SendDataIntegrationEvent", Method: http.MethodPost, Path: "/api-data/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-events", Body: []byte(`{"eventType":"DataSetLoad"}`)},
		{Name: "GetDataIntegrationEvent", Method: http.MethodGet, Path: "/api-data/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-events/event-000001"},
		{Name: "ListDataIntegrationEvents", Method: http.MethodGet, Path: "/api-data/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-events?eventType=DataSetLoad&maxResults=10"},
		{Name: "GetDataIntegrationFlowExecution", Method: http.MethodGet, Path: "/api-data/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows/" + url.PathEscape(flow) + "/executions/exec-000001"},
		{Name: "ListDataIntegrationFlowExecutions", Method: http.MethodGet, Path: "/api-data/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows/" + url.PathEscape(flow) + "/executions?maxResults=10"},
		{Name: "CreateBillOfMaterialsImportJob", Method: http.MethodPost, Path: "/api/configuration/instances/" + url.PathEscape(instanceID) + "/bill-of-materials-import-jobs", Body: []byte(`{}`)},
		{Name: "GetBillOfMaterialsImportJob", Method: http.MethodGet, Path: "/api/configuration/instances/" + url.PathEscape(instanceID) + "/bill-of-materials-import-jobs/bom-000001"},
		{Name: "TagResource", Method: http.MethodPost, Path: "/api/tags/" + resourceArn, Body: []byte(`{"tags":{"env":"example","team":"qa"}}`)},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/api/tags/" + resourceArn},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/api/tags/" + resourceArn + "?tagKeys=team"},
		{Name: "DeleteDataIntegrationFlow", Method: http.MethodDelete, Path: "/api/data-integration/instance/" + url.PathEscape(instanceID) + "/data-integration-flows/" + url.PathEscape(flow)},
		{Name: "DeleteDataLakeDataset", Method: http.MethodDelete, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace) + "/datasets/" + url.PathEscape(dataset)},
		{Name: "DeleteDataLakeNamespace", Method: http.MethodDelete, Path: "/api/datalake/instance/" + url.PathEscape(instanceID) + "/namespaces/" + url.PathEscape(namespace)},
		{Name: "DeleteInstance", Method: http.MethodDelete, Path: "/api/instance/" + url.PathEscape(instanceID)},
	}

	for _, call := range calls {
		status, body, err := supplyChainRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Body)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status != http.StatusOK {
			exitf("%s expected 200, got %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		if strings.Contains(string(body), "NotImplemented") {
			exitf("%s returned NotImplemented: %s", call.Name, strings.TrimSpace(string(body)))
		}

		logf("%s returned %d", call.Name, status)
		trimmed := strings.TrimSpace(string(body))
		if trimmed != "" {
			fmt.Println(trimmed)
		}
	}

	fmt.Println("Done.")
}

func supplyChainRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	body []byte,
) (int, []byte, error) {
	base, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return 0, nil, err
	}
	cleanPath, rawQuery, _ := strings.Cut(path, "?")
	base.Path = cleanPath
	base.RawQuery = rawQuery

	if body == nil {
		body = []byte{}
	}
	req, err := http.NewRequestWithContext(ctx, method, base.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "scn", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
