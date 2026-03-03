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
	clusterName := getenv("STACKYARD_CLUSTER_NAME", "eks-basic-cluster")
	roleARN := getenv("STACKYARD_ROLE_ARN", "arn:aws:iam::123456789012:role/stackyard-eks")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard EKS basic client using %s\n", endpoint)

	createPayload := map[string]any{
		"name":    clusterName,
		"roleArn": roleARN,
		"resourcesVpcConfig": map[string]any{
			"subnetIds":            []string{"subnet-12345678"},
			"endpointPublicAccess": true,
		},
	}
	status, body, err := eksRequest(ctx, endpoint, region, creds, http.MethodPost, "/clusters", createPayload)
	if err != nil {
		exitf("create cluster: %v", err)
	}
	expectStatus("CreateCluster", status, body, http.StatusOK)
	logf("CreateCluster returned %d", status)

	status, body, err = eksRequest(ctx, endpoint, region, creds, http.MethodGet, "/clusters", nil)
	if err != nil {
		exitf("list clusters: %v", err)
	}
	expectStatus("ListClusters", status, body, http.StatusOK)
	logf("ListClusters returned %d", status)

	status, body, err = eksRequest(ctx, endpoint, region, creds, http.MethodGet, "/clusters/"+clusterName, nil)
	if err != nil {
		exitf("describe cluster: %v", err)
	}
	expectStatus("DescribeCluster", status, body, http.StatusOK)
	logf("DescribeCluster returned %d", status)

	status, body, err = eksRequest(ctx, endpoint, region, creds, http.MethodDelete, "/clusters/"+clusterName, nil)
	if err != nil {
		exitf("delete cluster: %v", err)
	}
	expectStatus("DeleteCluster", status, body, http.StatusOK)
	logf("DeleteCluster returned %d", status)

	fmt.Println("Done.")
}

func eksRequest(
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

	requestURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "eks", region, time.Now()); err != nil {
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

func expectStatus(action string, status int, body []byte, expected int) {
	if status != expected {
		exitf("expected %s to return %d, got %d: %s", action, expected, status, strings.TrimSpace(string(body)))
	}
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
