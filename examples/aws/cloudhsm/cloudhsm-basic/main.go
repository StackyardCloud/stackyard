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
	hsmType := getenv("STACKYARD_HSM_TYPE", "hsm1.medium")
	subnetID := getenv("STACKYARD_SUBNET_ID", "subnet-12345678")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard CloudHSM basic client using %s\n", endpoint)

	createStatus, createBody, err := cloudhsmRequest(ctx, endpoint, region, creds, "CreateCluster", map[string]any{
		"HsmType":   hsmType,
		"SubnetIds": []string{subnetID},
		"TagList": []map[string]string{
			{"Key": "env", "Value": "dev"},
		},
	})
	if err != nil {
		exitf("create cluster: %v", err)
	}
	if err := expectHTTPStatus("CreateCluster", createStatus, http.StatusOK, createBody); err != nil {
		exitf("create cluster: %v", err)
	}
	clusterID, err := extractString(createBody, "Cluster", "ClusterId")
	if err != nil {
		exitf("parse create cluster response: %v", err)
	}
	logf("created cluster: %s", clusterID)

	describeStatus, describeBody, err := cloudhsmRequest(ctx, endpoint, region, creds, "DescribeClusters", map[string]any{"MaxResults": 10})
	if err != nil {
		exitf("describe clusters: %v", err)
	}
	if err := expectHTTPStatus("DescribeClusters", describeStatus, http.StatusOK, describeBody); err != nil {
		exitf("describe clusters: %v", err)
	}
	logf("DescribeClusters succeeded (%d)", describeStatus)

	deleteStatus, deleteBody, err := cloudhsmRequest(ctx, endpoint, region, creds, "DeleteCluster", map[string]any{"ClusterId": clusterID})
	if err != nil {
		exitf("delete cluster: %v", err)
	}
	if err := expectHTTPStatus("DeleteCluster", deleteStatus, http.StatusOK, deleteBody); err != nil {
		exitf("delete cluster: %v", err)
	}
	logf("DeleteCluster succeeded (%d)", deleteStatus)

	fmt.Println("Done.")
}

func cloudhsmRequest(
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

	requestURL := strings.TrimRight(endpoint, "/") + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "BaldrApiService."+action)

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "cloudhsm", region, time.Now()); err != nil {
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

func expectHTTPStatus(action string, status, expected int, body []byte) error {
	if status != expected {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, expected, status, strings.TrimSpace(string(body)))
	}
	return nil
}

func extractString(body []byte, keys ...string) (string, error) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	current := payload
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("key path %v: non-object segment", keys)
		}
		next, ok := object[key]
		if !ok {
			return "", fmt.Errorf("missing key %s", key)
		}
		current = next
	}
	value, ok := current.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("key path %v is empty or non-string", keys)
	}
	return value, nil
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
