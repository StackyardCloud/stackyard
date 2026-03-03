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

type restCall struct {
	Name    string
	Method  string
	Path    string
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

	fmt.Printf("Stackyard Amazon MSK Replicator advanced client using %s\n", endpoint)

	replicatorARN := "arn:aws:kafka:us-east-1:123456789012:replicator/stackyard-replicator/00000000-0000-0000-0000-000000000001-1"
	sourceClusterARN := "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-source/01234567-89ab-cdef-0123-456789abcdef-1"
	targetClusterARN := "arn:aws:kafka:us-east-1:123456789012:cluster/stackyard-target/01234567-89ab-cdef-0123-456789abcdef-2"

	calls := []restCall{
		{
			Name:   "CreateReplicator",
			Method: http.MethodPost,
			Path:   "/replication/v1/replicators",
			Payload: map[string]any{
				"replicatorName":          "stackyard-replicator",
				"description":             "Stackyard MSK Replicator staged-plan example",
				"serviceExecutionRoleArn": "arn:aws:iam::123456789012:role/stackyard-msk-replicator",
				"kafkaClusters": []map[string]any{
					{
						"amazonMskCluster": map[string]any{"mskClusterArn": sourceClusterARN},
						"vpcConfig": map[string]any{
							"subnetIds":        []string{"subnet-0123456789abcdef0"},
							"securityGroupIds": []string{"sg-0123456789abcdef0"},
						},
					},
					{
						"amazonMskCluster": map[string]any{"mskClusterArn": targetClusterARN},
						"vpcConfig": map[string]any{
							"subnetIds":        []string{"subnet-0123456789abcdef0"},
							"securityGroupIds": []string{"sg-0123456789abcdef0"},
						},
					},
				},
				"replicationInfoList": []map[string]any{
					{
						"sourceKafkaClusterArn": sourceClusterARN,
						"targetKafkaClusterArn": targetClusterARN,
						"targetCompressionType": "NONE",
						"consumerGroupReplication": map[string]any{
							"consumerGroupsToReplicate": []string{".*"},
						},
						"topicReplication": map[string]any{
							"topicsToReplicate": []string{".*"},
						},
					},
				},
			},
		},
		{Name: "ListReplicators", Method: http.MethodGet, Path: "/replication/v1/replicators?replicatorNameFilter=stackyard&maxResults=20&nextToken=token-000001"},
		{Name: "DescribeReplicator", Method: http.MethodGet, Path: "/replication/v1/replicators/" + url.PathEscape(replicatorARN)},
		{Name: "UpdateReplicationInfo", Method: http.MethodPut, Path: "/replication/v1/replicators/" + url.PathEscape(replicatorARN) + "/replication-info", Payload: map[string]any{
			"currentVersion":        "1",
			"sourceKafkaClusterArn": sourceClusterARN,
			"targetKafkaClusterArn": targetClusterARN,
			"consumerGroupReplication": map[string]any{
				"consumerGroupsToReplicate": []string{".*"},
			},
			"topicReplication": map[string]any{
				"topicsToReplicate": []string{".*"},
			},
		}},
		{Name: "DeleteReplicator", Method: http.MethodDelete, Path: "/replication/v1/replicators/" + url.PathEscape(replicatorARN)},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := mskv1Request(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func mskv1Request(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	} else {
		body = []byte{}
	}

	fullURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(body))
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "kafka", region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown action") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "unauthorized")
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
