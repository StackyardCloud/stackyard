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

type policyResponse struct {
	PolicyVersion string `json:"policyVersion"`
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	accountID := getenv("AWS_ACCOUNT_ID", "123456789012")
	clusterID := normalizeClusterIdentifier(getenv("STACKYARD_CLUSTER_ID", "aurora-dsql-cluster"))

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	clusterARN := fmt.Sprintf("arn:aws:dsql:%s:%s:cluster/%s", region, accountID, clusterID)
	fmt.Printf("Stackyard Aurora DSQL advanced client using %s\n", endpoint)

	createPayload := map[string]any{
		"identifier":                clusterID,
		"clientToken":               "stackyard-dsql-create",
		"deletionProtectionEnabled": false,
		"tags":                      map[string]string{"env": "dev", "example": "aurora-dsql"},
	}
	status, body, err := dsqlRequest(ctx, endpoint, region, creds, http.MethodPost, "/cluster", createPayload)
	if err != nil {
		exitf("CreateCluster: %v", err)
	}
	expect2xx("CreateCluster", status, body)

	var created cluster
	_ = json.Unmarshal(body, &created)
	if created.Identifier != "" {
		clusterID = created.Identifier
	}
	if created.ARN != "" {
		clusterARN = created.ARN
	}
	logf("created cluster identifier=%s arn=%s", clusterID, clusterARN)

	updatePayload := map[string]any{
		"clientToken":               "stackyard-dsql-update",
		"deletionProtectionEnabled": false,
	}
	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/cluster/"+url.PathEscape(clusterID),
		updatePayload,
	)
	if err != nil {
		exitf("UpdateCluster: %v", err)
	}
	expect2xx("UpdateCluster", status, body)

	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodGet,
		"/clusters/"+url.PathEscape(clusterID)+"/vpc-endpoint-service-name",
		nil,
	)
	if err != nil {
		exitf("GetVpcEndpointServiceName: %v", err)
	}
	expect2xx("GetVpcEndpointServiceName", status, body)

	policyDocumentBytes, _ := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Sid":       "AllowRead",
				"Effect":    "Allow",
				"Principal": map[string]string{"AWS": "*"},
				"Action":    []string{"dsql:GetCluster", "dsql:ListTagsForResource"},
				"Resource":  clusterARN,
			},
		},
	})
	putPolicyPayload := map[string]any{
		"clientToken": "stackyard-dsql-policy-put",
		"policy":      string(policyDocumentBytes),
	}
	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/cluster/"+url.PathEscape(clusterID)+"/policy",
		putPolicyPayload,
	)
	if err != nil {
		exitf("PutClusterPolicy: %v", err)
	}
	expect2xx("PutClusterPolicy", status, body)
	policyVersion := extractPolicyVersion(body)

	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodGet,
		"/cluster/"+url.PathEscape(clusterID)+"/policy",
		nil,
	)
	if err != nil {
		exitf("GetClusterPolicy: %v", err)
	}
	expect2xx("GetClusterPolicy", status, body)
	if v := extractPolicyVersion(body); v != "" {
		policyVersion = v
	}

	resourcePath := "/tags/" + url.PathEscape(clusterARN)
	tagPayload := map[string]any{
		"tags": map[string]string{
			"owner": "stackyard",
			"tier":  "integration",
		},
	}
	status, body, err = dsqlRequest(ctx, endpoint, region, creds, http.MethodPost, resourcePath, tagPayload)
	if err != nil {
		exitf("TagResource: %v", err)
	}
	expect2xx("TagResource", status, body)

	status, body, err = dsqlRequest(ctx, endpoint, region, creds, http.MethodGet, resourcePath, nil)
	if err != nil {
		exitf("ListTagsForResource: %v", err)
	}
	expect2xx("ListTagsForResource", status, body)

	untagQuery := url.Values{}
	untagQuery.Add("tagKeys", "tier")
	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodDelete,
		resourcePath+"?"+untagQuery.Encode(),
		nil,
	)
	if err != nil {
		exitf("UntagResource: %v", err)
	}
	expect2xx("UntagResource", status, body)

	deletePolicyQuery := url.Values{}
	deletePolicyQuery.Set("clientToken", "stackyard-dsql-policy-delete")
	if policyVersion != "" {
		deletePolicyQuery.Set("expectedPolicyVersion", policyVersion)
	}
	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodDelete,
		"/cluster/"+url.PathEscape(clusterID)+"/policy?"+deletePolicyQuery.Encode(),
		nil,
	)
	if err != nil {
		exitf("DeleteClusterPolicy: %v", err)
	}
	expect2xx("DeleteClusterPolicy", status, body)

	deleteClusterQuery := url.Values{}
	deleteClusterQuery.Set("clientToken", "stackyard-dsql-delete")
	status, body, err = dsqlRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodDelete,
		"/cluster/"+url.PathEscape(clusterID)+"?"+deleteClusterQuery.Encode(),
		nil,
	)
	if err != nil {
		exitf("DeleteCluster: %v", err)
	}
	expect2xx("DeleteCluster", status, body)

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

func extractPolicyVersion(body []byte) string {
	var parsed policyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return parsed.PolicyVersion
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
		logf("%s returned %d", action, status)
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
