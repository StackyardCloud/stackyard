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
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000)
	keyspace := fmt.Sprintf("%s_%s", getenv("STACKYARD_KEYSPACES_KEYSPACE", "stackyard_keyspace_basic"), suffix)
	table := fmt.Sprintf("%s_%s", getenv("STACKYARD_KEYSPACES_TABLE", "stackyard_table_basic"), suffix)
	tableRestore := table + "_restore"
	typeName := fmt.Sprintf("%s_%s", getenv("STACKYARD_KEYSPACES_TYPE", "stackyard_type_basic"), suffix)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Keyspaces basic client using %s\n", endpoint)

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "CreateKeyspace", map[string]any{
		"keyspaceName": keyspace,
	}); err != nil {
		exitf("CreateKeyspace failed: %v", err)
	}

	createTableOut, err := runKeyspacesAction(ctx, endpoint, region, creds, "CreateTable", map[string]any{
		"keyspaceName": keyspace,
		"tableName":    table,
		"schemaDefinition": map[string]any{
			"allColumns":    []map[string]string{{"name": "pk", "type": "text"}},
			"partitionKeys": []map[string]string{{"name": "pk"}},
		},
		"autoScalingSpecification": map[string]any{
			"readCapacityAutoScaling": map[string]any{"minimumUnits": 1, "maximumUnits": 5},
		},
	})
	if err != nil {
		exitf("CreateTable failed: %v", err)
	}

	tableARN, _ := createTableOut["resourceArn"].(string)
	if tableARN == "" {
		exitf("CreateTable response missing resourceArn")
	}

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"resourceArn": tableARN,
		"tags":        []map[string]string{{"key": "env", "value": "basic"}},
	}); err != nil {
		exitf("TagResource failed: %v", err)
	}

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "ListTagsForResource", map[string]any{
		"resourceArn": tableARN,
		"maxResults":  10,
	}); err != nil {
		exitf("ListTagsForResource failed: %v", err)
	}

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "GetTableAutoScalingSettings", map[string]any{
		"keyspaceName": keyspace,
		"tableName":    table,
	}); err != nil {
		exitf("GetTableAutoScalingSettings failed: %v", err)
	}

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "RestoreTable", map[string]any{
		"sourceKeyspaceName": keyspace,
		"sourceTableName":    table,
		"targetKeyspaceName": keyspace,
		"targetTableName":    tableRestore,
	}); err != nil {
		exitf("RestoreTable failed: %v", err)
	}

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "CreateType", map[string]any{
		"keyspaceName":     keyspace,
		"typeName":         typeName,
		"fieldDefinitions": []map[string]string{{"name": "f1", "type": "text"}},
	}); err != nil {
		exitf("CreateType failed: %v", err)
	}

	if _, err := runKeyspacesAction(ctx, endpoint, region, creds, "ListTypes", map[string]any{
		"keyspaceName": keyspace,
		"maxResults":   10,
	}); err != nil {
		exitf("ListTypes failed: %v", err)
	}

	cleanup := []struct {
		action  string
		payload map[string]any
	}{
		{action: "DeleteType", payload: map[string]any{"keyspaceName": keyspace, "typeName": typeName}},
		{action: "DeleteTable", payload: map[string]any{"keyspaceName": keyspace, "tableName": tableRestore}},
		{action: "DeleteTable", payload: map[string]any{"keyspaceName": keyspace, "tableName": table}},
		{action: "DeleteKeyspace", payload: map[string]any{"keyspaceName": keyspace}},
	}
	for _, step := range cleanup {
		if _, err := runKeyspacesAction(ctx, endpoint, region, creds, step.action, step.payload); err != nil {
			exitf("%s failed during cleanup: %v", step.action, err)
		}
	}

	fmt.Println("Done.")
}

func runKeyspacesAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (map[string]any, error) {
	status, body, err := keyspacesRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return nil, err
	}
	if err := expectOK(action, status, body); err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", action, err)
	}
	return out, nil
}

func keyspacesRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
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
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "KeyspacesService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cassandra", region, time.Now()); err != nil {
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

func expectOK(action string, status int, body []byte) error {
	if status >= 200 && status < 300 {
		fmt.Printf("%s returned %d\n", action, status)
		return nil
	}
	return fmt.Errorf("expected %s to return 2xx, got %d: %s", action, status, strings.TrimSpace(string(body)))
}

func hashSHA256(body []byte) string {
	sum := sha256.Sum256(body)
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
