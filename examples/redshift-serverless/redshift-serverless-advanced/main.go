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

	fmt.Printf("Stackyard Redshift Serverless advanced client using %s\n", endpoint)

	namespaceName := "stage-redshiftserverless-ns"
	workgroupName := "stage-redshiftserverless-wg"
	snapshotName := "stage-redshiftserverless-snapshot"
	resourceARN := "arn:aws:redshift-serverless:us-east-1:123456789012:workgroup/" + workgroupName

	calls := []rpcCall{
		{Name: "CreateNamespace", Action: "CreateNamespace", Payload: map[string]any{"namespaceName": namespaceName, "adminUsername": "admin", "dbName": "dev"}},
		{Name: "CreateWorkgroup", Action: "CreateWorkgroup", Payload: map[string]any{"workgroupName": workgroupName, "namespaceName": namespaceName, "baseCapacity": 64}},
		{Name: "ListNamespaces", Action: "ListNamespaces", Payload: map[string]any{}},
		{Name: "ListWorkgroups", Action: "ListWorkgroups", Payload: map[string]any{}},
		{Name: "GetCredentials", Action: "GetCredentials", Payload: map[string]any{"workgroupName": workgroupName, "dbUser": "admin"}},
		{Name: "CreateSnapshot", Action: "CreateSnapshot", Payload: map[string]any{"namespaceName": namespaceName, "snapshotName": snapshotName}},
		{Name: "ListSnapshots", Action: "ListSnapshots", Payload: map[string]any{}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"resourceArn": resourceARN, "tags": []map[string]any{{"key": "env", "value": "stage"}}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"resourceArn": resourceARN}},
		{Name: "ListTracks", Action: "ListTracks", Payload: map[string]any{}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) error {
	status, body, err := redshiftServerlessRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return fmt.Errorf("HTTP %d: %s", status, trimmed)
	}
	logf("%s returned %d", call.Name, status)
	return nil
}

func redshiftServerlessRequest(
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
	req.Header.Set("X-Amz-Target", "RedshiftServerless."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "redshift-serverless", region, time.Now()); err != nil {
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
