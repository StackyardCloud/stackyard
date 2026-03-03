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

type cluster struct {
	ARN        string `json:"arn"`
	Identifier string `json:"identifier"`
	Status     string `json:"status"`
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	clusterID := normalizeClusterIdentifier(getenv("STACKYARD_CLUSTER_ID", "aurora-dsql-basic-cluster"))

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Aurora DSQL basic client using %s\n", endpoint)

	createPayload := map[string]any{
		"identifier":                clusterID,
		"clientToken":               "stackyard-dsql-basic-create",
		"deletionProtectionEnabled": false,
		"tags":                      map[string]string{"env": "dev", "example": "aurora-dsql-basic"},
	}

	status, body, err := dsqlRequest(ctx, endpoint, region, creds, http.MethodPost, "/cluster", createPayload)
	if err != nil {
		exitf("CreateCluster: %v", err)
	}
	expect2xx("CreateCluster", status, body)

	var created cluster
	_ = json.Unmarshal(body, &created)
	if created.Identifier == "" {
		created.Identifier = clusterID
	}
	logf("created cluster identifier=%s arn=%s", created.Identifier, created.ARN)

	status, body, err = dsqlRequest(ctx, endpoint, region, creds, http.MethodGet, "/cluster?maxResults=20", nil)
	if err != nil {
		exitf("ListClusters: %v", err)
	}
	expect2xx("ListClusters", status, body)
	logf("ListClusters returned %d", status)

	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodGet,
		"/cluster/"+url.PathEscape(created.Identifier),
		nil,
	)
	if err != nil {
		exitf("GetCluster: %v", err)
	}
	expect2xx("GetCluster", status, body)
	logf("GetCluster returned %d", status)

	deletePath := "/cluster/" + url.PathEscape(created.Identifier) + "?clientToken=stackyard-dsql-basic-delete"
	status, body, err = dsqlRequest(ctx, endpoint, region, creds, http.MethodDelete, deletePath, nil)
	if err != nil {
		exitf("DeleteCluster: %v", err)
	}
	expect2xx("DeleteCluster", status, body)
	logf("DeleteCluster returned %d", status)

	fmt.Println("Done.")
}

func dsqlRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload any,
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

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "dsql", region, time.Now()); err != nil {
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

func normalizeClusterIdentifier(seed string) string {
	filtered := strings.Builder{}
	for _, ch := range strings.ToLower(seed) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			filtered.WriteRune(ch)
		}
	}
	out := filtered.String()
	if out == "" {
		out = "auroradsqlcluster"
	}
	const requiredLen = 26
	for len(out) < requiredLen {
		out += "0123456789"
	}
	return out[:requiredLen]
}

func expect2xx(action string, status int, body []byte) {
	if status >= 200 && status < 300 {
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
