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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type requestCase struct {
	Action  string
	Method  string
	Path    string
	Headers map[string]string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	sourceGraphName := getenv("STACKYARD_NEPTUNEANALYTICS_SOURCE_GRAPH_NAME", "stackyard-neptuneanalytics-advanced-source")
	restoredGraphName := getenv("STACKYARD_NEPTUNEANALYTICS_RESTORED_GRAPH_NAME", "stackyard-neptuneanalytics-advanced-restored")
	importGraphName := getenv("STACKYARD_NEPTUNEANALYTICS_IMPORT_GRAPH_NAME", "stackyard-neptuneanalytics-advanced-import")
	snapshotName := getenv("STACKYARD_NEPTUNEANALYTICS_SNAPSHOT_NAME", "stackyard-neptuneanalytics-advanced-snapshot")
	importSource := getenv("STACKYARD_NEPTUNEANALYTICS_IMPORT_SOURCE", "s3://stackyard-neptuneanalytics-advanced/import")
	exportDestination := getenv("STACKYARD_NEPTUNEANALYTICS_EXPORT_DESTINATION", "s3://stackyard-neptuneanalytics-advanced/export")
	roleArn := getenv("STACKYARD_NEPTUNEANALYTICS_ROLE_ARN", "arn:aws:iam::123456789012:role/stackyard-neptuneanalytics")
	kmsKeyIdentifier := getenv("STACKYARD_NEPTUNEANALYTICS_KMS_KEY_IDENTIFIER", "alias/aws/neptune-graph")
	privateEndpointVpcID := getenv("STACKYARD_NEPTUNEANALYTICS_PRIVATE_ENDPOINT_VPC_ID", "vpc-0123456789abcdef0")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Neptune Analytics API advanced client using %s\n", endpoint)

	createSourceStatus, createSourceBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "CreateGraph",
		Method: http.MethodPost,
		Path:   "/graphs",
		Payload: map[string]any{
			"graphName":          sourceGraphName,
			"provisionedMemory":  128,
			"publicConnectivity": true,
			"replicaCount":       1,
		},
	})
	if err != nil {
		exitf("CreateGraph(source) failed: %v", err)
	}
	if err := expectStatus("CreateGraph(source)", createSourceStatus, createSourceBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	sourceGraphID := jsonStringField(createSourceBody, "id")
	sourceGraphARN := jsonStringField(createSourceBody, "arn")
	if sourceGraphID == "" || sourceGraphARN == "" {
		exitf("CreateGraph(source) response missing id/arn: %s", strings.TrimSpace(string(createSourceBody)))
	}
	logf("Created source graph: %s", sourceGraphID)

	createSnapshotStatus, createSnapshotBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "CreateGraphSnapshot",
		Method: http.MethodPost,
		Path:   "/snapshots",
		Payload: map[string]any{
			"graphIdentifier": sourceGraphID,
			"snapshotName":    snapshotName,
			"tags": map[string]string{
				"env":  "advanced",
				"team": "platform",
			},
		},
	})
	if err != nil {
		exitf("CreateGraphSnapshot failed: %v", err)
	}
	if err := expectStatus("CreateGraphSnapshot", createSnapshotStatus, createSnapshotBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	snapshotID := jsonStringField(createSnapshotBody, "id")
	if snapshotID == "" {
		exitf("CreateGraphSnapshot response missing id: %s", strings.TrimSpace(string(createSnapshotBody)))
	}
	logf("Created snapshot: %s", snapshotID)

	restoreStatus, restoreBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "RestoreGraphFromSnapshot",
		Method: http.MethodPost,
		Path:   "/snapshots/" + url.PathEscape(snapshotID) + "/restore",
		Payload: map[string]any{
			"graphName":          restoredGraphName,
			"provisionedMemory":  128,
			"publicConnectivity": false,
		},
	})
	if err != nil {
		exitf("RestoreGraphFromSnapshot failed: %v", err)
	}
	if err := expectStatus("RestoreGraphFromSnapshot", restoreStatus, restoreBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	restoredGraphID := jsonStringField(restoreBody, "id")
	if restoredGraphID == "" {
		exitf("RestoreGraphFromSnapshot response missing id: %s", strings.TrimSpace(string(restoreBody)))
	}
	logf("Restored graph: %s", restoredGraphID)

	startExportStatus, startExportBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "StartExportTask",
		Method: http.MethodPost,
		Path:   "/exporttasks",
		Payload: map[string]any{
			"graphIdentifier":  sourceGraphID,
			"roleArn":          roleArn,
			"format":           "CSV",
			"destination":      exportDestination,
			"kmsKeyIdentifier": kmsKeyIdentifier,
			"parquetType":      "COLUMNAR",
		},
	})
	if err != nil {
		exitf("StartExportTask failed: %v", err)
	}
	if err := expectStatus("StartExportTask", startExportStatus, startExportBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	exportTaskID := jsonStringField(startExportBody, "taskId")
	if exportTaskID == "" {
		exitf("StartExportTask response missing taskId: %s", strings.TrimSpace(string(startExportBody)))
	}
	logf("Started export task: %s", exportTaskID)

	stage3Requests := []requestCase{
		{Action: "GetExportTask", Method: http.MethodGet, Path: "/exporttasks/" + url.PathEscape(exportTaskID)},
		{Action: "ListExportTasks", Method: http.MethodGet, Path: "/exporttasks?maxResults=25"},
		{Action: "CancelExportTask", Method: http.MethodDelete, Path: "/exporttasks/" + url.PathEscape(exportTaskID)},
	}
	for _, reqCase := range stage3Requests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	createGraphUsingImportTaskStatus, createGraphUsingImportTaskBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "CreateGraphUsingImportTask",
		Method: http.MethodPost,
		Path:   "/importtasks",
		Payload: map[string]any{
			"graphName":            importGraphName,
			"source":               importSource,
			"roleArn":              roleArn,
			"format":               "CSV",
			"maxProvisionedMemory": 128,
		},
	})
	if err != nil {
		exitf("CreateGraphUsingImportTask failed: %v", err)
	}
	if err := expectStatus("CreateGraphUsingImportTask", createGraphUsingImportTaskStatus, createGraphUsingImportTaskBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	importTaskID := jsonStringField(createGraphUsingImportTaskBody, "taskId")
	importGraphID := jsonStringField(createGraphUsingImportTaskBody, "graphId")
	if importTaskID == "" || importGraphID == "" {
		exitf("CreateGraphUsingImportTask response missing taskId/graphId: %s", strings.TrimSpace(string(createGraphUsingImportTaskBody)))
	}
	logf("Created graph via import task: graphId=%s taskId=%s", importGraphID, importTaskID)

	status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "GetImportTask",
		Method: http.MethodGet,
		Path:   "/importtasks/" + url.PathEscape(importTaskID),
	})
	if err != nil {
		exitf("GetImportTask request failed: %v", err)
	}
	if err := expectStatus("GetImportTask", status, body, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	logf("GetImportTask succeeded (%d)", status)

	createEndpointStatus, createEndpointBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "CreatePrivateGraphEndpoint",
		Method: http.MethodPost,
		Path:   "/graphs/" + url.PathEscape(sourceGraphID) + "/endpoints",
		Payload: map[string]any{
			"vpcId":               privateEndpointVpcID,
			"subnetIds":           []string{"subnet-11111111", "subnet-22222222"},
			"vpcSecurityGroupIds": []string{"sg-12345678"},
		},
	})
	if err != nil {
		exitf("CreatePrivateGraphEndpoint failed: %v", err)
	}
	if err := expectStatus("CreatePrivateGraphEndpoint", createEndpointStatus, createEndpointBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	endpointVpcID := jsonStringField(createEndpointBody, "vpcId")
	if endpointVpcID == "" {
		exitf("CreatePrivateGraphEndpoint response missing vpcId: %s", strings.TrimSpace(string(createEndpointBody)))
	}
	logf("Created private endpoint for VPC: %s", endpointVpcID)

	stage5Requests := []requestCase{
		{
			Action: "GetPrivateGraphEndpoint",
			Method: http.MethodGet,
			Path:   "/graphs/" + url.PathEscape(sourceGraphID) + "/endpoints/" + url.PathEscape(endpointVpcID),
		},
		{
			Action: "ListPrivateGraphEndpoints",
			Method: http.MethodGet,
			Path:   "/graphs/" + url.PathEscape(sourceGraphID) + "/endpoints?maxResults=25",
		},
	}
	for _, reqCase := range stage5Requests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	escapedGraphARN := url.PathEscape(sourceGraphARN)
	status, body, err = neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "TagResource",
		Method: http.MethodPost,
		Path:   "/tags/" + escapedGraphARN,
		Payload: map[string]any{
			"tags": map[string]string{
				"env":  "advanced",
				"team": "platform",
			},
		},
	})
	if err != nil {
		exitf("TagResource request failed: %v", err)
	}
	if err := expectStatus("TagResource", status, body, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	logf("TagResource succeeded (%d)", status)

	tagRequests := []requestCase{
		{Action: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + escapedGraphARN},
		{Action: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + escapedGraphARN + "?tagKeys=team"},
		{Action: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + escapedGraphARN},
	}
	for _, reqCase := range tagRequests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	stage46Requests := []requestCase{
		{
			Action: "GetGraphSummary",
			Method: http.MethodGet,
			Path:   "/summary?mode=DETAILED",
			Headers: map[string]string{
				"graphIdentifier": sourceGraphID,
			},
		},
		{
			Action: "ResetGraph",
			Method: http.MethodPut,
			Path:   "/graphs/" + url.PathEscape(sourceGraphID),
			Payload: map[string]any{
				"skipSnapshot": true,
			},
		},
	}
	for _, reqCase := range stage46Requests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	cleanupRequests := []requestCase{
		{
			Action: "DeletePrivateGraphEndpoint",
			Method: http.MethodDelete,
			Path:   "/graphs/" + url.PathEscape(sourceGraphID) + "/endpoints/" + url.PathEscape(endpointVpcID),
		},
		{Action: "DeleteGraphSnapshot", Method: http.MethodDelete, Path: "/snapshots/" + url.PathEscape(snapshotID)},
		{Action: "DeleteGraph(source)", Method: http.MethodDelete, Path: "/graphs/" + url.PathEscape(sourceGraphID) + "?skipSnapshot=true"},
		{Action: "DeleteGraph(restored)", Method: http.MethodDelete, Path: "/graphs/" + url.PathEscape(restoredGraphID) + "?skipSnapshot=true"},
		{Action: "DeleteGraph(import)", Method: http.MethodDelete, Path: "/graphs/" + url.PathEscape(importGraphID) + "?skipSnapshot=true"},
	}
	for _, reqCase := range cleanupRequests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

func neptuneAnalyticsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	reqCase requestCase,
) (int, []byte, error) {
	var body []byte
	if reqCase.Payload != nil {
		encoded, err := json.Marshal(reqCase.Payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	req, err := http.NewRequestWithContext(
		ctx,
		reqCase.Method,
		strings.TrimRight(endpoint, "/")+reqCase.Path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Accept", "application/json")
	if reqCase.Payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range reqCase.Headers {
		req.Header.Set(k, v)
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "neptune-graph", region, time.Now()); err != nil {
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

func expectStatus(action string, status int, body []byte, want int) error {
	if status != want {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, want, status, strings.TrimSpace(string(body)))
	}
	return nil
}

func jsonStringField(body []byte, key string) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
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
