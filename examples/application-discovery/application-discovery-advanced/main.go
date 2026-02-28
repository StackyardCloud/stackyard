package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type rpcCall struct {
	Name    string
	Action  string
	Payload map[string]any
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

	fmt.Printf("Stackyard Application Discovery advanced client using %s\n", endpoint)

	applicationID := ""
	createAppOut, err := runCallWithPayload(ctx, endpoint, region, creds, rpcCall{
		Name:   "CreateApplication",
		Action: "CreateApplication",
		Payload: map[string]any{
			"name":        "advanced-discovery-app",
			"description": "advanced discovery app",
		},
	})
	if err != nil {
		exitf("CreateApplication failed: %v", err)
	}
	applicationID = payloadString(createAppOut, "configurationId")
	if strings.TrimSpace(applicationID) == "" {
		exitf("CreateApplication did not return configurationId")
	}

	calls := []rpcCall{
		{Name: "AssociateConfigurationItemsToApplication", Action: "AssociateConfigurationItemsToApplication", Payload: map[string]any{"applicationConfigurationId": applicationID, "configurationIds": []string{"srv-000001", "srv-000002"}}},
		{Name: "UpdateApplication", Action: "UpdateApplication", Payload: map[string]any{"configurationId": applicationID, "name": "advanced-discovery-app-updated", "description": "updated"}},
		{Name: "CreateTags", Action: "CreateTags", Payload: map[string]any{"configurationIds": []string{applicationID}, "tags": []map[string]string{{"key": "env", "value": "advanced"}, {"key": "owner", "value": "qa"}}}},
		{Name: "DescribeTags", Action: "DescribeTags", Payload: map[string]any{}},
		{Name: "DescribeAgents", Action: "DescribeAgents", Payload: map[string]any{}},
		{Name: "StartDataCollectionByAgentIds", Action: "StartDataCollectionByAgentIds", Payload: map[string]any{"agentIds": []string{"agent-000001"}}},
		{Name: "StopDataCollectionByAgentIds", Action: "StopDataCollectionByAgentIds", Payload: map[string]any{"agentIds": []string{"agent-000001"}}},
		{Name: "BatchDeleteAgents", Action: "BatchDeleteAgents", Payload: map[string]any{"deleteAgents": []map[string]any{{"agentId": "agent-999999"}}}},
		{Name: "StartContinuousExport", Action: "StartContinuousExport", Payload: map[string]any{}},
		{Name: "DescribeContinuousExports", Action: "DescribeContinuousExports", Payload: map[string]any{}},
		{Name: "StopContinuousExport", Action: "StopContinuousExport", Payload: map[string]any{}},
		{Name: "StartExportTask", Action: "StartExportTask", Payload: map[string]any{"exportDataFormat": "CSV"}},
		{Name: "DescribeExportTasks", Action: "DescribeExportTasks", Payload: map[string]any{}},
		{Name: "ExportConfigurations", Action: "ExportConfigurations", Payload: map[string]any{"exportDataFormat": "CSV"}},
		{Name: "DescribeExportConfigurations", Action: "DescribeExportConfigurations", Payload: map[string]any{}},
		{Name: "StartImportTask", Action: "StartImportTask", Payload: map[string]any{"name": "advanced-import", "importUrl": "s3://stackyard/advanced-import.csv", "clientRequestToken": "advanced-import-token"}},
		{Name: "DescribeImportTasks", Action: "DescribeImportTasks", Payload: map[string]any{}},
		{Name: "BatchDeleteImportData", Action: "BatchDeleteImportData", Payload: map[string]any{"importTaskIds": []string{"advanced-import-token"}}},
		{Name: "StartBatchDeleteConfigurationTask", Action: "StartBatchDeleteConfigurationTask", Payload: map[string]any{"configurationType": "SERVER", "configurationIds": []string{"srv-000001"}}},
		{Name: "DescribeBatchDeleteConfigurationTask", Action: "DescribeBatchDeleteConfigurationTask", Payload: map[string]any{}},
		{Name: "DescribeConfigurations", Action: "DescribeConfigurations", Payload: map[string]any{"configurationIds": []string{"srv-000001", applicationID}}},
		{Name: "ListConfigurations", Action: "ListConfigurations", Payload: map[string]any{}},
		{Name: "ListServerNeighbors", Action: "ListServerNeighbors", Payload: map[string]any{"configurationId": "srv-000001"}},
		{Name: "GetDiscoverySummary", Action: "GetDiscoverySummary", Payload: map[string]any{}},
		{Name: "DeleteTags", Action: "DeleteTags", Payload: map[string]any{"configurationIds": []string{applicationID}, "tags": []string{"owner"}}},
		{Name: "DisassociateConfigurationItemsFromApplication", Action: "DisassociateConfigurationItemsFromApplication", Payload: map[string]any{"applicationConfigurationId": applicationID, "configurationIds": []string{"srv-000002"}}},
		{Name: "DeleteApplications", Action: "DeleteApplications", Payload: map[string]any{"configurationIds": []string{applicationID}}},
	}

	for _, call := range calls {
		if _, err := runCallWithPayload(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCallWithPayload(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) (map[string]any, error) {
	status, body, err := discoveryRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return nil, fmt.Errorf("HTTP %d: %s", status, trimmed)
	}

	fmt.Printf("%s returned %d\n", call.Name, status)

	payload := map[string]any{}
	if len(strings.TrimSpace(string(body))) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return payload, nil
}

func discoveryRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte("{}")
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSPoseidonService_V2015_11_01."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "discovery", region, time.Now()); err != nil {
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

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
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
