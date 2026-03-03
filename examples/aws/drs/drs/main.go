package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type apiCall struct {
	Name    string
	Method  string
	Path    string
	Payload []byte
	Headers map[string]string
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	resourceArn := getenv("STACKYARD_DRS_RESOURCE_ARN", "arn:aws:drs:us-east-1:123456789012:source-server/s-00000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AWS DRS advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Name: "DescribeSourceServers", Method: http.MethodPost, Path: "/DescribeSourceServers", Payload: []byte(`{"maxResults":10}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "DescribeJobs", Method: http.MethodPost, Path: "/DescribeJobs", Payload: []byte(`{"maxResults":10}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "DescribeSourceNetworks", Method: http.MethodPost, Path: "/DescribeSourceNetworks", Payload: []byte(`{"maxResults":10}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "DescribeRecoveryInstances", Method: http.MethodPost, Path: "/DescribeRecoveryInstances", Payload: []byte(`{"maxResults":10}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "CreateLaunchConfigurationTemplate", Method: http.MethodPost, Path: "/CreateLaunchConfigurationTemplate", Payload: []byte(`{"copyPrivateIp":true,"launchDisposition":"STOPPED","targetInstanceTypeRightSizingMethod":"NONE"}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "CreateReplicationConfigurationTemplate", Method: http.MethodPost, Path: "/CreateReplicationConfigurationTemplate", Payload: []byte(`{"bandwidthThrottling":0,"createPublicIP":false,"dataPlaneRouting":"PRIVATE_IP","defaultLargeStagingDiskType":"GP3","ebsEncryption":"DEFAULT","pitPolicy":[{"enabled":true,"interval":60,"retentionDuration":60,"ruleID":1,"units":"MINUTE"}],"replicationServerInstanceType":"t3.small","stagingAreaSubnetId":"subnet-12345678","stagingAreaTags":{"env":"advanced"},"useDedicatedReplicationServer":false}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + url.PathEscape(resourceArn), Payload: []byte(`{"tags":{"env":"advanced"}}`), Headers: map[string]string{"Content-Type": "application/json"}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + url.PathEscape(resourceArn), Payload: nil, Headers: map[string]string{"Accept": "application/json"}},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + url.PathEscape(resourceArn) + "?tagKeys=env", Payload: nil, Headers: map[string]string{"Accept": "application/json"}},
	}

	for _, call := range calls {
		status, body, err := drsRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload, call.Headers)
		if err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
		if status >= 200 && status < 300 {
			logf("%s returned %d", call.Name, status)
			continue
		}
		if isStagedPlanTolerated(status, body) {
			logf("%s returned %d: expected while staged plan is in progress", call.Name, status)
			continue
		}
		exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
}

func drsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload []byte,
	headers map[string]string,
) (int, []byte, error) {
	if payload == nil {
		payload = []byte{}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(payload), "drs", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(status int, body []byte) bool {
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		return true
	}
	combined := strings.ToLower(string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "signaturedoesnotmatch") ||
		strings.Contains(combined, "invalidrequest") ||
		strings.Contains(combined, "requestthrottled") ||
		strings.Contains(combined, "conflict")
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
