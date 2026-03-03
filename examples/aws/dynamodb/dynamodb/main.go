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

type createTableResponse struct {
	TableDescription struct {
		TableARN string `json:"TableArn"`
	} `json:"TableDescription"`
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	accountID := getenv("AWS_ACCOUNT_ID", "123456789012")
	tableName := getenv("STACKYARD_DDB_TABLE", "stackyard-ddb")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard DynamoDB advanced client using %s\n", endpoint)

	createTable := map[string]any{
		"TableName": tableName,
		"AttributeDefinitions": []map[string]string{
			{"AttributeName": "pk", "AttributeType": "S"},
			{"AttributeName": "sk", "AttributeType": "S"},
		},
		"KeySchema": []map[string]string{
			{"AttributeName": "pk", "KeyType": "HASH"},
			{"AttributeName": "sk", "KeyType": "RANGE"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	}
	status, body, err := dynamodbRequest(ctx, endpoint, region, creds, "CreateTable", createTable)
	if err != nil {
		exitf("CreateTable: %v", err)
	}
	expect2xx("CreateTable", status, body)

	tableARN := fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", region, accountID, tableName)
	var created createTableResponse
	if err := json.Unmarshal(body, &created); err == nil && created.TableDescription.TableARN != "" {
		tableARN = created.TableDescription.TableARN
	}

	batchWrite := map[string]any{
		"RequestItems": map[string]any{
			tableName: []map[string]any{
				{
					"PutRequest": map[string]any{
						"Item": map[string]any{
							"pk":     map[string]string{"S": "acct#1"},
							"sk":     map[string]string{"S": "order#1"},
							"amount": map[string]string{"N": "100"},
							"status": map[string]string{"S": "PENDING"},
						},
					},
				},
				{
					"PutRequest": map[string]any{
						"Item": map[string]any{
							"pk":     map[string]string{"S": "acct#1"},
							"sk":     map[string]string{"S": "order#2"},
							"amount": map[string]string{"N": "65"},
							"status": map[string]string{"S": "PENDING"},
						},
					},
				},
			},
		},
	}
	mustCall(ctx, endpoint, region, creds, "BatchWriteItem", batchWrite)

	queryPayload := map[string]any{
		"TableName":              tableName,
		"KeyConditionExpression": "pk = :pk",
		"ExpressionAttributeValues": map[string]any{
			":pk": map[string]string{"S": "acct#1"},
		},
	}
	mustCall(ctx, endpoint, region, creds, "Query", queryPayload)

	transactWrite := map[string]any{
		"TransactItems": []map[string]any{
			{
				"Update": map[string]any{
					"TableName": tableName,
					"Key": map[string]any{
						"pk": map[string]string{"S": "acct#1"},
						"sk": map[string]string{"S": "order#1"},
					},
					"UpdateExpression": "SET #s = :s",
					"ExpressionAttributeNames": map[string]string{
						"#s": "status",
					},
					"ExpressionAttributeValues": map[string]any{
						":s": map[string]string{"S": "PAID"},
					},
				},
			},
			{
				"Put": map[string]any{
					"TableName": tableName,
					"Item": map[string]any{
						"pk":     map[string]string{"S": "acct#1"},
						"sk":     map[string]string{"S": "order#3"},
						"amount": map[string]string{"N": "125"},
						"status": map[string]string{"S": "PAID"},
					},
				},
			},
		},
	}
	mustCall(ctx, endpoint, region, creds, "TransactWriteItems", transactWrite)

	transactGet := map[string]any{
		"TransactItems": []map[string]any{
			{
				"Get": map[string]any{
					"TableName": tableName,
					"Key": map[string]any{
						"pk": map[string]string{"S": "acct#1"},
						"sk": map[string]string{"S": "order#1"},
					},
				},
			},
			{
				"Get": map[string]any{
					"TableName": tableName,
					"Key": map[string]any{
						"pk": map[string]string{"S": "acct#1"},
						"sk": map[string]string{"S": "order#2"},
					},
				},
			},
		},
	}
	mustCall(ctx, endpoint, region, creds, "TransactGetItems", transactGet)

	tagPayload := map[string]any{
		"ResourceArn": tableARN,
		"Tags": []map[string]string{
			{"Key": "env", "Value": "dev"},
			{"Key": "owner", "Value": "stackyard"},
		},
	}
	mustCall(ctx, endpoint, region, creds, "TagResource", tagPayload)

	mustCall(ctx, endpoint, region, creds, "ListTagsOfResource", map[string]any{"ResourceArn": tableARN})
	mustCall(ctx, endpoint, region, creds, "UntagResource", map[string]any{"ResourceArn": tableARN, "TagKeys": []string{"owner"}})

	ttlPayload := map[string]any{
		"TableName": tableName,
		"TimeToLiveSpecification": map[string]any{
			"Enabled":       true,
			"AttributeName": "expiresAt",
		},
	}
	mustCall(ctx, endpoint, region, creds, "UpdateTimeToLive", ttlPayload)

	mustCall(ctx, endpoint, region, creds, "DeleteTable", map[string]any{"TableName": tableName})

	fmt.Println("Done.")
}

func mustCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, action string, payload map[string]any) {
	status, body, err := dynamodbRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	expect2xx(action, status, body)
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

func expect2xx(action string, status int, body []byte) {
	if status >= 200 && status < 300 {
		fmt.Printf("%s returned %d\n", action, status)
		return
	}
	exitf("expected %s to return 2xx, got %d: %s", action, status, strings.TrimSpace(string(body)))
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
