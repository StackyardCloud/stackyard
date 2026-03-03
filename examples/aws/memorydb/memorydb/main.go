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

type requestCase struct {
	Action  string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	clusterName := getenv("STACKYARD_MEMORYDB_CLUSTER", "memorydb-cluster")
	aclName := getenv("STACKYARD_MEMORYDB_ACL", "memorydb-acl")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard MemoryDB advanced client using %s\n", endpoint)

	requests := []requestCase{
		{
			Action: "CreateUser",
			Payload: map[string]any{
				"UserName":     "memorydb-user",
				"AccessString": "on ~* +@all",
				"AuthenticationMode": map[string]any{
					"Type":      "password",
					"Passwords": []string{"secret"},
				},
			},
		},
		{
			Action: "CreateACL",
			Payload: map[string]any{
				"ACLName":   aclName,
				"UserNames": []string{"memorydb-user"},
			},
		},
		{
			Action: "CreateCluster",
			Payload: map[string]any{
				"ClusterName": clusterName,
				"NodeType":    "db.t4g.small",
				"NumShards":   1,
				"ACLName":     aclName,
			},
		},
		{
			Action: "DescribeClusters",
			Payload: map[string]any{
				"ClusterName": clusterName,
			},
		},
		{
			Action: "UpdateCluster",
			Payload: map[string]any{
				"ClusterName":            clusterName,
				"SnapshotRetentionLimit": 1,
			},
		},
		{Action: "DeleteCluster", Payload: map[string]any{"ClusterName": clusterName}},
		{Action: "DeleteACL", Payload: map[string]any{"ACLName": aclName}},
		{Action: "DeleteUser", Payload: map[string]any{"UserName": "memorydb-user"}},
	}

	for _, req := range requests {
		status, body, err := memorydbRequest(ctx, endpoint, region, creds, req.Action, req.Payload)
		if err != nil {
			exitf("%s request failed: %v", req.Action, err)
		}
		if err := expectSuccess(req.Action, status, body); err != nil {
			exitf("%s response check failed: %v", req.Action, err)
		}
		fmt.Printf("%s succeeded (%d)\n", req.Action, status)
	}

	fmt.Println("Done.")
}

func memorydbRequest(
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
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonMemoryDB."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "memorydb", region, time.Now()); err != nil {
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
	if status == http.StatusOK {
		return nil
	}
	return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
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
