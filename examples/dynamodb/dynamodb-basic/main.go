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
	tableName := getenv("STACKYARD_DDB_TABLE", "stackyard-ddb-basic")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard DynamoDB basic client using %s\n", endpoint)

	createTable := map[string]any{
		"TableName": tableName,
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
		},
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	}
	if err := callDynamoDB(ctx, endpoint, region, creds, "CreateTable", createTable); err != nil {
		exitf("CreateTable: %v", err)
	}

	putItem := map[string]any{
		"TableName": tableName,
		"Item": map[string]any{
			"pk":     map[string]string{"S": "user#1"},
			"name":   map[string]string{"S": "Stackyard"},
			"status": map[string]string{"S": "ACTIVE"},
		},
	}
	if err := callDynamoDB(ctx, endpoint, region, creds, "PutItem", putItem); err != nil {
		exitf("PutItem: %v", err)
	}

	getItem := map[string]any{
		"TableName": tableName,
		"Key": map[string]any{
			"pk": map[string]string{"S": "user#1"},
		},
		"ConsistentRead": true,
	}
	if err := callDynamoDB(ctx, endpoint, region, creds, "GetItem", getItem); err != nil {
		exitf("GetItem: %v", err)
	}

	deleteTable := map[string]any{"TableName": tableName}
	if err := callDynamoDB(ctx, endpoint, region, creds, "DeleteTable", deleteTable); err != nil {
		exitf("DeleteTable: %v", err)
	}

	fmt.Println("Done.")
}

func callDynamoDB(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) error {
	status, body, err := dynamodbRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return fmt.Errorf("expected %s to return 2xx, got %d: %s", action, status, strings.TrimSpace(string(body)))
	}
	fmt.Printf("%s returned %d\n", action, status)
	return nil
}

func dynamodbRequest(
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
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "dynamodb", region, time.Now()); err != nil {
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
