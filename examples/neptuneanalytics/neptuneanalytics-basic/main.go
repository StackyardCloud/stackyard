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
	graphName := getenv("STACKYARD_NEPTUNEANALYTICS_GRAPH_NAME", "stackyard-neptuneanalytics-basic")
	importSource := getenv("STACKYARD_NEPTUNEANALYTICS_IMPORT_SOURCE", "s3://stackyard-neptuneanalytics-basic/import")
	roleArn := getenv("STACKYARD_NEPTUNEANALYTICS_ROLE_ARN", "arn:aws:iam::123456789012:role/stackyard-neptuneanalytics")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Neptune Analytics API basic client using %s\n", endpoint)

	createStatus, createBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "CreateGraph",
		Method: http.MethodPost,
		Path:   "/graphs",
		Payload: map[string]any{
			"graphName":          graphName,
			"provisionedMemory":  128,
			"publicConnectivity": true,
		},
	})
	if err != nil {
		exitf("CreateGraph request failed: %v", err)
	}
	if err := expectStatus("CreateGraph", createStatus, createBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}

	graphID := jsonStringField(createBody, "id")
	if graphID == "" {
		exitf("CreateGraph response missing id: %s", strings.TrimSpace(string(createBody)))
	}
	logf("CreateGraph succeeded (id=%s)", graphID)

	stage1Requests := []requestCase{
		{Action: "ListGraphs", Method: http.MethodGet, Path: "/graphs?maxResults=25"},
		{Action: "GetGraph", Method: http.MethodGet, Path: "/graphs/" + url.PathEscape(graphID)},
		{
			Action: "UpdateGraph",
			Method: http.MethodPatch,
			Path:   "/graphs/" + url.PathEscape(graphID),
			Payload: map[string]any{
				"publicConnectivity": false,
				"provisionedMemory":  256,
			},
		},
		{Action: "StopGraph", Method: http.MethodPost, Path: "/graphs/" + url.PathEscape(graphID) + "/stop"},
		{Action: "StartGraph", Method: http.MethodPost, Path: "/graphs/" + url.PathEscape(graphID) + "/start"},
	}

	for _, reqCase := range stage1Requests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	startImportStatus, startImportBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "StartImportTask",
		Method: http.MethodPost,
		Path:   "/graphs/" + url.PathEscape(graphID) + "/importtasks",
		Payload: map[string]any{
			"source":  importSource,
			"roleArn": roleArn,
			"format":  "CSV",
		},
	})
	if err != nil {
		exitf("StartImportTask request failed: %v", err)
	}
	if err := expectStatus("StartImportTask", startImportStatus, startImportBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	importTaskID := jsonStringField(startImportBody, "taskId")
	if importTaskID == "" {
		exitf("StartImportTask response missing taskId: %s", strings.TrimSpace(string(startImportBody)))
	}
	logf("StartImportTask succeeded (taskId=%s)", importTaskID)

	stage3Requests := []requestCase{
		{Action: "GetImportTask", Method: http.MethodGet, Path: "/importtasks/" + url.PathEscape(importTaskID)},
		{Action: "ListImportTasks", Method: http.MethodGet, Path: "/importtasks?maxResults=25"},
		{Action: "CancelImportTask", Method: http.MethodDelete, Path: "/importtasks/" + url.PathEscape(importTaskID)},
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

	executeQueryStatus, executeQueryBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "ExecuteQuery",
		Method: http.MethodPost,
		Path:   "/queries",
		Headers: map[string]string{
			"graphIdentifier": graphID,
		},
		Payload: map[string]any{
			"query":    "MATCH (n) RETURN n LIMIT 1",
			"language": "OPEN_CYPHER",
		},
	})
	if err != nil {
		exitf("ExecuteQuery request failed: %v", err)
	}
	if err := expectStatus("ExecuteQuery", executeQueryStatus, executeQueryBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	queryID := jsonStringField(executeQueryBody, "queryId")
	if queryID == "" {
		exitf("ExecuteQuery response missing queryId: %s", strings.TrimSpace(string(executeQueryBody)))
	}
	logf("ExecuteQuery succeeded (queryId=%s)", queryID)

	stage4Requests := []requestCase{
		{
			Action: "GetQuery",
			Method: http.MethodGet,
			Path:   "/queries/" + url.PathEscape(queryID),
			Headers: map[string]string{
				"graphIdentifier": graphID,
			},
		},
		{
			Action: "ListQueries",
			Method: http.MethodGet,
			Path:   "/queries?maxResults=25",
			Headers: map[string]string{
				"graphIdentifier": graphID,
			},
		},
		{
			Action: "CancelQuery",
			Method: http.MethodDelete,
			Path:   "/queries/" + url.PathEscape(queryID),
			Headers: map[string]string{
				"graphIdentifier": graphID,
			},
		},
		{
			Action: "GetGraphSummary",
			Method: http.MethodGet,
			Path:   "/summary?mode=BASIC",
			Headers: map[string]string{
				"graphIdentifier": graphID,
			},
		},
	}

	for _, reqCase := range stage4Requests {
		status, body, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, reqCase)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectStatus(reqCase.Action, status, body, http.StatusOK); err != nil {
			exitf("%v", err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	deleteStatus, deleteBody, err := neptuneAnalyticsRequest(ctx, endpoint, region, creds, requestCase{
		Action: "DeleteGraph",
		Method: http.MethodDelete,
		Path:   "/graphs/" + url.PathEscape(graphID) + "?skipSnapshot=true",
	})
	if err != nil {
		exitf("DeleteGraph request failed: %v", err)
	}
	if err := expectStatus("DeleteGraph", deleteStatus, deleteBody, http.StatusOK); err != nil {
		exitf("%v", err)
	}
	logf("DeleteGraph succeeded (%d)", deleteStatus)

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
