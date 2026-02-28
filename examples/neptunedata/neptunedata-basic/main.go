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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Neptune Data API basic client using %s\n", endpoint)

	status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, http.MethodGet, "/status", nil)
	if err != nil {
		exitf("GetEngineStatus request failed: %v", err)
	}
	if err := expectSuccess("GetEngineStatus", status, body); err != nil {
		exitf("GetEngineStatus response validation failed: %v", err)
	}
	logf("GetEngineStatus succeeded (%d)", status)

	status, body, err = neptuneDataRequest(ctx, endpoint, region, creds, http.MethodPost, "/gremlin", map[string]any{
		"gremlin": "g.V().limit(1)",
	})
	if err != nil {
		exitf("ExecuteGremlinQuery request failed: %v", err)
	}
	if err := expectSuccess("ExecuteGremlinQuery", status, body); err != nil {
		exitf("ExecuteGremlinQuery response validation failed: %v", err)
	}
	logf("ExecuteGremlinQuery succeeded (%d)", status)

	status, body, err = neptuneDataRequest(ctx, endpoint, region, creds, http.MethodGet, "/gremlin/status", nil)
	if err != nil {
		exitf("ListGremlinQueries request failed: %v", err)
	}
	if err := expectSuccess("ListGremlinQueries", status, body); err != nil {
		exitf("ListGremlinQueries response validation failed: %v", err)
	}
	logf("ListGremlinQueries succeeded (%d)", status)

	status, body, err = neptuneDataRequest(ctx, endpoint, region, creds, http.MethodPost, "/loader", map[string]any{
		"source":         "s3://stackyard-neptunedata/basic/input.csv",
		"format":         "csv",
		"s3BucketRegion": "us-east-1",
		"iamRoleArn":     "arn:aws:iam::123456789012:role/stackyard-neptunedata",
	})
	if err != nil {
		exitf("StartLoaderJob request failed: %v", err)
	}
	if err := expectSuccess("StartLoaderJob", status, body); err != nil {
		exitf("StartLoaderJob response validation failed: %v", err)
	}
	logf("StartLoaderJob succeeded (%d)", status)

	status, body, err = neptuneDataRequest(ctx, endpoint, region, creds, http.MethodGet, "/propertygraph/statistics", nil)
	if err != nil {
		exitf("GetPropertygraphStatistics request failed: %v", err)
	}
	if err := expectSuccess("GetPropertygraphStatistics", status, body); err != nil {
		exitf("GetPropertygraphStatistics response validation failed: %v", err)
	}
	logf("GetPropertygraphStatistics succeeded (%d)", status)

	status, body, err = neptuneDataRequest(ctx, endpoint, region, creds, http.MethodPost, "/ml/dataprocessing", map[string]any{
		"id":                      "basic-ml-dp",
		"inputDataS3Location":     "s3://stackyard-neptunedata/basic/raw",
		"processedDataS3Location": "s3://stackyard-neptunedata/basic/processed",
	})
	if err != nil {
		exitf("StartMLDataProcessingJob request failed: %v", err)
	}
	if err := expectSuccess("StartMLDataProcessingJob", status, body); err != nil {
		exitf("StartMLDataProcessingJob response validation failed: %v", err)
	}
	logf("StartMLDataProcessingJob succeeded (%d)", status)

	status, body, err = neptuneDataRequest(ctx, endpoint, region, creds, http.MethodPost, "/system", map[string]any{
		"action": "initiateDatabaseReset",
	})
	if err != nil {
		exitf("ExecuteFastReset request failed: %v", err)
	}
	if err := expectSuccess("ExecuteFastReset", status, body); err != nil {
		exitf("ExecuteFastReset response validation failed: %v", err)
	}
	logf("ExecuteFastReset succeeded (%d)", status)

	fmt.Println("Done.")
}

func neptuneDataRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "neptune-db", region, time.Now()); err != nil {
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

func expectSuccess(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	if strings.Contains(string(body), "NotImplemented") {
		return fmt.Errorf("expected %s to be implemented, got: %s", action, strings.TrimSpace(string(body)))
	}
	return nil
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
